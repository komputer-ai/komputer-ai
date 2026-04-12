"""Integration tests for secrets."""

import pytest


SECRET_NAME = "sdk-test-secret"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, SECRET_NAME)
    yield
    _safe_delete(client, SECRET_NAME)


def _safe_delete(client, name):
    try:
        client.delete_secret(name)
    except Exception:
        pass


class TestSecrets:
    def test_create_secret(self, client):
        resp = client.create_secret(
            name=SECRET_NAME,
            data={"API_KEY": "test-key-123", "TOKEN": "test-token"},
        )
        assert resp.name == SECRET_NAME
        assert "API_KEY" in resp.keys
        assert "TOKEN" in resp.keys

    def test_list_secrets_contains_created(self, client):
        client.create_secret(name=SECRET_NAME, data={"KEY": "val"})

        secrets = client.list_secrets()
        names = [s.name for s in secrets.secrets]
        assert SECRET_NAME in names

    def test_update_secret(self, client):
        client.create_secret(name=SECRET_NAME, data={"OLD_KEY": "old"})

        resp = client.update_secret(SECRET_NAME, data={"NEW_KEY": "new-value"})
        assert "NEW_KEY" in resp.keys

    def test_delete_secret(self, client):
        client.create_secret(name=SECRET_NAME, data={"KEY": "val"})

        client.delete_secret(SECRET_NAME)

        secrets = client.list_secrets()
        names = [s.name for s in secrets.secrets]
        assert SECRET_NAME not in names
