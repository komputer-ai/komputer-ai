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


def _install_skills() -> list[HookMatcher] | None:
    """Install skills from the SKILLS_DIR as pre-tool-use hooks."""
    import logging
    if not SKILLS_DIR.exists():
        logging.info("skills: no skills directory found")
        return None

    hook_matchers = []
    for skill_dir in SKILLS_DIR.iterdir():
        if not skill_dir.is_dir():
            continue
        instructions_file = skill_dir / "SKILL.md"
        if instructions_file.exists():
            skill_text = instructions_file.read_text()
            hook_matchers.append(HookMatcher(
                tool_name=None,
                hook_type="pre_tool_use",
                instructions=f"<skill name='{skill_dir.name}'>\n{skill_text}\n</skill>",
            ))
            logging.info(f"skills: installed {skill_dir.name}")
    return hook_matchers or None


async def run_agent(instructions: str, model: str, publisher, system_prompt: str = None):
    """Run the Claude Agent SDK with the given instructions and publish events."""
    import state

    # Build options
    options = ClaudeAgentOptions(
        model=model,
    )

    if system_prompt:
        options.system_prompt = system_prompt

    # Install skills as hooks
    hooks = _install_skills()
    if hooks:
        options.hook_matchers = hooks

    # Check for MCP config to enable all tools
    mcp_config = Path(os.environ.get("CLAUDE_CONFIG_DIR", Path.home() / ".claude")) / "mcp_servers.json"
    if mcp_config.exists():
        import json, logging
        try:
            config = json.loads(mcp_config.read_text())
            if config:
                logging.info(f"MCP config found with servers: {list(config.keys())}")
                options.mcp_servers = config
        except Exception as e:
            import logging
            logging.warning(f"Failed to read MCP config: {e}")

    # Check for permission prompt tool config
    permission_file = Path(os.environ.get("CLAUDE_CONFIG_DIR", Path.home() / ".claude")) / ".permissions"
    if permission_file.exists():
        import json, logging
        try:
            permissions = json.loads(permission_file.read_text())
            allow_list = permissions.get("allow", [])
            if allow_list:
                options.permission_prompt_tool_allowlist = allow_list
                logging.info(f"Loaded {len(allow_list)} allowed tools from permissions")
        except Exception as e:
            logging.warning(f"Failed to read permissions: {e}")

    # Load existing session ID for continuation
    session_id = _load_session_id()
    if session_id:
        options.resume = session_id

    from prompts import build_prompt
    full_prompt = build_prompt(instructions)

    # Create a fresh queue for this session and register it in shared state
    # so server.py can push steer messages from the FastAPI thread.
    session_steer_queue: asyncio.Queue = asyncio.Queue()
    state.steer_queue = session_steer_queue

    # Event signalled when receive_response() is done — tells the generator
    # to stop blocking on the queue and exit cleanly.
    session_done = asyncio.Event()

    async def message_generator():
        """Yield the initial task, then yield each follow-up steer message in order.

        Races queue.get() against session_done so we never deadlock: if the SDK
        finishes processing (receive_response yields ResultMessage) while the
        generator is waiting for a steer, session_done fires and we return.
        """
        yield {"type": "user", "message": {"role": "user", "content": full_prompt}}
        while True:
            # Race: either a steer message arrives or the session completes.
            get_task = asyncio.ensure_future(session_steer_queue.get())
            done_task = asyncio.ensure_future(session_done.wait())
            done, pending = await asyncio.wait(
                {get_task, done_task}, return_when=asyncio.FIRST_COMPLETED
            )
            for t in pending:
                t.cancel()
            if done_task in done:
                # Session finished — no more messages to yield.
                return
            msg = get_task.result()
            if msg is None:
                return
            yield {"type": "user", "message": {"role": "user", "content": msg}}

    result = None

    async with ClaudeSDKClient(options=options) as client:
        # Register the client so signal handlers can interrupt it.
        state.set_active_client(client)

        # Streaming input mode: pass the async generator so the SDK can pull
        # follow-up messages on demand without restarting the session.
        await client.query(message_generator())

        last_usage = None  # Track last assistant message's usage for context size
        async for message in client.receive_response():
            if isinstance(message, AssistantMessage):
                usage = message.usage  # dict | None, keys: input_tokens, output_tokens, cache_*
                last_usage = usage
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

        # Signal the generator to stop waiting for steers and exit.
        session_done.set()

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
                "last_usage": last_usage,
                "context_window": _fetch_context_window(model),
            })

    # Clear the queue reference so server.py won't try to push to a dead queue.
    state.steer_queue = None

    return result


def run_agent_sync(instructions: str, model: str, publisher):
    """Synchronous wrapper for run_agent, for use in threads."""
    asyncio.run(run_agent(instructions, model, publisher))
