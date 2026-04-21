# 04 — Manager with Sub-Agents

A manager agent that orchestrates multiple worker agents in parallel, then synthesizes their results.

## What it does

A manager agent spawns three worker agents concurrently — each researching a different domain of AI in healthcare. Once all workers complete, the manager synthesizes the findings into a final report.

> **Note:** You don't need to explicitly tell a manager to use sub-agents. Managers autonomously decide how to approach a task — they'll spawn sub-agents, parallelize work, and synthesize results whenever they judge it's the most efficient approach. The instructions in this example are deliberately specific to illustrate what's happening, but in practice you can just describe the goal and let the manager figure out the strategy.

## Run it

```bash
# Apply the manager agent
kubectl apply -f manager.yaml

# Watch the manager (you'll see it spawn sub-agents)
komputer watch research-manager

# The manager creates workers automatically — list them
kubectl get komputeragents

# Or via the API
curl http://localhost:8080/api/v1/agents
```

## What gets created

When the manager runs, it creates sub-agents via the komputer.ai MCP tools. The platform:

1. Creates `KomputerAgent` CRs for each sub-agent
2. Automatically groups them under a `KomputerOffice`
3. The manager polls sub-agent status and reads their results

```bash
# See the office that gets auto-created
kubectl get komputeroffices
komputer office list
komputer office get research-manager
```

## How manager orchestration works

Managers have access to MCP tools:
- `create_agent` — spawn a sub-agent with instructions
- `get_agent` — check a sub-agent's status and last message
- `list_agents` — see all agents in the namespace

Workers get `Bash` and `WebSearch` only. Managers get the orchestration tools on top.

## Key concepts

- **`role: manager`** — enables the MCP orchestration tools
- **`KomputerOffice`** — auto-created group tracking manager + all its sub-agents
- **`claude-opus-4-7`** — use a more capable model for managers that need to reason and coordinate
- Sub-agents default to `role: worker` when spawned by a manager
- Managers can read sub-agent results from `/workspace` via shared naming conventions, or by reading the `lastTaskMessage` status field
