"""When the agent runs against Bedrock, _fetch_context_window must skip
the Anthropic /v1/models lookup — there is no API key in that mode and
the model identifier is a Bedrock ID anyway."""

import os
import sys
from unittest.mock import patch

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import agent


def test_fetch_context_window_returns_none_when_bedrock_enabled(monkeypatch):
    # Set ANTHROPIC_API_KEY as well so the test verifies the Bedrock guard
    # fires *before* the API-key-missing guard (otherwise the test would
    # pass for the wrong reason — there's no key in the test env by default).
    monkeypatch.setenv("CLAUDE_CODE_USE_BEDROCK", "1")
    monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-should-not-be-used")
    with patch("agent.httpx.get") as mock_get:
        result = agent._fetch_context_window("anything")
    assert result is None
    mock_get.assert_not_called()


def test_fetch_context_window_calls_anthropic_when_bedrock_disabled(monkeypatch):
    """Sanity check the guard doesn't fire when Bedrock is off."""
    monkeypatch.delenv("CLAUDE_CODE_USE_BEDROCK", raising=False)
    monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-test")

    class FakeResp:
        status_code = 200

        def json(self):
            # The real parser reads `max_input_tokens` from the response.
            return {"max_input_tokens": 200000}

        def raise_for_status(self):
            pass

    with patch("agent.httpx.get", return_value=FakeResp()) as mock_get:
        result = agent._fetch_context_window("claude-sonnet-4-6")
    assert result == 200000
    mock_get.assert_called_once()
