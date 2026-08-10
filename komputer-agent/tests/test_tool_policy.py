"""Unit tests for the agent tool allow/deny policy."""
import pytest

from tool_policy import DEFAULT_TOOLS, is_allowed, matches_pattern, parse_tool_list


def test_default_tools_is_exactly_todays_list():
    assert DEFAULT_TOOLS == [
        "Bash", "WebSearch", "WebFetch", "Read", "Write", "Edit", "Glob", "Grep", "Skill"
    ]


@pytest.mark.parametrize("tool,pattern,expected", [
    ("Read", "Read", True),
    ("Read", "Write", False),
    ("read", "Read", False),                                  # case-sensitive
    ("mcp__figma__get_design_context", "mcp__figma__*", True),
    ("mcp__figma__get_design_context", "mcp__figma__get_design_context", True),
    ("mcp__notion__search", "mcp__figma__*", False),
    ("Bash", "*", True),
    ("mcp__figma__x", "mcp__figma__y", False),
])
def test_matches_pattern(tool, pattern, expected):
    assert matches_pattern(tool, pattern) is expected


def test_empty_allowlist_permits_everything():
    assert is_allowed("Bash", []) is True
    assert is_allowed("mcp__figma__anything", []) is True


def test_allowlist_permits_only_listed_tools():
    allowed = ["Read", "mcp__figma__*"]
    assert is_allowed("Read", allowed) is True
    assert is_allowed("mcp__figma__get_design_context", allowed) is True
    assert is_allowed("Bash", allowed) is False
    assert is_allowed("mcp__notion__search", allowed) is False


@pytest.mark.parametrize("raw", [None, "", "   ", "not json", "{}", "[1, 2]", '"Read"'])
def test_parse_tool_list_falls_back_to_empty(raw):
    assert parse_tool_list(raw) == []


def test_parse_tool_list_reads_json_array():
    assert parse_tool_list('["Read", "Grep"]') == ["Read", "Grep"]


def test_parse_tool_list_strips_blank_entries():
    assert parse_tool_list('["Read", "", "  ", "Grep"]') == ["Read", "Grep"]
