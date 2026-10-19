# 06 — Sleeping Agents

Use `lifecycle: Sleep` to run cost-efficient agents that spin down between tasks but keep their workspace.

## What it does

Creates an agent with `lifecycle: Sleep`. After each task completes, the pod is deleted — saving compute costs. When you send the next task, the operator starts a new pod and the agent resumes with its workspace intact.

## Run it

```bash
# Apply the agent (first task runs immediately)
kubectl apply -f agent.yaml

# Watch it work
komputer watch data-processor

# After the task completes, the pod is deleted (Phase → Sleeping)
kubectl get komputeragents data-processor

# Send another task — the operator wakes it up automatically
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "data-processor", "instructions": "Now generate charts from /workspace/output/summary.json"}'

# Or with the CLI
komputer create data-processor "Now generate charts from /workspace/output/summary.json"
```

## Lifecycle comparison

| Mode | Pod after task | PVC after task | Best for |
|------|---------------|----------------|---------|
| Default (empty) | Stays running | Kept | Interactive, frequent tasks |
| `Sleep` | Deleted | Kept | Infrequent tasks, cost-sensitive |
| `AutoDelete` | Deleted | Deleted | One-shot jobs |
| `Archive` | Deleted | Deleted (agent kept as a record) | One-shot jobs you want to audit |

## When to use Sleep

- **Batch jobs** — process files on a schedule, don't need the pod sitting idle between runs
- **Cost-sensitive workloads** — you're billed for pod CPU/memory even when idle
- **Data pipelines** — agent needs workspace persistence but only runs occasionally

## Key concepts

- **`lifecycle: Sleep`** — pod is deleted after each task; PVC and CR are kept
- **Phase transitions**: `Running → Sleeping` after task, `Sleeping → Running` when woken
- The operator detects the new task (via CR `instructions` change) and restarts the pod
- Workspace at `/workspace` is always available — the PVC survives across sleep cycles
- `pip install` and `npm install -g` packages in `/workspace` are also preserved
