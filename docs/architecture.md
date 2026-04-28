---
title: Architecture
description: How komputer.ai is built — components, protocols, data flow, and the design choices that make it stateless and Kubernetes-native.
---

komputer.ai is a Kubernetes-native platform for running long-lived Claude AI agents as first-class cluster resources. It is **stateless by design** — there is no external database. Agents, templates, and configuration are Kubernetes Custom Resources stored in etcd; agent runtime state lives on the CR's `.status` field. The only out-of-cluster dependency is the Anthropic API.

## System overview

The platform is composed of four main services plus Redis. Each service is independently scalable and stateless (except for the agent pods, which own a PVC for their workspace).

```
                                    ┌──────────────────────────────────────┐
                                    │           Kubernetes API             │
                                    │   (etcd — source of truth for all    │
                                    │    CRs and their .status fields)     │
                                    └───────────┬──────────────────────────┘
                                                │
              ┌─────────────────────────────────┼─────────────────────────────────┐
              │                                 │                                 │
              ▼                                 ▼                                 ▼
   ┌──────────────────┐            ┌──────────────────────┐           ┌────────────────────┐
   │  komputer-ui     │   REST     │    komputer-api      │  watch /  │ komputer-operator  │
   │  (Next.js)       │ ─────────▶ │   (Go, stateless)    │  patch    │   (Go, controller- │
   │                  │   WS       │                      │ ◀────────▶│    runtime)        │
   │  browser         │ ◀─────────▶│  REST + WebSocket    │           │                    │
   └──────────────────┘            └─────────┬────────────┘           └─────────┬──────────┘
                                             │                                  │ reconcile
                                             │ XREAD                            │
                                             ▼                                  ▼
                                    ┌──────────────────┐              ┌──────────────────┐
                                    │      Redis       │   XADD       │   Agent Pod(s)   │
                                    │  (streams only — │ ◀────────────│  komputer-agent  │
                                    │   not storage)   │              │  (Python +       │
                                    └──────────────────┘              │   Claude SDK)    │
                                                                       │  + PVC workspace │
                                                                       └──────────────────┘
                                                                                │
                                                                                ▼
                                                                       ┌──────────────────┐
                                                                       │   Anthropic API  │
                                                                       └──────────────────┘
```

## Components

### komputer-ui

A Next.js dashboard for creating, watching, and managing agents. Pure client of the API — no server-side state, no database. Talks REST for control plane operations and opens a WebSocket per agent for live event streaming. Renders the agent's token-by-token output, tool calls, costs, and lifecycle transitions in real time.

### komputer-api

A stateless Go service exposing both REST (control plane) and WebSocket (event plane). It is the only component that external clients (UI, CLI, SDKs, manager agents) interact with.

- **Reads** from the Kubernetes API to list/get agents, templates, secrets, connectors, etc.
- **Writes** CRs and patches `.status` fields on behalf of clients
- **Subscribes** to Redis streams produced by agent pods and rebroadcasts events to connected WebSocket clients
- Holds **no state** between requests — restart the API at any time, lose nothing

The API can be replicated horizontally. Each replica independently watches the same Redis streams; WebSocket clients can connect to any replica and use a `?group=<name>` query parameter to share a Redis consumer group across replicas (see [WebSocket events](/docs/integration/websocket)).

### komputer-operator

A Kubernetes controller (built with [Operator SDK](https://sdk.operatorframework.io/)) that watches `KomputerAgent`, `KomputerSquad`, `KomputerSchedule`, and template CRs and reconciles them into Pods, PVCs, ConfigMaps, and Secrets. Owns the lifecycle: create, sleep (delete pod, keep PVC), wake, auto-delete on completion.

Like the API, the operator is stateless — its only memory is the CRs it reconciles. Restart it and it picks up exactly where it left off.

### komputer-agent

A thin Python pod (one per agent) that runs the Claude Agent SDK against the user's task and publishes every event — token, tool call, cost, completion — to a Redis stream named after the agent. It also exposes a small HTTP server for the API to send new tasks, wake signals, and inject runtime config.

The agent is intentionally minimal: all orchestration, state persistence, and policy lives in the API and operator. The agent's job is to talk to Anthropic and emit events. See [Agents](/docs/concepts/agents) for the design rationale.

### Redis

Used **exclusively as a message bus**. Each agent has a Redis stream (`komputer-events:<agent>`) that the agent writes to via `XADD` and the API reads via `XREAD`. Recent events are also kept on a `komputer-history:<agent>` list for short-window WebSocket replay on reconnect. Redis is **not** a database, cache, or source of truth — drop the Redis volume and only in-flight live events are lost; all persistent state lives in CRs.

The chart ships a Redis HA subchart by default; an external Redis can be configured via `externalRedis.address`.

## Data flow

A typical agent lifecycle:

1. **Create** — Client (UI/CLI/SDK) calls `POST /api/v1/agents`. The API writes a `KomputerAgent` CR to the Kubernetes API.
2. **Reconcile** — The operator's watch fires. It resolves the referenced template, creates a PVC for the workspace, mounts secrets and MCP connector configs, and creates a Pod.
3. **Run** — The agent pod boots, loads the system prompt, skills, and memories, then connects back to the API to receive its task. It runs the Claude SDK loop.
4. **Stream** — As the SDK emits events (text, tool_use, tool_result, task_completed, cost), the agent calls `XADD komputer-events:<agent> ...`.
5. **Broadcast** — The API's per-stream goroutine reads via `XREAD BLOCK` and fans events out to every connected WebSocket subscriber. The same goroutine appends to `komputer-history:<agent>` for replay.
6. **Persist** — On task completion, the API patches the agent's `.status` (TaskStatus, SessionID, TotalCostUSD, etc.) so the state survives Redis loss.
7. **Lifecycle** — Based on the agent's lifecycle mode, the operator may put the pod to sleep (PVC retained), keep it running, or auto-delete on completion.

The control plane (REST → CR write → operator reconcile) and the data plane (agent → Redis → WebSocket) are fully decoupled. Either can be restarted without disrupting the other.

## Protocols and APIs

| Surface | Protocol | Used by | Purpose |
|---------|----------|---------|---------|
| `komputer-api` REST | HTTP/JSON | UI, CLI, SDKs, manager agents | CRUD for agents, templates, secrets, connectors, schedules, costs |
| `komputer-api` WebSocket | WS | UI, CLI, SDKs | Live per-agent event stream (text, tool calls, costs) |
| Agent ↔ API | HTTP | API → agent pod | Send task, wake signal, inject config |
| Agent → Redis | RESP (XADD) | Agent | Publish events |
| API ← Redis | RESP (XREAD) | API worker goroutine | Consume events |
| Operator ↔ K8s | HTTP/Protobuf | Operator | Watch + patch CRs, manage Pods/PVCs |
| Agent → Anthropic | HTTPS | Agent | Claude SDK calls |
| MCP connectors | stdio / SSE / HTTP | Agent ↔ MCP servers | External tool integration |

OpenAPI spec for the REST + WS surface lives in [`komputer-sdk/openapi.yaml`](https://github.com/komputer-ai/komputer-ai/blob/main/komputer-sdk/openapi.yaml) and powers the generated Python, Go, and TypeScript SDKs.

## Tech stack

| Layer | Technology |
|-------|------------|
| **API service** | Go, Gin (HTTP), gorilla/websocket, client-go, structured logging (`slog`) |
| **Operator** | Go, [Operator SDK](https://sdk.operatorframework.io/) |
| **Agent** | Python 3.12, [Claude Agent SDK](https://github.com/anthropics/anthropic-sdk-python), FastAPI, redis-py |
| **CLI** | Go, Cobra |
| **UI** | Next.js, React, Tailwind, TypeScript |
| **SDKs** | Python (`komputer-ai-sdk`), Go, TypeScript — generated from OpenAPI |
| **State storage** | Kubernetes etcd (via CRs) — no external database |
| **Message transport** | Redis 7 (streams) |
| **Packaging** | Helm chart (`oci://ghcr.io/komputer-ai/charts/komputer-ai`), CRDs bundled |
| **Observability** | Prometheus metrics + ServiceMonitor, structured JSON logs, Grafana dashboard |
| **Container images** | Multi-arch (amd64, arm64), published to `ghcr.io/komputer-ai/*` |

## Scalability and performance

**Horizontal scaling.** The API and operator are both stateless and can be scaled horizontally. The operator uses leader election so only one replica reconciles at a time; standby replicas take over instantly on failure. The API has no leader — every replica serves requests independently.

**Streaming latency.** Agent → Redis → API → WebSocket delivery is typically sub-50ms within a cluster. There is no polling anywhere in the live path.

**Distributed WebSocket consumption.** When clients pass `?group=<name>` on the WebSocket URL, the API uses a Redis consumer group so that across N API replicas, each event is delivered exactly once to that group. This is how the SDKs scale event consumption across multiple consumers without duplicates.

**Agent isolation.** Each agent runs in its own pod with its own PVC. Agents do not share memory, filesystem, or network namespaces. CPU and memory limits are configurable per template.

**Sleep mode.** Idle agents can be put to sleep — the pod is deleted, the PVC retained. On wake, the operator recreates the pod and the agent resumes from the persisted workspace and session ID. This reduces idle cost to near-zero (PVC storage only) while keeping wake time under a few seconds.

**No database bottleneck.** Because all persistent state is in CRs, scaling the platform is bounded by the Kubernetes control plane, not by a database tier.

## Why stateless?

The single most important design decision is the absence of an external database. This was deliberate:

- **One source of truth.** CRs already describe the desired state of agents. Putting runtime state on `.status` means there is one place to look, one place to back up, one consistency model.
- **Zero migration burden.** Schema changes are CRD versions, with conversion webhooks if needed. No ORM, no migrations table, no drift between code and DB.
- **Operationally simple.** Restart any component, lose nothing. Disaster recovery is `etcd` backup + Helm install.
- **Native to the environment.** Kubernetes operators already have idiomatic patterns for this (Operator SDK, server-side apply, status subresource). Using them costs nothing and gives us reconciliation, watch, and RBAC for free.

The trade-off is that Kubernetes etcd has practical limits on object size (~1.5 MiB) and total object count. We design `.status` fields to stay well under these limits, and event history that exceeds them is the one thing we keep elsewhere — in Redis, with bounded retention.
