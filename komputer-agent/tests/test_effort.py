"""Tests for the reasoning-effort override (issue #185).

THE CRITICAL INVARIANT: today the agent never sets `effort`, so the bundled
Claude CLI uses its own default ("auto") effort. That must not change —
when effort is unset/empty, ClaudeAgentOptions.effort must stay None (the
SDK then never emits --effort to the CLI). Only an explicit non-empty value
should be passed through.
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest

import agent
import state
from claude_agent_sdk import ResultMessage


class _FakeClient:
    """Stands in for ClaudeSDKClient: captures the options it was built with,
    no-op query, yields one ResultMessage so run_agent completes normally."""

    captured_options = None

    def __init__(self, options=None):
        _FakeClient.captured_options = options
        self._result = ResultMessage(
            subtype="success",
            duration_ms=100,
            duration_api_ms=90,
            is_error=False,
            num_turns=1,
            session_id="sess-effort-test",
            stop_reason=None,
            total_cost_usd=0.001,
            usage={"input_tokens": 1, "output_tokens": 1},
        )

    async def __aenter__(self):
        return self

    async def __aexit__(self, *exc):
        return False

    async def query(self, *_args, **_kwargs):
        return None

    async def receive_response(self):
        yield self._result


@pytest.fixture(autouse=True)
def _reset_state(monkeypatch):
    state.interrupted = False
    state.steered = False
    state.steer_queue = None
    _FakeClient.captured_options = None

    monkeypatch.setattr(agent, "ClaudeSDKClient", _FakeClient)
    monkeypatch.setattr(agent, "_load_session_id", lambda: None)
    monkeypatch.setattr(agent, "_save_session_id", lambda session_id: None)
    monkeypatch.setattr(agent, "_fetch_context_window", lambda model: None)

    async def fake_wait_for_steer(queue, timeout):
        return False

    monkeypatch.setattr(agent, "_wait_for_steer", fake_wait_for_steer)
    yield
    state.interrupted = False
    state.steered = False
    state.steer_queue = None


class _FakePublisher:
    def __init__(self):
        self.published = []

    def publish(self, event_type, payload):
        self.published.append((event_type, payload))


@pytest.mark.asyncio
async def test_effort_unset_leaves_claude_agent_options_effort_none():
    """The critical invariant: no effort kwarg means ClaudeAgentOptions.effort
    stays None, so the SDK never passes --effort to the CLI (CLI default "auto")."""
    publisher = _FakePublisher()
    await agent.run_agent("do the thing", "claude-sonnet-4-6", publisher)

    assert _FakeClient.captured_options is not None
    assert _FakeClient.captured_options.effort is None


@pytest.mark.asyncio
async def test_effort_empty_string_also_leaves_effort_none():
    """An empty-string effort (the API/CR's "unset" representation) must behave
    identically to omitting the parameter entirely — never passed through."""
    publisher = _FakePublisher()
    await agent.run_agent("do the thing", "claude-sonnet-4-6", publisher, effort="")

    assert _FakeClient.captured_options.effort is None


@pytest.mark.asyncio
async def test_effort_set_is_passed_through_to_claude_agent_options():
    publisher = _FakePublisher()
    await agent.run_agent("do the thing", "claude-sonnet-4-6", publisher, effort="high")

    assert _FakeClient.captured_options.effort == "high"


@pytest.mark.parametrize("effort_value", ["low", "medium", "high", "xhigh", "max"])
@pytest.mark.asyncio
async def test_every_allowed_effort_value_is_passed_through(effort_value):
    publisher = _FakePublisher()
    await agent.run_agent("do the thing", "claude-sonnet-4-6", publisher, effort=effort_value)

    assert _FakeClient.captured_options.effort == effort_value
