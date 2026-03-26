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
    description="Block until a sub-agent finishes its task, then return its final result and last few events. This is the PREFERRED way to wait for sub-agents — it uses Redis stream subscription instead of polling, saving time and tokens. Call this after create_agent. Supports waiting for multiple agents at once.",
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
                event_type = fields.get(b"type", fields.get("type", b"")).decode() if isinstance(fields.get(b"type", fields.get("type", b"")), bytes) else fields.get(b"type", fields.get("type", ""))
                if event_type in terminal_types:
                    payload_str = fields.get(b"payload", fields.get("payload", b"{}"))
                    if isinstance(payload_str, bytes):
                        payload_str = payload_str.decode()
                    results[info["display_name"]] = {
                        "status": event_type,
                        "payload": json.loads(payload_str),
                    }
                    del pending[full_name]
                    break
        except redis.RedisError as e:
            results[info["display_name"]] = {"status": "error", "payload": {"error": f"Redis error: {e}"}}
            del pending[full_name]

    # Blocking loop: XREAD on remaining streams until all finish or timeout.
    while pending and time.time() < deadline:
        remaining_ms = int((deadline - time.time()) * 1000)
        if remaining_ms <= 0:
            break

        block_ms = min(remaining_ms, 5000)
        streams = {}
        for info in pending.values():
            streams[info["stream_key"]] = info["last_id"]

        try:
            resp = _redis_client.xread(streams, block=block_ms, count=100)
        except redis.RedisError as e:
            # Transient error, retry after brief pause.
            time.sleep(0.5)
            continue

        if not resp:
            continue

        for stream_key_bytes, entries in resp:
            stream_key = stream_key_bytes.decode() if isinstance(stream_key_bytes, bytes) else stream_key_bytes
            # Find which agent this stream belongs to.
            matched_full_name = None
            for full_name, info in pending.items():
                if info["stream_key"] == stream_key:
                    matched_full_name = full_name
                    break
            if not matched_full_name:
                continue

            for entry_id, fields in entries:
                pending[matched_full_name]["last_id"] = entry_id.decode() if isinstance(entry_id, bytes) else entry_id
                event_type = fields.get(b"type", fields.get("type", b""))
                if isinstance(event_type, bytes):
                    event_type = event_type.decode()
                if event_type in terminal_types:
                    payload_str = fields.get(b"payload", fields.get("payload", b"{}"))
                    if isinstance(payload_str, bytes):
                        payload_str = payload_str.decode()
                    results[pending[matched_full_name]["display_name"]] = {
                        "status": event_type,
                        "payload": json.loads(payload_str),
                    }
                    del pending[matched_full_name]
                    break

    # Any still pending after timeout.
    for full_name, info in pending.items():
        results[info["display_name"]] = {"status": "timeout", "payload": {"error": f"Agent did not complete within {timeout_seconds}s"}}

    # Fetch the last few events for each completed agent (for context).
    for name in names:
        full_name = _sub_name(name)
        stream_key = f"{_stream_prefix}:{full_name}"
        try:
            recent = _redis_client.xrevrange(stream_key, "+", "-", count=5)
            recent_events = []
            for _, fields in reversed(recent):
                etype = fields.get(b"type", fields.get("type", b""))
                if isinstance(etype, bytes):
                    etype = etype.decode()
                pstr = fields.get(b"payload", fields.get("payload", b"{}"))
                if isinstance(pstr, bytes):
                    pstr = pstr.decode()
                recent_events.append({"type": etype, "payload": json.loads(pstr)})
            if name in results:
                results[name]["recent_events"] = recent_events
        except redis.RedisError:
            pass

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
