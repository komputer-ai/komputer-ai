"""S3 session store: save the agent's Claude session to S3, restore it when missing.

The operator turns this on by injecting KOMPUTER_S3_* env vars (only when
KomputerConfig.spec.s3 is configured). One gzip tarball per agent holds
`.claude/**` (minus the skills dir, regenerated from env on every start) and
`.komputer-session`, so the resume pointer and the transcript it points to always
travel together.

Neither function ever raises: S3 must never block or fail a task.
"""
import logging
import os
import shutil
import tarfile
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path

logger = logging.getLogger(__name__)

WORKSPACE = Path("/workspace")
# Backoff before each save attempt (the first is immediate).
_RETRY_DELAYS = (0, 1, 2)

# Result of this pod's startup restore, reported on the next task_started.
last_restore: dict | None = None


def enabled() -> bool:
    return bool(os.environ.get("KOMPUTER_S3_BUCKET"))


def _bucket_and_key() -> tuple[str, str]:
    return os.environ["KOMPUTER_S3_BUCKET"], os.environ["KOMPUTER_S3_KEY"]


def _client():
    import boto3
    from botocore.config import Config

    force_path_style = os.environ.get("KOMPUTER_S3_FORCE_PATH_STYLE", "").lower() == "true"
    # The store's own keys are KOMPUTER_S3_*, apart from AWS_*, so they never
    # clash with the agent's other AWS use (Bedrock). Without them, boto3's
    # default credential chain applies (IAM role, or AWS_* you set yourself).
    access_key = os.environ.get("KOMPUTER_S3_ACCESS_KEY_ID") or None
    secret_key = os.environ.get("KOMPUTER_S3_SECRET_ACCESS_KEY") or None
    return boto3.client(
        "s3",
        endpoint_url=os.environ.get("KOMPUTER_S3_ENDPOINT") or None,
        region_name=os.environ.get("KOMPUTER_S3_REGION") or None,
        aws_access_key_id=access_key if access_key and secret_key else None,
        aws_secret_access_key=secret_key if access_key and secret_key else None,
        config=Config(
            s3={"addressing_style": "path"} if force_path_style else {},
            connect_timeout=3,
            read_timeout=5,
            retries={"max_attempts": 2},
        ),
    )


def _skip_skills(info: tarfile.TarInfo):
    if info.name == ".claude/skills" or info.name.startswith(".claude/skills/"):
        return None
    return info


def _restore_filter(info: tarfile.TarInfo, dest: str):
    # Drop links (Claude Code leaves some under .claude, e.g. debug/latest ->
    # an absolute path, which the "data" filter would reject outright), then
    # apply "data": no ownership/permission bits from the archive, no unsafe paths.
    if info.issym() or info.islnk():
        return None
    return tarfile.data_filter(info, dest)


class _KeepOpen:
    """Hands a file to boto3 without letting it close it: s3transfer closes the
    file object when an upload fails, which would break every retry."""

    def __init__(self, f):
        self._f = f

    def read(self, *args):
        return self._f.read(*args)

    def seek(self, *args):
        return self._f.seek(*args)

    def tell(self):
        return self._f.tell()

    def close(self):
        pass


def _now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def save(workspace: Path = WORKSPACE, client=None) -> dict:
    """Upload the session tarball. Returns {"ok", "at", "bytes", "error"}."""
    at = _now()
    try:
        bucket, key = _bucket_and_key()
        with tempfile.TemporaryFile() as f:
            with tarfile.open(fileobj=f, mode="w:gz") as tar:
                for name in (".claude", ".komputer-session"):
                    if (workspace / name).exists():
                        tar.add(workspace / name, arcname=name, filter=_skip_skills)
            size = f.tell()
            s3 = client or _client()
            last_err = None
            for delay in _RETRY_DELAYS:
                if delay:
                    time.sleep(delay)
                try:
                    f.seek(0)
                    s3.upload_fileobj(_KeepOpen(f), bucket, key)
                    logger.info("session saved to S3", extra={"bytes": size})
                    return {"ok": True, "at": at, "bytes": size, "error": None}
                except Exception as e:  # noqa: BLE001 — any failure is reported, never raised
                    last_err = e
            raise last_err
    except Exception as e:  # noqa: BLE001
        logger.warning("session save to S3 failed", extra={"error": str(e)})
        return {"ok": False, "at": at, "bytes": 0, "error": str(e)}


def restore_if_needed(workspace: Path = WORKSPACE, client=None) -> dict:
    """Restore the session from S3 if this workspace has none. Returns {"restored", "error"}."""
    global last_restore
    result = {"restored": False, "error": None}
    try:
        if (workspace / ".komputer-session").exists():
            return result
        bucket, key = _bucket_and_key()
        s3 = client or _client()
        try:
            s3.head_object(Bucket=bucket, Key=key)
        except Exception as e:  # noqa: BLE001
            code = getattr(e, "response", {}).get("Error", {}).get("Code")
            if code in ("404", "NoSuchKey", "NotFound"):
                return result  # brand-new agent: nothing saved yet
            raise
        # Extract into a scratch dir on the same filesystem, then move into
        # place: a failed or truncated download never leaves half a session.
        staging = Path(tempfile.mkdtemp(prefix=".komputer-restore-", dir=workspace))
        try:
            with tempfile.TemporaryFile() as f:
                s3.download_fileobj(bucket, key, f)
                f.seek(0)
                with tarfile.open(fileobj=f, mode="r:gz") as tar:
                    tar.extractall(staging, filter=_restore_filter)
            for name in (".claude", ".komputer-session"):
                if (staging / name).exists():
                    shutil.move(str(staging / name), str(workspace / name))
        finally:
            shutil.rmtree(staging, ignore_errors=True)
        result["restored"] = True
        logger.info("session restored from S3")
    except Exception as e:  # noqa: BLE001
        logger.warning("session restore from S3 failed; starting fresh", extra={"error": str(e)})
        result = {"restored": False, "error": str(e)}
    finally:
        last_restore = result
    return result


def task_end_fields() -> dict:
    """The session_store block for a task-end event: saves the session first.
    Empty when the store isn't configured. Every path that ends a task uses it,
    so "the task ended" always comes with a save attempt."""
    return {"session_store": save()} if enabled() else {}
