import os

import httpx
from claude_agent_sdk import tool, create_sdk_mcp_server

API_URL = os.environ.get("KOMPUTER_API_URL", "http://komputer-api:8080")
AGENT_NAME = os.environ.get("KOMPUTER_AGENT_NAME", "unknown")


def _sub_name(name: str) -> str:
    """Prefix sub-agent name with manager name."""
    return f"{AGENT_NAME}-sub-{name}"


def _ok(text: str) -> dict:
    return {"content": [{"type": "text", "text": text}]}


def _err(text: str) -> dict:
    return {"content": [{"type": "text", "text": text}], "isError": True}


async def _request(method: str, path: str, timeout: int = 10, **kwargs) -> dict:
    """Make an HTTP request to the komputer API and return a tool response."""
    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.request(method, f"{API_URL}{path}", **kwargs)
            if resp.status_code >= 400:
                return _err(f"API error {resp.status_code}: {resp.text}")
            return _ok(resp.text)
    except httpx.HTTPError as exc:
        return _err(f"Request failed: {exc}")


@tool(
    name="create_agent",
    description="Create a sub-agent to handle a specific task. The agent will start working immediately. Sub-agents are always workers (no orchestration tools).",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Short descriptive name for the sub-agent (e.g. 'researcher', 'writer')"},
            "instructions": {"type": "string", "description": "Detailed task instructions for the sub-agent"},
            "model": {"type": "string", "description": "Claude model to use (optional, defaults to claude-sonnet)"},
        },
        "required": ["name", "instructions"],
    },
)
async def create_agent(args):
    full_name = _sub_name(args["name"])
    payload = {
        "name": full_name,
        "instructions": args["instructions"],
        "role": "worker",
    }
    if args.get("model"):
        payload["model"] = args["model"]
    return await _request("POST", "/api/v1/agents", timeout=30, json=payload)


@tool(
    name="get_agent_status",
    description="Get the current status of a sub-agent. Use to check if it's still working (taskStatus='Busy') or done (taskStatus='Idle').",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The sub-agent name (as passed to create_agent)"},
        },
        "required": ["name"],
    },
)
async def get_agent_status(args):
    full_name = _sub_name(args["name"])
    return await _request("GET", f"/api/v1/agents/{full_name}")


@tool(
    name="get_agent_events",
    description="Get the event history of a sub-agent, including its final result. Use after the agent's taskStatus is 'Idle'.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The sub-agent name (as passed to create_agent)"},
            "limit": {"type": "integer", "description": "Max events to return (default 50)"},
        },
        "required": ["name"],
    },
)
async def get_agent_events(args):
    full_name = _sub_name(args["name"])
    limit = args.get("limit", 50)
    return await _request("GET", f"/api/v1/agents/{full_name}/events", params={"limit": limit})


@tool(
    name="delete_agent",
    description="Delete a sub-agent and clean up its resources. Use after collecting its results.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The sub-agent name (as passed to create_agent)"},
        },
        "required": ["name"],
    },
)
async def delete_agent(args):
    full_name = _sub_name(args["name"])
    return await _request("DELETE", f"/api/v1/agents/{full_name}")


def create_manager_server():
    """Create the MCP server with all manager orchestration tools."""
    return create_sdk_mcp_server(
        name="komputer",
        tools=[create_agent, get_agent_status, get_agent_events, delete_agent],
    )
