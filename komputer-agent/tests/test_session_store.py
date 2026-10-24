"""S3 session store: save .claude at task end, restore it only when missing."""
import io
import os
import tarfile
import time
from pathlib import Path

import pytest
from botocore.exceptions import ClientError, EndpointConnectionError

import session_store


class FakeS3:
    """Records uploads in memory; can be told to fail."""

    def __init__(self, fail_times=0, download_exc=None, head_exc=None):
        self.objects = {}
        self.upload_calls = []
        self.put_object_calls = 0
        self.download_calls = 0
        self.fail_times = fail_times
        self.download_exc = download_exc
        self.head_exc = head_exc

    def upload_fileobj(self, fileobj, bucket, key):
        self.upload_calls.append((fileobj, bucket, key))
        if self.fail_times > 0:
            self.fail_times -= 1
            raise RuntimeError("AccessDenied: no write")
        self.objects[(bucket, key)] = fileobj.read()

    def put_object(self, **kwargs):
        self.put_object_calls += 1

    def head_object(self, Bucket, Key):
        if self.head_exc:
            raise self.head_exc
        if (Bucket, Key) not in self.objects:
            raise ClientError({"Error": {"Code": "404", "Message": "Not Found"}}, "HeadObject")
        return {}

    def download_fileobj(self, bucket, key, fileobj):
        self.download_calls += 1
        if self.download_exc:
            raise self.download_exc
        fileobj.write(self.objects[(bucket, key)])


@pytest.fixture
def s3_env(monkeypatch):
    monkeypatch.setenv("KOMPUTER_S3_BUCKET", "sessions")
    monkeypatch.setenv("KOMPUTER_S3_KEY", "komputer/team-a/a1/u-1/session.tar.gz")
    monkeypatch.setattr(session_store, "_RETRY_DELAYS", (0, 0, 0))


@pytest.fixture
def workspace(tmp_path):
    (tmp_path / ".claude" / "projects" / "-workspace").mkdir(parents=True)
    (tmp_path / ".claude" / "projects" / "-workspace" / "s1.jsonl").write_text('{"type":"user"}\n')
    (tmp_path / ".claude" / "skills" / "x").mkdir(parents=True)
    (tmp_path / ".claude" / "skills" / "x" / "SKILL.md").write_text("skill")
    (tmp_path / ".komputer-session").write_text("s1")
    (tmp_path / "repo.txt").write_text("workspace file, not part of the session")
    return tmp_path


def _members(blob: bytes) -> set:
    with tarfile.open(fileobj=io.BytesIO(blob), mode="r:gz") as tar:
        return set(tar.getnames())


def test_disabled_without_bucket(monkeypatch):
    monkeypatch.delenv("KOMPUTER_S3_BUCKET", raising=False)
    assert session_store.enabled() is False


def test_save_tarball_contents(s3_env, workspace):
    s3 = FakeS3()
    result = session_store.save(workspace, client=s3)

    assert result["ok"] is True
    assert result["error"] is None
    assert result["bytes"] > 0
    blob = s3.objects[("sessions", "komputer/team-a/a1/u-1/session.tar.gz")]
    names = _members(blob)
    assert ".claude/projects/-workspace/s1.jsonl" in names
    assert ".komputer-session" in names
    assert not any(n.startswith(".claude/skills") for n in names)
    assert "repo.txt" not in names


def test_save_streams_from_tempfile(s3_env, workspace):
    s3 = FakeS3()
    session_store.save(workspace, client=s3)

    assert s3.put_object_calls == 0
    assert len(s3.upload_calls) == 1
    fileobj = s3.upload_calls[0][0]
    assert hasattr(fileobj, "read") and not isinstance(fileobj, (bytes, bytearray))


def test_save_failure_returns_error_after_retries(s3_env, workspace):
    s3 = FakeS3(fail_times=5)
    result = session_store.save(workspace, client=s3)

    assert result["ok"] is False
    assert "AccessDenied" in result["error"]
    assert len(s3.upload_calls) == 3


def test_save_never_raises_when_client_cannot_be_built(s3_env, workspace, monkeypatch):
    def boom():
        raise RuntimeError("no credentials")

    monkeypatch.setattr(session_store, "_client", boom)
    result = session_store.save(workspace)
    assert result["ok"] is False
    assert "no credentials" in result["error"]


def test_restore_skips_when_local_session_exists(s3_env, workspace):
    s3 = FakeS3()
    result = session_store.restore_if_needed(workspace, client=s3)

    assert result == {"restored": False, "error": None}
    assert s3.download_calls == 0


def test_restore_extracts_when_missing(s3_env, workspace, tmp_path_factory):
    s3 = FakeS3()
    assert session_store.save(workspace, client=s3)["ok"] is True

    fresh = tmp_path_factory.mktemp("fresh")
    result = session_store.restore_if_needed(fresh, client=s3)

    assert result == {"restored": True, "error": None}
    assert (fresh / ".komputer-session").read_text() == "s1"
    assert (fresh / ".claude" / "projects" / "-workspace" / "s1.jsonl").exists()
    assert session_store.last_restore == result


def test_restore_missing_object_starts_fresh(s3_env, tmp_path):
    result = session_store.restore_if_needed(tmp_path, client=FakeS3())
    assert result == {"restored": False, "error": None}
    assert not (tmp_path / ".komputer-session").exists()


def test_restore_unreachable_starts_fresh_quickly(s3_env, tmp_path):
    s3 = FakeS3(head_exc=EndpointConnectionError(endpoint_url="http://s3.invalid"))
    start = time.monotonic()
    result = session_store.restore_if_needed(tmp_path, client=s3)

    assert time.monotonic() - start < 10
    assert result["restored"] is False
    assert result["error"]
    assert list(tmp_path.iterdir()) == []


def test_restore_files_writable_by_agent(s3_env, workspace, tmp_path_factory):
    s3 = FakeS3()
    session_store.save(workspace, client=s3)
    fresh = tmp_path_factory.mktemp("fresh")
    session_store.restore_if_needed(fresh, client=s3)

    for p in [fresh / ".komputer-session", fresh / ".claude" / "projects" / "-workspace" / "s1.jsonl"]:
        assert os.access(p, os.W_OK), p
        assert p.stat().st_uid == os.getuid()


def test_client_uses_endpoint_and_path_style(monkeypatch):
    monkeypatch.setenv("KOMPUTER_S3_ENDPOINT", "http://minio:9000")
    monkeypatch.setenv("KOMPUTER_S3_REGION", "us-east-1")
    monkeypatch.setenv("KOMPUTER_S3_FORCE_PATH_STYLE", "true")
    monkeypatch.setenv("AWS_ACCESS_KEY_ID", "ak")
    monkeypatch.setenv("AWS_SECRET_ACCESS_KEY", "sk")
    c = session_store._client()
    assert c.meta.endpoint_url == "http://minio:9000"
    assert c.meta.config.s3["addressing_style"] == "path"


def test_client_uses_store_credentials_over_generic(monkeypatch):
    """KOMPUTER_S3_* keys win, so the store never borrows (or clobbers) Bedrock's AWS_* keys."""
    monkeypatch.setenv("KOMPUTER_S3_ACCESS_KEY_ID", "store-ak")
    monkeypatch.setenv("KOMPUTER_S3_SECRET_ACCESS_KEY", "store-sk")
    monkeypatch.setenv("AWS_ACCESS_KEY_ID", "bedrock-ak")
    monkeypatch.setenv("AWS_SECRET_ACCESS_KEY", "bedrock-sk")
    c = session_store._client()
    assert c._request_signer._credentials.access_key == "store-ak"


class ClosingFakeS3(FakeS3):
    """Like s3transfer: a failed upload closes the file object it was given."""

    def upload_fileobj(self, fileobj, bucket, key):
        try:
            super().upload_fileobj(fileobj, bucket, key)
        except Exception:
            fileobj.close()
            raise


def test_save_retries_survive_a_closed_file(s3_env, workspace):
    s3 = ClosingFakeS3(fail_times=1)
    result = session_store.save(workspace, client=s3)
    assert result["ok"] is True, result
    assert len(s3.upload_calls) == 2


def test_save_reports_the_real_error(s3_env, workspace):
    s3 = ClosingFakeS3(fail_times=5)
    result = session_store.save(workspace, client=s3)
    assert result["ok"] is False
    assert "AccessDenied" in result["error"], result["error"]


def _tarball_with(members):
    """members: list of (name, kind, payload) — kind "file" or "symlink"."""
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w:gz") as tar:
        for name, kind, payload in members:
            info = tarfile.TarInfo(name)
            if kind == "symlink":
                info.type = tarfile.SYMTYPE
                info.linkname = payload
                tar.addfile(info)
            else:
                data = payload.encode()
                info.size = len(data)
                tar.addfile(info, io.BytesIO(data))
    return buf.getvalue()


def test_restore_drops_links_instead_of_failing(s3_env, tmp_path):
    """Claude Code can leave symlinks under .claude (e.g. debug/latest -> an absolute path)."""
    s3 = FakeS3()
    s3.objects[("sessions", os.environ["KOMPUTER_S3_KEY"])] = _tarball_with([
        (".komputer-session", "file", "s1"),
        (".claude/projects/-workspace/s1.jsonl", "file", "{}\n"),
        (".claude/debug/latest", "symlink", "/home/komputer/.claude/debug/x.txt"),
    ])
    result = session_store.restore_if_needed(tmp_path, client=s3)
    assert result == {"restored": True, "error": None}
    assert (tmp_path / ".komputer-session").read_text() == "s1"
    assert not (tmp_path / ".claude" / "debug" / "latest").exists()


def test_restore_failure_leaves_no_partial_session(s3_env, tmp_path):
    s3 = FakeS3()
    blob = _tarball_with([
        (".claude/projects/-workspace/s1.jsonl", "file", "{}\n"),
        (".komputer-session", "file", "s1"),
    ])
    s3.objects[("sessions", os.environ["KOMPUTER_S3_KEY"])] = blob[: len(blob) // 2]  # truncated download
    result = session_store.restore_if_needed(tmp_path, client=s3)
    assert result["restored"] is False and result["error"]
    assert not (tmp_path / ".komputer-session").exists()
    assert not (tmp_path / ".claude").exists()


def test_save_with_real_boto3_reports_the_real_error(s3_env, workspace, monkeypatch):
    """Real s3transfer closes the file on failure; the reported error must be the cause, not 'seek of closed file'."""
    monkeypatch.setenv("KOMPUTER_S3_ENDPOINT", "http://127.0.0.1:1")
    monkeypatch.setenv("KOMPUTER_S3_REGION", "us-east-1")
    monkeypatch.setenv("KOMPUTER_S3_ACCESS_KEY_ID", "ak")
    monkeypatch.setenv("KOMPUTER_S3_SECRET_ACCESS_KEY", "sk")
    result = session_store.save(workspace)
    assert result["ok"] is False
    assert "closed file" not in result["error"], result["error"]
    assert "127.0.0.1:1" in result["error"] or "Connect" in result["error"], result["error"]
