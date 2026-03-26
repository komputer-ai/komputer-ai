import asyncio
import os
from pathlib import Path

from claude_agent_sdk import (
    AssistantMessage,
    ClaudeAgentOptions,
    HookMatcher,
    ResultMessage,
    StreamEvent,
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

    is_manager = os.environ.get("KOMPUTER_ROLE") == "manager"

    options = ClaudeAgentOptions(
        tools=["Bash", "WebSearch"],
        permission_mode="bypassPermissions",
        model=model,
        cwd="/workspace",
        # Enable streaming for real-time events. Disabled for managers
        # due to incompatibility with MCP servers in current SDK version.
        include_partial_messages=not is_manager,
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

    result = None
    # Track which content we've already published via StreamEvent
    # so we don't duplicate when the full AssistantMessage arrives.
    streamed_turn = False

    async for message in query(prompt=instructions, options=options):
        if isinstance(message, StreamEvent):
            # Real-time streaming event from Claude — publish immediately.
            evt = message.event
            evt_type = evt.get("type", "")

            if evt_type == "content_block_start":
                block = evt.get("content_block", {})
                block_type = block.get("type", "")
                if block_type == "text":
                    # Text block starting — content comes in deltas
                    pass
                elif block_type == "thinking":
                    pass
                elif block_type == "tool_use":
                    publisher.publish("tool_call", {
                        "id": block.get("id", ""),
                        "tool": block.get("name", ""),
                        "input": {},
                    })
                    streamed_turn = True

            elif evt_type == "content_block_stop":
                block = evt.get("content_block", {})
                block_type = block.get("type", "")
                if block_type == "text" and block.get("text"):
                    publisher.publish("text", {"content": block["text"]})
                    streamed_turn = True
                elif block_type == "thinking" and block.get("thinking"):
                    publisher.publish("thinking", {"content": block["thinking"][:500]})
                    streamed_turn = True

        elif isinstance(message, AssistantMessage):
            # Full turn message — only publish if we didn't already stream it.
            if not streamed_turn:
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
            streamed_turn = False

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
