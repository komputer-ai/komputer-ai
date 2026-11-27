"""Terminal task events carry Claude Code's per-model usage (model_usage).

The API reads its costBasis to warn when Claude Code couldn't price a model,
e.g. a Bedrock inference-profile ARN without bedrock:GetInferenceProfile.
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest

import agent
import state
from claude_agent_sdk import AssistantMessage, ResultMessage, TextBlock

ARN = "arn:aws:bedrock:eu-central-1:123456789012:application-inference-profile/abc"
MODEL_USAGE = {ARN: {"inputTokens": 23, "outputTokens": 169, "costUSD": 0.065, "canonicalModel": ARN, "costBasis": "unknown"}}


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


async def _run(monkeypatch, subtype="success", interrupted=False):
    messages = [
        AssistantMessage(content=[TextBlock(text="done")], model="anthropic.claude-haiku-4-5-20251001-v1:0"),
        ResultMessage(subtype=subtype, duration_ms=10, duration_api_ms=10, is_error=subtype != "success",
                      num_turns=1, session_id="s", stop_reason=None, total_cost_usd=0.065, usage={},
                      model_usage=MODEL_USAGE),
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
        if interrupted:
            state.interrupted = True
        return False

    monkeypatch.setattr(agent, "_wait_for_steer", no_steer)
    await agent.run_agent("go", ARN, FakePublisher())
    return published


@pytest.mark.asyncio
async def test_task_completed_carries_model_usage(monkeypatch):
    published = await _run(monkeypatch)
    [payload] = [p for t, p in published if t == "task_completed"]
    assert payload["model_usage"] == MODEL_USAGE


@pytest.mark.asyncio
async def test_budget_stop_error_carries_model_usage(monkeypatch):
    published = await _run(monkeypatch, subtype="error_max_budget_usd")
    [payload] = [p for t, p in published if t == "error"]
    assert payload["model_usage"] == MODEL_USAGE


@pytest.mark.asyncio
async def test_task_cancelled_carries_model_usage(monkeypatch):
    published = await _run(monkeypatch, interrupted=True)
    [payload] = [p for t, p in published if t == "task_cancelled"]
    assert payload["model_usage"] == MODEL_USAGE
