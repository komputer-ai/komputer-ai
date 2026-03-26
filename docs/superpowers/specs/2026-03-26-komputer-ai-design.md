# komputer.ai Design Specification

## Context

komputer.ai is a platform for running distributed Claude-agent-based AI agents on Kubernetes. The goal is to provide a simple API to create persistent, autonomous agents that can execute arbitrary tasks using Claude's capabilities (bash, web search) within their own workspace.

## Architecture Overview

```
                    ┌─────────────────┐
                    │   User / CLI    │
                    └────────┬────────┘
                             │ POST /api/v1/agents
                    ┌────────▼────────┐
                    │  komputer-api   │
                    │  (Gin / Go)     │
                    │                 │
                    │ • HTTP endpoints│
                    │ • K8s client    │───── Creates/queries KomputerAgent CRs
                    │ • Redis worker  │◄──── Reads from Redis queue (logs only)
                    └─────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
    ┌─────────▼──────┐  ┌───▼────┐  ┌──────▼──────────┐
    │ KomputerAgent  │  │ Redis  │  │ KomputerAgent   │
    │ Template CRD   │  │        │  │ CRD             │
    └─────────┬──────┘  └───▲────┘  └──────┬──────────┘
              │              │              │
              │    ┌─────────┤              │
              │    │         │              │
       ┌──────▼────▼─────────┼──────────────┘
       │ komputer-operator   │
       │ (operator-sdk / Go) │
       │                     │
       │ Reconciles CRs →    │
       │ creates Pods + PVCs │
       └──────────┬──────────┘
                  │
       ┌──────────▼──────────┐
       │ Agent Pod           │
       │ (single container)  │
       │ • Python + Claude   │
       │ • FastAPI server    │──── Pushes events to Redis
       │ • Bash/web tools    │
       │ • PVC at /workspace │
       └─────────────────────┘
```

## Repository Structure

Monorepo with strict isolation — no shared code between components. Each subdirectory is a fully self-contained project that can be extracted to its own repo later.

```
komputer-ai/
├── komputer-api/          # Go module (gin)
├── komputer-operator/     # Go module (operator-sdk)
├── komputer-agent/        # Python project
└── docs/
```

## Component 1: komputer-operator

**Built with operator-sdk.** The project must be scaffolded using `operator-sdk init` and `operator-sdk create api` commands. All CRDs, controllers, and RBAC are generated and managed through the operator-sdk toolchain — no freestyle k8s controller code.

### CRDs

#### KomputerRedisConfig (v1alpha1)

Singleton cluster resource holding Redis connection details. The operator auto-discovers the single instance — no explicit reference needed from agents.

```yaml
apiVersion: komputer.ai/v1alpha1
kind: KomputerRedisConfig
metadata:
  name: default
spec:
  address: "redis:6379"
  db: 0
  queue: "komputer-events"
  passwordSecret:
    name: "redis-secret"       # k8s Secret name
    key: "password"            # key within the Secret
```

#### KomputerAgentTemplate (v1alpha1)

Full `corev1.PodSpec` passthrough so users can configure any pod-level setting (tolerations, node selectors, volumes, etc.).

```yaml
apiVersion: komputer.ai/v1alpha1
kind: KomputerAgentTemplate
metadata:
  name: default
spec:
  podSpec:                       # corev1.PodSpec
    containers:
      - name: agent
        image: komputer-agent:latest
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "2Gi"
    tolerations: []
    nodeSelector: {}
  storage:
    size: "5Gi"
    storageClassName: ""         # optional
```

#### KomputerAgent (v1alpha1)

Claude agent configuration. K8s infra config comes from the template; this CRD only holds agent-level settings.

```yaml
apiVersion: komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: my-research-agent
  labels:
    komputer.ai/agent-name: "my-research-agent"
spec:
  templateRef: "default"         # optional, defaults to "default"
  instructions: "Research quantum computing and write a summary"
  model: "claude-sonnet-4-20250514"

status:
  phase: Pending | Running | Succeeded | Failed
  podName: "my-research-agent-pod"
  pvcName: "my-research-agent-pvc"
  startTime: "2026-03-26T..."
  completionTime: "2026-03-26T..."
  message: ""
```

### Reconciliation Logic

1. Watch `KomputerAgent` CRs
2. On create:
   - Resolve `templateRef` → get KomputerAgentTemplate pod spec
   - Auto-discover the single `KomputerRedisConfig` in the cluster
   - Create PVC: `{agent-name}-pvc` with size from template
   - Create Pod from template pod spec, injecting:
     - `KOMPUTER_INSTRUCTIONS` env var (from CR spec)
     - `KOMPUTER_MODEL` env var (from CR spec)
     - `KOMPUTER_AGENT_NAME` env var (from CR metadata)
     - Config (including Redis) mounted as `/etc/komputer/config.json`
     - PVC mounted at `/workspace`
   - Set label `komputer.ai/agent-name` on the pod
3. On pod termination: recreate the pod (keep agent alive, similar to Deployment)
4. Update CR status based on pod state

### Project Structure

```
komputer-operator/
├── cmd/
│   └── main.go
├── api/
│   └── v1alpha1/
│       ├── komputeragent_types.go
│       ├── komputeragenttemplate_types.go
│       ├── komputerredisconfig_types.go
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── internal/
│   └── controller/
│       ├── komputeragent_controller.go
│       ├── komputeragenttemplate_controller.go
│       └── komputerredisconfig_controller.go
├── config/
│   ├── crd/
│   ├── rbac/
│   ├── manager/
│   └── samples/
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

## Component 2: komputer-api

### Endpoints

```
POST   /api/v1/agents    # Create or trigger agent (upsert by name)
GET    /api/v1/agents     # List all agents
```

### POST /api/v1/agents — Upsert Behavior

1. Check if agent with `name` exists (label selector: `komputer.ai/agent-name=<name>`)
2. **If exists:** get pod IP from CR status → podName → k8s pod API, then forward instructions to the agent pod's FastAPI endpoint (`POST http://<pod-ip>:8000/task`) to trigger a new task
3. **If not exists:** create a new `KomputerAgent` CR

**Request body:**
```json
{
  "name": "my-research-agent",
  "instructions": "Research quantum computing and write a summary",
  "model": "claude-sonnet-4-20250514",
  "templateRef": "default"
}
```

Required: `name`, `instructions`
Optional: `model` (default: `claude-sonnet-4-20250514`), `templateRef` (default: `"default"`)

**Response (201 Created / 200 OK):**
```json
{
  "name": "my-research-agent",
  "namespace": "default",
  "model": "claude-sonnet-4-20250514",
  "status": "Pending",
  "createdAt": "2026-03-26T10:00:00Z"
}
```

### GET /api/v1/agents — List

Returns all `KomputerAgent` CRs with their status.

**Response (200):**
```json
{
  "agents": [
    {
      "name": "my-research-agent",
      "namespace": "default",
      "model": "claude-sonnet-4-20250514",
      "status": "Running",
      "createdAt": "2026-03-26T10:00:00Z"
    }
  ]
}
```

### Redis Event Worker

Runs as a goroutine alongside the HTTP server. Consumes messages from the Redis queue defined in `KomputerRedisConfig` and logs them. Designed to be easily extracted into a separate `komputer-event-handler` project later.

### Project Structure

```
komputer-api/
├── main.go              # entrypoint: starts HTTP server + Redis worker
├── handler.go           # Gin route handlers
├── k8s.go               # K8s client: create/list/query KomputerAgent CRs
├── worker.go            # Redis consumer goroutine (logs messages)
├── go.mod
├── go.sum
└── Dockerfile
```

## Component 3: komputer-agent

### Behavior

Single-container Python application that runs a Claude agent with bash and web search tools in a persistent workspace.

1. **On startup:** reads config from env vars and mounted files, runs initial task
2. **After initial task completes:** FastAPI server stays running, accepting new tasks via `/task`
3. **All events** (tool calls, messages, completions, errors) published to Redis
4. **Workspace:** `/workspace` (PVC) — agent installs whatever tools it needs, persists across restarts

### FastAPI Endpoint

```
POST /task
{
  "instructions": "Now do this other thing..."
}
```

### Environment Variables (injected by operator)

- `KOMPUTER_INSTRUCTIONS` — initial task instructions
- `KOMPUTER_MODEL` — Claude model to use
- `KOMPUTER_AGENT_NAME` — agent identifier for events

### Config (mounted at /etc/komputer/config.json)

Contains Redis and any future configuration. Assembled by the operator from cluster CRDs.

```json
{
  "redis": {
    "address": "redis:6379",
    "password": "...",
    "db": 0,
    "queue": "komputer-events"
  }
}
```

### Event Format

```json
{
  "agentName": "my-research-agent",
  "type": "tool_call",
  "timestamp": "2026-03-26T10:00:01Z",
  "payload": {
    "tool": "bash",
    "input": "npm init -y",
    "output": "..."
  }
}
```

Event types: `tool_call`, `message`, `completion`, `error`

### Project Structure

```
komputer-agent/
├── main.py              # entrypoint: starts FastAPI + runs initial task
├── server.py            # FastAPI endpoint for /task
├── agent.py             # Claude agent runner (claude-agent-sdk)
├── events.py            # Redis event publisher
├── requirements.txt     # claude-agent-sdk, redis, fastapi, uvicorn
└── Dockerfile
```

## Build Order

1. **komputer-operator** — defines CRDs that everything else depends on
2. **komputer-api** — creates CRs and consumes events
3. **komputer-agent** — runs inside pods created by the operator

Each component gets its own implementation plan.

## Future Work (out of scope for v1)

- Per-agent memory
- Collective memory / agent communication
- Orchestrator / manager agent
- Team of agents (one orchestrator creates member agents)
- `komputer-event-handler` as separate service
