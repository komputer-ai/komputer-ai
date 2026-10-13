"""A cancelled (interrupted) run must publish task_cancelled WITH the cost/usage
fields the ResultMessage carries, not just a bare reason.

Before this fix, the interrupted branch in run_agent() published only
{"reason": "Cancelled by user"}, even though a ResultMessage (with
total_cost_usd, usage, duration_ms, num_turns, session_id) was already in
hand — so the API worker had nothing to read and cancelled tasks never
recorded spend in status.lastTaskCostUSD / status.totalCostUSD.
"""
import asyncio
import os
import sys
from unittest.mock import patch

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest

import agent
import state
from claude_agent_sdk import ResultMessage


class _FakeClient:
    """Stands in for ClaudeSDKClient: no-op query, yields one ResultMessage."""

    def __init__(self, result):
        self._result = result

    async def __aenter__(self):
        return self

    async def __aexit__(self, *exc):
        return False

    async def query(self, *_args, **_kwargs):
        return None

    async def receive_response(self):
        yield self._result


@pytest.fixture(autouse=True)
def _reset_state():
    state.interrupted = False
    state.steered = False
    state.steer_queue = None
    yield
    state.interrupted = False
    state.steered = False
    state.steer_queue = None


@pytest.mark.asyncio
async def test_interrupted_run_publishes_cost_on_task_cancelled(monkeypatch):
    result = ResultMessage(
        subtype="success",
        duration_ms=4200,
        duration_api_ms=4000,
        is_error=False,
        num_turns=3,
        session_id="sess-123",
        stop_reason=None,
        total_cost_usd=0.0456,
        usage={"input_tokens": 10, "output_tokens": 20},
    )

    published = []

    class FakePublisher:
        def publish(self, event_type, payload):
            published.append((event_type, payload))

    # Mark the run as interrupted — simulates a user cancel or an operator
    # taskTimeout cancel landing before the ResultMessage is processed.
    def fake_client_factory(options=None):
        state.interrupted = True
        return _FakeClient(result)

    monkeypatch.setattr(agent, "ClaudeSDKClient", fake_client_factory)
    monkeypatch.setattr(agent, "_load_session_id", lambda: None)
    monkeypatch.setattr(agent, "_save_session_id", lambda session_id: None)
    monkeypatch.setattr(agent, "_fetch_context_window", lambda model: None)

    # No steer arrives — fall straight through to the terminal publish.
    async def fake_wait_for_steer(queue, timeout):
        return False

    monkeypatch.setattr(agent, "_wait_for_steer", fake_wait_for_steer)

    publisher = FakePublisher()
    await agent.run_agent("do the thing", "claude-sonnet-4-6", publisher)

    cancelled_events = [p for t, p in published if t == "task_cancelled"]
    assert len(cancelled_events) == 1
    payload = cancelled_events[0]

    assert payload["reason"] == "Cancelled by user"
    assert payload["cost_usd"] == 0.0456
    assert payload["duration_ms"] == 4200
    assert payload["turns"] == 3
    assert payload["session_id"] == "sess-123"
    assert payload["usage"] == {"input_tokens": 10, "output_tokens": 20}

    # The completed-only event must not also fire.
    assert not [p for t, p in published if t == "task_completed"]
