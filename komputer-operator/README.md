# komputer-operator

Kubernetes operator built with [operator-sdk](https://sdk.operatorframework.io/) that manages the lifecycle of Claude AI agents. It watches `KomputerAgent` custom resources and creates the necessary pods, persistent volumes, and configuration for each agent.

## CRDs

### KomputerConfig

Cluster-scoped singleton holding platform configuration — Redis connection and internal API URL. The operator auto-discovers this resource.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerConfig
metadata:
  name: default
spec:
  redis:
    address: "redis.default:6379"
    db: 0
    streamPrefix: "komputer-events"
    passwordSecret:
      name: "redis-secret"
      key: "password"
  apiURL: "http://komputer-api.default.svc.cluster.local:8080"
```

The `apiURL` is used by manager agents to create and manage sub-agents via the komputer-api.

### KomputerAgentClusterTemplate

Cluster-scoped reusable pod configuration with full `corev1.PodSpec` passthrough. Shared across all namespaces.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
metadata:
  name: default
spec:
  maxConcurrentAgents: 0   # 0 = no cap; set >0 to queue agents above the limit
  podSpec:
    containers:
      - name: agent
        image: komputer-agent:latest
        env:
          - name: ANTHROPIC_API_KEY
            valueFrom:
              secretKeyRef:
                name: anthropic-api-key
                key: api-key
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "2Gi"
  storage:
    size: "5Gi"
```

`maxConcurrentAgents` caps how many agents using this template can be in `Phase=Running` per namespace at once. Excess agents enter `Phase=Queued` and are admitted in priority order (see `KomputerAgent.spec.priority`). Counted against both namespaced and cluster templates of the same name.

### KomputerAgentTemplate

Namespace-scoped version of the template. If a `KomputerAgentTemplate` exists in the agent's namespace with the same name as a `KomputerAgentClusterTemplate`, the namespaced template takes precedence.

### KomputerAgent

An agent instance. The operator creates a pod and PVC when this resource is created.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: my-agent
spec:
  templateRef: "default"          # optional, defaults to "default"
  instructions: "Research AI news"
  model: "claude-sonnet-4-6"     # optional, has default
  role: "manager"                 # "manager" or "worker"
  lifecycle: "Sleep"              # "", "Sleep", or "AutoDelete"
  priority: 0                     # higher = admitted first when template's maxConcurrentAgents is reached
  secrets:                        # optional list of K8s Secret names
    - my-agent-secrets
  podSpec:                        # optional — merged into the template's PodSpec at pod build
    containers:                   # containers matched by name; only set fields override
      - name: agent
        resources:
          requests: { cpu: "4", memory: "8Gi" }
          limits:   { cpu: "4", memory: "8Gi" }
  storage:                        # optional — overrides template storage; expands existing PVC if class supports it
    size: 50Gi
```

- `role`: `manager` agents get orchestration tools (MCP) to create and manage sub-agents, while `worker` agents only have bash and web search tools.
- `lifecycle`: Controls what happens after task completion. Default (`""`) keeps the pod running. `Sleep` deletes the pod but keeps the PVC — the agent wakes up on the next task. `AutoDelete` deletes the entire agent.
- `priority`: Signed int32, PodPriority-style. Higher = admitted first when the template's `maxConcurrentAgents` cap is reached. Defaults to 0.
- `secrets`: List of K8s Secret names. Each key in each secret is injected as an env var into the agent pod. The operator deduplicates env vars — agent secrets override template env vars with the same name.
- `podSpec` / `storage`: Per-agent overrides. Apply to the **next** pod build — running pods are not mutated. PodSpec containers are merged by name (only the non-zero fields you set override the template). Storage size growth triggers in-place PVC expansion if the StorageClass has `allowVolumeExpansion: true`.

**Status fields:**

```
kubectl get komputeragents
NAME       PHASE      TASK         COST     MODEL              AGE
my-agent   Running    InProgress   0.0842   claude-sonnet-4-6  5m
sleepy     Sleeping   Complete     0.0231   claude-sonnet-4-6  1h
```

- `phase` — Pod lifecycle: Pending, Running, Queued, Sleeping, Succeeded, Failed
- `taskStatus` — Agent activity: Complete, InProgress, Error (managed by the API worker)
- `lastTaskCostUSD` — Cost of the most recent task
- `totalCostUSD` — Cumulative cost of all tasks (shown as Cost column)
- `podName`, `pvcName` — Names of created resources
- `startTime`, `completionTime` — Timestamps
- `lastTaskMessage` — Latest event summary
- `sessionId` — Claude session ID for conversation continuity
- `queuePosition`, `queueReason` — When `phase=Queued`, the 1-based position in the template's admission queue and the reason text

### KomputerOffice

Tracks a group of agents working under a manager. Created automatically when a manager agent creates sub-agents.

```
kubectl get komputeroffices
NAME         PHASE        MANAGER      AGENTS   ACTIVE   COST     AGE
research     InProgress   researcher   4        2        0.3210   10m
```

- `phase` — InProgress, Complete, Error
- `members` — Per-agent status (name, role, task status, cost)
- `totalCostUSD` — Aggregate cost across all members

### KomputerSchedule

Runs agent tasks on a cron schedule.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerSchedule
metadata:
  name: nightly-report
spec:
  schedule: "0 2 * * *"           # 2:00 AM daily
  timezone: "America/New_York"
  instructions: "Generate the daily performance report"
  agent:
    model: "claude-sonnet-4-6"
    lifecycle: "Sleep"             # default for schedules
    role: "worker"
```

```
kubectl get komputerschedules
NAME             SCHEDULE      PHASE    AGENT            RUNS   COST     NEXT                   AGE
nightly-report   0 2 * * *     Active   nightly-report   7      0.5600   2026-03-30T06:00:00Z   7d
```

- `suspended` — Pause without deleting
- `autoDelete` — Delete the schedule after first successful run
- `keepAgents` — When auto-deleting, keep the created agents alive
- Tracks run count, success/fail counts, per-run and total cost

## Reconciliation Logic

When a `KomputerAgent` CR is created:

1. Resolves the `templateRef` to get the pod spec (checks namespaced template first, then cluster template)
2. Applies per-agent overrides — `spec.podSpec` containers are merged by name into the template, and `spec.storage` replaces the template's storage block
3. Auto-discovers the singleton `KomputerConfig`
4. **Admission gate** — if the resolved template sets `maxConcurrentAgents > 0`, the operator counts how many other agents in the namespace using the same template are in `Phase=Running`. If the count is at or above the cap, this agent is moved to `Phase=Queued` with `queuePosition` set, and reconciliation returns without creating a pod. When other agents transition out of Running (or are deleted), a sibling watch re-enqueues queued agents and the highest-priority one (by `spec.priority`, then creation timestamp, then name) is admitted.
5. Creates a PVC (`{name}-pvc`) for the agent's persistent workspace; if it already exists and `spec.storage.size` has grown, patches it for in-place expansion (no-op if the StorageClass doesn't support `allowVolumeExpansion`)
6. Creates a ConfigMap (`{name}-pod-config`) with Redis config at `/etc/komputer/config.json`
7. Creates a Pod from the template, injecting:
   - `KOMPUTER_INSTRUCTIONS`, `KOMPUTER_MODEL`, `KOMPUTER_AGENT_NAME`, `KOMPUTER_NAMESPACE` env vars
   - For managers: `KOMPUTER_ROLE`, `KOMPUTER_API_URL`
   - Env vars from `spec.secrets` (each key in each referenced K8s Secret)
   - Workspace PVC at `/workspace`
   - Config at `/etc/komputer`
8. Manages pod lifecycle based on `spec.lifecycle`:
   - Default: keeps the pod alive, recreates on termination
   - Sleep: deletes the pod after task completion, preserves PVC
   - AutoDelete: deletes the entire agent after task completion
9. Updates CR status based on pod state

## Development

### Prerequisites

- Go 1.22+
- operator-sdk v1.42+
- A Kubernetes cluster

### Build and test

```bash
make generate    # Regenerate deepcopy code
make manifests   # Regenerate CRD manifests
make test        # Run integration tests with envtest
go build ./...   # Build
```

### Install CRDs

```bash
make install     # Uses server-side apply (required for large PodSpec CRD)
```

### Run locally

```bash
KOMPUTER_API_URL=http://localhost:8080 make run
```

For HA deployments, enable leader election:

```bash
make run ARGS="--leader-elect"
```

### Deploy to cluster

```bash
make docker-build IMG=<registry>/komputer-operator:latest
make docker-push IMG=<registry>/komputer-operator:latest
make deploy IMG=<registry>/komputer-operator:latest
```

## Project Structure

```
komputer-operator/
├── api/v1alpha1/                    # CRD type definitions
│   ├── komputeragent_types.go
│   ├── komputeragenttemplate_types.go
│   ├── komputeragentclustertemplate_types.go
│   ├── komputerconfig_types.go
│   └── komputerredisconfig_types.go  # Legacy, kept for migration
├── internal/controller/
│   ├── komputeragent_controller.go      # Reconciliation logic
│   └── komputeragent_controller_test.go # Integration tests
├── cmd/main.go                      # Manager entrypoint
├── config/
│   ├── crd/bases/                   # Generated CRD manifests
│   ├── rbac/                        # RBAC rules
│   └── samples/                     # Example CRs
├── Makefile
└── Dockerfile
```
