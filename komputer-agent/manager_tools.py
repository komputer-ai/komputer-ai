import asyncio
import json
import os
import re
import time

import httpx
import redis
from claude_agent_sdk import tool, create_sdk_mcp_server

API_URL = os.environ.get("KOMPUTER_API_URL", "http://komputer-api:8080")
AGENT_NAME = os.environ.get("KOMPUTER_AGENT_NAME", "unknown")

# Set by create_manager_server() from the agent's Redis config.
_redis_client: redis.Redis | None = None
_stream_prefix: str = "komputer-events"


def _sub_name(name: str) -> str:
    """Prefix sub-agent name with manager name, sanitized for K8s."""
    sanitized = re.sub(r'[^a-z0-9-]', '', name.lower())[:50]
    if not sanitized:
        raise ValueError(f"Invalid sub-agent name: {name}")
    return f"{AGENT_NAME}-sub-{sanitized}"


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


def _field(fields: dict, key: str) -> str:
    """Extract a string field from Redis stream entry, handling bytes."""
    val = fields.get(key.encode(), fields.get(key, b""))
    return val.decode() if isinstance(val, bytes) else str(val)


def _poll_once(pending: dict, results: dict, terminal_types: set):
    """Non-blocking: check each pending stream for terminal events."""
    if not pending:
        return

    # Check each stream individually using XRANGE (works even if stream is new).
    # XREAD with block=0 can behave unexpectedly with non-existent streams.
    for full_name, info in list(pending.items()):
        try:
            # Read entries starting from our last checkpoint.
            # Use the last_id directly — XRANGE is inclusive, but we track
            # the last processed ID so we might re-read it. We skip dupes below.
            min_id = info["last_id"] if info["last_id"] != "0-0" else "-"
            entries = _redis_client.xrange(info["stream_key"], min_id, "+", count=100)
        except redis.RedisError:
            continue

        for entry_id, fields in entries:
            info["last_id"] = entry_id.decode() if isinstance(entry_id, bytes) else entry_id
            etype = _field(fields, "type")
            if etype in terminal_types:
                results[info["display_name"]] = {
                    "status": etype,
                    "payload": json.loads(_field(fields, "payload") or "{}"),
                }
                del pending[full_name]
                break


def _fetch_recent_events(names: list[str]) -> dict:
    """Fetch the last few events for each agent."""
    recent_map = {}
    for name in names:
        full_name = _sub_name(name)
        stream_key = f"{_stream_prefix}:{full_name}"
        try:
            recent = _redis_client.xrevrange(stream_key, "+", "-", count=5)
            recent_events = []
            for _, fields in reversed(recent):
                recent_events.append({
                    "type": _field(fields, "type"),
                    "payload": json.loads(_field(fields, "payload") or "{}"),
                })
            recent_map[name] = recent_events
        except redis.RedisError:
            pass
    return recent_map


@tool(
    name="create_agent",
    description="Create a sub-agent to handle a specific task. The agent will start working immediately. Sub-agents are always workers (no orchestration tools). After creating agents, use wait_for_completion to block until they finish — this is much more efficient than polling.",
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
    name="wait_for_completion",
    description="Block until one or more sub-agents finish their tasks, then return their final results and last few events. This is the PREFERRED way to wait for sub-agents — it subscribes to Redis streams directly instead of polling, saving time and tokens. Supports waiting for multiple agents at once.",
    input_schema={
        "type": "object",
        "properties": {
            "names": {
                "type": "array",
                "items": {"type": "string"},
                "description": "List of sub-agent names to wait for (as passed to create_agent). Waits until ALL complete.",
            },
            "timeout_seconds": {
                "type": "integer",
                "description": "Max seconds to wait (default 300 = 5 minutes). Returns partial results on timeout.",
            },
        },
        "required": ["names"],
    },
)
async def wait_for_completion(args):
    if _redis_client is None:
        return _err("Redis not configured for manager tools")

    names = args["names"]
    timeout_seconds = args.get("timeout_seconds", 300)
    deadline = time.time() + timeout_seconds

    # Build stream keys and track which agents are still pending.
    pending = {}
    for name in names:
        full_name = _sub_name(name)
        stream_key = f"{_stream_prefix}:{full_name}"
        pending[full_name] = {"stream_key": stream_key, "display_name": name, "last_id": "0-0"}

    results = {}
    terminal_types = {"task_completed", "error", "task_cancelled"}

    # First pass: check existing stream entries for already-completed agents.
    for full_name, info in list(pending.items()):
        try:
            entries = _redis_client.xrange(info["stream_key"], "-", "+")
            for entry_id, fields in entries:
                info["last_id"] = entry_id.decode() if isinstance(entry_id, bytes) else entry_id
                if _field(fields, "type") in terminal_types:
                    results[info["display_name"]] = {
                        "status": _field(fields, "type"),
                        "payload": json.loads(_field(fields, "payload") or "{}"),
                    }
                    del pending[full_name]
                    break
        except redis.RedisError as e:
            results[info["display_name"]] = {"status": "error", "payload": {"error": f"Redis error: {e}"}}
            del pending[full_name]

    # Async polling loop: non-blocking XREAD + asyncio.sleep.
    # We use short sleeps (2s) so the MCP server's event loop stays responsive
    # and the stdio transport doesn't time out.
    while pending and time.time() < deadline:
        _poll_once(pending, results, terminal_types)
        if not pending:
            break
        await asyncio.sleep(2)

    # Any still pending after timeout.
    for full_name, info in pending.items():
        results[info["display_name"]] = {"status": "timeout", "payload": {"error": f"Agent did not complete within {timeout_seconds}s"}}

    # Fetch the last few events for each agent (for context).
    recent_map = _fetch_recent_events(names)
    for name, events in recent_map.items():
        if name in results:
            results[name]["recent_events"] = events

    return _ok(json.dumps(results, indent=2))


@tool(
    name="get_agent_status",
    description="Get the current status of a sub-agent. Use to check if it's still working (taskStatus='Busy') or done (taskStatus='Idle'). Prefer wait_for_completion instead of polling this.",
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
    full_name = _sub_name(args["name"])
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
    full_name = _sub_name(args["name"])
    return await _request("DELETE", f"/api/v1/agents/{full_name}")


def create_manager_server(redis_config: dict | None = None):
    """Create the MCP server with all manager orchestration tools.

    Args:
        redis_config: Redis configuration dict from /etc/komputer/config.json.
            If provided, enables the wait_for_completion tool to subscribe
            directly to Redis streams instead of polling the API.
    """
    global _redis_client, _stream_prefix

    if redis_config:
        password = redis_config.get("password") or None
        _redis_client = redis.Redis(
            host=redis_config["address"].split(":")[0],
            port=int(redis_config["address"].split(":")[1]),
            password=password,
            db=redis_config.get("db", 0),
        )
        _stream_prefix = redis_config.get("stream_prefix", "komputer-events")

    return create_sdk_mcp_server(
        name="komputer",
        tools=[create_agent, wait_for_completion, get_agent_status, get_agent_events, delete_agent],
    )
