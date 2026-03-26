# komputer-operator

Kubernetes operator built with [operator-sdk](https://sdk.operatorframework.io/) that manages the lifecycle of Claude AI agents. It watches `KomputerAgent` custom resources and creates the necessary pods, persistent volumes, and configuration for each agent.

## CRDs

### KomputerRedisConfig

Cluster-scoped singleton holding Redis connection details. The operator auto-discovers this resource — no explicit reference needed.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerRedisConfig
metadata:
  name: default
spec:
  address: "redis:6379"
  db: 0
  queue: "komputer-events"
  passwordSecret:
    name: "redis-secret"
    key: "password"
```

### KomputerAgentTemplate

Reusable pod configuration with full `corev1.PodSpec` passthrough. Supports any pod-level settings — tolerations, node selectors, resource limits, env vars, etc.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentTemplate
metadata:
  name: default
spec:
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
  model: "claude-sonnet-4-20250514"  # optional, has default
```

**Status fields:**

```
kubectl get komputeragents
NAME       PHASE     TASK     MODEL                      AGE
my-agent   Running   ● Busy   claude-sonnet-4-20250514   5m
```

- `phase` — Pod lifecycle: Pending, Running, Succeeded, Failed
- `taskStatus` — Agent activity: Idle, Busy, Error (managed by the API worker)
- `podName`, `pvcName` — Names of created resources
- `startTime`, `completionTime` — Timestamps
- `lastTaskMessage` — Latest event summary

## Reconciliation Logic

When a `KomputerAgent` CR is created:

1. Resolves the `templateRef` to get the pod spec
2. Auto-discovers the singleton `KomputerRedisConfig`
3. Creates a PVC (`{name}-pvc`) for the agent's persistent workspace
4. Creates a ConfigMap (`{name}-pod-config`) with Redis config at `/etc/komputer/config.json`
5. Creates a Pod from the template, injecting:
   - `KOMPUTER_INSTRUCTIONS`, `KOMPUTER_MODEL`, `KOMPUTER_AGENT_NAME` env vars
   - Workspace PVC at `/workspace`
   - Config at `/etc/komputer`
6. Keeps the pod alive — recreates on termination
7. Updates CR status based on pod state

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
make run         # Runs against current kubeconfig cluster
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
│   └── komputerredisconfig_types.go
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
