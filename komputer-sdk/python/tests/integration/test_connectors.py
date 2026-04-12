"""Integration tests for connectors."""

import pytest


CONNECTOR_NAME = "sdk-test-connector"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, CONNECTOR_NAME)
    yield
    _safe_delete(client, CONNECTOR_NAME)


def _safe_delete(client, name):
    try:
        client.delete_connector(name)
    except Exception:
        pass


class TestConnectors:
    def test_create_connector(self, client):
        resp = client.create_connector(
            name=CONNECTOR_NAME,
            service="custom",
            url="https://example.com/mcp",
            display_name="SDK Test Connector",
        )
        assert resp.name == CONNECTOR_NAME
        assert resp.service == "custom"

    def test_list_connectors(self, client):
        client.create_connector(
            name=CONNECTOR_NAME,
            service="custom",
            url="https://example.com/mcp",
        )

        connectors = client.list_connectors()
        names = [c.name for c in connectors]
        assert CONNECTOR_NAME in names

    def test_get_connector(self, client):
        client.create_connector(
            name=CONNECTOR_NAME,
            service="custom",
            url="https://example.com/mcp",
        )

        resp = client.get_connector(CONNECTOR_NAME)
        assert resp.name == CONNECTOR_NAME
        assert resp.url == "https://example.com/mcp"

    def test_delete_connector(self, client):
        client.create_connector(
            name=CONNECTOR_NAME,
            service="custom",
            url="https://example.com/mcp",
        )

        client.delete_connector(CONNECTOR_NAME)

        with pytest.raises(Exception):
            client.get_connector(CONNECTOR_NAME)
