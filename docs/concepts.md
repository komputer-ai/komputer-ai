# Concepts

This document explains the core entities in komputer.ai, how they relate to each other, and the role each one plays in the system.

## Kubernetes as the Database

komputer.ai is stateless — it has no external database. All system state is stored as Kubernetes Custom Resources (CRs) in etcd, the cluster's built-in key-value store. Agents, templates, and config are all CRs. Agent status, task progress, session IDs, and pod references are all persisted as CR status fields.

This means the Kubernetes API server is the source of truth. The operator watches CRs and reconciles them into pods and volumes. The API server reads and patches CRs to reflect task status. If the operator or API restarts, they simply re-read the CRs and resume — there's nothing else to recover.

Redis is used only as a transient event bus for real-time streaming, not as persistent storage.

## Agents

An **agent** is the central entity in komputer.ai. It represents a persistent Claude AI instance running inside a Kubernetes pod with its own isolated workspace.

When you create an agent, you give it a name, a task (instructions), and optionally a model and role. The operator provisions a pod and a persistent volume for the agent. The agent executes the task using Claude's capabilities — bash commands, web search, and more — and streams events back in real-time.

Agents are **persistent**. After completing a task, the pod stays running and the workspace is preserved. You can send the same agent new tasks, and it picks up where it left off — same files, same environment. Claude also maintains conversation continuity across tasks via session IDs.

### Roles

Agents have one of two roles:

- **Manager** — Has orchestration tools that allow it to create, monitor, and manage sub-agents. When you give a manager a complex task, it can break it down and delegate parts to worker agents. Managers are the default role for agents created via the API or CLI.
- **Worker** — Has only bash and web search tools. Workers are focused executors that handle a single task. Sub-agents created by managers are always workers.

## Templates

Templates define **how** an agent pod is configured — container image, resource limits, environment variables, tolerations, node selectors, and storage size. They use full Kubernetes PodSpec passthrough, so anything you can put in a pod spec, you can put in a template.

There are two kinds of templates:

- **KomputerAgentClusterTemplate** — Cluster-scoped. Shared across all namespaces. This is where you typically define your default agent configuration.
- **KomputerAgentTemplate** — Namespace-scoped. If a namespace-scoped template exists with the same name as a cluster template, the namespace-scoped one takes precedence. This lets teams customize agent configuration without affecting the rest of the cluster.

When an agent is created, it references a template by name (defaulting to `"default"`). The operator resolves the template — checking the agent's namespace first, then falling back to cluster scope — and uses it to build the pod.

**Important:** The template must include the `ANTHROPIC_API_KEY` environment variable (typically via a Kubernetes Secret reference). Without it, agents cannot communicate with the Claude API and will fail to start. This is the one mandatory piece of configuration in every template.

## Config

**KomputerConfig** is a cluster-scoped singleton that holds platform-wide settings:

- **Redis connection** — Address, database number, stream prefix, and optional password secret. Redis is the event bus that connects agents to the API.
- **API URL** — The internal cluster URL of the komputer-api service. Manager agents use this to create and manage sub-agents via HTTP.

The operator auto-discovers this resource — agents and templates don't need to reference it explicitly.

## Secrets

Agents often need credentials to do their work — API keys, tokens, passwords. komputer.ai handles this through Kubernetes Secrets:

- When creating an agent, you can pass key-value secrets (e.g. `GITHUB=ghp_xxx`).
- The API creates a Kubernetes Secret and links it to the agent CR.
- The operator injects each key as a `SECRET_*` environment variable into the agent pod (e.g. `SECRET_GITHUB`).
- The agent's system prompt instructs Claude to check `SECRET_*` env vars when credentials are needed.
- When the agent is deleted, its secrets are automatically cleaned up via Kubernetes owner references.

Secrets from the template (like `ANTHROPIC_API_KEY`) and agent-specific secrets are merged at pod creation time. If there's a conflict, agent secrets take precedence.

## Namespaces

komputer.ai is fully namespace-aware. Namespaces provide isolation boundaries for agents and their resources:

- **Agents** are namespace-scoped — two teams can each have an agent named `researcher` without conflict.
- **Templates** can be namespace-scoped (per-team overrides) or cluster-scoped (shared defaults).
- **Config** is cluster-scoped — one Redis and API configuration for the whole platform.
- **Secrets** live in the same namespace as their agent.

When creating an agent, the namespace is auto-created if it doesn't exist. The default template and required secrets are copied into the new namespace automatically.

All API endpoints and CLI commands support namespace selection. If no namespace is specified, the server's default namespace is used.

## How They Fit Together

```
KomputerConfig (cluster)
    │
    ├── Redis connection settings
    └── API URL for manager agents

KomputerAgentClusterTemplate (cluster)
    │
    └── Default pod spec, image, resources, storage
         │
         └── overridden by ──▶ KomputerAgentTemplate (per namespace)

KomputerAgent (per namespace)
    │
    ├── references ──▶ Template (by name)
    ├── owns ──▶ Pod, PVC, ConfigMap, Secrets
    ├── role: manager ──▶ gets MCP tools to create sub-agents
    └── role: worker ──▶ gets bash + web search only
```

The typical flow:

1. Platform admin sets up **KomputerConfig** (Redis, API URL) and a **KomputerAgentClusterTemplate** (default pod configuration)
2. External system creates a **KomputerAgent** via the API, optionally with secrets
3. The operator resolves the template, creates a pod and workspace, and starts the agent
4. The agent executes the task, streaming events through Redis to the API
5. The external system consumes events via WebSocket or polls the events endpoint
6. The agent stays alive for future tasks, or is deleted when no longer needed
