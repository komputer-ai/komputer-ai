"""Tests for agent._serialize_tool_output, used by the PostToolUse hook.

str(tool_response) on a dict/list produces Python repr (single quotes,
True/False/None) instead of JSON, which breaks the UI's JSON viewer and
differs from what's stored in the backfilled session JSONL. These tests
pin the fix: dicts/lists are JSON-serialized, strings pass through
unchanged, and truncation to 1000 chars happens after serialization.
"""
import json

import agent


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
    value = {"data": "x" * 2000}
    output, truncated = agent._serialize_tool_output(value)
    assert len(output) == 1000
    assert truncated is True
    # Truncation happens post-serialization, so it still starts as valid JSON prefix.
    assert output.startswith('{"data": "')


def test_long_string_is_truncated():
    output, truncated = agent._serialize_tool_output("y" * 1500)
    assert len(output) == 1000
    assert truncated is True


def test_exactly_1000_chars_is_not_truncated():
    output, truncated = agent._serialize_tool_output("z" * 1000)
    assert len(output) == 1000
    assert truncated is False
