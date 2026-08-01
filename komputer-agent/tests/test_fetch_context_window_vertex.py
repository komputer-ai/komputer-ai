"""When the agent runs against Vertex AI, _fetch_context_window must skip the
Anthropic /v1/models lookup — there is no API key in that mode and the model
identifier is a Vertex ID anyway."""

from unittest.mock import patch

import agent


def test_fetch_context_window_returns_none_when_vertex_enabled(monkeypatch):
    # Set ANTHROPIC_API_KEY as well so the test verifies the Vertex guard fires
    # *before* the API-key-missing guard (otherwise the test would pass for the
    # wrong reason — there's no key in the test env by default).
    monkeypatch.setenv("CLAUDE_CODE_USE_VERTEX", "1")
    monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-should-not-be-used")
    with patch("agent.httpx.get") as mock_get:
        result = agent._fetch_context_window("claude-sonnet-4-5@20250929")
    assert result is None
    mock_get.assert_not_called()
