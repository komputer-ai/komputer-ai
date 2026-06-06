---
title: Squads overview
description: A named group of agents that share a single Kubernetes Pod, giving every member direct read/write access to each other's workspaces.
---

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

## Inline members — create a squad with new agents

Each `members` entry can be a full inline agent spec. This is the one-shot path: declare the squad and its brand-new members in a single CR.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSquad
metadata:
  name: review-pair
  namespace: team-platform
spec:
  members:
    - name: coder                       # KomputerAgent will be created with this name
      spec:
        instructions: "Implement the feature described in /agents/reviewer/workspace/SPEC.md."
        model: claude-sonnet-4-6
        role: worker
        lifecycle: Sleep
        templateRef: default
    - name: reviewer
      spec:
        instructions: "Drop a SPEC.md in your workspace, then review the coder's diff and write feedback."
        model: claude-sonnet-4-6
        role: worker
        lifecycle: Sleep
        templateRef: default
```

Inside the squad Pod each member sees both workspaces:

```
/agents/coder/workspace        # coder's own PVC
/agents/reviewer/workspace     # reviewer's PVC, also mounted in the coder container (and vice-versa)
/workspace                     # alias for the current member's own workspace
```

## Member refs — bring existing agents into a squad

Use `ref` when the agents already exist as `KomputerAgent` CRs and you just want to co-locate them. Mix `ref` and inline `spec` freely in the same `members` list.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSquad
metadata:
  name: rebuild-strike-team
spec:
  members:
    - ref: { name: senior-architect }     # existing agent
    - ref: { name: legacy-explorer }      # existing agent
    - name: scribe                         # plus one brand-new member inline
      spec:
        instructions: "Take notes from the conversation in /agents/*/workspace and write a refactor plan."
        role: worker
        lifecycle: Sleep
```

A given agent can only be a member of **one** squad at a time — see `caveats.md`.

## Tuning lifecycle

```yaml
spec:
  orphanTTL: "30m"          # delete the squad if every member is asleep this long (default: 10m)
  breakUpRequested: true    # gracefully dissolve once all members are sleeping; PVCs survive
  members: [...]
```

See `lifecycle.md` for the state-machine details, `manager-tools.md` for the MCP tools a manager agent uses to spin up squads on the fly, and `gitops.md` for the `ref`-vs-`spec` tradeoffs in declarative setups.
