"""Tool allow/deny policy for agents.

Enforcement notes (verified against claude-agent-sdk 0.2.83 / CLI 2.1.146):
  - ``disallowed_tools`` genuinely removes tools, including MCP tools.
  - ``allowed_tools`` only auto-approves permission prompts; narrowing it does
    NOT deny, even under ``permission_mode="dontAsk"``. An allowlist therefore
    has to be enforced by a PreToolUse hook.
  - Deny beats allow unconditionally, so "deny mcp__x__* then allow
    mcp__x__one" yields nothing. Subsetting a connector must use the allowlist.
"""

import json
import logging

logger = logging.getLogger(__name__)

# The built-in tools komputer grants when no allowlist is configured.
# Order is preserved so the default path stays byte-for-byte as before.
DEFAULT_TOOLS = [
    "Bash", "WebSearch", "WebFetch", "Read", "Write", "Edit", "Glob", "Grep", "Skill",
]


def matches_pattern(tool_name: str, pattern: str) -> bool:
    """Match a tool name against one pattern.

    A trailing ``*`` is a prefix match (``mcp__figma__*``); anything else is an
    exact, case-sensitive match.
    """
    if pattern.endswith("*"):
        return tool_name.startswith(pattern[:-1])
    return tool_name == pattern


def is_allowed(tool_name: str, allowed: list[str]) -> bool:
    """Whether ``tool_name`` passes the allowlist.

    An empty allowlist means "unrestricted" — that is the default path.
    """
    if not allowed:
        return True
    return any(matches_pattern(tool_name, p) for p in allowed)


def parse_tool_list(raw: str | None) -> list[str]:
    """Parse a JSON array env var into a tool list.

    Returns ``[]`` for anything unusable so a malformed value degrades to
    default behavior rather than a wrongly-restricted or wrongly-denied agent.
    """
    if not raw or not raw.strip():
        return []
    try:
        parsed = json.loads(raw)
    except (json.JSONDecodeError, TypeError):
        logger.warning("ignoring malformed tool list env var", extra={"raw": raw[:200]})
        return []
    if not isinstance(parsed, list) or not all(isinstance(x, str) for x in parsed):
        logger.warning("ignoring non-string-array tool list env var", extra={"raw": raw[:200]})
        return []
    return [x.strip() for x in parsed if x.strip()]


def is_denied(tool_name: str, disallowed: list[str]) -> bool:
    """Whether ``tool_name`` matches any deny pattern."""
    return any(matches_pattern(tool_name, p) for p in disallowed)


def build_tool_options(
    allowed: list[str], disallowed: list[str], mcp_server_names: list[str]
) -> dict:
    """Compute SDK tool options from the agent's configured policy.

    With no allowlist this reproduces komputer's historical configuration
    exactly: the default built-ins plus an ``mcp__<name>__*`` entry per server.

    With an allowlist, the list fully REPLACES the defaults and the MCP
    wildcards are not auto-appended — the caller's list is the whole story.

    Deny beats allow. The SDK's ``disallowed_tools`` already enforces that, but
    denied tools are also stripped from ``tools`` and from the allowlist the
    hook enforces, so no single mechanism is load-bearing on its own.
    """
    if allowed:
        # Built-ins are whatever the user listed that isn't an MCP tool; `tools`
        # only governs built-ins, so MCP entries must not leak into it.
        effective = [t for t in allowed if not is_denied(t, disallowed)]
        builtins = [t for t in effective if not t.startswith("mcp__")]
        return {
            "tools": builtins,
            "allowed_tools": list(effective),
            "disallowed_tools": list(disallowed),
            "enforce_allowlist": True,
            "effective_allowed": effective,
        }

    tools = [t for t in DEFAULT_TOOLS if not is_denied(t, disallowed)]
    allowed_tools = list(tools)
    for name in mcp_server_names:
        allowed_tools.append(f"mcp__{name}__*")
    return {
        "tools": tools,
        "allowed_tools": allowed_tools,
        "disallowed_tools": list(disallowed),
        "enforce_allowlist": False,
        "effective_allowed": [],
    }


def make_allowlist_deny_hook(allowed: list[str]):
    """Build a PreToolUse hook that denies any tool outside ``allowed``.

    Required because the SDK's ``allowed_tools`` does not deny on its own. The
    reason string is surfaced to the model so it adapts instead of failing
    opaquely.
    """

    async def allowlist_hook(input, session_id, ctx):
        tool_name = input.get("tool_name", "")
        if is_allowed(tool_name, allowed):
            return {}
        return {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": (
                    f"{tool_name} is not in this agent's allowedTools"
                ),
            }
        }

    return allowlist_hook
