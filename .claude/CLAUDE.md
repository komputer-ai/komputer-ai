# komputer-ai Architecture Rules

## 1. Thin Agent

The `komputer-agent` (Python) should contain minimal logic — just enough to run the Claude SDK and publish events to Redis. All business logic, state management, and orchestration belongs in `komputer-api` or `komputer-operator`.

When adding features, default to implementing in the API or operator. Only add code to the agent if it requires in-pod execution (Claude SDK interaction, workspace filesystem access).

## 2. CR Status as Database

`KomputerAgentStatus` is the single source of truth for agent state. Treat `.status` fields as a database — read from them for queries, write to them for state changes. No separate database.

To persist new agent state: add a field to `KomputerAgentStatus` in `komputer-operator/api/v1alpha1/komputeragent_types.go` and regenerate the CRD.

## 3. Redis is Just a Queue

Redis is exclusively a message transport (streams) for forwarding events from the agent to the API worker. It is NOT a data store, cache, or source of truth.

- Agent publishes events via `XADD` to Redis streams
- API worker consumes via `XREAD`, then writes state to the CR status
- Do not add Redis keys for storing state, config, or lookups
- `komputer-history:*` lists are for real-time WebSocket replay only, not a queryable store

## 4. CR Status Field Ownership

Each status field has exactly one writer. No component writes fields it doesn't own.

| Owner | Fields |
|-------|--------|
| **Operator** | Phase, PodName, PvcName, StartTime, CompletionTime, Message |
| **API worker** | TaskStatus, LastTaskMessage, SessionID, LastTaskCostUSD, TotalCostUSD |

When adding new status fields, decide the owner upfront and document it in the field comment.

## 5. Full-Stack Field Consistency

When adding a new field to `KomputerAgentSpec` or `KomputerAgentStatus`, it must be propagated to **all** layers:

| # | Layer | What to update |
|---|-------|----------------|
| 1 | **CRD types** | `komputer-operator/api/v1alpha1/komputeragent_types.go` |
| 2 | **CRD YAML** | Regenerate CRD + copy to `helm/komputer-ai/crds/` |
| 3 | **API request struct** | `CreateAgentRequest` in `komputer-api/handler.go` |
| 4 | **API response struct** | `AgentResponse` in `komputer-api/handler.go` |
| 5 | **API handlers** | All response paths in create, get, list, and wake handlers |
| 6 | **K8s client** | `komputer-api/k8s.go` — pass field when creating/updating CR |
| 7 | **CLI** | `komputer-cli/main.go` — add flag + include in request/display |
| 8 | **UI types** | `komputer-ui/src/lib/types.ts` — update `AgentResponse` / `CreateAgentRequest` |
| 9 | **UI components** | `komputer-ui/src/components/` — display/accept the field where relevant |

Do not merge a new field unless all layers are updated. A missing layer means clients can't see or set the field.

## 6. Full-Stack Feature Consistency

When adding a new capability (e.g. a new API endpoint, agent action, or operator behavior), ensure it is exposed across all relevant surfaces:

- **API** — endpoint + request/response structs
- **CLI** — command or flag
- **UI** — page, button, or display element
- **Operator** — reconciliation logic if it affects CR lifecycle

Not every feature needs all four — use judgment — but the default is to expose everywhere unless there's a reason not to.

## 7. Helm RBAC Must Match K8s API Usage

When adding a new Kubernetes API call in `komputer-api` or `komputer-operator` (e.g. creating a new resource type, reading a new API group), update the matching RBAC template in Helm:

- `komputer-api` → `helm/komputer-ai/templates/komputer-api/rbac.yaml`
- `komputer-operator` → `helm/komputer-ai/templates/komputer-operator/rbac.yaml`
- `komputer-agent` → `helm/komputer-ai/templates/komputer-agent/rbac.yaml`

Add the resource, API group, and verbs needed. Without this, the component will get `403 Forbidden` at runtime.

## 8. Tags Must Include Release Notes

When creating a git tag, always create a GitHub release with release notes. Summarize changes since the last tag using the commit history, grouped by category (features, fixes, docs, etc.). Use `gh release create` with a descriptive body.

## 9. Minimal Prompt Changes

When modifying agent system prompts (`komputer-api/prompt.go`, `komputer-agent/prompts.py`), keep additions as short as possible — a sentence or two, not whole sections. Prompts accumulate fast and directly impact token cost and context limits. Before adding, check if an existing line can be tweaked instead.

## 10. Surgical Changes, Cloud-Native Mindset

When implementing a feature or fix, make the smallest clean change that solves the problem. Do not bundle refactors, renames, or "improvements" unless the problem specifically requires them. Default to cloud-native patterns (CRDs, controllers, reconciliation loops, declarative config) — avoid inventing custom state machines or orchestration when Kubernetes primitives already handle it.
