"""Integration tests for memories."""

import pytest


MEMORY_NAME = "sdk-test-memory"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, MEMORY_NAME)
    yield
    _safe_delete(client, MEMORY_NAME)


def _safe_delete(client, name):
    try:
        client.delete_memory(name)
    except Exception:
        pass


class TestMemories:
    def test_create_memory(self, client):
        resp = client.create_memory(
            name=MEMORY_NAME,
            content="SDK integration test content",
            description="Created by Python SDK tests",
        )
        assert resp.name == MEMORY_NAME
        assert resp.content == "SDK integration test content"

    def test_list_memories_contains_created(self, client):
        client.create_memory(
            name=MEMORY_NAME,
            content="list test",
            description="for listing",
        )

        memories = client.list_memories()
        names = [m.name for m in memories]
        assert MEMORY_NAME in names

    def test_get_memory(self, client):
        client.create_memory(
            name=MEMORY_NAME,
            content="get test content",
            description="for get",
        )

        resp = client.get_memory(MEMORY_NAME)
        assert resp.name == MEMORY_NAME
        assert resp.content == "get test content"

    def test_patch_memory(self, client):
        client.create_memory(
            name=MEMORY_NAME,
            content="original",
            description="original desc",
        )

        resp = client.patch_memory(MEMORY_NAME, content="updated content")
        assert resp.content == "updated content"

    def test_delete_memory(self, client):
        client.create_memory(
            name=MEMORY_NAME,
            content="to delete",
            description="delete me",
        )

        client.delete_memory(MEMORY_NAME)

        with pytest.raises(Exception):
            client.get_memory(MEMORY_NAME)
