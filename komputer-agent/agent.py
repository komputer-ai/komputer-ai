import asyncio
import os
from pathlib import Path

import httpx

from claude_agent_sdk import (
    AssistantMessage,
    ClaudeAgentOptions,
    ClaudeSDKClient,
    HookMatcher,
    ResultMessage,
    TextBlock,
    ThinkingBlock,
    ToolUseBlock,
    query,
)

# Point Claude config to the workspace PVC so sessions survive pod restarts.
os.environ.setdefault("CLAUDE_CONFIG_DIR", "/workspace/.claude")

SESSION_FILE = Path("/workspace/.komputer-session")
SKILLS_DIR = Path(os.environ.get("CLAUDE_CONFIG_DIR", Path.home() / ".claude")) / "skills"


def _load_session_id() -> str | None:
    """Load the last session ID from the workspace."""
    try:
        return SESSION_FILE.read_text().strip() or None
    except FileNotFoundError:
        return None


def _save_session_id(session_id: str):
    """Save the session ID to the workspace for future tasks."""
    SESSION_FILE.write_text(session_id)


def _fetch_context_window(model: str) -> int | None:
    """Fetch the context window size for a model from the Anthropic API."""
    import logging
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        logging.warning("_fetch_context_window: ANTHROPIC_API_KEY not set")
        return None
    try:
        resp = httpx.get(
            f"https://api.anthropic.com/v1/models/{model}",
            headers={"x-api-key": api_key, "anthropic-version": "2023-06-01"},
            timeout=5,
        )
        resp.raise_for_status()
        data = resp.json()
        logging.info(f"_fetch_context_window: model={model} response={data}")
        return data.get("max_input_tokens")
    except Exception as e:
        logging.warning(f"_fetch_context_window: failed for model={model}: {e}")
        return None


def _write_skills(skills: dict[str, str | dict[str, str]]):
    """Write Claude skills to ~/.claude/skills/<name>/SKILL.md."""
    for name, skill in skills.items():
        if isinstance(skill, dict):
            content = f"---\nname: {name}\ndescription: {skill['description']}\n---\n\n{skill['content']}"
        else:
            content = skill

        skill_dir = SKILLS_DIR / name
        skill_dir.mkdir(parents=True, exist_ok=True)
        (skill_dir / "SKILL.md").write_text(content)


async def run_agent(instructions: str, model: str, publisher, system_prompt: str = None):
    """Run a Claude agent with the given instructions using the Claude Agent SDK."""
    import state
    session_id = _load_session_id()

    # Strip system prompt from the event — only show the user's task
    user_task = instructions
    task_marker = "## Your Task\n"
    if task_marker in instructions:
        user_task = instructions.split(task_marker, 1)[1]

    publisher.publish("task_started", {
        "instructions": user_task,
        "resuming_session": session_id is not None,
    })

    async def post_tool_hook(input, session_id, ctx):
        publisher.publish(
            "tool_result",
            {
                "tool": input.get("tool_name", ""),
                "input": input.get("tool_input", {}),
                "output": str(input.get("tool_response", ""))[:1000],
            },
        )
        return {}

    options = ClaudeAgentOptions(
        tools=["Bash", "WebSearch", "WebFetch", "Read", "Write", "Edit", "Glob", "Grep", "Skill"],
        allowed_tools=["Bash", "WebSearch", "WebFetch", "Read", "Write", "Edit", "Glob", "Grep", "Skill"],
        setting_sources=["user", "project"],
        permission_mode="bypassPermissions",
        model=model,
        cwd="/workspace",
        hooks={
            "PostToolUse": [
                HookMatcher(matcher=None, hooks=[post_tool_hook]),
            ],
        },
    )

    # Set system prompt via SDK (replaces previous system prompt, doesn't accumulate in history)
    if system_prompt:
        options.system_prompt = system_prompt

    # Register MCP servers: manager tools + connectors from KOMPUTER_MCP_SERVERS env.
    mcp_servers = {}
    if os.environ.get("KOMPUTER_ROLE") == "manager":
        from manager_tools import create_manager_server
        mcp_servers["komputer"] = create_manager_server()

    mcp_env = os.environ.get("KOMPUTER_MCP_SERVERS")
    if mcp_env:
        import json as _json
        try:
            for name, cfg in _json.loads(mcp_env).items():
                # Resolve tokenEnv → read env var → set Authorization header
                token_env = cfg.pop("tokenEnv", None)
                auth_type = cfg.pop("authType", "token")
                if token_env:
                    raw = os.environ.get(token_env, "")
                    if raw:
                        if auth_type == "oauth":
                            try:
                                token_data = _json.loads(raw)
                                token = token_data.get("access_token", "")
                                cfg["_oauth_connector"] = name  # stash connector name for refresh
                            except _json.JSONDecodeError:
                                token = raw  # fallback to raw
                        else:
                            token = raw
                        if token:
                            cfg["headers"] = {"Authorization": f"Bearer {token}"}
                mcp_servers[name] = cfg
        except Exception as e:
            print(f"[komputer] failed to parse KOMPUTER_MCP_SERVERS: {e}")

    if mcp_servers:
        options.mcp_servers = mcp_servers
        # Allow all MCP tools from connected servers.
        for name in mcp_servers:
            options.allowed_tools.append(f"mcp__{name}__*")
        # Log server config (redact auth tokens)
        debug_servers = {}
        for n, c in mcp_servers.items():
            if isinstance(c, dict):
                d = {k: v for k, v in c.items() if k != "headers"}
                if "headers" in c:
                    d["headers"] = {k: v[:10] + "..." for k, v in c["headers"].items()}
                debug_servers[n] = d
            else:
                debug_servers[n] = "<sdk_server>"
        print(f"[komputer] registered MCP servers: {debug_servers}")
        print(f"[komputer] allowed_tools: {options.allowed_tools}")

    # Resume previous session if one exists
    if session_id:
        options.resume = session_id

    from prompts import build_prompt
    full_prompt = build_prompt(instructions)

    result = None

    async with ClaudeSDKClient(options=options) as client:
        # Register the client so signal handlers can interrupt it.
        state.set_active_client(client)

        await client.query(full_prompt)


        async for message in client.receive_response():
            if isinstance(message, AssistantMessage):
                usage = message.usage  # dict | None, keys: input_tokens, output_tokens, cache_*
                for block in message.content:
                    if isinstance(block, TextBlock):
                        publisher.publish("text", {"content": block.text, "usage": usage})
                    elif isinstance(block, ThinkingBlock):
                        publisher.publish("thinking", {"content": block.thinking[:500], "usage": usage})
                    elif isinstance(block, ToolUseBlock):
                        publisher.publish("tool_call", {
                            "id": block.id,
                            "tool": block.name,
                            "input": block.input,
                        })
            elif isinstance(message, ResultMessage):
                result = message

        if result:
            # Persist session ID for future tasks
            _save_session_id(result.session_id)
            publisher.publish("task_completed", {
                "cost_usd": result.total_cost_usd,
                "duration_ms": result.duration_ms,
                "turns": result.num_turns,
                "stop_reason": result.stop_reason,
                "session_id": result.session_id,
                "usage": result.usage,
                "context_window": _fetch_context_window(model),
            })

    return result


def run_agent_sync(instructions: str, model: str, publisher):
    """Synchronous wrapper for run_agent, for use in threads."""
    asyncio.run(run_agent(instructions, model, publisher))
