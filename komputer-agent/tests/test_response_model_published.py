"""The model the API reports is published alongside the configured one.

With a Bedrock application inference profile the configured model is an opaque
ARN; the model on each AssistantMessage (and the CLI's per-model usage on the
ResultMessage) is what says which model actually answered and was priced.
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest

import agent
import state
from claude_agent_sdk import AssistantMessage, ResultMessage, TextBlock, ThinkingBlock, ToolUseBlock

PROFILE_ARN = "arn:aws:bedrock:eu-central-1:123456789012:application-inference-profile/abc123"
RESPONSE_MODEL = "claude-sonnet-5"
MODEL_USAGE = {RESPONSE_MODEL: {"inputTokens": 10, "outputTokens": 5, "cacheReadInputTokens": 100,
                                "cacheCreationInputTokens": 20, "costUSD": 0.0123}}


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


async def _run(monkeypatch, interrupted=False):
    messages = [
        AssistantMessage(content=[
            ThinkingBlock(thinking="hmm", signature="sig"),
            TextBlock(text="hello"),
            ToolUseBlock(id="t1", name="Bash", input={"command": "ls"}),
        ], model=RESPONSE_MODEL),
        ResultMessage(subtype="success", duration_ms=10, duration_api_ms=10, is_error=False,
                      num_turns=1, session_id="s", stop_reason=None, total_cost_usd=0.0123,
                      usage={}, model_usage=MODEL_USAGE),
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
        # run_agent clears state.interrupted on entry, so a cancel has to land mid-run.
        if interrupted:
            state.interrupted = True
        return False

    monkeypatch.setattr(agent, "_wait_for_steer", no_steer)

    await agent.run_agent("do it", PROFILE_ARN, FakePublisher())
    return dict((t, p) for t, p in published)


@pytest.mark.asyncio
async def test_progress_events_carry_response_model(monkeypatch):
    events = await _run(monkeypatch)
    for event_type in ("thinking", "text", "tool_call"):
        assert events[event_type]["model"] == RESPONSE_MODEL, event_type


@pytest.mark.asyncio
@pytest.mark.parametrize("interrupted,event_type", [(False, "task_completed"), (True, "task_cancelled")])
async def test_end_of_task_events_carry_response_model_and_usage(monkeypatch, interrupted, event_type):
    events = await _run(monkeypatch, interrupted=interrupted)
    payload = events[event_type]
    assert payload["model"] == PROFILE_ARN  # configured model is unchanged
    assert payload["response_model"] == RESPONSE_MODEL
    assert payload["model_usage"] == MODEL_USAGE
