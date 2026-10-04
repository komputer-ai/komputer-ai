---
title: REST API
description: HTTP endpoints to create, list, inspect, cancel, and delete agents and connectors.
---

## Create an Agent / Send a Task

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
| `systemPrompt` | no | Custom system prompt prepended before the built-in role prompt |
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

## List Agents

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

## Get Agent Details

```
GET /api/v1/agents/:name?namespace=production
```

Returns the same `AgentResponse` object as the list endpoint, for a single agent.

## Get Agent Events (History)

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

## Cancel a Task

```
POST /api/v1/agents/:name/cancel?namespace=production
```

Gracefully cancels the running task. The agent pod stays alive for future tasks.

## Compact a Conversation

Manually trigger compaction on the agent's active task. Older turns are summarized to free context space. Only works while the agent is actively running a task — returns `409 Conflict` otherwise.

```
POST /api/v1/agents/:name/compact?namespace=production
Content-Type: application/json
```

The body is optional. If you want to guide the compactor:

```json
{ "instructions": "preserve all code blocks and recent file paths" }
```

**Response (200 OK):**
```json
{ "status": "compacting", "name": "my-agent" }
```

When the compaction actually happens, a `compaction` event is emitted on the agent's event stream with `payload.trigger = "manual"`. See [Agents → Compaction](../concepts/agents.md#compaction) for the full picture.

## Delete an Agent

```
DELETE /api/v1/agents/:name?namespace=production
```

Deletes the agent CR, which triggers the operator to clean up the pod, PVC, Secrets and ConfigMap.

## Connectors

### List Connectors

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

### Create a Connector

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

`authType` selects how the secret is sent: `token` (default, `Authorization: Bearer <secret>`), `oauth`, or `header`. For `header`, also set `headerName` (e.g. `"X-API-Key"`) — the secret value is sent verbatim in that header with no `Bearer` prefix:

```json
{
  "name": "amigo-mcp",
  "service": "custom",
  "url": "https://mcp.example.com/mcp",
  "authType": "header",
  "headerName": "X-API-Key",
  "authSecretName": "amigo-mcp-credentials",
  "authSecretKey": "token"
}
```

### Get a Connector

```
GET /api/v1/connectors/:name?namespace=default
```

### Update a Connector Token or Disabled State

Replace the auth token of a `token` or `header` connector, toggle whether it's disabled, or both — without recreating it:

```
PATCH /api/v1/connectors/:name?namespace=default
```

```json
{
  "token": "ghp_newtoken"
}
```

```json
{
  "disabled": true
}
```

At least one of `token` / `disabled` is required. A `token` is written into the secret key the connector already references; other keys in that secret are left alone. If the connector has no `authSecretKeyRef`, a managed `<name>-credentials` secret is created and the connector is switched to `token` auth. OAuth connectors return `400` for a `token` update — reconnect them via `/api/v1/oauth/authorize` instead — but `disabled` can still be toggled on them. Returns the connector, which now includes `"disabled": true|false`.

Running agents pick up a new token on their next pod start (sleep + wake); sleeping agents get it automatically on wake. A `disabled` connector is excluded from MCP server resolution the next time an agent's config is resolved (new pod, wake, or a live PATCH to a running agent); an already-running pod keeps what it was given until it restarts. A disabled connector also can't be newly attached to an agent, schedule, or squad member — see [Disabling a connector](../concepts/connectors.md#disabling-a-connector).

### Delete a Connector

```
DELETE /api/v1/connectors/:name?namespace=default
```

### Attach/Remove Connectors from a Running Agent

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

## Schedules

A schedule runs an agent task on a cron cadence. See [Schedules](../concepts/schedules.md) for the concept overview.

### Create a Schedule

```
POST /api/v1/schedules
Content-Type: application/json
```

```json
{
  "name": "nightly-report",
  "schedule": "0 2 * * *",
  "instructions": "Summarize yesterday's signups and post to Slack.",
  "timezone": "America/New_York",
  "agent": { "lifecycle": "Sleep", "model": "claude-sonnet-4-6" }
}
```

### List / Get / Delete Schedules

```
GET    /api/v1/schedules?namespace=default
GET    /api/v1/schedules/:name?namespace=default
DELETE /api/v1/schedules/:name?namespace=default
```

The response includes the schedule's `instructions`, current `phase`, `nextRunTime`, `runCount`, `successfulRuns`, `failedRuns`, and cost totals.

### Update a Schedule

```
PATCH /api/v1/schedules/:name
Content-Type: application/json
```

```json
{
  "schedule": "0 9 * * 1-5",
  "instructions": "Updated task prompt..."
}
```

Both fields are optional; pass either or both.

### Trigger a Schedule Manually

Fire a schedule immediately, outside its cron cadence. The schedule's normal next run is unaffected.

```
POST /api/v1/schedules/:name/trigger?namespace=default
```

**Response (200 OK):**
```json
{ "status": "triggered", "name": "nightly-report", "agentName": "nightly-report-agent" }
```

Returns `409 Conflict` if the previous run is still in progress.
