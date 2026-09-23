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

## Timeouts — Auto-Sleep, Auto-Delete, and Task Caps

Lifecycle modes fire the instant a task finishes. TTLs instead act on **elapsed time**, so an agent you forget about doesn't hold a pod or a PVC forever, and `taskTimeout` caps how long any single task may run:

- **`spec.sleepTTL`** — An **idle timeout**. Once the agent has gone this long without activity, the operator deletes its pod and sets `Phase=Sleeping` (workspace preserved), exactly like a manual sleep. The clock is `status.lastActivityAt`, first stamped when the agent reports `task_started` and refreshed on every later event, on wake, and when a task is forwarded — so any work resets the countdown. It never fires mid-task, and never on an already-sleeping agent. **An agent that has never started a task has no idle clock and is never auto-slept**, so a `sleepTTL` shorter than pod startup can't sleep an agent before its first task runs; use `deleteTTL` to reclaim agents that are never used.
- **`spec.deleteTTL`** — An **absolute lifetime** measured from `metadata.creationTimestamp`. When it elapses the whole agent is deleted (pod + PVC, via owner references). Unlike `sleepTTL` it does not reset on wake and applies in every phase — including `Sleeping`, which is what makes it a cap rather than a suggestion. It will interrupt a running task.
- **`spec.taskTimeout`** — A **per-task wall-clock cap**. Once a single task has been running this long, the operator cancels it — the same interruption a manual cancel produces — and the agent stays alive for the next task. The clock is `status.taskStartedAt`, stamped once when the task starts and never refreshed, so **steering does not extend the deadline**; this is a hard cap, not an idle timeout. It only runs while a task is in progress on a running pod, and `status.taskExpiresAt` shows when it will fire. A cancelled task reports `taskStatus=Complete` like any other cancel, so any `lifecycle` you set still applies afterwards.

All three take Go duration strings (`30m`, `2h`, `1h30m`). Note that Go has no day unit — use `24h`, not `1d`. All three are optional and independent; set `sleepTTL` below `deleteTTL` for the natural "keep warm → hibernate → clean up" sequence (the operator logs a warning if `deleteTTL <= sleepTTL`, since the agent would be deleted before it ever slept).

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: nightly-report
spec:
  instructions: "Generate the nightly report"
  sleepTTL: 30m   # hibernate after 30 minutes idle
  deleteTTL: 24h  # hard-delete a day after creation
```

The operator publishes the projected transition times as `status.sleepExpiresAt` and `status.deleteExpiresAt`, and requeues to wake exactly when the next TTL comes due. `sleepExpiresAt` is empty whenever the idle clock isn't running — mid-task, or already asleep.

**Interaction with `lifecycle`.** A TTL turns its matching lifecycle action into a delayed one. `lifecycle: Sleep` alone sleeps immediately on task completion; adding `sleepTTL: 30m` gives it a 30-minute grace period instead. Likewise `deleteTTL` defers `lifecycle: AutoDelete`. So `lifecycle: AutoDelete` + `deleteTTL: 7d` means "stay usable, but clean yourself up after a week".

**Template-level defaults.** `sleepTTL` and `deleteTTL` can also be set on a `KomputerAgentTemplate` (or `KomputerAgentClusterTemplate`) to apply to every agent using it. A per-agent value overrides the template's, each field independently. This is how scheduled agents pick up TTLs, since `ScheduleAgentSpec` has no TTL field of its own.

**Squad members.** TTLs work for squad members too, set on the member's own agent spec (or inherited from its template) — including inline member specs in a `create_squad` call. The squad controller enforces them, with two differences forced by the shared pod:

- Sleeping a member only sets `Phase=Sleeping`; there is no per-member pod to delete. The shared squad pod is torn down once **every** member is asleep, the same path a manual member sleep takes.
- A member deleted by `deleteTTL` is removed from the squad automatically — its dangling ref is pruned on the next reconcile, which then feeds the normal [empty-squad and single-member-shrinkage handling](squads/lifecycle.md). Squads also have their own `orphanTTL` for reclaiming a squad once it is empty.

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

When an agent in `Phase=Running` transitions to `Sleeping`/`Succeeded`/`Failed` or is deleted, the operator re-evaluates queued siblings sharing the same template and admits the highest-priority one. A Running agent counts against the cap regardless of `taskStatus` — so an idle agent (taskStatus `Complete` but pod still alive) keeps holding its slot. Use `lifecycle: Sleep` if you want completed agents to free their slot immediately, or `sleepTTL` to free it after a grace period.

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

## Tool Permissions

By default an agent gets komputer.ai's standard built-in tool set, plus **every** tool from each attached connector:

```
Bash  WebSearch  WebFetch  Read  Write  Edit  Glob  Grep  Skill
```

Two optional spec fields change that. `disallowedTools` removes specific tools; `allowedTools` restricts the agent to an explicit list. Both are optional — leave them unset and the agent behaves exactly as it always has.

### `disallowedTools` — remove specific tools

Purely **subtractive**, and the safer of the two. The agent keeps all nine built-ins and all connector tools, minus exactly what you name.

```yaml
spec:
  disallowedTools:
    - Bash
```

That agent still has WebSearch, WebFetch, Read, Write, Edit, Glob, Grep, Skill and every connector tool — it simply has no shell. This is usually what you want when the goal is "stop it doing one specific thing".

### `allowedTools` — restrict to an explicit list

> **⚠ Warning — `allowedTools` REPLACES the default tool set. It does not add to it.**
>
> The moment you set this field, the agent loses the use of **every built-in tool you did not list**, and **every connector tool**. An agent given `allowedTools: ["Read"]` cannot run commands, write files, edit files, search the web, or use any connector — it can only read.
>
> Most agents still need several of the defaults. Start from the full list below and delete what you don't want, rather than writing a short list from scratch:
>
> ```yaml
> spec:
>   allowedTools:
>     - Bash
>     - WebSearch
>     - WebFetch
>     - Read
>     - Write
>     - Edit
>     - Glob
>     - Grep
>     - Skill
> ```
>
> If the agent has connectors attached, you must list those too — they are **not** re-added automatically. See [Connector tools](#connector-tools) below.

A read-only researcher, written out in full:

```yaml
spec:
  allowedTools:
    - Read
    - Glob
    - Grep
    - WebSearch
    - WebFetch
```

That agent can explore a workspace and search the web, but cannot modify anything or run commands.

### Connector tools

Connector (MCP) tools are named `mcp__<connector>__<tool>`, where `<connector>` is the KomputerConnector name **exactly as written** — hyphens are preserved, so a connector named `internal-search` exposes `mcp__internal-search__*`, not `mcp__internal_search__*`. Wildcards are supported: a trailing `*` is a prefix match.

Run `komputer connector tools <name>` to list a connector's exact tool names before writing a policy — a pattern that doesn't match any real tool fails silently, granting nothing rather than erroring.

**Allow one whole connector and nothing else:**

```yaml
spec:
  allowedTools:
    - Read
    - mcp__figma__*
```

**Allow only specific tools from a connector** — the common case for giving an agent read access to a service without write access:

```yaml
spec:
  allowedTools:
    - Read
    - mcp__figma__get_design_context
    - mcp__figma__get_screenshot
```

**Keep everything, but block a few connector tools:**

```yaml
spec:
  disallowedTools:
    - mcp__figma__use_figma
```

**Block a connector entirely**, leaving the rest of the agent untouched:

```yaml
spec:
  disallowedTools:
    - mcp__figma__*
```

### Deny always beats allow

> **⚠ Important:** Combining a wildcard deny with a narrower allow does **not** give you a subset — it gives you nothing.
>
> ```yaml
> # WRONG — this yields ZERO Figma tools, not one.
> spec:
>   allowedTools:
>     - mcp__figma__get_design_context
>   disallowedTools:
>     - mcp__figma__*
> ```
>
> To admit a subset of a connector's tools, use `allowedTools` **on its own** and list exactly the tools you want.

### Setting tool permissions

**CLI** — at creation, or later with `komputer config`:

```bash
# Restrict to an explicit set (replaces the defaults)
komputer create researcher --instructions "Audit the repo" \
  --allow-tool Read --allow-tool Glob --allow-tool Grep \
  --allow-tool 'mcp__figma__get_design_context'

# Just remove a couple of tools, keeping everything else
komputer create writer --instructions "Draft the docs" \
  --disallow-tool Bash --disallow-tool 'mcp__figma__*'

# Change it later (applies on the agent's next start)
komputer config researcher --disallow-tool Bash

# Clear a restriction and go back to the defaults
komputer config researcher --allow-tool ''
```

Quote patterns containing `*` so your shell doesn't expand them.

**REST**:

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "researcher",
    "instructions": "Audit the repo",
    "allowedTools": ["Read", "Glob", "Grep"],
    "disallowedTools": ["Bash"]
  }'
```

**SDK** (Python; Go and TypeScript expose the same options):

```python
client.create_agent(
    "researcher",
    "Audit the repo",
    allowed_tools=["Read", "Glob", "Grep", "mcp__figma__get_design_context"],
)
```

**Manager agents** can set both fields when creating sub-agents via the `create_agent` MCP tool, and change them later with `patch_agent`.

### How the two fields enforce differently

They both stop a tool being used, but not in the same way — and the difference matters for token cost:

| | `disallowedTools` | `allowedTools` |
|---|---|---|
| Tool visible to the agent? | **No** — removed from its context entirely | **Yes** — it still sees non-listed tools |
| Tool callable? | No | No — the call is refused at invocation |
| What the agent is told | The tool simply doesn't exist | `<tool> is not in this agent's allowedTools` |

So `disallowedTools` is also the better choice when you want to keep a large connector out of the agent's context altogether, since unseen tools don't consume tokens. `allowedTools` is the only way to express "just these two tools from this connector" — it refuses the rest at call time rather than hiding them.

In both cases the agent cannot execute the tool; the difference is whether it knows the tool exists.

### Notes

- Tool names are **case-sensitive**: `Read`, not `read`.
- A pattern ending in `*` is a prefix match; anything else must match exactly.
- Both fields apply to squad members too — squad members use the same agent spec.
- Changes to an existing agent take effect the **next time the agent starts**, because the tool policy is applied when the agent's Claude session is constructed. Put the agent to sleep and wake it to apply immediately.
- When a tool is blocked, the agent is told which tool was denied and why, so it adapts and reports the limitation instead of failing silently.
- Scheduled agents support both fields too — set them under `spec.agent` on the `KomputerSchedule`, which accepts the full agent config. See [Schedules → Agent configuration](./schedules.md#agent-configuration).
