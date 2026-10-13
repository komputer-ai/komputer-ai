"""Tool-call duration measured in the agent.

Event timestamps have one-second precision, so a duration derived from them is
useless for short tool calls. The agent times each call itself — from when it
publishes tool_call to when PostToolUse fires — and sends duration_ms with the
tool_result.
"""

import agent


def test_duration_measured_from_start_and_entry_consumed():
    starts = {"toolu_1": 100.0}
    assert agent._pop_tool_duration_ms(starts, "toolu_1", now=104.4) == 4400
    assert "toolu_1" not in starts  # each start is used once


def test_unknown_tool_use_id_yields_none():
    assert agent._pop_tool_duration_ms({}, "toolu_missing", now=5.0) is None


def test_missing_tool_use_id_yields_none():
    assert agent._pop_tool_duration_ms({"toolu_1": 1.0}, None, now=5.0) is None


def test_clock_skew_never_goes_negative():
    assert agent._pop_tool_duration_ms({"toolu_1": 10.0}, "toolu_1", now=9.0) == 0
