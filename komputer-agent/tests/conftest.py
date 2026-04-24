"""Shared fixtures for manager_tools tests.

The mock_api fixture monkey-patches the module-level httpx.AsyncClient call
inside manager_tools._request, so every tool talks to an in-memory dict of
canned responses keyed by (METHOD, PATH).

sys.path is extended here (once, before collection) so that `import manager_tools`
works from any test file without repeating the path setup.
"""
import json
import os
import sys

# Add komputer-agent to path so `import manager_tools` works from tests dir.
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import manager_tools
import pytest


@pytest.fixture(autouse=True)
def env_setup(monkeypatch):
    """Every test runs as if it's an agent named 'self' in namespace 'default'."""
    monkeypatch.setenv("KOMPUTER_AGENT_NAME", "self")
    monkeypatch.setenv("KOMPUTER_NAMESPACE", "default")
    monkeypatch.setenv("KOMPUTER_API_URL", "http://test-api")


@pytest.fixture
def mock_api(monkeypatch):
    """Mock the API calls _request makes.

    Usage:
        def test_x(mock_api):
            mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "skills": []})
            mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
            ...
            assert mock_api.calls == [("GET", "/api/v1/agents/foo"),
                                      ("PATCH", "/api/v1/agents/foo")]
            assert mock_api.last_json == {"skills": ["s1"]}
    """
    class FakeAPI:
        def __init__(self):
            self.responses = {}   # (method, path) -> (status, body_dict)
            self.calls = []       # list of (method, path) actually invoked
            self.last_json = None # JSON body of the most recent request

        def set(self, method, path, body, status=200):
            self.responses[(method.upper(), path)] = (status, body)

    fake = FakeAPI()

    class FakeResponse:
        def __init__(self, status, body):
            self.status_code = status
            self._body = body
            self.text = json.dumps(body)
        def json(self):
            return self._body

    class FakeClient:
        def __init__(self, *a, **kw):
            pass
        async def __aenter__(self):
            return self
        async def __aexit__(self, *a):
            return False
        async def request(self, method, url, **kwargs):
            # Strip the base URL to match the registered path.
            base = os.environ["KOMPUTER_API_URL"]
            path = url.replace(base, "", 1)
            fake.calls.append((method.upper(), path))
            fake.last_json = kwargs.get("json")
            key = (method.upper(), path)
            if key not in fake.responses:
                return FakeResponse(404, {"error": f"no mock for {key}"})
            status, body = fake.responses[key]
            return FakeResponse(status, body)

    # Patch httpx and the module-level API_URL/NAMESPACE constants in manager_tools.
    # API_URL and NAMESPACE are captured at import time from env vars, so we must
    # patch the module attributes directly in addition to setting env vars.
    monkeypatch.setattr(manager_tools, "API_URL", os.environ["KOMPUTER_API_URL"])
    monkeypatch.setattr(manager_tools, "NAMESPACE", os.environ["KOMPUTER_NAMESPACE"])
    monkeypatch.setattr(manager_tools.httpx, "AsyncClient", FakeClient)

    return fake
