<p align="center">
  <img src="docs/assets/logo.png" alt="komputer.ai logo" width="600" />
</p>

<h1 align="center">Komputer.AI</h1>

<p align="center">
  <strong>Distributed Claude AI agents on Kubernetes</strong>
</p>

<p align="center">
  <a href="https://github.com/kontroloop-ai/komputer-ai/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT" /></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go 1.22+" /></a>
  <a href="https://www.python.org/"><img src="https://img.shields.io/badge/Python-3.12+-3776AB?logo=python&logoColor=white" alt="Python 3.12+" /></a>
  <a href="https://kubernetes.io/"><img src="https://img.shields.io/badge/Kubernetes-operator-326CE5?logo=kubernetes&logoColor=white" alt="Kubernetes" /></a>
  <a href="https://www.anthropic.com/"><img src="https://img.shields.io/badge/Powered%20by-Claude-D4A574?logo=anthropic&logoColor=white" alt="Powered by Claude" /></a>
</p>

<p align="center">
  A stateless, Kubernetes-native platform for running persistent Claude AI agents.<br/>
  Built on CRDs, operators, and the Kubernetes API — agents are first-class cluster resources.<br/>
  Designed to be driven by external systems. Create agents, send tasks, and stream real-time results via REST + WebSocket.
</p>

<p align="center">
  <img src="docs/demo.gif" alt="komputer.ai CLI demo" width="800" />
</p>

---

## Components

| Component | Language | Description |
|-----------|----------|-------------|
| [komputer-operator](komputer-operator/) | Go | Kubernetes operator that manages agent lifecycle — creates pods, PVCs, and config for each agent |
| [komputer-api](komputer-api/) | Go | REST + WebSocket API for creating agents, listing status, and streaming real-time events |
| [komputer-agent](komputer-agent/) | Python | The agent runtime — runs Claude with bash/web tools in a persistent workspace |
| [komputer-cli](komputer-cli/) | Go | Beautiful CLI for interacting with the platform |

Each component is fully self-contained with no shared code, making it easy to extract into separate repositories.

## Documentation

1. [Concepts](docs/concepts.md) — Agents, templates, config, secrets, namespaces — how the system fits together
2. [Installation](#installation) — Deploy to any Kubernetes cluster in minutes
3. [Integration Guide](docs/integration-guide.md) — How to connect external systems via HTTP API and WebSocket events
4. [Custom Agent Images](docs/custom-agent-image.md) — Build custom agent images with your own packages and tools
5. [Local Development](docs/local-development.md) — Build and run from source on a local cluster
6. [Architecture](#architecture) — System diagram and component interactions
7. Komputer Components
   1. [komputer-api](komputer-api/README.md) — REST & WebSocket API reference, Swagger UI, configuration
   2. [komputer-operator](komputer-operator/README.md) — CRD definitions, reconciliation logic, operator development guide
   3. [komputer-agent](komputer-agent/README.md) — Agent runtime, Claude SDK integration, manager tools, event format
   4. [komputer-cli](komputer-cli/README.md) — CLI commands, flags, usage examples

## Installation

### Prerequisites

- Kubernetes cluster (Docker Desktop, kind, minikube, EKS, GKE, etc.)
- `kubectl` configured
- `helm` 3.x installed
- An [Anthropic API key](https://console.anthropic.com/)

### 1. Create the Anthropic API key secret

```bash
kubectl create namespace komputer-ai
kubectl create secret generic anthropic-api-key \
  --from-literal=api-key=sk-ant-... \
  -n komputer-ai
```

### 2. Install with Helm

```bash
helm install komputer-ai oci://ghcr.io/kontroloop-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  --namespace komputer-ai
```

This deploys the operator, API, Redis, CRDs, and a default agent template — everything you need.

### 3. Install the CLI

Download from [GitHub Releases](https://github.com/kontroloop-ai/komputer-ai/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/kontroloop-ai/komputer-ai/releases/latest/download/komputer-darwin-arm64 -o komputer
chmod +x komputer && sudo mv komputer /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/kontroloop-ai/komputer-ai/releases/latest/download/komputer-linux-amd64 -o komputer
chmod +x komputer && sudo mv komputer /usr/local/bin/
```

### 4. Connect and run your first agent

```bash
# Port-forward the API (or use an Ingress)
kubectl port-forward svc/komputer-ai-api 8080:8080 -n komputer-ai &

komputer login http://localhost:8080
komputer run my-agent "Write a haiku about Kubernetes"
```

### Custom installation

For external Redis, custom resource limits, or other configuration:

```bash
# Use external Redis
helm install komputer-ai oci://ghcr.io/kontroloop-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  --set redis.enabled=false \
  --set externalRedis.address=redis.prod:6379 \
  --namespace komputer-ai

# See all options
helm show values oci://ghcr.io/kontroloop-ai/charts/komputer-ai
```

For building from source, see the [Local Development](docs/local-development.md) guide.

## Custom Resources

For a conceptual overview of these resources, see [Concepts](docs/concepts.md).

**KomputerConfig** (cluster-scoped, singleton) — Platform configuration with Redis and API settings:
```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerConfig
metadata:
  name: default
spec:
  redis:
    address: "redis.default:6379"
    db: 0
    streamPrefix: "komputer-events"
    passwordSecret:
      name: redis-secret
      key: password
  apiURL: "http://komputer-api.default.svc.cluster.local:8080"
```

**KomputerAgentClusterTemplate** (cluster-scoped) — Reusable pod configuration shared across all namespaces:
```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
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

**KomputerAgentTemplate** (namespaced) — Namespace-scoped pod configuration. Takes precedence over a cluster template with the same name.

**KomputerAgent** — An agent instance with Claude configuration:
```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: my-agent
spec:
  instructions: "Research quantum computing and write a summary"
  model: "claude-sonnet-4-6"
  templateRef: "default"
  role: "manager"    # or "worker" — managers get orchestration tools
  lifecycle: "Sleep" # "", "Sleep", or "AutoDelete"
  secrets:           # optional list of K8s Secret names
    - my-agent-secrets
```

**KomputerOffice** — Tracks a group of agents under a manager. Created automatically when managers create sub-agents.

**KomputerSchedule** — Runs agent tasks on a cron schedule with timezone support, auto-delete, and cost tracking.

## CLI Usage

```bash
komputer login <endpoint>           # Save API endpoint
komputer create <name> <prompt>     # Create agent or send task
komputer run <name> <prompt>        # Create + stream output live
komputer chat <name>                # Interactive turn-by-turn conversation
komputer list                       # List all agents
komputer get <name>                 # Get agent details + recent events
komputer watch <name>               # Stream live events (WebSocket)
komputer cancel <name>              # Cancel running task
komputer delete <name> [name...]    # Delete one or more agents

# Flags
--api <url>                         # Override saved endpoint
--model <model>                     # Override Claude model per task
-n, --namespace <ns>                # Target Kubernetes namespace
--secret KEY=VALUE                  # Pass secrets (repeatable)
```

### Secrets

Pass credentials to agents at creation time. Secrets are stored as K8s Secrets and injected as `SECRET_*` env vars:

```bash
# Single secret
komputer run github-bot "create a PR" --secret GITHUB=ghp_xxx

# Multiple secrets
komputer run deploy-agent "deploy to prod" \
  --secret GITHUB=ghp_xxx \
  --secret SLACK=xoxb-xxx \
  --secret AWS_KEY=AKIA...

# Agent sees: SECRET_GITHUB, SECRET_SLACK, SECRET_AWS_KEY as env vars
```

The agent is instructed to check `SECRET_*` env vars when credentials are needed. If a required secret is missing, the agent completes what it can and reports which credential is needed.

## How It Works

1. **Create** — CLI/API creates a `KomputerAgent` CR in Kubernetes
2. **Reconcile** — Operator detects the CR, creates a PVC (persistent workspace) and Pod
3. **Execute** — Agent pod starts, runs Claude with the given instructions
4. **Stream** — Agent publishes structured events to Redis (tool calls, messages, results)
5. **Consume** — API worker reads events, updates CR status (`InProgress`/`Complete`), broadcasts via WebSocket
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

## Architecture

```
                    ┌─────────────────┐
                    │  komputer-cli   │
                    │  (--namespace)  │
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
    │ ClusterTemplate│  │        │  │ (manager/worker)│
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

## Project Structure

```
komputer-ai/
├── komputer-operator/     # K8s operator (Go, operator-sdk)
│   ├── api/v1alpha1/      # CRD types (Config, Agent, Templates)
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
