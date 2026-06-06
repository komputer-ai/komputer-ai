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

### Get a Connector

```
GET /api/v1/connectors/:name?namespace=default
```

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
