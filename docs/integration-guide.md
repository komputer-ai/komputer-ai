# Integration Guide

komputer.ai is designed from the ground up to be plugged into external systems. The platform itself provides a simple HTTP + WebSocket API that serves as the control plane for AI agents on Kubernetes. Your systems are the ones that create agents, assign tasks, and consume results.

Whether it's a vibecoding platform running coding agents on Kubernetes, a debugging platform that spins up agents to investigate issues in your production environment, or a marketing system that utilizes AI agents to generate and optimize campaigns — komputer.ai is the backend that makes it work.

## Overview

```
┌──────────────────┐       HTTP REST        ┌─────────────────┐       ┌─────────────────┐
│  Your System     │ ────────────────────▶   │  komputer-api   │ ────▶ │  Agent Pods     │
│                  │                         │                 │       │  (Claude AI)    │
│  Coding platform │ ◀──── WebSocket ──────  │  :8080          │ ◀──── │  on Kubernetes  │
│  Debugging tools │       (real-time)       └─────────────────┘       └─────────────────┘
│  Marketing apps  │
│  DevOps systems  │
│  Custom apps     │
└──────────────────┘
```

Two integration surfaces:

- **HTTP REST** — create agents, send tasks, check status, get results, delete agents
- **WebSocket** — stream real-time events as agents work (thinking, tool calls, text output, completion)

## Important: komputer.ai is an Internal Backend

komputer.ai is designed to be **wrapped by your system**, not exposed directly to end users. Think of it as an internal service — like a database or message queue — that your application talks to behind the scenes. This has several important implications:

### Your system owns authentication

komputer.ai does not implement authentication or authorization. It is an internal service that should live inside your cluster network, accessible only to your backend systems. Your wrapper application is responsible for authenticating users and deciding who can create agents or access results. Use Kubernetes NetworkPolicies, service mesh, or VPN to ensure komputer-api is not publicly reachable.

### Your system owns message persistence

Agent events (thinking, tool calls, text output, task completion) are streamed in real-time via WebSocket and buffered temporarily in Redis. **komputer.ai does not provide long-term storage of agent messages.** Claude maintains its own internal conversation history for session continuity, but that is not accessible to external systems.

If you need to store agent activity — for audit logs, user-facing chat history, billing, or analytics — **your system must collect events from the WebSocket (or the `/events` endpoint) and persist them in your own database.** This is by design: komputer.ai stays simple and stateless, while your wrapper handles the storage strategy that fits your use case.

### Your system owns the user experience

komputer.ai ships with a CLI ([komputer-cli](../komputer-cli/README.md)) and a web dashboard ([komputer-ui](../komputer-ui/README.md)) for direct interaction with the platform, but neither implements authentication. For production use, your wrapper application should handle auth and sit in front of komputer.ai. You can use the built-in UI as an internal operations dashboard, or build your own user-facing interface on top of the API.

## Base URL

The komputer-api listens on port `8080` by default. When deployed in-cluster, use the Kubernetes service:

```
http://komputer-api.<namespace>.svc.cluster.local:8080
```

For local development:

```
http://localhost:8080
```

## Namespace Selection

All endpoints support an optional `?namespace=` query parameter. If omitted, the server's default namespace is used. For the create endpoint, namespace can also be passed in the request body.

---

## HTTP API

### Create an Agent / Send a Task

```
POST /api/v1/agents
Content-Type: application/json
```

```json
{
  "name": "my-agent",
  "instructions": "Analyze the latest sales data and produce a summary report",
  "model": "claude-sonnet-4-6",
  "namespace": "production"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Agent identifier (lowercase, hyphens, max 63 chars) |
| `instructions` | yes | The task prompt for Claude |
| `model` | no | Claude model (default: `claude-sonnet-4-6`) |
| `templateRef` | no | Pod template to use (default: `default`) |
| `role` | no | `manager` (can orchestrate sub-agents) or `worker` (default: `manager`) |
| `connectors` | no | List of `KomputerConnector` names to attach |
| `namespace` | no | Target Kubernetes namespace |

**Behavior:**
- If the agent doesn't exist, it is created and starts working immediately
- If the agent exists and is idle, the new task is assigned to it
- If the agent exists and is busy, returns `409 Conflict`

**Response (201 Created):**
```json
{
  "name": "my-agent",
  "namespace": "production",
  "model": "claude-sonnet-4-6",
  "status": "Pending",
  "createdAt": "2026-03-27T10:00:00Z"
}
```

### List Agents

```
GET /api/v1/agents?namespace=production
```

**Response:**
```json
{
  "agents": [
    {
      "name": "my-agent",
      "namespace": "production",
      "model": "claude-sonnet-4-6",
      "status": "Running",
      "taskStatus": "InProgress",
      "lastTaskMessage": "Calling Bash: python analyze.py",
      "createdAt": "2026-03-27T10:00:00Z"
    }
  ]
}
```

### Get Agent Details

```
GET /api/v1/agents/:name?namespace=production
```

Returns the same `AgentResponse` object as the list endpoint, for a single agent.

### Get Agent Events (History)

```
GET /api/v1/agents/:name/events?limit=10&namespace=production
```

Returns the most recent events from the agent's Redis stream. `limit` defaults to 50, max 200.

**Response:**
```json
{
  "agent": "my-agent",
  "events": [
    {"agentName": "my-agent", "type": "task_started", "timestamp": "...", "payload": {"instructions": "..."}},
    {"agentName": "my-agent", "type": "text", "timestamp": "...", "payload": {"content": "Here is the report..."}},
    {"agentName": "my-agent", "type": "task_completed", "timestamp": "...", "payload": {"result": "...", "cost_usd": 0.12, "duration_ms": 45000, "turns": 3, "stop_reason": "end_turn", "session_id": "sess_01abc..."}}
  ]
}
```

### Cancel a Task

```
POST /api/v1/agents/:name/cancel?namespace=production
```

Gracefully cancels the running task. The agent pod stays alive for future tasks.

### Delete an Agent

```
DELETE /api/v1/agents/:name?namespace=production
```

Deletes the agent CR, which triggers the operator to clean up the pod, PVC, Secrets and ConfigMap.

---

### Connectors

#### List Connectors

```
GET /api/v1/connectors?namespace=default
```

**Response:**
```json
{
  "connectors": [
    {
      "name": "github",
      "namespace": "default",
      "service": "github",
      "displayName": "GitHub",
      "url": "https://api.githubcopilot.com/mcp/",
      "attachedAgents": 2,
      "agentNames": ["dev-agent", "review-agent"],
      "createdAt": "2026-04-01T10:00:00Z"
    }
  ]
}
```

#### Create a Connector

```
POST /api/v1/connectors
Content-Type: application/json
```

```json
{
  "name": "github",
  "service": "github",
  "displayName": "GitHub",
  "url": "https://api.githubcopilot.com/mcp/",
  "authSecretName": "github-credentials",
  "authSecretKey": "token",
  "namespace": "default"
}
```

#### Get a Connector

```
GET /api/v1/connectors/:name?namespace=default
```

#### Delete a Connector

```
DELETE /api/v1/connectors/:name?namespace=default
```

#### Attach/Remove Connectors from a Running Agent

Connectors can be changed on a running agent without restarting the pod. The change takes effect on the next task:

```bash
# Attach connectors
curl -X PATCH http://localhost:8080/api/v1/agents/my-agent \
  -H "Content-Type: application/json" \
  -d '{"connectors": ["github", "linear"]}'

# Remove all connectors
curl -X PATCH http://localhost:8080/api/v1/agents/my-agent \
  -H "Content-Type: application/json" \
  -d '{"connectors": []}'
```

---

## WebSocket — Real-Time Events

Connect to stream events as the agent works:

```
ws://localhost:8080/api/v1/agents/:name/ws
```

The connection stays open until you disconnect. Events arrive as JSON lines:

```json
{"agentName":"my-agent","namespace":"production","type":"task_started","timestamp":"2026-03-27T10:00:01Z","payload":{"instructions":"Analyze the latest sales data..."}}
{"agentName":"my-agent","namespace":"production","type":"thinking","timestamp":"2026-03-27T10:00:02Z","payload":{"content":"I need to look at the sales data..."}}
{"agentName":"my-agent","namespace":"production","type":"tool_call","timestamp":"2026-03-27T10:00:03Z","payload":{"id":"tc_01","tool":"Bash","input":{"command":"python analyze.py"}}}
{"agentName":"my-agent","namespace":"production","type":"tool_result","timestamp":"2026-03-27T10:00:05Z","payload":{"tool":"Bash","output":"Total revenue: $1.2M..."}}
{"agentName":"my-agent","namespace":"production","type":"text","timestamp":"2026-03-27T10:00:06Z","payload":{"content":"Based on the analysis, here is the summary report..."}}
{"agentName":"my-agent","namespace":"production","type":"task_completed","timestamp":"2026-03-27T10:00:07Z","payload":{"result":"...","cost_usd":0.08,"duration_ms":6000,"turns":2}}
```

### Event Types

| Type | Description | Key Payload Fields |
|------|-------------|-------------------|
| `task_started` | Agent begins working | `instructions` |
| `thinking` | Claude's internal reasoning | `content` |
| `tool_call` | Tool invocation (Bash, WebSearch, etc.) | `id`, `tool`, `input` |
| `tool_result` | Tool execution output | `tool`, `output` |
| `text` | Claude's text response | `content` |
| `task_completed` | Task finished successfully | `result`, `cost_usd`, `duration_ms`, `turns`, `stop_reason`, `session_id` |
| `task_cancelled` | Task was cancelled | `reason` |
| `error` | Error occurred | `error` |

### Cost and Usage Tracking

Every `task_completed` event includes usage metrics that your system should collect:

```json
{
  "type": "task_completed",
  "payload": {
    "result": "Here is the summary report...",
    "cost_usd": 0.08,
    "duration_ms": 6000,
    "turns": 2,
    "stop_reason": "end_turn",
    "session_id": "sess_01abc..."
  }
}
```

| Field | Description |
|-------|-------------|
| `cost_usd` | Total Anthropic API cost for this task (input + output tokens) |
| `duration_ms` | Wall-clock time from task start to completion |
| `turns` | Number of agent turns (a turn = one Claude request/response cycle, including tool use) |
| `stop_reason` | Why the agent stopped (`end_turn`, `max_tokens`, etc.) |
| `session_id` | Claude session ID — the same agent reuses this across tasks for conversation continuity |

**komputer.ai does not aggregate or store these metrics.** Your system is responsible for collecting them from the WebSocket stream (or the `/events` endpoint) and building whatever analytics you need. Common use cases:

- **Cost dashboards** — track spend per agent, namespace, team, or time period
- **Budgeting and alerts** — set cost thresholds per namespace and alert when exceeded
- **Performance tracking** — monitor task duration and turn count to detect regressions or inefficient prompts
- **Chargeback / billing** — attribute costs to internal teams or external customers
- **Capacity planning** — correlate agent usage patterns with cluster resource consumption

A minimal collector looks like this:

```python
# Collect cost data from the WebSocket stream
ws = websocket.WebSocket()
ws.connect(f"ws://localhost:8080/api/v1/agents/{agent_name}/ws")

for msg in ws:
    event = json.loads(msg)
    if event["type"] == "task_completed":
        payload = event["payload"]
        db.insert({
            "agent": event["agentName"],
            "namespace": event["namespace"],
            "timestamp": event["timestamp"],
            "cost_usd": payload["cost_usd"],
            "duration_ms": payload["duration_ms"],
            "turns": payload["turns"],
        })
```

---

## Integration Patterns

### Pattern 1: Fire and Forget

Create an agent, don't wait for results. Poll later or check events.

```bash
# Create the agent
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "reporter", "instructions": "Generate the weekly report"}'

# Check later
curl http://localhost:8080/api/v1/agents/reporter
curl http://localhost:8080/api/v1/agents/reporter/events?limit=5
```

### Pattern 2: Create and Stream

Create an agent and immediately connect via WebSocket to stream results.

```python
import requests
import websocket
import json

API = "http://localhost:8080"

# Create the agent
requests.post(f"{API}/api/v1/agents", json={
    "name": "analyst",
    "instructions": "Analyze server logs for anomalies"
})

# Stream events
ws = websocket.WebSocket()
ws.connect(f"ws://localhost:8080/api/v1/agents/analyst/ws")

while True:
    event = json.loads(ws.recv())
    print(f"[{event['type']}] {event.get('payload', {})}")
    if event["type"] in ("task_completed", "error"):
        break

ws.close()
```

### Pattern 3: Reusable Agents

Create an agent once, then send it multiple tasks over time. The agent keeps its workspace (PVC) between tasks.

```bash
# First task
curl -X POST http://localhost:8080/api/v1/agents \
  -d '{"name": "dev-agent", "instructions": "Clone the repo and set up the project"}'

# Wait for completion, then send another task to the same agent
curl -X POST http://localhost:8080/api/v1/agents \
  -d '{"name": "dev-agent", "instructions": "Run the test suite and fix any failures"}'
```

### Pattern 4: Multi-Namespace Isolation

Run isolated agent pools per team or environment.

> ⚠️ **Each namespace needs the Anthropic API key secret.** Before deploying agents to a new namespace, create the secret there or agents will fail to start:
> ```bash
> kubectl create secret generic anthropic-api-key --from-literal=api-key=sk-ant-... -n prod-agents
> kubectl create secret generic anthropic-api-key --from-literal=api-key=sk-ant-... -n staging-agents
> ```

```bash
# Production agents
curl -X POST "http://localhost:8080/api/v1/agents" \
  -d '{"name": "monitor", "instructions": "Check system health", "namespace": "prod-agents"}'

# Staging agents
curl -X POST "http://localhost:8080/api/v1/agents" \
  -d '{"name": "monitor", "instructions": "Check system health", "namespace": "staging-agents"}'

# List per namespace
curl "http://localhost:8080/api/v1/agents?namespace=prod-agents"
```

---

## Error Handling

| HTTP Code | Meaning |
|-----------|---------|
| `201` | Agent created successfully |
| `200` | Task forwarded to existing agent / successful read |
| `400` | Bad request (missing fields, invalid role) |
| `404` | Agent not found |
| `409` | Agent is busy or has no running pod yet |
| `500` | Internal error (cluster issue, pod unreachable) |

All error responses follow this format:

```json
{"error": "description of what went wrong"}
```
