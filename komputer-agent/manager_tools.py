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
    description="Create a sub-agent to handle a task. Default role is 'worker' (Bash+WebSearch only). Set role='manager' for complex tasks that need their own sub-agents. Use lifecycle='AutoDelete' for one-shot tasks.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Unique name for the sub-agent (lowercase, hyphens, no spaces). Used for all operations (wait, status, delete)."},
            "instructions": {"type": "string", "description": "Detailed task instructions for the sub-agent."},
            "role": {"type": "string", "enum": ["worker", "manager"], "description": "Agent role. 'worker' (default) has Bash+WebSearch only. 'manager' can create its own sub-agents."},
            "lifecycle": {"type": "string", "enum": ["", "Sleep", "AutoDelete"], "description": "Post-task behavior. Empty=pod stays running, 'Sleep'=pod deleted/PVC kept, 'AutoDelete'=everything deleted."},
            "model": {"type": "string", "description": "Claude model override (optional)."},
            "templateRef": {"type": "string", "description": "Pod template name (optional, defaults to 'default')."},
            "systemPrompt": {"type": "string", "description": "Custom system prompt defining the sub-agent's behavior, persona, or constraints (optional)."},
        },
        "required": ["name", "instructions"],
    },
)
async def create_agent(args):
    full_name = _sanitize_name(args["name"])
    payload = {
        "name": full_name,
        "instructions": args["instructions"],
        "role": args.get("role", "worker"),
        "namespace": NAMESPACE,
    }

    if args.get("lifecycle"):
        payload["lifecycle"] = args["lifecycle"]
    if args.get("model"):
        payload["model"] = args["model"]
    if args.get("templateRef"):
        payload["templateRef"] = args["templateRef"]
    if args.get("systemPrompt"):
        payload["systemPrompt"] = args["systemPrompt"]

    # Identify this agent as the office manager so the operator can create/join an Office.
    manager_name = os.environ.get("KOMPUTER_AGENT_NAME", "")
    if manager_name:
        payload["officeManager"] = manager_name

    # Secrets are inherited automatically by the operator from the office manager.

    return await _request("POST", "/api/v1/agents", timeout=30, json=payload)


@tool(
    name="schedule_agent",
    description="Schedule a recurring or one-time agent task on a cron schedule. Creates a KomputerSchedule that triggers an agent at scheduled times. Use for reminders, periodic reports, monitoring, etc.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Unique name for the schedule (lowercase, hyphens, no spaces)."},
            "schedule": {"type": "string", "description": "Cron expression (5-field: min hour dom month dow). Examples: '0 9 * * MON-FRI' (weekday 9am), '*/30 * * * *' (every 30min), '0 0 1 * *' (monthly)."},
            "instructions": {"type": "string", "description": "Task instructions for the agent on each scheduled run."},
            "timezone": {"type": "string", "description": "IANA timezone (e.g. 'America/New_York', 'Asia/Jerusalem'). Defaults to UTC."},
            "auto_delete": {"type": "boolean", "description": "If true, the schedule deletes itself after the first successful run. Use for one-time future tasks."},
            "keep_agents": {"type": "boolean", "description": "When auto_delete is true, keep managed agents alive after the schedule deletes itself."},
            "agent_name": {"type": "string", "description": "Reference an existing agent instead of creating a new one. Use your own agent name to schedule a task for yourself."},
            "lifecycle": {"type": "string", "enum": ["", "Sleep", "AutoDelete"], "description": "Agent lifecycle. Defaults to 'Sleep' (recommended for schedules)."},
            "role": {"type": "string", "enum": ["worker", "manager"], "description": "Agent role. Default: 'worker'."},
            "model": {"type": "string", "description": "Claude model override."},
        },
        "required": ["name", "schedule", "instructions"],
    },
)
async def schedule_agent(args):
    sanitized = _sanitize_name(args["name"])
    payload = {
        "name": sanitized,
        "schedule": args["schedule"],
        "namespace": NAMESPACE,
    }

    if args.get("timezone"):
        payload["timezone"] = args["timezone"]
    if args.get("auto_delete") is not None:
        payload["autoDelete"] = args["auto_delete"]
    if args.get("keep_agents") is not None:
        payload["keepAgents"] = args["keep_agents"]

    if args.get("agent_name"):
        # Reference an existing agent.
        payload["agentName"] = _sanitize_name(args["agent_name"])
    else:
        # Build an agent spec for a new agent.
        agent_spec = {
            "lifecycle": args.get("lifecycle", "Sleep"),
            "role": args.get("role", "worker"),
        }
        if args.get("model"):
            agent_spec["model"] = args["model"]
        # Secrets are inherited automatically by the operator from the office manager.
        payload["agent"] = agent_spec

    # Instructions live at top level on the schedule.
    payload["instructions"] = args["instructions"]

    return await _request("POST", "/api/v1/schedules", timeout=30, json=payload)


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
    name="cancel_agent",
    description="Cancel a sub-agent's running task. Stops the current task but keeps the agent alive — you can send it a new task afterwards with create_agent using the same name. Use when a sub-agent is stuck, taking too long, or you want to redirect it.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The sub-agent name (as passed to create_agent)"},
        },
        "required": ["name"],
    },
)
async def cancel_agent(args):
    full_name = _sanitize_name(args["name"])
    return await _request("POST", f"/api/v1/agents/{full_name}/cancel")


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


@tool(
    name="delete_schedule",
    description="Delete a schedule and its managed agents. Use to stop recurring tasks.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "The schedule name (as passed to schedule_agent)"},
        },
        "required": ["name"],
    },
)
async def delete_schedule(args):
    full_name = _sanitize_name(args["name"])
    return await _request("DELETE", f"/api/v1/schedules/{full_name}")


@tool(
    name="create_memory",
    description="Create a KomputerMemory resource containing reusable knowledge (notes, context, instructions). Optionally attach it to yourself so it's included in your system prompt on next wake.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Unique name for the memory (lowercase, hyphens, no spaces)."},
            "content": {"type": "string", "description": "The memory content (markdown text, instructions, notes, etc.)."},
            "description": {"type": "string", "description": "Short human-readable description of what this memory contains."},
            "attach": {"type": "boolean", "description": "If true, also attach this memory to the current agent."},
        },
        "required": ["name", "content"],
    },
)
async def create_memory(args):
    sanitized = _sanitize_name(args["name"])
    payload = {
        "name": sanitized,
        "content": args["content"],
        "namespace": NAMESPACE,
    }
    if args.get("description"):
        payload["description"] = args["description"]

    result = await _request("POST", "/api/v1/memories", timeout=30, json=payload)
    if result.get("isError"):
        return result

    if args.get("attach"):
        self_name = os.environ.get("KOMPUTER_AGENT_NAME", "")
        if not self_name:
            return _err("Memory created but could not attach: KOMPUTER_AGENT_NAME not set.")
        # Get current memories, then append.
        get_result = await _request("GET", f"/api/v1/agents/{self_name}")
        if get_result.get("isError"):
            return _err(f"Memory created but failed to fetch agent for attach: {get_result['content'][0]['text']}")
        agent_data = json.loads(get_result["content"][0]["text"])
        current_memories = agent_data.get("memories") or []
        if sanitized not in current_memories:
            current_memories.append(sanitized)
        patch_result = await _request("PATCH", f"/api/v1/agents/{self_name}", timeout=30, json={"memories": current_memories})
        if patch_result.get("isError"):
            return _err(f"Memory created but attach failed: {patch_result['content'][0]['text']}")
        return _ok(f"Memory '{sanitized}' created and attached to '{self_name}'.")

    return result


@tool(
    name="attach_memory",
    description="Attach an existing KomputerMemory to an agent so it's included in the agent's system prompt on next wake.",
    input_schema={
        "type": "object",
        "properties": {
            "memory_name": {"type": "string", "description": "Name of the KomputerMemory to attach."},
            "agent_name": {"type": "string", "description": "Agent to attach the memory to. Defaults to the current agent."},
        },
        "required": ["memory_name"],
    },
)
async def attach_memory(args):
    memory = _sanitize_name(args["memory_name"])
    agent = args.get("agent_name")
    if agent:
        agent = _sanitize_name(agent)
    else:
        agent = os.environ.get("KOMPUTER_AGENT_NAME", "")
        if not agent:
            return _err("No agent_name provided and KOMPUTER_AGENT_NAME not set.")

    # Get current memories, then append.
    get_result = await _request("GET", f"/api/v1/agents/{agent}")
    if get_result.get("isError"):
        return get_result
    agent_data = json.loads(get_result["content"][0]["text"])
    current_memories = agent_data.get("memories") or []
    if memory in current_memories:
        return _ok(f"Memory '{memory}' is already attached to '{agent}'.")
    current_memories.append(memory)

    return await _request("PATCH", f"/api/v1/agents/{agent}", timeout=30, json={"memories": current_memories})


@tool(
    name="create_skill",
    description="Create a KomputerSkill resource containing a reusable skill (instructions, workflows, tool usage patterns). Optionally attach it to yourself so it's available in your next task.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Unique name for the skill (lowercase, hyphens, no spaces)."},
            "content": {"type": "string", "description": "The skill content (markdown text describing the skill workflow or instructions)."},
            "description": {"type": "string", "description": "Short human-readable description of what this skill does."},
            "attach": {"type": "boolean", "description": "If true, also attach this skill to the current agent."},
        },
        "required": ["name", "content"],
    },
)
async def create_skill(args):
    sanitized = _sanitize_name(args["name"])
    payload = {
        "name": sanitized,
        "content": args["content"],
        "namespace": NAMESPACE,
    }
    if args.get("description"):
        payload["description"] = args["description"]

    result = await _request("POST", "/api/v1/skills", timeout=30, json=payload)
    if result.get("isError"):
        return result

    if args.get("attach"):
        self_name = os.environ.get("KOMPUTER_AGENT_NAME", "")
        if not self_name:
            return _err("Skill created but could not attach: KOMPUTER_AGENT_NAME not set.")
        # Get current skills, then append.
        get_result = await _request("GET", f"/api/v1/agents/{self_name}")
        if get_result.get("isError"):
            return _err(f"Skill created but failed to fetch agent for attach: {get_result['content'][0]['text']}")
        agent_data = json.loads(get_result["content"][0]["text"])
        current_skills = agent_data.get("skills") or []
        if sanitized not in current_skills:
            current_skills.append(sanitized)
        patch_result = await _request("PATCH", f"/api/v1/agents/{self_name}", timeout=30, json={"skills": current_skills})
        if patch_result.get("isError"):
            return _err(f"Skill created but attach failed: {patch_result['content'][0]['text']}")
        return _ok(f"Skill '{sanitized}' created and attached to '{self_name}'.")

    return result


@tool(
    name="attach_skill",
    description="Attach an existing KomputerSkill to an agent so it's available in the agent's next task.",
    input_schema={
        "type": "object",
        "properties": {
            "skill_name": {"type": "string", "description": "Name of the KomputerSkill to attach."},
            "agent_name": {"type": "string", "description": "Agent to attach the skill to. Defaults to the current agent."},
        },
        "required": ["skill_name"],
    },
)
async def attach_skill(args):
    skill = _sanitize_name(args["skill_name"])
    agent = args.get("agent_name")
    if agent:
        agent = _sanitize_name(agent)
    else:
        agent = os.environ.get("KOMPUTER_AGENT_NAME", "")
        if not agent:
            return _err("No agent_name provided and KOMPUTER_AGENT_NAME not set.")

    # Get current skills, then append.
    get_result = await _request("GET", f"/api/v1/agents/{agent}")
    if get_result.get("isError"):
        return get_result
    agent_data = json.loads(get_result["content"][0]["text"])
    current_skills = agent_data.get("skills") or []
    if skill in current_skills:
        return _ok(f"Skill '{skill}' is already attached to '{agent}'.")
    current_skills.append(skill)

    return await _request("PATCH", f"/api/v1/agents/{agent}", timeout=30, json={"skills": current_skills})


def create_manager_server():
    """Create the MCP server with manager orchestration tools."""
    return create_sdk_mcp_server(
        name="komputer",
        tools=[create_agent, schedule_agent, get_agent_status, get_agent_events, cancel_agent, delete_agent, delete_schedule, create_memory, attach_memory, create_skill, attach_skill],
    )
