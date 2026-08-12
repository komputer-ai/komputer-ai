---
title: Schedules
description: Run agent tasks on a cron schedule.
---

A **KomputerSchedule** runs agent tasks on a cron schedule. Use it for recurring work — nightly reports, periodic monitoring, scheduled analysis.

Key features:

- **Cron expression** — Standard 5-field cron (`min hour dom month dow`)
- **Instructions** — The task prompt the agent runs every tick (editable after creation)
- **Timezone** — IANA timezone support (defaults to UTC)
- **Suspend/resume** — Pause schedules without deleting them
- **Auto-delete** — Optionally delete the schedule after the first successful run
- **Keep agents** — When auto-deleting, optionally keep the created agents alive
- **Agent configuration** — Configure created agents with the full agent spec: model, role, template, secrets, skills, memories, connectors, tool permissions, storage, and more
- **Cost tracking** — Tracks total cost and per-run cost across all scheduled runs
- **Manual trigger** — Fire a schedule immediately, outside its cron cadence (UI: "Run now"; CLI: `komputer schedule trigger <name>`)

Schedules created through the API, CLI, UI, or SDK default to `Sleep` lifecycle for their agents, so compute is only used during the actual task execution. See [Defaults](#defaults) for the exact behavior, including how it differs under a raw `kubectl apply`.

## Minimal schedule

A nightly summary, GitOps-friendly:

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSchedule
metadata:
  name: nightly-signups
spec:
  schedule: "0 2 * * *"                # 02:00 every day
  timezone: "America/New_York"
  instructions: >
    Pull yesterday's signups from the analytics warehouse and post a one-line
    summary to Slack #growth. Highlight any anomalies vs. the trailing 7-day avg.
  agent:
    model: claude-sonnet-4-6
    lifecycle: Sleep                   # pod is torn down between runs
```

The `agent` block lets the schedule create its own dedicated agent on first fire. If you want the schedule to drive an **existing** agent instead, set `spec.agentName` (and omit `spec.agent`).

## Agent configuration

`spec.agent` accepts **every field a `KomputerAgent` spec accepts except `instructions`** — the schedule supplies those from its own top-level `instructions`, so the same task text is used whether a run creates a fresh agent or wakes an existing one.

The full set:

| Field | Purpose |
|---|---|
| `templateRef` | Which [template](./templates.md) to build the agent from |
| `systemPrompt` | Custom system prompt, appended to the internal one |
| `model` | Claude model to run |
| `role` | `worker` or `manager` (see [Roles](./agents.md#roles)) |
| `secrets` | K8s Secret names injected as env vars |
| `skills` | [Skills](./skills.md) to attach |
| `memories` | [Memories](./memories.md) to attach |
| `connectors` | [Connectors](./connectors.md) to attach |
| `allowedTools` | Restrict to an explicit tool list (see [Tool permissions](./agents.md#tool-permissions)) |
| `disallowedTools` | Remove specific tools |
| `lifecycle` | `""`, `Sleep`, or `AutoDelete` |
| `priority` | Admission order under a template's concurrency cap |
| `podSpec` | Pod overrides (resources, image, env) |
| `storage` | PVC size and storage class |
| `labels` | User-defined labels propagated to child resources |

Each behaves exactly as it does on a `KomputerAgent` — the schedule inlines the same underlying spec rather than mirroring a subset of it, so anything you can configure on an agent you can configure on a scheduled agent.

A weekday-9am stand-up bot that uses a custom template, references existing secrets and connectors, attaches a skill, and denies the shell. This is the shape you'd reach for in production.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSchedule
metadata:
  name: weekday-standup
  namespace: team-product
spec:
  schedule: "0 9 * * 1-5"              # 09:00 Mon–Fri
  timezone: "Europe/Berlin"
  instructions: >
    Fetch yesterday's merged PRs from Linear, summarise per assignee, and
    post the digest to Slack #product-standup. Use the linear and slack
    connector tools.
  agent:
    model: claude-sonnet-4-6
    lifecycle: Sleep
    role: worker
    templateRef: lightweight           # see concepts/templates.md
    secrets:
      - linear-credentials
      - slack-bot-token
    connectors:
      - linear
      - slack
    skills:
      - standup-format
    disallowedTools:
      - Bash                           # this bot only needs connector tools
    storage:
      size: 20Gi
```

### Defaults

A scheduled agent is a self-contained job, so two fields default differently than they would on a hand-written agent. When you omit them, the API fills in — and because the CLI, UI, SDK, and manager MCP tools all go through the API, every one of those clients gets the same behavior:

- **`role: worker`** — a scheduled agent runs one task; it isn't there to orchestrate sub-agents. Set `role: manager` explicitly if you want it to.
- **`lifecycle: Sleep`** — the pod is torn down between runs and the workspace PVC is preserved, so compute is only spent during the run itself.

Every other field falls back to the agent CRD's own default (`model: claude-sonnet-4-6`, `templateRef: default`, `priority: 0`) or is simply unset.

> **⚠ Raw `kubectl apply` gets `role: manager`, not `worker`.**
>
> The `worker` default is applied by the API, CLI, UI, SDK, and MCP paths — not by the CRD. `spec.agent` inlines the same shared config the agent spec uses, which carries the agent's own CRD default of `manager`, and the Kubernetes API server stamps CRD defaults on every write. The operator therefore never observes an empty `role` and cannot tell "unset" from "deliberately manager".
>
> This is inherent to sharing one spec between agents and schedules, not an oversight — removing the CRD default would change the agents CRD and break the parity this feature is built on. **If you apply a schedule with `kubectl` and want a worker, say so explicitly:**
>
> ```yaml
>   agent:
>     role: worker
> ```
>
> `lifecycle` has no CRD default, so its `Sleep` default is likewise API-side only; a `kubectl`-applied schedule gets the empty lifecycle (pod stays running after each run).

### `spec.agent` applies at agent creation only

`spec.agent` is a **template read once**, when a run finds no agent and creates one. It is not reconciled onto an agent that already exists: the wake path patches only `instructions`.

So editing `spec.agent.model` — or `skills`, `connectors`, `storage`, or any other field — on a schedule whose agent has already been created **has no effect on that agent**. The schedule will keep waking the same agent with its original configuration, and nothing will report an error.

To apply a changed `spec.agent`, delete the agent and let the next scheduled run recreate it from the updated template:

```bash
kubectl delete komputeragent <agent-name>
# or
komputer agent delete <agent-name>
```

The schedule's `status.agentName` tells you which agent it is currently driving. Schedules whose agents use `lifecycle: AutoDelete` are unaffected by this — each run creates a fresh agent, so a `spec.agent` edit lands on the very next run.

## One-off scheduled run

Use `autoDelete: true` to schedule a single future run that cleans itself up afterwards. Combine with `keepAgents: true` if you want the agent it created to survive.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSchedule
metadata:
  name: launch-eod-recap
spec:
  schedule: "0 18 30 6 *"              # 18:00 on June 30 (single instant)
  timezone: "UTC"
  autoDelete: true
  keepAgents: true                     # keep the agent it spawns
  instructions: "Compile a launch-day recap into /workspace/recap.md."
  agent:
    model: claude-sonnet-4-6
    lifecycle: Sleep
```

## Editing a schedule

The cron expression, the instructions, and the agent configuration can all be updated after creation. In the UI, the schedule detail page has inline edit controls for the cron expression, the instructions, and the core agent fields (model, lifecycle, role, template) — the full field set is available when you create the schedule. From the CLI:

```bash
komputer schedule update my-schedule --cron "0 9 * * 1-5"
komputer schedule update my-schedule --instructions "Summarize yesterday's signups."
komputer schedule update my-schedule --model claude-opus-4-6 --skill markdown-reports
```

`komputer schedule create` and `komputer schedule update` take the same agent flags — `--model`, `--role`, `--template`, `--secret`, `--skill`, `--memory`, `--allow-tool`, `--disallow-tool`, `--system-prompt`, `--priority`, `--cpu`, `--memory-limit`, `--storage`, `--image`, `--label`, `--lifecycle`. Attaching connectors to a scheduled agent is currently only available through the API, UI, and SDK, not the CLI.

Remember that agent-config edits only affect agents created **after** the edit — see [`spec.agent` applies at agent creation only](#specagent-applies-at-agent-creation-only).

> **⚠ A raw `PATCH` replaces `spec.agent` wholesale — it does not merge.**
>
> `PATCH /api/v1/schedules/{name}` overwrites the entire `agent` object with what you send. Posting `{"agent": {"model": "claude-opus-4-6"}}` to a schedule that had skills, connectors, and a storage override **clears all of them**.
>
> The CLI and UI compensate for you — `komputer schedule update` reads the current spec, applies your flags, and writes the merged result back. If you are calling the API directly, do the same: `GET` the schedule, merge your change into the returned `agent` object, then `PATCH` the whole thing.

## Manual trigger

You can fire a schedule outside its normal cron cadence. The schedule's next cron run is unaffected — this is purely an extra one-off run.

```bash
komputer schedule trigger my-schedule
# or
curl -X POST http://komputer-api/api/v1/schedules/my-schedule/trigger
```

The trigger returns `409 Conflict` if the schedule's last run is still in progress.

## Failure behavior

If a scheduled run fails (the agent returns an error), the schedule keeps firing at the next cron tick — failures do not pause the schedule. Only three things stop future runs:

- The schedule is suspended (`spec.suspended: true`)
- The schedule is `autoDelete: true` and has completed (it deletes itself)
- The cron expression is invalid (phase becomes `Error`)

The `failedRuns` counter in the status surfaces past failures so you can spot a repeatedly-failing schedule without it ever blocking itself.
