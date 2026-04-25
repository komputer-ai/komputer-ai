"""Structured logging for the agent. JSON when not on a TTY, colored console
when running interactively. Override with LOG_FORMAT=json|text."""

import logging
import os
import sys
from datetime import datetime, timezone

from pythonjsonlogger import jsonlogger


def _iso_timestamp(record: logging.LogRecord) -> str:
    """ISO 8601 with millisecond precision and UTC offset, matching the API's
    zap output so logs from both components correlate cleanly."""
    dt = datetime.fromtimestamp(record.created, tz=timezone.utc)
    millis = f"{dt.microsecond // 1000:03d}"
    return f"{dt.strftime('%Y-%m-%dT%H:%M:%S')}.{millis}Z"


_COMPONENT = "komputer-agent"


class _ConsoleFormatter(logging.Formatter):
    """Human-readable formatter with ANSI colors per level."""

    COLORS = {
        "DEBUG": "\033[36m",   # cyan
        "INFO": "\033[32m",    # green
        "WARNING": "\033[33m", # yellow
        "ERROR": "\033[31m",   # red
        "CRITICAL": "\033[35m",# magenta
    }
    RESET = "\033[0m"

    def format(self, record: logging.LogRecord) -> str:
        color = self.COLORS.get(record.levelname, "")
        ts = _iso_timestamp(record)
        extras = {
            k: v for k, v in record.__dict__.items()
            if k not in (
                "args", "asctime", "created", "exc_info", "exc_text", "filename",
                "funcName", "levelname", "levelno", "lineno", "module", "msecs",
                "message", "msg", "name", "pathname", "process", "processName",
                "relativeCreated", "stack_info", "thread", "threadName",
                "taskName",
            )
        }
        extras_str = " ".join(f"{k}={v}" for k, v in extras.items())
        msg = record.getMessage()
        line = f"{ts} {color}{record.levelname:5}{self.RESET} {msg}" + (f"  {extras_str}" if extras_str else "")
        if record.exc_info:
            line += "\n" + self.formatException(record.exc_info)
        return line


class _JsonFormatter(jsonlogger.JsonFormatter):
    """JSON formatter that always includes timestamp + component."""

    def add_fields(self, log_record: dict, record: logging.LogRecord, message_dict: dict) -> None:
        super().add_fields(log_record, record, message_dict)
        log_record["timestamp"] = _iso_timestamp(record)
        log_record["level"] = record.levelname.lower()
        log_record["component"] = _COMPONENT
        if "message" not in log_record and record.getMessage():
            log_record["message"] = record.getMessage()


def init_logger() -> logging.Logger:
    """Configure the root logger. Idempotent — safe to call multiple times."""
    level_name = os.environ.get("LOG_LEVEL", "info").upper()
    level = getattr(logging, level_name, logging.INFO)

    fmt = os.environ.get("LOG_FORMAT", "").lower()
    if fmt == "json":
        use_json = True
    elif fmt == "text":
        use_json = False
    else:
        use_json = not sys.stdout.isatty()

    handler = logging.StreamHandler(sys.stdout)
    if use_json:
        handler.setFormatter(_JsonFormatter())
    else:
        handler.setFormatter(_ConsoleFormatter())

    root = logging.getLogger()
    # Clear any existing handlers (uvicorn / claude-sdk install their own)
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(level)

    # Quiet noisy libraries
    logging.getLogger("httpx").setLevel(logging.WARNING)
    logging.getLogger("httpcore").setLevel(logging.WARNING)
    logging.getLogger("uvicorn.access").setLevel(logging.WARNING)

    return root
