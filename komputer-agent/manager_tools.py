import json
import os
import re

import httpx
from claude_agent_sdk import tool, create_sdk_mcp_server

API_URL = os.environ.get("KOMPUTER_API_URL", "http://komputer-api:8080")
NAMESPACE = os.environ.get("KOMPUTER_NAMESPACE", "default")


def _sanitize_name(name: str) -> str:
    """Sanitize agent name for K8s resource naming."""
    sanitized = re.sub(r'[^a-z0-9-]', '', name.lower())[:63]
    if not sanitized:
        raise ValueError(f"Invalid agent name: {name}")
    return sanitized


def _ok(text: str) -> dict:
    return {"content": [{"type": "text", "text": text}]}


def _err(text: str) -> dict:
    return {"content": [{"type": "text", "text": text}], "isError": True}


async def _request(method: str, path: str, timeout: int = 10, **kwargs) -> dict:
    """Make an HTTP request to the komputer API and return a tool response."""
    try:
        # Always include namespace in query params.
        params = kwargs.pop("params", {})
        params["namespace"] = NAMESPACE
        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.request(method, f"{API_URL}{path}", params=params, **kwargs)
            if resp.status_code >= 400:
                return _err(f"API error {resp.status_code}: {resp.text}")
            return _ok(resp.text)
    except httpx.HTTPError as exc:
        return _err(f"Request failed: {exc}")


@tool(
    name="create_agent",
    description="Create a sub-agent to handle a specific task. The agent will start working immediately. Sub-agents are always workers (no orchestration tools). You can pass secrets to sub-agents if they need credentials (e.g. for git clone with auth).",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Unique name for the sub-agent (e.g. 'bitcoin-researcher', 'weather-agent'). Use lowercase with hyphens, no spaces. This exact name is used for all subsequent operations."},
            "instructions": {"type": "string", "description": "Detailed task instructions for the sub-agent"},
            "model": {"type": "string", "description": "Claude model to use (optional, defaults to claude-sonnet)"},
            "secrets": {"type": "object", "description": "Key-value secrets to pass to the sub-agent (e.g. {\"GITHUB\": \"ghp_xxx\"}). They become SECRET_KEY env vars in the sub-agent pod.", "additionalProperties": {"type": "string"}},
        },
        "required": ["name", "instructions"],
    },
)
async def create_agent(args):
    full_name = _sanitize_name(args["name"])
    payload = {
        "name": full_name,
        "instructions": args["instructions"],
        "role": "worker",
        "namespace": NAMESPACE,
    }
    if args.get("secrets"):
        payload["secrets"] = args["secrets"]
    if args.get("model"):
        payload["model"] = args["model"]
    return await _request("POST", "/api/v1/agents", timeout=30, json=payload)


@tool(
    name="get_agent_status",
    description="Get the current status of a sub-agent. Returns taskStatus ('Busy', 'Idle', 'Error') and other details.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The sub-agent name (as passed to create_agent)"},
        },
        "required": ["name"],
    },
)
async def get_agent_status(args):
    full_name = _sanitize_name(args["name"])
    return await _request("GET", f"/api/v1/agents/{full_name}")


@tool(
    name="get_agent_events",
    description="Get the last few events from a sub-agent. Returns only the 5 most recent events by default to save context. Use for checking progress or getting results after completion.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The sub-agent name (as passed to create_agent)"},
            "limit": {"type": "integer", "description": "Max events to return (default 5, max 200)"},
        },
        "required": ["name"],
    },
)
async def get_agent_events(args):
    full_name = _sanitize_name(args["name"])
    limit = args.get("limit", 5)
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
    full_name = _sanitize_name(args["name"])
    return await _request("DELETE", f"/api/v1/agents/{full_name}")


def create_manager_server():
    """Create the MCP server with manager orchestration tools."""
    return create_sdk_mcp_server(
        name="komputer",
        tools=[create_agent, get_agent_status, get_agent_events, delete_agent],
    )
