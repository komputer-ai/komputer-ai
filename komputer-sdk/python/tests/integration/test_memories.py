"""Integration tests for memories."""

import pytest
from komputer_ai.models import CreateMemoryRequest, PatchMemoryRequest


MEMORY_NAME = "sdk-test-memory"


@pytest.fixture(autouse=True)
def cleanup(client):
    """Delete test memory before and after each test module run."""
    _safe_delete(client, MEMORY_NAME)
    yield
    _safe_delete(client, MEMORY_NAME)


def _safe_delete(client, name):
    try:
        client.memories.delete_memory(name)
    except Exception:
        pass


class TestMemories:
    def test_create_memory(self, client):
        req = CreateMemoryRequest(
            name=MEMORY_NAME,
            content="SDK integration test content",
            description="Created by Python SDK tests",
        )
        resp = client.memories.create_memory(req)
        assert resp.name == MEMORY_NAME
        assert resp.content == "SDK integration test content"

    def test_list_memories_contains_created(self, client):
        # Create first
        req = CreateMemoryRequest(
            name=MEMORY_NAME,
            content="list test",
            description="for listing",
        )
        client.memories.create_memory(req)

        memories = client.memories.list_memories()
        names = [m.name for m in memories]
        assert MEMORY_NAME in names

    def test_get_memory(self, client):
        req = CreateMemoryRequest(
            name=MEMORY_NAME,
            content="get test content",
            description="for get",
        )
        client.memories.create_memory(req)

        resp = client.memories.get_memory(MEMORY_NAME)
        assert resp.name == MEMORY_NAME
        assert resp.content == "get test content"

    def test_patch_memory(self, client):
        req = CreateMemoryRequest(
            name=MEMORY_NAME,
            content="original",
            description="original desc",
        )
        client.memories.create_memory(req)

        patch = PatchMemoryRequest(content="updated content")
        resp = client.memories.patch_memory(MEMORY_NAME, patch)
        assert resp.content == "updated content"

    def test_delete_memory(self, client):
        req = CreateMemoryRequest(
            name=MEMORY_NAME, content="to delete", description="delete me"
        )
        client.memories.create_memory(req)

        client.memories.delete_memory(MEMORY_NAME)

        # Verify it's gone
        with pytest.raises(Exception):
            client.memories.get_memory(MEMORY_NAME)
