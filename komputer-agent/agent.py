import asyncio

from claude_agent_sdk import (
    AssistantMessage,
    ClaudeAgentOptions,
    HookMatcher,
    ResultMessage,
    query,
)


async def run_agent(instructions: str, model: str, publisher):
    """Run a Claude agent with the given instructions using the Claude Agent SDK."""
    publisher.publish("message", {"content": f"Starting task: {instructions[:100]}"})

    async def post_tool_hook(input, session_id, ctx):
        publisher.publish(
            "tool_call",
            {
                "tool": input.get("tool_name", ""),
                "input": str(input.get("tool_input", ""))[:500],
                "output": str(input.get("tool_response", ""))[:500],
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

    result = None
    async for message in query(prompt=instructions, options=options):
        if isinstance(message, AssistantMessage):
            content = str(message.content)[:1000]
            publisher.publish("message", {"content": content})
        elif isinstance(message, ResultMessage):
            result = message

    if result:
        publisher.publish("completion", {"result": str(result)[:2000]})

    return result


def run_agent_sync(instructions: str, model: str, publisher):
    """Synchronous wrapper for run_agent, for use in threads."""
    asyncio.run(run_agent(instructions, model, publisher))
