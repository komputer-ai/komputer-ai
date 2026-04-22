# 07 — Custom Agent Image

Extend the base komputer-agent image with additional system tools and Python packages baked in.

## What it does

Builds a custom Docker image on top of `ghcr.io/komputer-ai/komputer-agent:latest` that includes PostgreSQL client, AWS CLI, FFmpeg, and common data science Python packages. Then creates a `KomputerAgentClusterTemplate` that uses this image and references it from an agent.

## Build the image

```bash
# Build your custom image
docker build -t my-registry/data-science-agent:latest .

# Push to your registry
docker push my-registry/data-science-agent:latest
```

## Apply the template and agent

```bash
# Create the cluster template (admin operation, once per cluster)
kubectl apply -f agent.yaml

# The agent references the template
komputer watch data-analyst
```

## Why extend instead of installing at runtime

Agents can install packages at runtime (`apt-get`, `pip`, `cargo`, etc.), but:

- **System packages** (`apt-get`) are **not persisted** — lost on pod restart
- **pip/npm installs** to `/workspace/.local` are persisted via PVC, but slow to reinstall on first run after a sleep cycle

Baking tools into the image means:
- Faster cold starts — no install step at the start of every task
- Reliable availability — no network dependency at runtime
- Consistent versions across all tasks

## Template precedence

The operator resolves templates in this order:
1. `KomputerAgentTemplate` (namespace-scoped) — for per-namespace overrides
2. `KomputerAgentClusterTemplate` (cluster-scoped) — for cluster-wide defaults

Use `KomputerAgentClusterTemplate` for images that all namespaces can share.

## Key concepts

- **`FROM ghcr.io/komputer-ai/komputer-agent:latest`** — always extend the official base
- **`USER komputer`** at the end — required; the Claude CLI refuses to run as root
- **`KomputerAgentClusterTemplate`** — cluster-scoped template, any namespace can reference it by name
- The agent's `templateRef: data-science` matches the template's `metadata.name`
- See [docs/custom-agent-image.md](../../docs/custom-agent-image.md) for the full guide
