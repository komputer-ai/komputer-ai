# komputer.ai

A platform for running distributed Claude AI agents on Kubernetes. Create persistent, autonomous agents that execute tasks using Claude's capabilities — bash commands, web search, and more — each in their own isolated workspace.

## Architecture

```
                    ┌─────────────────┐
                    │   komputer-cli  │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  komputer-api   │
                    │  (Go / Gin)     │
                    │                 │
                    │  REST + WS API  │───── Creates KomputerAgent CRs
                    │  Redis worker   │◄──── Consumes agent events
                    └─────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
    ┌─────────▼──────┐  ┌───▼────┐  ┌──────▼──────────┐
    │ AgentTemplate  │  │ Redis  │  │ KomputerAgent   │
    │ CRD            │  │        │  │ CRD             │
    └─────────┬──────┘  └───▲────┘  └──────┬──────────┘
              │              │              │
       ┌──────▼──────────────┼──────────────┘
       │ komputer-operator   │
       │ (Go / operator-sdk) │
       │                     │
       │ Reconciles CRs →    │
       │ creates Pods + PVCs │
       └──────────┬──────────┘
                  │
       ┌──────────▼──────────┐
       │ Agent Pod           │
       │ (Python / Claude)   │
       │                     │
       │ Bash + Web Search   │──── Events → Redis
       │ PVC at /workspace   │
       │ FastAPI on :8000    │
       └─────────────────────┘
```

## Components

| Component | Language | Description |
|-----------|----------|-------------|
| [komputer-operator](komputer-operator/) | Go | Kubernetes operator that manages agent lifecycle — creates pods, PVCs, and config for each agent |
| [komputer-api](komputer-api/) | Go | REST + WebSocket API for creating agents, listing status, and streaming real-time events |
| [komputer-agent](komputer-agent/) | Python | The agent runtime — runs Claude with bash/web tools in a persistent workspace |
| [komputer-cli](komputer-cli/) | Go | Beautiful CLI for interacting with the platform |

Each component is fully self-contained with no shared code, making it easy to extract into separate repositories.

## Custom Resources

**KomputerRedisConfig** (cluster-scoped, singleton) — Redis connection details used by all agents:
```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerRedisConfig
metadata:
  name: default
spec:
  address: "redis:6379"
  queue: "komputer-events"
  passwordSecret:
    name: redis-secret
    key: password
```

**KomputerAgentTemplate** — Reusable pod configuration (full PodSpec passthrough):
```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentTemplate
metadata:
  name: default
spec:
  podSpec:
    containers:
      - name: agent
        image: komputer-agent:latest
        resources:
          limits:
            cpu: "2"
            memory: "2Gi"
  storage:
    size: "5Gi"
```

**KomputerAgent** — An agent instance with Claude configuration:
```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: my-agent
spec:
  instructions: "Research quantum computing and write a summary"
  model: "claude-sonnet-4-20250514"
  templateRef: "default"
```

## Quick Start

### Prerequisites

- Kubernetes cluster (Docker Desktop, kind, minikube, or cloud)
- `kubectl` configured
- `operator-sdk` installed
- Go 1.22+
- Docker
- Redis deployed in the cluster
- An [Anthropic API key](https://console.anthropic.com/)

### 1. Install CRDs

```bash
cd komputer-operator
make install
```

### 2. Deploy Redis

```bash
kubectl run redis --image=redis:7-alpine --port=6379
kubectl expose pod redis --port=6379 --name=redis
```

### 3. Create secrets

```bash
# Redis password (empty for no auth)
kubectl apply -f komputer-operator/config/samples/redis-secret.yaml

# Anthropic API key
kubectl create secret generic anthropic-api-key \
  --from-literal=api-key=sk-ant-...
```

### 4. Apply base resources

```bash
kubectl apply -f komputer-operator/config/samples/komputer_v1alpha1_komputerredisconfig.yaml
kubectl apply -f komputer-operator/config/samples/komputer_v1alpha1_komputeragenttemplate.yaml
```

### 5. Build and load the agent image

```bash
docker build -t komputer-agent:latest komputer-agent/
# For kind:
kind load docker-image komputer-agent:latest --name <cluster-name>
```

### 6. Run the operator

```bash
cd komputer-operator
make run
```

### 7. Run the API

```bash
# In another terminal — port-forward Redis first
kubectl port-forward svc/redis 16379:6379 &
cd komputer-api
REDIS_ADDRESS=localhost:16379 go run .
```

### 8. Use the CLI

```bash
cd komputer-cli
go build -o komputer .

./komputer login http://localhost:8080
./komputer run my-agent "Write a haiku about Kubernetes"
```

## CLI Usage

```bash
komputer login <endpoint>           # Save API endpoint
komputer create <name> <prompt>     # Create agent or send task
komputer run <name> <prompt>        # Create + stream output live
komputer list                       # List all agents
komputer get <name>                 # Get agent details
komputer watch <name>               # Stream live events (WebSocket)
komputer cancel <name>              # Cancel running task
komputer delete <name>              # Delete agent and resources

# Flags
--api <url>                         # Override saved endpoint
--model <model>                     # Override Claude model per task
```

## How It Works

1. **Create** — CLI/API creates a `KomputerAgent` CR in Kubernetes
2. **Reconcile** — Operator detects the CR, creates a PVC (persistent workspace) and Pod
3. **Execute** — Agent pod starts, runs Claude with the given instructions
4. **Stream** — Agent publishes structured events to Redis (tool calls, messages, results)
5. **Consume** — API worker reads events, updates CR status (`Busy`/`Idle`), broadcasts via WebSocket
6. **Persist** — Agent pod stays running after task completion, accepting new tasks via FastAPI

### Event Types

Events published by agents and streamed via WebSocket:

| Type | Description | Payload |
|------|-------------|---------|
| `task_started` | Agent begins a task | `{instructions}` |
| `thinking` | Claude's reasoning | `{content}` |
| `tool_call` | Tool invocation | `{id, tool, input}` |
| `tool_result` | Tool execution result | `{tool, input, output}` |
| `text` | Claude's text response | `{content}` |
| `task_completed` | Task finished | `{result, cost_usd, duration_ms, turns}` |
| `task_cancelled` | Task was cancelled | `{reason}` |
| `error` | Error occurred | `{error}` |

## Project Structure

```
komputer-ai/
├── komputer-operator/     # K8s operator (Go, operator-sdk)
│   ├── api/v1alpha1/      # CRD type definitions
│   ├── internal/          # Controller logic
│   └── config/            # RBAC, CRDs, samples
├── komputer-api/          # HTTP + WebSocket API (Go, Gin)
│   ├── handler.go         # REST endpoints
│   ├── worker.go          # Redis event consumer
│   └── ws.go              # WebSocket hub
├── komputer-agent/        # Agent runtime (Python)
│   ├── agent.py           # Claude Agent SDK integration
│   ├── server.py          # FastAPI task endpoint
│   └── events.py          # Redis event publisher
├── komputer-cli/          # CLI (Go, Cobra + Lipgloss)
│   └── main.go            # All commands in one file
└── docs/                  # Design specs and plans
```

## License

[MIT](LICENSE)
