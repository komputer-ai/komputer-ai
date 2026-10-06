"""Agent runtime config — loaded from env vars on startup, updatable via /config endpoint.

The config file lives at /tmp/komputer/agent-config.json (outside workspace PVC).
It resets on pod restart; env vars are the source of truth for fresh pods.
"""

import json
import os
from pathlib import Path
from threading import Lock

CONFIG_PATH = Path("/tmp/komputer/agent-config.json")
_lock = Lock()


def _defaults_from_env() -> dict:
    """Build initial config from environment variables."""
    return {
        "model": os.getenv("KOMPUTER_MODEL", "claude-opus-5-5"),
        "lifecycle": os.getenv("KOMPUTER_LIFECYCLE", ""),
        "role": os.getenv("KOMPUTER_ROLE", "manager"),
        "instructions": os.getenv("KOMPUTER_INSTRUCTIONS", ""),
        "templateRef": os.getenv("KOMPUTER_TEMPLATE_REF", "default"),
    }


def init():
    """Initialize config file from env vars. Called once on agent startup."""
    CONFIG_PATH.parent.mkdir(parents=True, exist_ok=True)
    config = _defaults_from_env()
    with _lock:
        CONFIG_PATH.write_text(json.dumps(config, indent=2))


def load() -> dict:
    """Load current config from disk."""
    with _lock:
        try:
            return json.loads(CONFIG_PATH.read_text())
        except (FileNotFoundError, json.JSONDecodeError):
            # Fallback to env vars if file is missing/corrupt
            return _defaults_from_env()


def apply(updates: dict):
    """Merge partial updates into the config file and save."""
    with _lock:
        try:
            current = json.loads(CONFIG_PATH.read_text())
        except (FileNotFoundError, json.JSONDecodeError):
            current = _defaults_from_env()
        current.update({k: v for k, v in updates.items() if v is not None})
        CONFIG_PATH.write_text(json.dumps(current, indent=2))
