---
title: MCP Server
description: Drive komputer-ai from other Claude agents via the built-in MCP server.
---

`komputer-api` exposes its capabilities over the [Model Context Protocol](https://modelcontextprotocol.io/) so external Claude agents (Claude Code, custom Claude SDK clients, anything MCP-aware) can list, create, and trigger komputer-ai resources as tools.

## Endpoint

All MCP traffic goes to a single path on the API server:

```
POST <komputer-api-base-url>/mcp
```

**You must configure your client with this exact path** — it's not the API root, it's `/mcp`. For example, if your API is reachable at `https://komputer.example.com`, the MCP endpoint is `https://komputer.example.com/mcp`.

The server uses the standard MCP streamable HTTP transport. No special headers required beyond the MCP protocol handshake.

## Authentication

There is **no auth** on the `/mcp` endpoint today — it matches the rest of the API's posture. Anyone who can reach the endpoint can drive your cluster's komputer-ai resources. Lock down access at the network or ingress layer (e.g. behind a VPN, internal-only ingress, network policy).

## Configuring a client

### Claude Code

Add a custom MCP server in `~/.claude/settings.json` (or your project's `.claude/settings.json`):

```json
{
  "mcpServers": {
    "komputer-ai": {
      "type": "http",
      "url": "https://komputer.example.com/mcp"
    }
  }
}
```

Restart Claude Code; the tools below appear under the `komputer-ai` server.

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

Then attach `remote-komputer` to any agent that should be able to drive the remote cluster.

### Generic MCP client

Point any streamable-HTTP MCP client at `<api-url>/mcp`. The server reports `serverInfo: { name: "komputer-ai", version: "v1" }` during the `initialize` handshake.

## Tools

The current tool set is focused on the most useful remote-control surface. All tools accept an optional `namespace` argument; when omitted, the API's default namespace is used.

| Tool | Purpose |
|------|---------|
| `list_agents` | List agents in a namespace (or all namespaces) |
| `get_agent` | Full spec + status + costs for one agent |
| `compact_agent` | Trigger manual conversation compaction on an agent's active task |
| `list_schedules` | List schedules in a namespace |
| `get_schedule` | Full schedule including current `instructions` |
| `trigger_schedule` | Fire a schedule immediately, outside its cron cadence |
| `list_memories` | List `KomputerMemory` resources |
| `get_memory` | Get a memory's content and description |
| `list_skills` | List `KomputerSkill` resources |
| `get_skill` | Get a skill's full body |
| `list_connectors` | List configured MCP connectors |
| `list_secrets` | List secret names (values are never returned) |
| `list_namespaces` | List Kubernetes namespaces visible to the API |
| `list_templates` | List `KomputerAgentTemplate` / `KomputerAgentClusterTemplate` resources |

More write-side tools (create agent, attach memory, etc.) are likely to be added — open an issue if there's a specific one you need.

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
