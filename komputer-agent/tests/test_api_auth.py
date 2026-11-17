import pytest

import manager_tools
import oauth_refresh


@pytest.mark.asyncio
async def test_request_sends_bearer_token_when_set(mock_api, monkeypatch):
    monkeypatch.setenv("KOMPUTER_API_TOKEN", "s3cret")
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo"})
    await manager_tools._request("GET", "/api/v1/agents/foo")
    assert mock_api.last_headers == {"Authorization": "Bearer s3cret"}


@pytest.mark.asyncio
async def test_request_sends_no_auth_header_without_token(mock_api, monkeypatch):
    monkeypatch.delenv("KOMPUTER_API_TOKEN", raising=False)
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo"})
    await manager_tools._request("GET", "/api/v1/agents/foo")
    assert mock_api.last_headers == {}


def test_oauth_refresh_auth_headers(monkeypatch):
    monkeypatch.setenv("KOMPUTER_API_TOKEN", " s3cret ")
    assert oauth_refresh._auth_headers() == {"Authorization": "Bearer s3cret"}
    monkeypatch.setenv("KOMPUTER_API_TOKEN", "")
    assert oauth_refresh._auth_headers() == {}
