"""Tests for how the tool policy maps onto SDK options and the deny hook."""
import asyncio

from tool_policy import DEFAULT_TOOLS, build_tool_options, make_allowlist_deny_hook


def test_default_path_is_unchanged():
    """Both fields empty must reproduce today's exact configuration."""
    opts = build_tool_options([], [], ["figma", "notion"])
    assert opts["tools"] == DEFAULT_TOOLS
    assert opts["allowed_tools"] == DEFAULT_TOOLS + ["mcp__figma__*", "mcp__notion__*"]
    assert opts["disallowed_tools"] == []
    assert opts["enforce_allowlist"] is False


def test_default_path_without_mcp_servers():
    opts = build_tool_options([], [], [])
    assert opts["tools"] == DEFAULT_TOOLS
    assert opts["allowed_tools"] == DEFAULT_TOOLS
    assert opts["enforce_allowlist"] is False


def test_disallowed_only_keeps_every_other_default_and_mcp_wildcards():
    """disallowedTools is purely subtractive: only the named tool goes away."""
    opts = build_tool_options([], ["Bash"], ["figma"])
    expected = [t for t in DEFAULT_TOOLS if t != "Bash"]
    assert opts["tools"] == expected
    assert opts["allowed_tools"] == expected + ["mcp__figma__*"]
    assert opts["disallowed_tools"] == ["Bash"]
    assert opts["enforce_allowlist"] is False
    # The other eight defaults survive — this is the low-blast-radius knob.
    assert len(opts["tools"]) == 8
    assert "Read" in opts["tools"] and "Write" in opts["tools"]


def test_disallowed_connector_tool_leaves_builtins_intact():
    opts = build_tool_options([], ["mcp__figma__use_figma"], ["figma"])
    assert opts["tools"] == DEFAULT_TOOLS
    assert opts["disallowed_tools"] == ["mcp__figma__use_figma"]
    assert "mcp__figma__*" in opts["allowed_tools"]


def test_allowed_replaces_defaults_and_skips_mcp_autoappend():
    opts = build_tool_options(["Read", "mcp__figma__get_design_context"], [], ["figma", "notion"])
    assert opts["tools"] == ["Read"]                      # built-ins only
    assert opts["allowed_tools"] == ["Read", "mcp__figma__get_design_context"]
    assert opts["enforce_allowlist"] is True


def test_allowed_with_only_mcp_entries_yields_no_builtins():
    opts = build_tool_options(["mcp__figma__*"], [], ["figma"])
    assert opts["tools"] == []
    assert opts["allowed_tools"] == ["mcp__figma__*"]
    assert opts["enforce_allowlist"] is True


def test_deny_wins_over_allow():
    opts = build_tool_options(["Read", "Bash"], ["Bash"], [])
    assert opts["disallowed_tools"] == ["Bash"]
    assert "Bash" not in opts["tools"]
    assert "Bash" not in opts["allowed_tools"]
    assert opts["effective_allowed"] == ["Read"]


def test_deny_wildcard_wins_over_specific_allow():
    """The documented trap: denying a connector wildcard beats a narrower allow."""
    opts = build_tool_options(
        ["Read", "mcp__figma__get_design_context"], ["mcp__figma__*"], ["figma"]
    )
    assert opts["effective_allowed"] == ["Read"]
    assert "mcp__figma__get_design_context" not in opts["allowed_tools"]


def test_hook_honors_deny_over_allow():
    opts = build_tool_options(["Read", "Bash"], ["Bash"], [])
    hook = make_allowlist_deny_hook(opts["effective_allowed"])
    out = _run(hook({"tool_name": "Bash"}, "sess", None))
    assert out["hookSpecificOutput"]["permissionDecision"] == "deny"


def _run(coro):
    return asyncio.run(coro)


def test_deny_hook_blocks_unlisted_tool():
    hook = make_allowlist_deny_hook(["Read"])
    out = _run(hook({"tool_name": "Bash"}, "sess", None))
    decision = out["hookSpecificOutput"]
    assert decision["hookEventName"] == "PreToolUse"
    assert decision["permissionDecision"] == "deny"
    assert "Bash" in decision["permissionDecisionReason"]


def test_deny_hook_allows_listed_tool():
    hook = make_allowlist_deny_hook(["Read", "mcp__figma__*"])
    assert _run(hook({"tool_name": "Read"}, "sess", None)) == {}
    assert _run(hook({"tool_name": "mcp__figma__get_design_context"}, "sess", None)) == {}


def test_deny_hook_blocks_other_mcp_servers():
    hook = make_allowlist_deny_hook(["mcp__figma__*"])
    out = _run(hook({"tool_name": "mcp__notion__search"}, "sess", None))
    assert out["hookSpecificOutput"]["permissionDecision"] == "deny"
