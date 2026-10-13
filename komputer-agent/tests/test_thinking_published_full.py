"""Thinking is published in full, like assistant text.

It used to be cut to 500 characters at publish time while text events were
not cut at all; the cap protected nothing (the history and stream bounds are by
event count, not size) and just hid the agent's reasoning. The UI collapses long
thinking instead.
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest

import agent
import state
from claude_agent_sdk import AssistantMessage, ResultMessage, ThinkingBlock


class _FakeClient:
    def __init__(self, messages):
        self._messages = messages

    async def __aenter__(self):
        return self

    async def __aexit__(self, *exc):
        return False

    async def query(self, *_args, **_kwargs):
        return None

    async def receive_response(self):
        for m in self._messages:
            yield m


@pytest.fixture(autouse=True)
def _reset_state():
    state.interrupted = False
    state.steered = False
    state.steer_queue = None
    yield
    state.interrupted = False


@pytest.mark.asyncio
async def test_long_thinking_is_published_untruncated(monkeypatch):
    long_thought = "considering the options carefully. " * 100  # ~3500 chars
    messages = [
        AssistantMessage(content=[ThinkingBlock(thinking=long_thought, signature="sig")], model="claude-opus-5-5"),
        ResultMessage(subtype="success", duration_ms=10, duration_api_ms=10, is_error=False,
                      num_turns=1, session_id="s", stop_reason=None, total_cost_usd=0.01, usage={}),
    ]
    published = []

    class FakePublisher:
        def publish(self, event_type, payload):
            published.append((event_type, payload))

    monkeypatch.setattr(agent, "ClaudeSDKClient", lambda options=None: _FakeClient(messages))
    monkeypatch.setattr(agent, "_load_session_id", lambda: None)
    monkeypatch.setattr(agent, "_save_session_id", lambda session_id: None)
    monkeypatch.setattr(agent, "_fetch_context_window", lambda model: None)

    async def no_steer(queue, timeout):
        return False

    monkeypatch.setattr(agent, "_wait_for_steer", no_steer)

    await agent.run_agent("think hard", "claude-opus-5-5", FakePublisher())

    thinking = [p for t, p in published if t == "thinking"]
    assert len(thinking) == 1
    assert thinking[0]["content"] == long_thought
