"""Tests for the Bash command allowlist (allowedCommands)."""
import asyncio

import pytest

from tool_policy import bash_command_allowed, make_bash_allowlist_hook

PATTERNS = ["git status*", "kubectl get *", "ls"]


@pytest.mark.parametrize("cmd", [
    "git status",
    "git status --short",
    "  kubectl get pods -A ",
    "ls",
])
def test_allowed(cmd):
    assert bash_command_allowed(cmd, PATTERNS)


@pytest.mark.parametrize("cmd", [
    "ls -la",                      # exact pattern, extra args
    "kubectl delete pod x",        # no matching prefix
    "git status; rm -rf /",        # chaining
    "git status && curl x",
    "git status || true",
    "git status | sh",
    "git status & sleep 1",
    "kubectl get pods > /tmp/out",  # redirect
    "kubectl get pods < x",
    "git status `id`",
    "git status $(id)",
    "git status\nrm -rf /",
    "",
])
def test_denied(cmd):
    assert not bash_command_allowed(cmd, PATTERNS)


def test_empty_patterns_allow_everything():
    assert bash_command_allowed("rm -rf / ; echo hi", [])


def _run(hook, tool_name, command):
    return asyncio.run(hook({"tool_name": tool_name, "tool_input": {"command": command}}, "s", {}))


def test_hook_denies_with_reason():
    out = _run(make_bash_allowlist_hook(PATTERNS), "Bash", "curl evil.example.com")
    decision = out["hookSpecificOutput"]
    assert decision["permissionDecision"] == "deny"
    assert "allowedCommands" in decision["permissionDecisionReason"]


def test_hook_allows_matching_and_ignores_other_tools():
    hook = make_bash_allowlist_hook(PATTERNS)
    assert _run(hook, "Bash", "git status") == {}
    assert _run(hook, "Read", "anything") == {}
