import pytest
import json
import asyncio

import manager_tools


@pytest.mark.asyncio
async def test_request_helper_uses_mock(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools._request("GET", "/api/v1/agents/foo")
    assert not result.get("isError")
    assert mock_api.calls == [("GET", "/api/v1/agents/foo")]


# --- sleep_agent ---

@pytest.mark.asyncio
async def test_sleep_agent_patches_lifecycle(mock_api):
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo", "lifecycle": "Sleep"})
    result = await manager_tools.sleep_agent.handler({"name": "foo"})
    assert not result.get("isError")
    assert mock_api.last_json == {"lifecycle": "Sleep"}


@pytest.mark.asyncio
async def test_sleep_agent_defaults_to_self(mock_api):
    mock_api.set("PATCH", "/api/v1/agents/self", {"name": "self", "lifecycle": "Sleep"})
    result = await manager_tools.sleep_agent.handler({})
    assert not result.get("isError")
    assert mock_api.calls == [("PATCH", "/api/v1/agents/self")]


# --- wake_agent ---

@pytest.mark.asyncio
async def test_wake_agent_posts_with_instructions(mock_api):
    mock_api.set("POST", "/api/v1/agents", {"name": "foo", "status": "Pending"})
    result = await manager_tools.wake_agent.handler({"name": "foo", "instructions": "do X"})
    assert not result.get("isError")
    assert mock_api.last_json["name"] == "foo"
    assert mock_api.last_json["instructions"] == "do X"


@pytest.mark.asyncio
async def test_wake_agent_requires_instructions(mock_api):
    result = await manager_tools.wake_agent.handler({"name": "foo"})
    assert result.get("isError")


# --- list_agents ---

@pytest.mark.asyncio
async def test_list_agents(mock_api):
    mock_api.set("GET", "/api/v1/agents", {"agents": [{"name": "a"}, {"name": "b"}]})
    result = await manager_tools.list_agents.handler({})
    assert not result.get("isError")
    body = json.loads(result["content"][0]["text"])
    assert len(body["agents"]) == 2


# --- get_agent ---

@pytest.mark.asyncio
async def test_get_agent_returns_full_spec(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "model": "claude-haiku-4-5", "skills": ["a"]})
    result = await manager_tools.get_agent.handler({"name": "foo"})
    assert not result.get("isError")
    body = json.loads(result["content"][0]["text"])
    assert body["model"] == "claude-haiku-4-5"


@pytest.mark.asyncio
async def test_list_connectors(mock_api):
    mock_api.set("GET", "/api/v1/connectors", {"connectors": [{"name": "slack"}]})
    result = await manager_tools.list_connectors.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_list_connector_templates(mock_api):
    mock_api.set("GET", "/api/v1/connector-templates", {"templates": [{"id": "slack"}]})
    result = await manager_tools.list_connector_templates.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_get_connector(mock_api):
    mock_api.set("GET", "/api/v1/connectors/slack", {"name": "slack", "authType": "token"})
    result = await manager_tools.get_connector.handler({"name": "slack"})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_attach_connector_appends_to_existing(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "connectors": ["github"]})
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools.attach_connector.handler({"connector_name": "slack", "agent_name": "foo"})
    assert not result.get("isError")
    assert mock_api.last_json == {"connectors": ["github", "slack"]}


@pytest.mark.asyncio
async def test_attach_connector_idempotent(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "connectors": ["slack"]})
    result = await manager_tools.attach_connector.handler({"connector_name": "slack", "agent_name": "foo"})
    assert not result.get("isError")
    # No PATCH made because slack was already attached.
    assert ("PATCH", "/api/v1/agents/foo") not in mock_api.calls


@pytest.mark.asyncio
async def test_detach_connector(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "connectors": ["slack", "github"]})
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools.detach_connector.handler({"connector_name": "slack", "agent_name": "foo"})
    assert not result.get("isError")
    assert mock_api.last_json == {"connectors": ["github"]}


@pytest.mark.asyncio
async def test_list_secrets(mock_api):
    mock_api.set("GET", "/api/v1/secrets", {"secrets": [{"name": "OPENAI"}]})
    result = await manager_tools.list_secrets.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_create_secret(mock_api):
    mock_api.set("POST", "/api/v1/secrets", {"name": "OPENAI"})
    result = await manager_tools.create_secret.handler({"name": "OPENAI", "value": "sk-test"})
    assert not result.get("isError")
    assert mock_api.last_json["name"] == "OPENAI"
    assert mock_api.last_json["value"] == "sk-test"
    assert mock_api.last_json["namespace"] == "default"


@pytest.mark.asyncio
async def test_delete_secret(mock_api):
    mock_api.set("DELETE", "/api/v1/secrets/OPENAI", {"deleted": "OPENAI"})
    result = await manager_tools.delete_secret.handler({"name": "OPENAI"})
    assert not result.get("isError")
    assert ("DELETE", "/api/v1/secrets/OPENAI") in mock_api.calls


@pytest.mark.asyncio
async def test_attach_secret_appends(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "secrets": []})
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools.attach_secret.handler({"secret_name": "OPENAI", "agent_name": "foo"})
    assert not result.get("isError")
    # PATCH field is "secretRefs" (request field) but GET field is "secrets" (response field).
    assert mock_api.last_json == {"secretRefs": ["OPENAI"]}


@pytest.mark.asyncio
async def test_detach_secret(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "secrets": ["OPENAI", "GITHUB"]})
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools.detach_secret.handler({"secret_name": "OPENAI", "agent_name": "foo"})
    assert not result.get("isError")
    assert mock_api.last_json == {"secretRefs": ["GITHUB"]}


@pytest.mark.asyncio
async def test_attach_secret_rejects_empty_name(mock_api):
    result = await manager_tools.attach_secret.handler({"secret_name": "", "agent_name": "foo"})
    assert result.get("isError")


@pytest.mark.asyncio
async def test_detach_secret_rejects_empty_name(mock_api):
    result = await manager_tools.detach_secret.handler({"secret_name": "  ", "agent_name": "foo"})
    assert result.get("isError")


@pytest.mark.asyncio
async def test_list_skills(mock_api):
    mock_api.set("GET", "/api/v1/skills", {"skills": [{"name": "git"}]})
    result = await manager_tools.list_skills.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_get_skill(mock_api):
    mock_api.set("GET", "/api/v1/skills/git", {"name": "git", "content": "..."})
    result = await manager_tools.get_skill.handler({"name": "git"})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_update_skill(mock_api):
    mock_api.set("PATCH", "/api/v1/skills/git", {"name": "git"})
    result = await manager_tools.update_skill.handler({"name": "git", "content": "new content"})
    assert not result.get("isError")
    assert mock_api.last_json == {"content": "new content"}


@pytest.mark.asyncio
async def test_update_skill_requires_a_field(mock_api):
    result = await manager_tools.update_skill.handler({"name": "git"})
    assert result.get("isError")


@pytest.mark.asyncio
async def test_delete_skill(mock_api):
    mock_api.set("DELETE", "/api/v1/skills/git", {"deleted": "git"})
    result = await manager_tools.delete_skill.handler({"name": "git"})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_detach_skill(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "skills": ["git", "docker"]})
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools.detach_skill.handler({"skill_name": "git", "agent_name": "foo"})
    assert not result.get("isError")
    assert mock_api.last_json == {"skills": ["docker"]}


# --- list_memories ---

@pytest.mark.asyncio
async def test_list_memories(mock_api):
    mock_api.set("GET", "/api/v1/memories", {"memories": [{"name": "preferences"}]})
    result = await manager_tools.list_memories.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_get_memory(mock_api):
    mock_api.set("GET", "/api/v1/memories/preferences", {"name": "preferences", "content": "..."})
    result = await manager_tools.get_memory.handler({"name": "preferences"})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_update_memory(mock_api):
    mock_api.set("PATCH", "/api/v1/memories/preferences", {"name": "preferences"})
    result = await manager_tools.update_memory.handler({"name": "preferences", "content": "new"})
    assert not result.get("isError")
    assert mock_api.last_json == {"content": "new"}


@pytest.mark.asyncio
async def test_update_memory_requires_a_field(mock_api):
    result = await manager_tools.update_memory.handler({"name": "preferences"})
    assert result.get("isError")


@pytest.mark.asyncio
async def test_delete_memory(mock_api):
    mock_api.set("DELETE", "/api/v1/memories/preferences", {"deleted": "preferences"})
    result = await manager_tools.delete_memory.handler({"name": "preferences"})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_detach_memory(mock_api):
    mock_api.set("GET", "/api/v1/agents/foo", {"name": "foo", "memories": ["a", "b"]})
    mock_api.set("PATCH", "/api/v1/agents/foo", {"name": "foo"})
    result = await manager_tools.detach_memory.handler({"memory_name": "a", "agent_name": "foo"})
    assert not result.get("isError")
    assert mock_api.last_json == {"memories": ["b"]}


# --- list_schedules ---

@pytest.mark.asyncio
async def test_list_schedules(mock_api):
    mock_api.set("GET", "/api/v1/schedules", {"schedules": [{"name": "morning"}]})
    result = await manager_tools.list_schedules.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_get_schedule(mock_api):
    mock_api.set("GET", "/api/v1/schedules/morning", {"name": "morning", "schedule": "0 9 * * *"})
    result = await manager_tools.get_schedule.handler({"name": "morning"})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_update_schedule_cron(mock_api):
    mock_api.set("PATCH", "/api/v1/schedules/morning", {"name": "morning"})
    result = await manager_tools.update_schedule.handler({"name": "morning", "schedule": "0 10 * * *"})
    assert not result.get("isError")
    assert mock_api.last_json == {"schedule": "0 10 * * *"}


@pytest.mark.asyncio
async def test_update_schedule_instructions(mock_api):
    mock_api.set("PATCH", "/api/v1/schedules/morning", {"name": "morning"})
    result = await manager_tools.update_schedule.handler({"name": "morning", "instructions": "new task"})
    assert not result.get("isError")
    assert mock_api.last_json == {"instructions": "new task"}


@pytest.mark.asyncio
async def test_update_schedule_requires_a_field(mock_api):
    result = await manager_tools.update_schedule.handler({"name": "morning"})
    assert result.get("isError")


# --- list_namespaces / list_templates ---

@pytest.mark.asyncio
async def test_list_namespaces(mock_api):
    mock_api.set("GET", "/api/v1/namespaces", {"namespaces": ["default", "team-a"]})
    result = await manager_tools.list_namespaces.handler({})
    assert not result.get("isError")


@pytest.mark.asyncio
async def test_list_templates(mock_api):
    mock_api.set("GET", "/api/v1/templates", {"templates": [{"name": "default"}, {"name": "gpu"}]})
    result = await manager_tools.list_templates.handler({})
    assert not result.get("isError")


def test_manager_server_registers_all_new_tools():
    server_dict = manager_tools.create_manager_server()
    # create_manager_server() returns a dict {"type": "sdk", "name": ..., "instance": <Server>}.
    # The MCP Server stores tools in its ListToolsRequest handler; call it synchronously via asyncio.
    import mcp.types
    inst = server_dict["instance"]
    handler = inst.request_handlers[mcp.types.ListToolsRequest]
    loop = asyncio.new_event_loop()
    try:
        result = loop.run_until_complete(
            handler(mcp.types.ListToolsRequest(method="tools/list"))
        )
    finally:
        loop.close()
    tool_names = {t.name for t in result.root.tools}

    # Spot-check a representative sample from each category.
    expected = {
        "create_agent", "sleep_agent", "wake_agent", "list_agents", "get_agent",
        "list_schedules", "update_schedule",
        "list_memories", "update_memory", "detach_memory",
        "list_skills", "delete_skill", "detach_skill",
        "list_connectors", "attach_connector", "detach_connector",
        "list_secrets", "create_secret", "attach_secret",
        "list_namespaces", "list_templates",
    }
    missing = expected - tool_names
    assert not missing, f"Missing tools in MCP server registration: {missing}"
