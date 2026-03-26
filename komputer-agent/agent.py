import asyncio
import json
import os
from pathlib import Path

from claude_agent_sdk import (
    AssistantMessage,
    ClaudeAgentOptions,
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


def _load_session_id() -> str | None:
    """Load the last session ID from the workspace."""
    try:
        return SESSION_FILE.read_text().strip() or None
    except FileNotFoundError:
        return None


def _save_session_id(session_id: str):
    """Save the session ID to the workspace for future tasks."""
    SESSION_FILE.write_text(session_id)


def _load_redis_config() -> dict | None:
    """Load Redis config from the agent's config file."""
    config_path = os.getenv("KOMPUTER_CONFIG_PATH", "/etc/komputer/config.json")
    try:
        with open(config_path) as f:
            return json.load(f).get("redis")
    except (FileNotFoundError, json.JSONDecodeError):
        return None


async def run_agent(instructions: str, model: str, publisher):
    """Run a Claude agent with the given instructions using the Claude Agent SDK."""
    session_id = _load_session_id()

    publisher.publish("task_started", {
        "instructions": instructions,
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
        tools=["Bash", "WebSearch"],
        permission_mode="bypassPermissions",
        model=model,
        cwd="/workspace",
        hooks={
            "PostToolUse": [
                HookMatcher(matcher=None, hooks=[post_tool_hook]),
            ],
        },
    )

    # Conditionally register manager orchestration tools
    if os.environ.get("KOMPUTER_ROLE") == "manager":
        from manager_tools import create_manager_server
        # Pass Redis config so wait_for_completion can subscribe to streams directly.
        redis_config = _load_redis_config()
        options.mcp_servers = {"komputer": create_manager_server(redis_config)}

    # Resume previous session if one exists
    if session_id:
        options.resume = session_id

    result = None
    async for message in query(prompt=instructions, options=options):
        if isinstance(message, AssistantMessage):
            for block in message.content:
                if isinstance(block, TextBlock):
                    publisher.publish("text", {"content": block.text})
                elif isinstance(block, ThinkingBlock):
                    publisher.publish("thinking", {"content": block.thinking[:500]})
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
            "result": result.result or "",
            "cost_usd": result.total_cost_usd,
            "duration_ms": result.duration_ms,
            "turns": result.num_turns,
            "stop_reason": result.stop_reason,
            "session_id": result.session_id,
        })

    return result


def run_agent_sync(instructions: str, model: str, publisher):
    """Synchronous wrapper for run_agent, for use in threads."""
    asyncio.run(run_agent(instructions, model, publisher))
