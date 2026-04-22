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

### Lifecycle Modes

By default, agent pods stay running after task completion. You can change this behavior with the `lifecycle` field:

- **Default (`""`)** — Pod stays running, ready for the next task immediately. Best for interactive use and agents that receive frequent tasks.
- **Sleep** — Pod is deleted after task completion, but the PVC (workspace) is preserved. When a new task is sent, the operator creates a fresh pod that reconnects to the same workspace. Saves compute costs for infrequent tasks.
- **AutoDelete** — The entire agent (CR, pod, PVC, secrets) is deleted after task completion. Best for one-shot tasks where nothing needs to persist.

Sleeping agents show a `Sleeping` phase in `kubectl get komputeragents`. When you send a new task to a sleeping agent, the API wakes it up automatically.

### Roles

Agents have one of two roles:

- **Manager** — Has orchestration tools that allow it to create, monitor, and manage sub-agents. When you give a manager a complex task, it can break it down and delegate parts to worker agents. Managers are the default role for agents created via the API or CLI.
- **Worker** — Has only bash and web search tools. Workers are focused executors that handle a single task. Sub-agents created by managers are always workers.

### Per-Agent Spec Overrides

Templates define a default pod configuration, but individual agents can override the resources, image, or storage of their pod inline on `spec.podSpec` and `spec.storage`. This avoids forking a new template every time one agent needs more memory or a different image.

- **`spec.podSpec`** — A `corev1.PodSpec` that's merged into the template's PodSpec. Containers are matched by name (typically `agent`), and only the non-zero fields you set override the template — so passing just `resources` keeps the template's image, env, command, etc.
- **`spec.storage`** — Overrides the template's storage block. If the underlying StorageClass supports `allowVolumeExpansion`, increasing `storage.size` also expands the existing PVC in place; storage classes that don't support expansion are tolerated (the operator logs and continues).

Overrides apply when the next pod is built. They don't mutate a running pod — for resource or image changes to take effect, the agent needs to Sleep+wake or be deleted and recreated.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: heavy-worker
spec:
  instructions: "Run the large batch job"
  templateRef: default
  podSpec:
    containers:
      - name: agent
        resources:
          requests: { cpu: "4", memory: "8Gi" }
          limits:   { cpu: "4", memory: "8Gi" }
  storage:
    size: 50Gi
```

Manager agents can apply overrides at runtime through the `update_agent` MCP tool — pass `cpu`, `memory`, `storage`, or `image`, and the manager builds the same shape and PATCHes the agent. Pass an empty string (e.g. `storage=""`) to remove an override and revert to the template default.

## Concurrency Control

Templates can cap how many of their agents run concurrently per namespace via `spec.maxConcurrentAgents`. When the cap is reached, new agents enter the `Queued` phase instead of having a pod created — they don't consume cluster resources while waiting.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
metadata:
  name: default
spec:
  maxConcurrentAgents: 10   # 0 = no cap (default)
  podSpec: { ... }
```

Queued agents are admitted in **priority order**:

- `KomputerAgent.spec.priority` is a signed int32 (matches Kubernetes PodPriority — higher number = admitted first)
- Default priority is `0`, so without explicit priority everyone competes equally
- Ties are broken by creation timestamp (older first), then by name

When an agent in `Phase=Running` transitions to `Sleeping`/`Succeeded`/`Failed` or is deleted, the operator re-evaluates queued siblings sharing the same template and admits the highest-priority one. A Running agent counts against the cap regardless of `taskStatus` — so an idle agent (taskStatus `Complete` but pod still alive) keeps holding its slot. Use `lifecycle: Sleep` if you want completed agents to free their slot automatically.

The agent's `status.phase` shows `Queued` and `status.queuePosition` exposes the 1-based position in the queue:

```bash
kubectl get komputeragents -o custom-columns=NAME:.metadata.name,PHASE:.status.phase,QUEUE:.status.queuePosition,REASON:.status.queueReason
# NAME    PHASE    QUEUE  REASON
# vc-1    Running  <none>
# vc-3    Queued   1      template "default" reached maxConcurrentAgents (1/1 running)
# vc-2    Queued   2      template "default" reached maxConcurrentAgents (1/1 running)
```

## Templates

Templates define **how** an agent pod is configured — container image, resource limits, environment variables, tolerations, node selectors, storage size, and an optional concurrency cap (`maxConcurrentAgents`). They use full Kubernetes PodSpec passthrough, so anything you can put in a pod spec, you can put in a template.

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

> ⚠️ **Anthropic API key required in every agent namespace.** If you create namespaces manually or deploy agents directly via `kubectl apply`, ensure the `anthropic-api-key` secret (or whatever secret your template references for `ANTHROPIC_API_KEY`) exists in that namespace. Agents will fail to start without it.
> ```bash
> kubectl create secret generic anthropic-api-key \
>   --from-literal=api-key=sk-ant-... \
>   -n <your-namespace>
> ```

All API endpoints and CLI commands support namespace selection. If no namespace is specified, the server's default namespace is used.

## Cost Tracking

Every agent tracks its Anthropic API usage in the CR status:

- **`lastTaskCostUSD`** — Cost of the most recent task
- **`totalCostUSD`** — Cumulative cost of all tasks run by this agent

These fields are updated by the API worker when a `task_completed` event arrives. You can see costs via `kubectl get komputeragents` (the Cost column), the CLI (`komputer get <name>`), or the API response fields.

Offices and schedules also aggregate costs across all their agents.

## Offices

A **KomputerOffice** represents a group of agents working together under a manager. When a manager agent creates sub-agents, the system tracks them as an office — providing a single view of the group's progress, active agents, and total cost.

Offices are created automatically when a manager agent creates its first sub-agent. The office status tracks:

- The manager agent and all its members
- Per-member task status and cost
- Aggregate counts (total, active, completed agents)
- Total cost across all members

This is primarily a status/observability resource — you don't create offices directly, they emerge from manager-worker interactions.

## Schedules

A **KomputerSchedule** runs agent tasks on a cron schedule. Use it for recurring work — nightly reports, periodic monitoring, scheduled analysis.

Key features:

- **Cron expression** — Standard 5-field cron (`min hour dom month dow`)
- **Timezone** — IANA timezone support (defaults to UTC)
- **Suspend/resume** — Pause schedules without deleting them
- **Auto-delete** — Optionally delete the schedule after the first successful run
- **Keep agents** — When auto-deleting, optionally keep the created agents alive
- **Agent configuration** — Specify model, role, lifecycle, template, and secrets for created agents
- **Cost tracking** — Tracks total cost and per-run cost across all scheduled runs

Schedules default to `Sleep` lifecycle for their agents, so compute is only used during the actual task execution.

## Memories

A **KomputerMemory** is a named, persistent knowledge resource that can be shared across agents. When an agent has memories attached, their content is injected into Claude's system prompt before each task — giving the agent persistent context without repeating it in every task prompt.

### When to use it

- **Runbooks** — Standard operating procedures, escalation paths, cluster-specific notes
- **Domain knowledge** — Background information an agent always needs (API docs, schemas, team conventions)
- **Shared context** — Facts that should be consistent across multiple agents (company policies, product details)

### How it works

1. Create a `KomputerMemory` CR with a `content` field (markdown supported) and an optional `description`
2. Reference it by name in `spec.memories` on a `KomputerAgent`
3. When the agent wakes up, the API resolves all attached memory names, fetches their content, and prepends it to the agent's system prompt
4. The `.status.attachedAgents` field tracks how many agents reference each memory

### Example

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerMemory
metadata:
  name: deployment-policy
  namespace: platform
spec:
  description: "Production deployment rules"
  content: |
    ## Deployment Policy
    - No direct pushes to main — all changes via PR
    - Run `make test` before every deploy
    - Rolling update strategy only (never Recreate in prod)
```

Attach to an agent:
```yaml
spec:
  memories:
    - deployment-policy
```

Agents can also create and attach memories dynamically at runtime using the `create_memory` and `attach_memory` manager tools.

## Skills

A **KomputerSkill** is a reusable, named skill written to the agent's filesystem as a Claude SDK skill file. Attached skills become available to the agent as slash commands it can invoke during task execution.

### When to use it

- **Repeatable workflows** — Step-by-step instructions the agent follows consistently (code review checklist, incident response steps)
- **Tool usage patterns** — How to use a particular API or CLI tool in your environment
- **Specializations** — Give a general-purpose agent deep expertise in a specific domain without changing its base instructions

### How it works

1. Create a `KomputerSkill` CR with a `description` (when to use it) and `content` (markdown instructions)
2. Reference it by name in `spec.skills` on a `KomputerAgent`
3. The operator writes each skill as a `.md` file to the agent's skill directory on startup
4. The Claude SDK discovers the skill files and makes them available as slash commands
5. The `.status.attachedAgents` field tracks how many agents reference each skill

### Example

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSkill
metadata:
  name: git-commit
  namespace: platform
spec:
  description: "Create well-formatted git commits following team conventions"
  content: |
    When creating a git commit:
    1. Run `git diff --staged` to review what's staged
    2. Write a subject line: `<type>: <short description>` (50 chars max)
    3. Types: feat, fix, docs, refactor, test, chore
    4. Add a body paragraph if the change needs explanation
    5. Never use --no-verify
```

Attach to an agent:
```yaml
spec:
  skills:
    - git-commit
```

Agents can also create and attach skills dynamically at runtime using the `create_skill` and `attach_skill` manager tools.

## Connectors

A **KomputerConnector** is a named MCP (Model Context Protocol) server connection. Connectors give agents access to external tools and data sources — GitHub repositories, Slack channels, Linear issues, and any service that exposes an MCP endpoint.

### When to use it

- **External integrations** — Let agents read and write to services like GitHub, Slack, or Linear without writing custom tools
- **Custom MCP servers** — Point to any MCP-compatible endpoint, self-hosted or remote
- **Shared credentials** — One connector definition can be attached to many agents; credentials are stored once as a K8s Secret

### How it works

1. Create a `KomputerConnector` CR with a URL and an optional auth secret reference
2. The UI can auto-create the K8s Secret from a token you paste in — you never handle the secret directly
3. Reference the connector by name in `spec.connectors` on a `KomputerAgent`
4. When the agent pod starts, the operator injects the MCP server config as `KOMPUTER_MCP_SERVERS` and mounts the auth token as a `CONNECTOR_<NAME>_TOKEN` env var
5. The agent runtime configures the Claude SDK with the MCP server, making all its tools available as `mcp__<name>__*` slash commands
6. If you attach or remove a connector from a **running** agent via PATCH, the change takes effect on the next task — no pod restart needed
7. The `.status.attachedAgents` field tracks how many agents reference each connector

### Example

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerConnector
metadata:
  name: github
  namespace: default
spec:
  service: github
  url: "https://api.githubcopilot.com/mcp/"
  authSecretKeyRef:
    name: github-credentials
    key: token
```

Attach to an agent:
```yaml
spec:
  connectors:
    - github
```

The agent can then use tools like `mcp__github__create_pull_request`, `mcp__github__search_code`, etc.

### Built-in connector templates

The dashboard includes templates for common services with step-by-step setup guides:

| Service | Auth type | Notes |
|---------|-----------|-------|
| GitHub | Personal Access Token | Full repo, issues, PR access |
| Slack | User OAuth Token | Channels, messages, threads |
| Linear | API Key | Issues, projects, cycles |
| Gmail | OAuth Access Token | Read, search, draft emails |
| Google Calendar | OAuth Access Token | Events, schedules, availability |
| Custom | Optional token | Any MCP-compatible URL |

> For services that require OAuth (Notion, Atlassian full access), see [`docs/connectors-mcp-status.md`](connectors-mcp-status.md).

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

KomputerMemory (per namespace)          KomputerSkill (per namespace)
    │                                       │
    └── content injected into system prompt └── written as skill file to agent fs

KomputerConnector (per namespace)
    │
    └── MCP server URL + auth secret → injected as env vars into agent pod

KomputerAgent (per namespace)
    │
    ├── references ──▶ Template (by name)
    ├── references ──▶ KomputerMemory names (injected into system prompt)
    ├── references ──▶ KomputerSkill names (written as skill files)
    ├── references ──▶ KomputerConnector names (MCP servers injected at pod start)
    ├── owns ──▶ Pod, PVC, ConfigMap, Secrets
    ├── lifecycle ──▶ Default (running) / Sleep (PVC only) / AutoDelete
    ├── role: manager ──▶ gets MCP tools to create sub-agents
    │                      └── creates ──▶ KomputerOffice (tracks the group)
    └── role: worker ──▶ gets bash + web search only

KomputerSchedule (per namespace)
    │
    ├── cron expression + timezone
    └── creates/triggers ──▶ KomputerAgent on schedule
```

The typical flow:

1. Platform admin sets up **KomputerConfig** (Redis, API URL) and a **KomputerAgentClusterTemplate** (default pod configuration)
2. External system creates a **KomputerAgent** via the API, optionally with secrets and a lifecycle mode
3. The operator resolves the template, creates a pod and workspace, and starts the agent
4. The agent executes the task, streaming events through Redis to the API
5. The external system consumes events via WebSocket or polls the events endpoint
6. Based on lifecycle: agent stays alive (default), sleeps (pod deleted, PVC kept), or auto-deletes
