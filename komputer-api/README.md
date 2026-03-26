# komputer-api

REST and WebSocket API gateway for the komputer.ai platform. Creates and manages Claude AI agents via Kubernetes CRs, consumes agent events from Redis, and streams them to clients in real-time.

## Endpoints

### REST

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/agents` | Create agent or send task to existing one |
| `GET` | `/api/v1/agents` | List all agents with status |
| `DELETE` | `/api/v1/agents/:name` | Delete agent and all its resources |
| `POST` | `/api/v1/agents/:name/cancel` | Cancel the running task on an agent |

### WebSocket

| Path | Description |
|------|-------------|
| `GET` `/api/v1/agents/:name/ws` | Stream real-time agent events |

### POST /api/v1/agents

Creates a new agent or sends a task to an existing one (upsert by name).

**Request:**
```json
{
  "name": "my-agent",
  "instructions": "Research quantum computing",
  "model": "claude-sonnet-4-20250514",
  "templateRef": "default"
}
```

Required: `name`, `instructions`. Optional: `model`, `templateRef` (both have defaults).

**Response (201 Created):**
```json
{
  "name": "my-agent",
  "namespace": "default",
  "model": "claude-sonnet-4-20250514",
  "status": "Pending",
  "createdAt": "2026-03-26T10:00:00Z"
}
```

If the agent already exists, the task is forwarded to its running pod. Returns `409` if the agent is busy.

### GET /api/v1/agents

**Response:**
```json
{
  "agents": [
    {
      "name": "my-agent",
      "namespace": "default",
      "model": "claude-sonnet-4-20250514",
      "status": "Running",
      "taskStatus": "Busy",
      "lastTaskMessage": "Calling WebSearch",
      "createdAt": "2026-03-26T10:00:00Z"
    }
  ]
}
```

### WebSocket /api/v1/agents/:name/ws

Connect to receive real-time events as the agent works:

```json
{"agentName":"my-agent","type":"task_started","timestamp":"...","payload":{"instructions":"..."}}
{"agentName":"my-agent","type":"thinking","timestamp":"...","payload":{"content":"..."}}
{"agentName":"my-agent","type":"tool_call","timestamp":"...","payload":{"tool":"WebSearch","input":{...}}}
{"agentName":"my-agent","type":"text","timestamp":"...","payload":{"content":"The answer is..."}}
{"agentName":"my-agent","type":"task_completed","timestamp":"...","payload":{"result":"...","cost_usd":0.08,"duration_ms":30000}}
```

## Redis Event Worker

The API runs a background goroutine that consumes agent events from a Redis queue. For each event it:

1. Logs the raw event
2. Broadcasts to WebSocket subscribers for that agent
3. Patches the `KomputerAgent` CR status (`taskStatus` and `lastTaskMessage`)

This worker is designed to be extracted into a separate `komputer-event-handler` service later.

## Configuration

All configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `NAMESPACE` | `default` | Kubernetes namespace for agent CRs |
| `REDIS_ADDRESS` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_QUEUE` | `komputer-events` | Redis queue name |

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
├── main.go       # Entrypoint: starts HTTP server + Redis worker
├── handler.go    # Gin route handlers (CRUD + cancel)
├── k8s.go        # Kubernetes client for CRs and pod operations
├── worker.go     # Redis consumer + CR status patcher
├── ws.go         # WebSocket hub and handler
├── go.mod
└── Dockerfile
```
