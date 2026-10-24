"""Tests for agent._serialize_tool_output, used by the PostToolUse hook.

str(tool_response) on a dict/list produces Python repr (single quotes,
True/False/None) instead of JSON, which breaks the UI's JSON viewer and
differs from what's stored in the backfilled session JSONL. These tests
pin the fix: dicts/lists are JSON-serialized, strings pass through
unchanged, and truncation to TOOL_OUTPUT_CAP happens after serialization.
"""
import json

import agent

CAP = agent.TOOL_OUTPUT_CAP


def test_dict_is_serialized_as_valid_json():
    output, truncated = agent._serialize_tool_output(
        {"ok": True, "count": 3, "note": None}
    )
    assert "'" not in output
    assert json.loads(output) == {"ok": True, "count": 3, "note": None}
    assert truncated is False


def test_list_is_serialized_as_valid_json():
    output, truncated = agent._serialize_tool_output([1, "two", False, None])
    assert json.loads(output) == [1, "two", False, None]
    assert truncated is False


def test_nested_dict_round_trips():
    value = {"items": [{"id": 1, "active": True}, {"id": 2, "active": False}]}
    output, truncated = agent._serialize_tool_output(value)
    assert json.loads(output) == value
    assert truncated is False


def test_string_passes_through_unchanged():
    output, truncated = agent._serialize_tool_output("plain text output")
    assert output == "plain text output"
    assert truncated is False


def test_non_string_non_collection_falls_back_to_str():
    output, truncated = agent._serialize_tool_output(42)
    assert output == "42"
    assert truncated is False


def test_long_dict_is_truncated_after_serialization():
    value = {"data": "x" * (CAP * 2)}
    output, truncated = agent._serialize_tool_output(value)
    assert len(output) <= CAP
    assert truncated is True
    # The long value is shortened inside the structure, so the result is still JSON.
    assert json.loads(output)["data"].endswith("…")


def test_long_string_is_truncated():
    output, truncated = agent._serialize_tool_output("y" * (CAP + 500))
    assert len(output) == CAP
    assert truncated is True


def test_exactly_cap_chars_is_not_truncated():
    output, truncated = agent._serialize_tool_output("z" * CAP)
    assert len(output) == CAP
    assert truncated is False


def test_truncated_dict_stays_valid_json():
    # The common case: a Bash result whose stdout alone exceeds the cap. Cutting
    # the serialized blob would hand the UI unparseable JSON and lose the
    # pretty-printing; long string values are shortened instead.
    bash = {"stdout": "line of build output\n" * (CAP // 10), "stderr": "", "interrupted": False}
    output, truncated = agent._serialize_tool_output(bash)
    assert truncated is True
    assert len(output) <= CAP
    parsed = json.loads(output)  # must still parse
    assert set(parsed) == {"stdout", "stderr", "interrupted"}
    assert parsed["interrupted"] is False
    assert parsed["stdout"].endswith("…")
    assert parsed["stdout"].startswith("line of build output")


def test_truncated_nested_structure_stays_valid_json():
    value = {"items": [{"name": f"n{i}", "body": "x" * (CAP // 2)} for i in range(5)]}
    output, truncated = agent._serialize_tool_output(value)
    assert truncated is True
    assert len(output) <= CAP
    assert len(json.loads(output)["items"]) == 5


def test_typical_bash_output_is_not_truncated():
    # A few KB of stdout (well past the old 1000-char cap) must arrive in full.
    bash = {"stdout": "line of build output\n" * 300, "stderr": "", "interrupted": False}
    output, truncated = agent._serialize_tool_output(bash)
    assert truncated is False
    assert json.loads(output) == bash
