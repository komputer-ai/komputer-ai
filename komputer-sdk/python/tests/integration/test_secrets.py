"""Integration tests for secrets."""

import pytest
from komputer_ai.models import CreateSecretRequest, UpdateSecretRequest


SECRET_NAME = "sdk-test-secret"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, SECRET_NAME)
    yield
    _safe_delete(client, SECRET_NAME)


def _safe_delete(client, name):
    try:
        client.secrets.delete_secret(name)
    except Exception:
        pass


class TestSecrets:
    def test_create_secret(self, client):
        req = CreateSecretRequest(
            name=SECRET_NAME,
            data={"API_KEY": "test-key-123", "TOKEN": "test-token"},
        )
        resp = client.secrets.create_secret(req)
        assert resp.name == SECRET_NAME
        # Keys should be returned, not values
        assert "API_KEY" in resp.keys
        assert "TOKEN" in resp.keys

    def test_list_secrets_contains_created(self, client):
        req = CreateSecretRequest(
            name=SECRET_NAME,
            data={"KEY": "val"},
        )
        client.secrets.create_secret(req)

        secrets = client.secrets.list_secrets()
        names = [s.name for s in secrets.secrets]
        assert SECRET_NAME in names

    def test_update_secret(self, client):
        req = CreateSecretRequest(
            name=SECRET_NAME,
            data={"OLD_KEY": "old"},
        )
        client.secrets.create_secret(req)

        update = UpdateSecretRequest(data={"NEW_KEY": "new-value"})
        resp = client.secrets.update_secret(SECRET_NAME, update)
        assert "NEW_KEY" in resp.keys

    def test_delete_secret(self, client):
        req = CreateSecretRequest(
            name=SECRET_NAME,
            data={"KEY": "val"},
        )
        client.secrets.create_secret(req)

        client.secrets.delete_secret(SECRET_NAME)

        # Verify gone — list shouldn't contain it
        secrets = client.secrets.list_secrets()
        names = [s.name for s in secrets.secrets]
        assert SECRET_NAME not in names
