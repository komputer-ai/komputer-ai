"""run_agent saves the session BEFORE publishing the task-end event.

The operator archives (deleting the PVC) or sleeps an agent the moment it sees
Complete, so the save has to land first — "Complete" must imply "saved".
"""
import pytest

import agent
import session_store
import state
from claude_agent_sdk import ResultMessage


class _FakeClient:
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


def _result():
    return ResultMessage(
        subtype="success", duration_ms=10, duration_api_ms=10, is_error=False,
        num_turns=1, session_id="sess-1", stop_reason=None, total_cost_usd=0.01,
        usage={"input_tokens": 1, "output_tokens": 1},
    )


@pytest.fixture
def run(monkeypatch):
    calls = []

    class FakePublisher:
        def publish(self, event_type, payload):
            calls.append(("publish", event_type, payload))

    saved = {"ok": True, "at": "2026-10-24T00:00:00Z", "bytes": 42, "error": None}

    def fake_save():
        calls.append(("save",))
        return saved

    monkeypatch.setattr(agent, "ClaudeSDKClient", lambda options=None: _FakeClient(_result()))
    monkeypatch.setattr(agent, "_load_session_id", lambda: None)
    monkeypatch.setattr(agent, "_save_session_id", lambda session_id: None)
    monkeypatch.setattr(agent, "_fetch_context_window", lambda model: None)

    async def no_steer(queue, timeout):
        return False

    monkeypatch.setattr(agent, "_wait_for_steer", no_steer)
    monkeypatch.setattr(session_store, "save", fake_save)
    return calls, saved, FakePublisher()


@pytest.mark.asyncio
async def test_run_agent_saves_before_publishing_task_completed(run, monkeypatch):
    calls, saved, publisher = run
    monkeypatch.setattr(session_store, "enabled", lambda: True)

    await agent.run_agent("do it", "claude-sonnet-4-6", publisher)

    order = [c[0] if c[0] == "save" else c[1] for c in calls]
    assert order.index("save") < order.index("task_completed")
    completed = [c[2] for c in calls if c[0] == "publish" and c[1] == "task_completed"][0]
    assert completed["session_store"] == saved


@pytest.mark.asyncio
async def test_run_agent_cancelled_also_saves_first(run, monkeypatch):
    calls, saved, publisher = run
    monkeypatch.setattr(session_store, "enabled", lambda: True)
    monkeypatch.setattr(agent, "ClaudeSDKClient", lambda options=None: (setattr(state, "interrupted", True), _FakeClient(_result()))[1])

    await agent.run_agent("do it", "claude-sonnet-4-6", publisher)

    order = [c[0] if c[0] == "save" else c[1] for c in calls]
    assert order.index("save") < order.index("task_cancelled")
    cancelled = [c[2] for c in calls if c[0] == "publish" and c[1] == "task_cancelled"][0]
    assert cancelled["session_store"] == saved


@pytest.mark.asyncio
async def test_run_agent_without_s3_does_not_save(run, monkeypatch):
    calls, _, publisher = run
    monkeypatch.setattr(session_store, "enabled", lambda: False)

    await agent.run_agent("do it", "claude-sonnet-4-6", publisher)

    assert ("save",) not in calls
    completed = [c[2] for c in calls if c[0] == "publish" and c[1] == "task_completed"][0]
    assert "session_store" not in completed
    started = [c[2] for c in calls if c[0] == "publish" and c[1] == "task_started"][0]
    assert "restored_session" not in started


@pytest.mark.asyncio
async def test_task_started_reports_restore(run, monkeypatch):
    calls, _, publisher = run
    monkeypatch.setattr(session_store, "enabled", lambda: True)
    monkeypatch.setattr(session_store, "last_restore", {"restored": True, "error": None})

    await agent.run_agent("do it", "claude-sonnet-4-6", publisher)

    started = [c[2] for c in calls if c[0] == "publish" and c[1] == "task_started"][0]
    assert started["restored_session"] is True
    assert started["restore_error"] is None


@pytest.mark.asyncio
async def test_task_started_reports_a_restore_from_previous(run, monkeypatch):
    calls, _, publisher = run
    monkeypatch.setattr(session_store, "enabled", lambda: True)
    monkeypatch.setattr(session_store, "last_restore",
                        {"restored": True, "from": "previous", "error": "latest: not a gzip file"})

    await agent.run_agent("do it", "claude-sonnet-4-6", publisher)

    started = [c[2] for c in calls if c[0] == "publish" and c[1] == "task_started"][0]
    assert started["restored_from"] == "previous"
    assert started["restore_error"] == "latest: not a gzip file"
