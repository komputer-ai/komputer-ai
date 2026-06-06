---
title: Agents
description: The persistent Claude AI instance running inside a Kubernetes pod with its own isolated workspace.
---

An **agent** is the central entity in komputer.ai. It represents a persistent Claude AI instance running inside a Kubernetes pod with its own isolated workspace.

When you create an agent, you give it a name, a task (instructions), and optionally a model and role. The operator provisions a pod and a persistent volume for the agent. The agent executes the task using Claude's capabilities — bash commands, web search, and more — and streams events back in real-time.

Agents are **persistent**. After completing a task, the pod stays running and the workspace is preserved. You can send the same agent new tasks, and it picks up where it left off — same files, same environment. Claude also maintains conversation continuity across tasks via session IDs.

## Lifecycle Modes

By default, agent pods stay running after task completion. You can change this behavior with the `lifecycle` field:

- **Default (`""`)** — Pod stays running, ready for the next task immediately. Best for interactive use and agents that receive frequent tasks.
- **Sleep** — Pod is deleted after task completion, but the PVC (workspace) is preserved. When a new task is sent, the operator creates a fresh pod that reconnects to the same workspace. Saves compute costs for infrequent tasks.
- **AutoDelete** — The entire agent (CR, pod, PVC, secrets) is deleted after task completion. Best for one-shot tasks where nothing needs to persist.

Sleeping agents show a `Sleeping` phase in `kubectl get komputeragents`. When you send a new task to a sleeping agent, the API wakes it up automatically.

## Roles

Agents have one of two roles:

- **Manager** — Has orchestration tools that allow it to create, monitor, and manage sub-agents. When you give a manager a complex task, it can break it down and delegate parts to worker agents. Managers are the default role for agents created via the API or CLI.
- **Worker** — Has only bash and web search tools. Workers are focused executors that handle a single task. Sub-agents created by managers are always workers.

## Per-Agent Spec Overrides

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
          requests:
            cpu: "4"
            memory: "8Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
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

## Compaction

When an agent's conversation history fills its model context window, the bundled Claude Code CLI **automatically compacts** the older turns into a summary — preserving recent context while freeing space to keep going. komputer-ai surfaces compaction in three ways:

- The chat UI shows a purple divider (**"Context auto-compacted"** or **"Context compacted manually"**) inline with the conversation at the moment compaction happens
- The CLI's interactive `komputer chat` prints a dim `── context auto-compacted ──` line in the same place
- The event stream emits a `compaction` event with `payload.trigger` set to `"auto"` or `"manual"` — useful for SDK consumers building custom dashboards

### Manual compaction

You can trigger compaction yourself while an agent is actively running a task. This is useful when you know a long paste or tool result is about to land and you'd rather compact preemptively than wait for the auto-trigger.

- **UI**: the purple Layers icon in the chat input footer, visible only while the agent is working
- **CLI**: `komputer agent compact <name>` (optionally `--instructions "preserve all code blocks"`)
- **REST**: `POST /api/v1/agents/<name>/compact` with optional `{"instructions": "..."}`
- **Manager MCP tool**: `compact_agent` (one of the manager tools a manager agent has access to)
- **External MCP**: the API's `/mcp` endpoint exposes `compact_agent` to external Claude / MCP-aware clients

Manual compaction only works while the agent is **actively running a task** — there's no conversation to compact otherwise. The endpoint returns 409 if the agent is idle.

### Where to tune compaction

There's no UI knob yet for the auto-compaction threshold or strategy — those stay at the bundled Claude Code CLI's defaults (compaction triggers around 95% of the model's context window). A future release may expose them as `KomputerAgentSpec` fields.
