---
title: MCP Server
description: Fully control komputer-ai from Claude Code, Claude Desktop, or any MCP-aware agent via the built-in MCP server.
---

`komputer-api` has a built-in [Model Context Protocol](https://modelcontextprotocol.io/) server. **Every REST API operation is available as an MCP tool**: create agents and send them tasks, read their events, cancel or delete them, and manage squads, schedules, memories, skills, secrets and connectors. External agents (Claude Code, Claude Desktop, custom Claude SDK agents, another komputer-ai cluster) can drive komputer the same way your services do over REST.

The tools are generated from the API's OpenAPI spec, and each tool call runs the same handler as the REST endpoint. Validation, defaults and error messages are therefore identical, and new API endpoints show up as tools automatically.

## Endpoint

All MCP traffic goes to a single path on the API server:

```
<komputer-api-base-url>/mcp
```

**You must configure your client with this exact path.** It's `/mcp`, not the API root. For example, if your API is reachable at `https://komputer.example.com`, the MCP endpoint is `https://komputer.example.com/mcp`. Locally, run `kubectl port-forward svc/komputer-api 8080:8080` and use `http://localhost:8080/mcp`.

The server uses the standard MCP streamable HTTP transport.

## Authentication

> **Warning:** there is **no auth** on `/mcp`, matching the rest of the API. The tools have **full write access**: anyone who can reach the endpoint can create and delete agents, create, overwrite and delete secrets, and spend your model budget. Keep komputer-api off the public internet and restrict access at the network or ingress layer (VPN, internal-only ingress, NetworkPolicy).

## Connecting a client

### Claude Code

```bash
claude mcp add --transport http komputer https://komputer.example.com/mcp
```

Add `--scope project` to share it with your team through `.mcp.json`, or `--scope user` to use it across all your projects. Run `claude mcp list` to confirm the connection; the tools appear as `mcp__komputer__<tool>`.

### Claude Desktop

Claude Desktop connects to local MCP servers through its config file. Use the `mcp-remote` bridge so the connection runs from your machine (this works with internal or port-forwarded URLs). Open **Settings → Developer → Edit Config** and add:

```json
{
  "mcpServers": {
    "komputer": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://komputer.example.com/mcp", "--allow-http"]
    }
  }
}
```

Restart Claude Desktop. `--allow-http` is only needed for plain `http://` URLs such as `http://localhost:8080/mcp`.

### Another komputer-ai cluster

Create a `KomputerConnector` pointing at the remote `/mcp`:

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerConnector
metadata:
  name: remote-komputer
spec:
  type: http
  url: https://komputer.example.com/mcp
```

Then attach `remote-komputer` to any agent that should be able to drive the remote cluster. This gives that agent full write access to the remote cluster — it can create and delete agents and manage secrets there — so restrict it with the agent's `disallowedTools` (e.g. `mcp__remote-komputer__delete_*`, `mcp__remote-komputer__*_secret`) unless it genuinely needs full control.

### Generic MCP client

Point any streamable-HTTP MCP client at `<api-url>/mcp`. The server reports `serverInfo: { name: "komputer-ai", version: "v1" }` during the `initialize` handshake.

## Tools

Tool names are the API's operation IDs in snake_case. Arguments are the endpoint's path and query parameters (for example `name`, `namespace`, `limit`), plus a `body` object for endpoints that take a JSON request body. `create_*` tools take `namespace` inside `body`; read, list and delete tools take it as a top-level argument. A non-2xx response comes back as a tool error that contains the HTTP status and the API's error message.

| Resource | Tools |
|---|---|
| Agents | `list_agents`, `create_agent` (also sends a task to an existing agent), `get_agent`, `patch_agent`, `delete_agent`, `cancel_agent_task`, `compact_agent`, `get_agent_events`, `get_agent_cost_breakdown` |
| Squads | `list_squads`, `create_squad`, `get_squad`, `patch_squad`, `delete_squad`, `add_squad_member`, `remove_squad_member`, `break_up_squad` |
| Offices | `list_offices`, `get_office`, `delete_office`, `get_office_events` |
| Schedules | `list_schedules`, `create_schedule`, `get_schedule`, `patch_schedule`, `delete_schedule`, `trigger_schedule` |
| Memories | `list_memories`, `create_memory`, `get_memory`, `patch_memory`, `delete_memory` |
| Skills | `list_skills`, `create_skill`, `get_skill`, `patch_skill`, `delete_skill` |
| Connectors | `list_connectors`, `create_connector`, `get_connector`, `update_connector`, `delete_connector`, `list_connector_tools`, `list_connector_templates` |
| Secrets | `list_secrets`, `create_secret`, `update_secret`, `delete_secret` (values are never returned) |
| Infra | `list_namespaces`, `list_templates` |

A typical run: `create_agent`, then poll `get_agent_events` with `after` set to the last event's timestamp until a `task_completed` event arrives — pass a small `limit` (e.g. `20`) while polling, since text events can be large. The live WebSocket stream, file downloads and the OAuth authorization flow are not available over MCP.

**Note:** if you used the earlier MCP server, tool names are unchanged; outputs now mirror the REST API responses (e.g. `get_agent`/`list_agents` report `status` instead of `phase`, `get_skill` returns `content` instead of `body`), and on list tools an omitted `namespace` now means all namespaces, as in REST.

## Quick smoke test

A bare `curl` round-trip confirms the endpoint is reachable and your client will see the tool list:

```bash
# 1. Initialize a session and capture the session id from the response headers.
curl -i -X POST https://komputer.example.com/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize",
       "params":{"protocolVersion":"2025-06-18","capabilities":{},
                 "clientInfo":{"name":"smoke-test","version":"0.1"}}}'

# 2. Use the Mcp-Session-Id header from above on subsequent calls.
curl -X POST https://komputer.example.com/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```
