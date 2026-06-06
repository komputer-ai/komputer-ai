---
title: Offices
description: Group of agents working under a manager — emerges from manager-worker interactions.
---

A **KomputerOffice** represents a group of agents working together under a manager. When a manager agent creates sub-agents, the system tracks them as an office — providing a single view of the group's progress, active agents, and total cost.

Offices are created automatically when a manager agent creates its first sub-agent. The office status tracks:

- The manager agent and all its members
- Per-member task status and cost
- Aggregate counts (total, active, completed agents)
- Total cost across all members

This is primarily a status/observability resource — you don't create offices directly, they emerge from manager-worker interactions.

## Inspecting an office

```bash
kubectl get komputeroffices -n team-platform
# NAME              PHASE        MANAGER        AGENTS  ACTIVE  COST
# release-cutover   InProgress   release-mgr    4       2       0.2418
```

A single office details view:

```bash
kubectl get komputeroffice release-cutover -n team-platform -o yaml
```

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerOffice
metadata:
  name: release-cutover
  namespace: team-platform
spec:
  manager: release-mgr           # the only field on .spec
status:
  phase: InProgress              # InProgress | Complete | Error
  manager:
    name: release-mgr
    role: manager
    taskStatus: InProgress
    lastTaskCostUSD: "0.0823"
  members:
    - name: changelog-writer
      role: worker
      taskStatus: Complete
      lastTaskCostUSD: "0.0411"
    - name: release-notes-poster
      role: worker
      taskStatus: InProgress
      lastTaskCostUSD: "0.0294"
    - name: smoke-tester
      role: worker
      taskStatus: Complete
      lastTaskCostUSD: "0.0890"
  totalAgents: 4
  activeAgents: 2
  completedAgents: 2
  totalCostUSD: "0.2418"
  totalTokens: 184223
  createdAt: "2026-06-06T14:02:11Z"
```

Or via the API:

```bash
curl http://komputer-api/api/v1/offices/release-cutover?namespace=team-platform
curl http://komputer-api/api/v1/offices?namespace=team-platform                # list
```

## Connector inheritance

When a manager creates a sub-agent, the operator copies the manager's `spec.connectors` onto the new worker so it inherits the same MCP tools (GitHub, Slack, etc.). You can override per-agent at create time. See `concepts/connectors.md`.
