---
title: Lifecycle
description: Creating, adding, leaving, breaking up — the full lifecycle of a squad and its members.
---

## Creating a squad

- **UI** — New Agent dialog → Squad tab or Team Up tab.
- **CLI** — `komputer squad create --agents agent-a,agent-b`
- **API** — `POST /api/v1/squads`

The squad spec requires at least two members. Each member is either a `ref` (name of an existing agent) or an embedded `spec` (the operator creates the agent automatically and converts the entry to a ref on first reconcile).

## Adding a member to a running squad

Adding an agent to a squad that is already Running injects it as an **ephemeral container** — no Pod restart, existing members are unaffected.

**The agent being adopted must be asleep.** This prevents leaving its solo Pod stranded next to the new squad container. The API returns `409 Conflict` with `agent "<name>" must be asleep in order to team up` if the partner is in any other phase. The UI Team Up dialog surfaces this preflight and offers an inline Sleep button per non-sleeping agent.

CLI: `komputer squad add <squad> <agent>`
API: `POST /api/v1/squads/<name>/members`

## Sleep & wake (per-member)

Sleep on a squad member cancels its task and sets `Phase=Sleeping`. The member's container keeps running idle inside the squad pod (Kubernetes can't selectively stop one container in a pod). Once **every** member is Sleeping, the operator deletes the squad pod entirely (PVCs are preserved).

Waking a sleeping member sends a task to it normally — the API forwards the task and clears `Phase=Sleeping`. If the squad pod was deleted (all members were sleeping), the operator rebuilds it; the woken member runs its task, while still-sleeping siblings come up with `KOMPUTER_WAKE_IDLE=true` so their containers expose HTTP without auto-running prior instructions.

UI: Sleep button on the agent detail page or "Sleep all" on the squad detail page.
CLI: `komputer agent patch <name> --lifecycle Sleep`
API: `PATCH /api/v1/agents/<name>` with `{"lifecycle": "Sleep"}`

Members can also sleep themselves on a timer via `spec.sleepTTL`, and delete themselves via `spec.deleteTTL` — set per member (or inherited from its template) exactly as for a solo agent. See [agent TTLs](../agents.md#ttls--auto-sleep-and-auto-delete). A TTL-slept member follows the same path as a manual sleep: phase only, with the shared pod torn down once every member is asleep. A TTL-deleted member is pruned from the squad on the next reconcile, which can in turn trigger single-member shrinkage or the orphan TTL below.

## Leave squad (single agent)

Removing one agent from a squad — the agent stays alive as a solo agent, its workspace is preserved, and any in-flight task on that member is cancelled.

UI: Leave Squad button on the agent detail page (replaces Team Up when in a squad).
CLI: `komputer squad remove <squad> <agent>`
API: `DELETE /api/v1/squads/<name>/members/<agent>`

## Break up the squad

Marks the squad for dissolution. The squad CR is deleted **once every member is asleep**; members then revert to solo agents (PVCs kept). Sending tasks to sleeping members in the meantime is allowed — they wake, run, return to Sleeping, and the break-up eventually completes.

UI: Break Up button on the squad detail page; the squad header shows a "Break-up pending" badge while waiting.
CLI: `komputer squad break-up <name>`
API: `POST /api/v1/squads/<name>/break-up`

## Empty squad — orphan TTL

When all members are removed the squad enters the `Orphaned` phase. After `orphanTTL` (default `10m`) the squad CR is deleted automatically. Set a custom TTL in the spec:

```yaml
spec:
  orphanTTL: 30m
```

## Single-member shrinkage

If a squad is reduced to exactly one member, it auto-dissolves: the squad Pod is deleted, the lone agent reverts to a solo Pod, and the squad CR is removed. No manual cleanup required.
