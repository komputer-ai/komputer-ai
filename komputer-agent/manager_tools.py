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
            "priority": {"type": "integer", "description": "Queue priority. Higher = admitted first when the template's maxConcurrentAgents cap is reached. Default: 0."},
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
    if args.get("priority") is not None:
        payload["priority"] = args["priority"]

    # Identify this agent as the office manager so the operator can create/join an Office.
    # The API inherits connectors and the operator inherits secrets from the manager automatically.
    manager_name = os.environ.get("KOMPUTER_AGENT_NAME", "")
    if manager_name:
        payload["officeManager"] = manager_name

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


@tool(
    name="update_agent",
    description="Update a sub-agent's spec (model, instructions, cpu, memory, storage, image). Changes apply to the next pod start — running pods are not mutated. Use Sleep+wake if you want changes to take effect now. To remove an override and revert to the template default, pass the field as an empty string (e.g. cpu='' or storage='').",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Sub-agent name (as passed to create_agent)."},
            "instructions": {"type": "string", "description": "New task instructions (optional)."},
            "model": {"type": "string", "description": "Override Claude model (optional)."},
            "cpu": {"type": "string", "description": "CPU (e.g. '2' or '500m'). Sets both requests and limits. Empty string clears the resources override."},
            "memory": {"type": "string", "description": "Memory (e.g. '4Gi'). Sets both requests and limits. Empty string clears the resources override."},
            "storage": {"type": "string", "description": "PVC size (e.g. '20Gi'). Empty string clears the storage override."},
            "image": {"type": "string", "description": "Override agent container image. Empty string clears the resources override."},
        },
        "required": ["name"],
    },
)
async def update_agent(args):
    full_name = _sanitize_name(args["name"])
    payload = {}
    if args.get("instructions"):
        payload["instructions"] = args["instructions"]
    if args.get("model"):
        payload["model"] = args["model"]

    # Storage: empty string ("") = clear; non-empty = set; missing key = no change.
    if "storage" in args:
        payload["storage"] = {} if args["storage"] == "" else {"size": args["storage"]}

    # Resources/image: any of cpu/memory/image present with empty value = clear;
    # any present with a value = build override. Mixing is treated as clear (safer default).
    keys = [k for k in ("cpu", "memory", "image") if k in args]
    if keys:
        if any(args[k] == "" for k in keys):
            payload["podSpec"] = {}
        else:
            container = {"name": "agent"}
            if args.get("image"):
                container["image"] = args["image"]
            if args.get("cpu") or args.get("memory"):
                rl = {}
                if args.get("cpu"):
                    rl["cpu"] = args["cpu"]
                if args.get("memory"):
                    rl["memory"] = args["memory"]
                # Same value for both requests and limits.
                container["resources"] = {"requests": dict(rl), "limits": dict(rl)}
            payload["podSpec"] = {"containers": [container]}

    if not payload:
        return _err("update_agent requires at least one field to update")
    return await _request("PATCH", f"/api/v1/agents/{full_name}", timeout=30, json=payload)


@tool(
    name="sleep_agent",
    description="Put an agent to sleep — pod is deleted, PVC (workspace) is preserved. The agent can be woken later with a new task. Defaults to the current agent if no name is given.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Agent to sleep. Defaults to the current agent."},
        },
    },
)
async def sleep_agent(args):
    name = args.get("name")
    if name:
        name = _sanitize_name(name)
    else:
        name = os.environ.get("KOMPUTER_AGENT_NAME", "")
        if not name:
            return _err("No name provided and KOMPUTER_AGENT_NAME not set.")
    return await _request("PATCH", f"/api/v1/agents/{name}", timeout=30, json={"lifecycle": "Sleep"})


@tool(
    name="wake_agent",
    description="Wake a sleeping agent (or trigger a new task on a running one) by sending it instructions. Equivalent to calling create_agent with the same name — the API routes the call to the existing agent.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Agent to wake."},
            "instructions": {"type": "string", "description": "Task instructions for this run."},
        },
        "required": ["name", "instructions"],
    },
)
async def wake_agent(args):
    if not args.get("instructions"):
        return _err("wake_agent requires 'instructions'.")
    name = _sanitize_name(args["name"])
    payload = {"name": name, "instructions": args["instructions"], "namespace": NAMESPACE}
    return await _request("POST", "/api/v1/agents", timeout=30, json=payload)


@tool(
    name="list_agents",
    description="List all agents in the current namespace, with name, status, lifecycle, model, and last task summary.",
    input_schema={"type": "object", "properties": {}},
)
async def list_agents(args):
    return await _request("GET", "/api/v1/agents")


@tool(
    name="get_agent",
    description="Get full details of an agent: spec (model, instructions, skills, memories, connectors, secrets, podSpec, storage, priority) and live status (phase, taskStatus, cost, tokens). Use this to inspect a sub-agent's full configuration.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Agent name. Defaults to the current agent."},
        },
    },
)
async def get_agent(args):
    name = args.get("name")
    if name:
        name = _sanitize_name(name)
    else:
        name = os.environ.get("KOMPUTER_AGENT_NAME", "")
        if not name:
            return _err("No name provided and KOMPUTER_AGENT_NAME not set.")
    return await _request("GET", f"/api/v1/agents/{name}")


@tool(
    name="list_connectors",
    description="List configured connectors (KomputerConnector resources) in the current namespace. Each connector defines an MCP server an agent can use (Slack, GitHub, etc.). Use this before attach_connector to see what's available.",
    input_schema={"type": "object", "properties": {}},
)
async def list_connectors(args):
    return await _request("GET", "/api/v1/connectors")


@tool(
    name="list_connector_templates",
    description="List built-in connector templates (catalog of known MCP integrations like Slack, GitHub, Linear). Use this when you want to set up a new connector and want to see what's supported.",
    input_schema={"type": "object", "properties": {}},
)
async def list_connector_templates(args):
    return await _request("GET", "/api/v1/connector-templates")


@tool(
    name="get_connector",
    description="Get full details of a connector: auth type, MCP server URL, and configuration.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Connector name."},
        },
        "required": ["name"],
    },
)
async def get_connector(args):
    name = _sanitize_name(args["name"])
    return await _request("GET", f"/api/v1/connectors/{name}")


async def _agent_list_field_patch(agent_name, field_name, mutator):
    """Helper: GET agent, mutate one of its list fields, PATCH it back.

    mutator(current_list) -> (new_list_or_None, message_if_no_op)
    Returns the tool result (either the PATCH result or _ok with message).
    """
    get_result = await _request("GET", f"/api/v1/agents/{agent_name}")
    if get_result.get("isError"):
        return get_result
    agent_data = json.loads(get_result["content"][0]["text"])
    current = agent_data.get(field_name) or []
    new_list, no_op_msg = mutator(list(current))
    if new_list is None:
        return _ok(no_op_msg)
    return await _request("PATCH", f"/api/v1/agents/{agent_name}", timeout=30, json={field_name: new_list})


def _resolve_agent(args):
    """Resolve agent_name arg or fall back to KOMPUTER_AGENT_NAME. Returns (name, error_dict_or_None)."""
    agent = args.get("agent_name")
    if agent:
        return _sanitize_name(agent), None
    agent = os.environ.get("KOMPUTER_AGENT_NAME", "")
    if not agent:
        return None, _err("No agent_name provided and KOMPUTER_AGENT_NAME not set.")
    return agent, None


@tool(
    name="attach_connector",
    description="Attach a connector (MCP server) to an agent. The connector's tools become available in the agent's next task.",
    input_schema={
        "type": "object",
        "properties": {
            "connector_name": {"type": "string", "description": "Name of the connector to attach."},
            "agent_name": {"type": "string", "description": "Agent to attach to. Defaults to the current agent."},
        },
        "required": ["connector_name"],
    },
)
async def attach_connector(args):
    connector = _sanitize_name(args["connector_name"])
    agent, err = _resolve_agent(args)
    if err:
        return err

    def mutator(current):
        if connector in current:
            return None, f"Connector '{connector}' is already attached to '{agent}'."
        current.append(connector)
        return current, None

    return await _agent_list_field_patch(agent, "connectors", mutator)


@tool(
    name="detach_connector",
    description="Detach a connector from an agent. Its tools will not be available in the agent's next task.",
    input_schema={
        "type": "object",
        "properties": {
            "connector_name": {"type": "string", "description": "Name of the connector to detach."},
            "agent_name": {"type": "string", "description": "Agent to detach from. Defaults to the current agent."},
        },
        "required": ["connector_name"],
    },
)
async def detach_connector(args):
    connector = _sanitize_name(args["connector_name"])
    agent, err = _resolve_agent(args)
    if err:
        return err

    def mutator(current):
        if connector not in current:
            return None, f"Connector '{connector}' is not attached to '{agent}'."
        current.remove(connector)
        return current, None

    return await _agent_list_field_patch(agent, "connectors", mutator)


def _sanitize_secret_name(name: str) -> str:
    """Secret names are env var names — uppercase + underscores. Strip whitespace only."""
    return (name or "").strip()


@tool(
    name="list_secrets",
    description="List managed secrets (Kubernetes Secret resources tagged for komputer.ai) in the current namespace. Returns names only — values are never exposed.",
    input_schema={"type": "object", "properties": {}},
)
async def list_secrets(args):
    return await _request("GET", "/api/v1/secrets")


@tool(
    name="create_secret",
    description="Create a managed secret in the current namespace. The secret can then be attached to any agent with attach_secret. Use this when an agent needs a new API key, token, or credential.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Secret name (env var style: UPPER_SNAKE_CASE, e.g. 'GITHUB_TOKEN')."},
            "value": {"type": "string", "description": "Secret value."},
        },
        "required": ["name", "value"],
    },
)
async def create_secret(args):
    name = _sanitize_secret_name(args["name"])
    if not name:
        return _err("Secret name cannot be empty.")
    payload = {"name": name, "value": args["value"], "namespace": NAMESPACE}
    return await _request("POST", "/api/v1/secrets", timeout=30, json=payload)


@tool(
    name="delete_secret",
    description="Delete a managed secret from the current namespace. Detach it from any agents first — leaving stale references can break the agent's pod start.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Secret name."},
        },
        "required": ["name"],
    },
)
async def delete_secret(args):
    name = _sanitize_secret_name(args["name"])
    return await _request("DELETE", f"/api/v1/secrets/{name}", timeout=30)


@tool(
    name="attach_secret",
    description="Attach a secret to an agent. The secret's value will be exposed as an env var (named after the secret) in the agent's pod on next start.",
    input_schema={
        "type": "object",
        "properties": {
            "secret_name": {"type": "string", "description": "Name of the secret to attach."},
            "agent_name": {"type": "string", "description": "Agent to attach to. Defaults to the current agent."},
        },
        "required": ["secret_name"],
    },
)
async def attach_secret(args):
    secret = _sanitize_secret_name(args["secret_name"])
    agent, err = _resolve_agent(args)
    if err:
        return err

    get_result = await _request("GET", f"/api/v1/agents/{agent}")
    if get_result.get("isError"):
        return get_result
    agent_data = json.loads(get_result["content"][0]["text"])
    current = list(agent_data.get("secrets") or [])
    if secret in current:
        return _ok(f"Secret '{secret}' is already attached to '{agent}'.")
    current.append(secret)
    return await _request("PATCH", f"/api/v1/agents/{agent}", timeout=30, json={"secretRefs": current})


@tool(
    name="detach_secret",
    description="Detach a secret from an agent. The env var will not be set in the next pod start.",
    input_schema={
        "type": "object",
        "properties": {
            "secret_name": {"type": "string", "description": "Name of the secret to detach."},
            "agent_name": {"type": "string", "description": "Agent to detach from. Defaults to the current agent."},
        },
        "required": ["secret_name"],
    },
)
async def detach_secret(args):
    secret = _sanitize_secret_name(args["secret_name"])
    agent, err = _resolve_agent(args)
    if err:
        return err

    get_result = await _request("GET", f"/api/v1/agents/{agent}")
    if get_result.get("isError"):
        return get_result
    agent_data = json.loads(get_result["content"][0]["text"])
    current = list(agent_data.get("secrets") or [])
    if secret not in current:
        return _ok(f"Secret '{secret}' is not attached to '{agent}'.")
    current.remove(secret)
    return await _request("PATCH", f"/api/v1/agents/{agent}", timeout=30, json={"secretRefs": current})


@tool(
    name="list_skills",
    description="List all KomputerSkill resources in the current namespace. Use this before attach_skill to see what's available.",
    input_schema={"type": "object", "properties": {}},
)
async def list_skills(args):
    return await _request("GET", "/api/v1/skills")


@tool(
    name="get_skill",
    description="Get full details of a skill (content, description, attached agents).",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Skill name."},
        },
        "required": ["name"],
    },
)
async def get_skill(args):
    name = _sanitize_name(args["name"])
    return await _request("GET", f"/api/v1/skills/{name}")


@tool(
    name="update_skill",
    description="Update a skill's content or description. Changes apply to all attached agents on their next pod start.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Skill name."},
            "content": {"type": "string", "description": "New skill content (markdown)."},
            "description": {"type": "string", "description": "New short description."},
        },
        "required": ["name"],
    },
)
async def update_skill(args):
    name = _sanitize_name(args["name"])
    payload = {}
    if args.get("content") is not None:
        payload["content"] = args["content"]
    if args.get("description") is not None:
        payload["description"] = args["description"]
    if not payload:
        return _err("update_skill requires at least one of: content, description.")
    return await _request("PATCH", f"/api/v1/skills/{name}", timeout=30, json=payload)


@tool(
    name="delete_skill",
    description="Delete a KomputerSkill from the current namespace. Any agents that have it attached will silently lose it on next pod start. Use list_skills first if unsure.",
    input_schema={
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Skill name."},
        },
        "required": ["name"],
    },
)
async def delete_skill(args):
    name = _sanitize_name(args["name"])
    return await _request("DELETE", f"/api/v1/skills/{name}", timeout=30)


@tool(
    name="detach_skill",
    description="Detach a skill from an agent. The skill will not be loaded into the agent's next task.",
    input_schema={
        "type": "object",
        "properties": {
            "skill_name": {"type": "string", "description": "Skill name."},
            "agent_name": {"type": "string", "description": "Agent to detach from. Defaults to the current agent."},
        },
        "required": ["skill_name"],
    },
)
async def detach_skill(args):
    skill = _sanitize_name(args["skill_name"])
    agent, err = _resolve_agent(args)
    if err:
        return err

    def mutator(current):
        if skill not in current:
            return None, f"Skill '{skill}' is not attached to '{agent}'."
        current.remove(skill)
        return current, None

    return await _agent_list_field_patch(agent, "skills", mutator)


def create_manager_server():
    """Create the MCP server with manager orchestration tools."""
    return create_sdk_mcp_server(
        name="komputer",
        tools=[create_agent, schedule_agent, get_agent_status, get_agent_events, cancel_agent, delete_agent, delete_schedule, create_memory, attach_memory, create_skill, attach_skill, update_agent, sleep_agent, wake_agent, list_agents, get_agent, list_connectors, list_connector_templates, get_connector, attach_connector, detach_connector, list_secrets, create_secret, delete_secret, attach_secret, detach_secret, list_skills, get_skill, update_skill, delete_skill, detach_skill],
    )
