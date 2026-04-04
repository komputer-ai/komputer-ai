# komputer-api

REST and WebSocket API gateway for the komputer.ai platform. Creates and manages Claude AI agents via Kubernetes CRs, consumes agent events from Redis, and streams them to clients in real-time.

## API Reference

Base path: `/api/v1`

**Swagger UI** is available at `/swagger/index.html` when the API is running. The OpenAPI spec is also available as [swagger.json](docs/swagger.json) and [swagger.yaml](docs/swagger.yaml).

All endpoints support namespace selection via the `?namespace=` query parameter. If omitted, the server's default namespace is used.

### Endpoints Overview

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agents` | Create agent or send task to existing one |
| `GET` | `/agents` | List all agents with status |
| `GET` | `/agents/:name` | Get agent details |
| `GET` | `/agents/:name/events` | Get agent event history |
| `DELETE` | `/agents/:name` | Delete agent and all its resources |
| `POST` | `/agents/:name/cancel` | Cancel the running task |
| `GET` | `/agents/:name/ws` | WebSocket — stream real-time events |
| `GET` | `/offices` | List all offices |
| `GET` | `/offices/:name` | Get office details |
| `DELETE` | `/offices/:name` | Delete office and its agents |
| `GET` | `/offices/:name/events` | Get office event history |
| `POST` | `/schedules` | Create a schedule |
| `GET` | `/schedules` | List all schedules |
| `GET` | `/schedules/:name` | Get schedule details |
| `DELETE` | `/schedules/:name` | Delete a schedule |

### Health Checks (outside `/api/v1`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness probe — always returns `{"status": "ok"}` |
| `GET` | `/readyz` | Readiness probe — checks Redis connectivity, returns 503 if unhealthy |

---

### POST /agents

Create a new agent or send a task to an existing one (upsert by name).

**Request body:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | yes | — | Agent identifier (lowercase, hyphens, max 63 chars) |
| `instructions` | string | yes | — | Task prompt for Claude |
| `model` | string | no | `claude-sonnet-4-6` | Claude model to use |
| `templateRef` | string | no | `default` | Pod template name |
| `role` | string | no | `manager` | `manager` (gets orchestration tools) or `worker` |
| `lifecycle` | string | no | `""` | `""` (pod stays running), `Sleep` (delete pod, keep PVC), `AutoDelete` (delete everything) |
| `namespace` | string | no | server default | Target Kubernetes namespace |
| `secrets` | object | no | — | Key-value pairs (e.g. `{"GITHUB": "ghp_xxx"}`) |

```json
{
  "name": "my-agent",
  "instructions": "Research quantum computing",
  "model": "claude-sonnet-4-6",
  "templateRef": "default",
  "role": "manager",
  "namespace": "my-namespace",
  "secrets": {
    "GITHUB": "ghp_xxx",
    "SLACK": "xoxb-xxx"
  }
}
```

When `secrets` is provided, the API creates a K8s Secret named `{agent-name}-secrets` with each key prefixed by `SECRET_` (e.g. `SECRET_GITHUB`). The operator injects the values as env vars into the agent pod.

**Responses:**

| Status | Condition |
|--------|-----------|
| `201 Created` | New agent created |
| `200 OK` | Task forwarded to existing idle agent |
| `400 Bad Request` | Missing required fields or invalid role |
| `409 Conflict` | Agent exists but is busy or has no running pod yet |
| `500 Internal Server Error` | Cluster or pod communication error |

**Response body (201/200):**

```json
{
  "name": "my-agent",
  "namespace": "default",
  "model": "claude-sonnet-4-6",
  "status": "Pending",
  "taskStatus": "",
  "lastTaskMessage": "",
  "createdAt": "2026-03-27T10:00:00Z"
}
```

---

### GET /agents

List all agents in a namespace.

**Query parameters:**

| Param | Description |
|-------|-------------|
| `namespace` | Target namespace (optional) |

**Response (200):**

```json
{
  "agents": [
    {
      "name": "my-agent",
      "namespace": "default",
      "model": "claude-sonnet-4-6",
      "status": "Running",
      "taskStatus": "InProgress",
      "lastTaskMessage": "Calling WebSearch",
      "createdAt": "2026-03-27T10:00:00Z"
    }
  ]
}
```

**Response fields:**

| Field | Description |
|-------|-------------|
| `name` | Agent identifier |
| `namespace` | Kubernetes namespace |
| `model` | Claude model |
| `status` | Pod phase: `Pending`, `Running`, `Sleeping`, `Succeeded`, `Failed` |
| `taskStatus` | Agent activity: `InProgress`, `Complete`, `Error`, or empty |
| `lifecycle` | Lifecycle mode: `""`, `Sleep`, or `AutoDelete` |
| `lastTaskMessage` | Most recent event summary (e.g. "Calling Bash", "Task completed") |
| `createdAt` | ISO 8601 creation timestamp |

---

### GET /agents/:name

Get details for a single agent. Returns the same response fields as the list endpoint.

**Query parameters:**

| Param | Description |
|-------|-------------|
| `namespace` | Target namespace (optional) |

**Responses:** `200 OK`, `404 Not Found`

---

### GET /agents/:name/events

Get event history from an agent's Redis stream, returned in chronological order.

**Query parameters:**

| Param | Default | Description |
|-------|---------|-------------|
| `namespace` | server default | Target namespace |
| `limit` | `50` | Max events to return (1–200) |

**Response (200):**

```json
{
  "agent": "my-agent",
  "events": [
    {
      "agentName": "my-agent",
      "namespace": "default",
      "type": "task_started",
      "timestamp": "2026-03-27T10:00:01Z",
      "payload": { "instructions": "Research quantum computing" }
    },
    {
      "agentName": "my-agent",
      "namespace": "default",
      "type": "text",
      "timestamp": "2026-03-27T10:00:05Z",
      "payload": { "content": "Here are the findings..." }
    },
    {
      "agentName": "my-agent",
      "namespace": "default",
      "type": "task_completed",
      "timestamp": "2026-03-27T10:00:06Z",
      "payload": { "result": "...", "cost_usd": 0.08, "duration_ms": 5000, "turns": 2, "session_id": "..." }
    }
  ]
}
```

---

### DELETE /agents/:name

Delete an agent and clean up all its resources (CR, Pod, PVC, ConfigMap, Secrets, Redis stream).

**Query parameters:**

| Param | Description |
|-------|-------------|
| `namespace` | Target namespace (optional) |

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` | `{"status": "deleted", "name": "my-agent"}` |
| `404 Not Found` | `{"error": "agent not found"}` |

---

### POST /agents/:name/cancel

Cancel the currently running task. The agent pod stays alive for future tasks.

**Query parameters:**

| Param | Description |
|-------|-------------|
| `namespace` | Target namespace (optional) |

**Responses:**

| Status | Body |
|--------|------|
| `200 OK` | `{"status": "cancelling", "name": "my-agent"}` |
| `404 Not Found` | `{"error": "agent not found"}` |
| `409 Conflict` | `{"error": "agent has no running pod"}` |

---

### GET /agents/:name/ws (WebSocket)

Upgrade to a WebSocket connection to stream real-time events as the agent works. The connection stays open until the client disconnects.

**Example:**
```
ws://localhost:8080/api/v1/agents/my-agent/ws
```

Each message is a JSON object:

```json
{"agentName":"my-agent","namespace":"default","type":"task_started","timestamp":"...","payload":{"instructions":"..."}}
{"agentName":"my-agent","namespace":"default","type":"thinking","timestamp":"...","payload":{"content":"..."}}
{"agentName":"my-agent","namespace":"default","type":"tool_call","timestamp":"...","payload":{"id":"tc_01","tool":"Bash","input":{"command":"ls"}}}
{"agentName":"my-agent","namespace":"default","type":"tool_result","timestamp":"...","payload":{"tool":"Bash","output":"file1.txt\nfile2.txt"}}
{"agentName":"my-agent","namespace":"default","type":"text","timestamp":"...","payload":{"content":"The answer is..."}}
{"agentName":"my-agent","namespace":"default","type":"task_completed","timestamp":"...","payload":{"result":"...","cost_usd":0.08,"duration_ms":30000,"turns":2,"session_id":"..."}}
```

### Event Types

| Type | Description | Key Payload Fields |
|------|-------------|-------------------|
| `task_started` | Agent begins working | `instructions` |
| `thinking` | Claude's internal reasoning | `content` |
| `tool_call` | Tool invocation | `id`, `tool`, `input` |
| `tool_result` | Tool execution output | `tool`, `output` |
| `text` | Claude's text response | `content` |
| `task_completed` | Task finished | `result`, `cost_usd`, `duration_ms`, `turns`, `session_id` |
| `task_cancelled` | Task was cancelled | `reason` |
| `error` | Error occurred | `error` |

---

### Error Format

All error responses use:

```json
{"error": "description of what went wrong"}
```

## Redis Event Worker

The API runs a background goroutine that consumes agent events from Redis Streams. For each event it:

1. Logs the raw event
2. Broadcasts to WebSocket subscribers for that agent
3. Patches the `KomputerAgent` CR status (`taskStatus` and `lastTaskMessage`) in the correct namespace

Events include a `namespace` field so the worker can update CRs across namespaces.

## Configuration

All configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `NAMESPACE` | `default` | Kubernetes namespace for agent CRs |
| `REDIS_ADDRESS` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_STREAM_PREFIX` | `komputer-events` | Redis stream prefix for agent events |

Requires a valid kubeconfig (in-cluster or `~/.kube/config`).

## Development

### Prerequisites

- Go 1.22+
- Access to a Kubernetes cluster
- Redis

### Build and run

```bash
go build ./...
REDIS_ADDRESS=localhost:6379 go run .
```

### Build Docker image

```bash
# From the monorepo root (needs komputer-operator for CRD types)
docker build -f komputer-api/Dockerfile .
```

## Project Structure

```
komputer-api/
├── main.go                 # Entrypoint: starts HTTP server + Redis worker
├── routes.go               # Route registration + shared helpers
├── handler_agents.go       # Agent CRUD, cancel, events handlers
├── handler_offices.go      # Office list, get, delete, events handlers
├── handler_schedules.go    # Schedule CRUD handlers
├── handler_memories.go     # Memory CRUD handlers
├── handler_skills.go       # Skill CRUD handlers
├── handler_secrets.go      # Secret CRUD handlers
├── handler_connectors.go   # Connector CRUD + MCP tool listing
├── handler_templates.go    # Template and namespace listing
├── k8s.go                  # Kubernetes client for CRs and pod operations
├── worker.go               # Redis consumer + CR status patcher
├── ws.go                   # WebSocket hub and handler
├── prompt.go               # Agent system prompts
├── go.mod
└── Dockerfile
```
