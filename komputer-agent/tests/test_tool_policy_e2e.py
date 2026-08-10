"""End-to-end checks that the tool policy reaches the SDK and actually bites.

Marked `e2e` because these spawn the bundled Claude CLI and need credentials.
Run explicitly: pytest tests/test_tool_policy_e2e.py -m e2e
"""

import asyncio
import textwrap

import pytest
from claude_agent_sdk import ClaudeAgentOptions, ClaudeSDKClient, HookMatcher

from tool_policy import build_tool_options, make_allowlist_deny_hook

pytestmark = pytest.mark.e2e

PROBE_SERVER = textwrap.dedent('''
    import sys, json
    TOOLS = [
        {"name": "ping_alpha", "description": "Returns ALPHA-OK", "inputSchema": {"type": "object", "properties": {}}},
        {"name": "ping_beta",  "description": "Returns BETA-OK",  "inputSchema": {"type": "object", "properties": {}}},
    ]
    def send(o):
        sys.stdout.write(json.dumps(o) + "\\n"); sys.stdout.flush()
    for line in sys.stdin:
        line = line.strip()
        if not line: continue
        req = json.loads(line)
        m, rid = req.get("method"), req.get("id")
        if m == "initialize":
            send({"jsonrpc":"2.0","id":rid,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"probe","version":"1.0"}}})
        elif m == "tools/list":
            send({"jsonrpc":"2.0","id":rid,"result":{"tools":TOOLS}})
        elif m == "tools/call":
            n = req["params"]["name"]
            send({"jsonrpc":"2.0","id":rid,"result":{"content":[{"type":"text","text":"ALPHA-OK" if n=="ping_alpha" else "BETA-OK"}]}})
        elif rid is not None:
            send({"jsonrpc":"2.0","id":rid,"result":{}})
''')


@pytest.fixture
def probe_server(tmp_path):
    path = tmp_path / "srv.py"
    path.write_text(PROBE_SERVER)
    return {"probe": {"command": "python3", "args": [str(path)]}}


def _run_with_policy(prompt, allowed, disallowed, mcp_servers):
    """Run one prompt under a given policy and return the agent's text.

    Mirrors exactly what agent.py does: build the policy, apply the three SDK
    fields, and install the deny hook only when an allowlist is configured.
    """
    policy = build_tool_options(allowed, disallowed, list(mcp_servers.keys()))
    hooks = {}
    if policy["enforce_allowlist"]:
        hooks["PreToolUse"] = [
            HookMatcher(matcher=None, hooks=[make_allowlist_deny_hook(policy["effective_allowed"])])
        ]
    opts = ClaudeAgentOptions(
        tools=policy["tools"],
        allowed_tools=policy["allowed_tools"],
        disallowed_tools=policy["disallowed_tools"],
        permission_mode="bypassPermissions",
        setting_sources=[],
        mcp_servers=mcp_servers,
        hooks=hooks or None,
    )

    async def run():
        out = []
        async with ClaudeSDKClient(options=opts) as c:
            await c.query(prompt)
            async for msg in c.receive_response():
                for blk in getattr(msg, "content", []) or []:
                    if getattr(blk, "text", None):
                        out.append(blk.text)
        return "\n".join(out)

    return asyncio.run(run())


def _visible_probe_tools(allowed, disallowed, mcp_servers):
    """Which mcp__probe tools the model can see under a given policy."""
    return _run_with_policy(
        "List every tool you have whose name starts with mcp__probe. "
        "Names only, one per line. If you have none, reply exactly NONE.",
        allowed, disallowed, mcp_servers,
    )


def _attempt_probe_call(tool, allowed, disallowed, mcp_servers):
    """Try to actually invoke `tool`, returning whatever the agent reports."""
    return _run_with_policy(
        f"Call the tool {tool} and report its exact output verbatim. "
        "If you are blocked, reply DENIED and quote the reason you were given verbatim.",
        allowed, disallowed, mcp_servers,
    )


def test_default_policy_exposes_all_connector_tools(probe_server):
    """The regression guard: no policy configured means nothing is restricted."""
    text = _visible_probe_tools([], [], probe_server)
    assert "ping_alpha" in text
    assert "ping_beta" in text


def test_disallowing_one_connector_tool_keeps_the_other(probe_server):
    text = _visible_probe_tools([], ["mcp__probe__ping_beta"], probe_server)
    assert "ping_alpha" in text
    assert "ping_beta" not in text


def test_allowlist_subsets_a_connector(probe_server):
    """The case native SDK flags cannot express — the deny hook makes it work.

    Note the asserted property: an allowlisted-out tool stays *visible* to the
    model (the hook intercepts at call time rather than removing it from
    context), but calling it is refused. Use disallowedTools when the tool
    should be invisible too.
    """
    denied = _attempt_probe_call("mcp__probe__ping_beta", ["mcp__probe__ping_alpha"], [], probe_server)
    assert "BETA-OK" not in denied, "ping_beta must not execute when it is not allowlisted"
    assert "not in this agent's allowedTools" in denied

    permitted = _attempt_probe_call("mcp__probe__ping_alpha", ["mcp__probe__ping_alpha"], [], probe_server)
    assert "ALPHA-OK" in permitted, "the allowlisted tool must still work"


def test_disallowing_whole_connector_removes_all_its_tools(probe_server):
    text = _visible_probe_tools([], ["mcp__probe__*"], probe_server)
    assert "ping_alpha" not in text
    assert "ping_beta" not in text
