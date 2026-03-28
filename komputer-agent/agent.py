import asyncio
import os
from pathlib import Path

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


def _load_session_id() -> str | None:
    """Load the last session ID from the workspace."""
    try:
        return SESSION_FILE.read_text().strip() or None
    except FileNotFoundError:
        return None


def _save_session_id(session_id: str):
    """Save the session ID to the workspace for future tasks."""
    SESSION_FILE.write_text(session_id)


async def run_agent(instructions: str, model: str, publisher):
    """Run a Claude agent with the given instructions using the Claude Agent SDK."""
    import state
    session_id = _load_session_id()

    # Extract just the user's task for the event (strip system prompt).
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
        options.mcp_servers = {"komputer": create_manager_server()}

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
