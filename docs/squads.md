# Squads

A squad is a named group of agents that share a single Kubernetes Pod, giving every member direct read/write access to each other's workspaces via the filesystem. The operator provisions one Pod per squad with all member containers and their PVCs mounted inside it — no coordination protocol, just shared files.

## When to use

Use a squad when agents need to exchange files directly:

- **Pair programming** — one agent writes code, another reviews or runs tests against the same files.
- **Pipeline workers** — an agent generates output files that the next agent in the chain reads.
- **Reviewer + coder** — one agent authors a diff, another applies feedback in place.

Do not use a squad when agents work on separate branches or unrelated tasks — solo agents are simpler and don't share resource lifecycle.

## Workspace layout

Inside the squad Pod each agent sees:

| Path | Contents |
|---|---|
| `/workspace` | The agent's own persistent workspace (its PVC) |
| `/agents/<sibling-name>/workspace` | Each sibling agent's workspace (read/write) |

All paths are read-write. There is no enforced isolation between members — any agent can write to a sibling's workspace.

## Lifecycle

### Creating a squad

- **UI** — New Agent dialog → Squad tab or Team Up tab.
- **CLI** — `komputer squad create --agents agent-a,agent-b`
- **API** — `POST /api/v1/squads`

The squad spec requires at least two members. Each member is either a `ref` (name of an existing agent) or an embedded `spec` (the operator creates the agent automatically and converts the entry to a ref on first reconcile).

### Adding a member to a running squad

Adding an agent to a squad that is already Running injects it as an **ephemeral container** — no Pod restart, existing members are unaffected.

**The agent being adopted must be asleep.** This prevents leaving its solo Pod stranded next to the new squad container. The API returns `409 Conflict` with `agent "<name>" must be asleep in order to team up` if the partner is in any other phase. The UI Team Up dialog surfaces this preflight and offers an inline Sleep button per non-sleeping agent.

CLI: `komputer squad add <squad> <agent>`
API: `POST /api/v1/squads/<name>/members`

### Sleep & wake (per-member)

Sleep on a squad member cancels its task and sets `Phase=Sleeping`. The member's container keeps running idle inside the squad pod (Kubernetes can't selectively stop one container in a pod). Once **every** member is Sleeping, the operator deletes the squad pod entirely (PVCs are preserved).

Waking a sleeping member sends a task to it normally — the API forwards the task and clears `Phase=Sleeping`. If the squad pod was deleted (all members were sleeping), the operator rebuilds it; the woken member runs its task, while still-sleeping siblings come up with `KOMPUTER_WAKE_IDLE=true` so their containers expose HTTP without auto-running prior instructions.

UI: Sleep button on the agent detail page or "Sleep all" on the squad detail page.
CLI: `komputer agent patch <name> --lifecycle Sleep`
API: `PATCH /api/v1/agents/<name>` with `{"lifecycle": "Sleep"}`

### Leave squad (single agent)

Removing one agent from a squad — the agent stays alive as a solo agent, its workspace is preserved, and any in-flight task on that member is cancelled.

UI: Leave Squad button on the agent detail page (replaces Team Up when in a squad).
CLI: `komputer squad remove <squad> <agent>`
API: `DELETE /api/v1/squads/<name>/members/<agent>`

### Break up the squad

Marks the squad for dissolution. The squad CR is deleted **once every member is asleep**; members then revert to solo agents (PVCs kept). Sending tasks to sleeping members in the meantime is allowed — they wake, run, return to Sleeping, and the break-up eventually completes.

UI: Break Up button on the squad detail page; the squad header shows a "Break-up pending" badge while waiting.
CLI: `komputer squad break-up <name>`
API: `POST /api/v1/squads/<name>/break-up`

### Empty squad — orphan TTL

When all members are removed the squad enters the `Orphaned` phase. After `orphanTTL` (default `10m`) the squad CR is deleted automatically. Set a custom TTL in the spec:

```yaml
spec:
  orphanTTL: 30m
```

### Single-member shrinkage

If a squad is reduced to exactly one member, it auto-dissolves: the squad Pod is deleted, the lone agent reverts to a solo Pod, and the squad CR is removed. No manual cleanup required.

## Limitations and caveats

- **One squad per agent.** An agent can belong to at most one squad at a time. This is enforced by an admission webhook — creating or patching a squad with an agent that is already in another squad is rejected.
- **Ephemeral container volume limitation.** When an agent is injected into a running Pod as an ephemeral container, Kubernetes does not allow adding new volumes to a running Pod. As a result the newly-added agent cannot mount its own PVC at `/workspace`. It *can* see all original members' workspaces at `/agents/<sibling>/workspace`. The agent's own `/workspace` becomes available after the next Pod restart (e.g. when another membership change triggers a recreate).
- **No resource requests for late-added members.** Ephemeral containers cannot declare resource requests/limits. Resources are only allocated on the next Pod restart when the agent becomes a regular container.
- **Pod name is the squad name.** `kubectl get pod` shows the squad Pod as `<squad-name>-pod`, not the individual agent names. Use `kubectl get pod <squad>-pod` to inspect it.
- **Cleanup finalizer.** Each squad carries a `komputer.ai/squad-cleanup` finalizer. Kubernetes blocks the actual deletion of a squad until the operator clears `Status.Squad` on every member — preventing orphaned members if the operator was down at delete time.

## GitOps

Members can be declared two ways in the squad CRD:

```yaml
spec:
  members:
    - ref:
        name: my-agent           # reference to an existing KomputerAgent
    - spec:                      # embedded spec — operator creates the agent
        task: "..."
```

**Prefer `ref` for GitOps.** When a member is declared with an embedded `spec`, the operator creates a `KomputerAgent` CR and rewrites the squad spec to a `ref` on first reconcile. This in-place mutation can surprise GitOps controllers (e.g. Argo CD, Flux) that detect drift. Declare agents separately and reference them by name to keep the squad spec stable.

## Manager tools

Manager agents can orchestrate squads using these MCP tools:

- `create_squad` — create a new squad
- `add_to_squad` — add an existing (sleeping) agent to a squad
- `remove_from_squad` — remove an agent from a squad
- `delete_squad` — delete a squad
- `list_squads` — list all squads

## Configuration

The validating admission webhook enforces the **one-squad-per-agent** constraint. It is enabled by default and **strongly recommended**:

```yaml
webhooks:
  enabled: true   # default; requires cert-manager
```

**With it on** — Kubernetes rejects conflicting squad create/update with a clear error like `agent "alice" is already in squad "squad-1"`. Requires cert-manager to be installed (the chart provisions a self-signed Issuer + Certificate automatically).

**With it off** — overlapping squads are allowed at the API. Two squads can both claim the same agent, causing the squad controller to race over `Phase=Squad` and Pod ownership: the agent flips between Pods and neither squad stabilizes. Only disable if cert-manager is unavailable and you accept this risk.
