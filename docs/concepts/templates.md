---
title: Templates
description: Define how an agent pod is configured — image, resources, env, storage, concurrency cap.
---

Templates define **how** an agent pod is configured — container image, resource limits, environment variables, tolerations, node selectors, storage size, and an optional concurrency cap (`maxConcurrentAgents`). They use full Kubernetes PodSpec passthrough, so anything you can put in a pod spec, you can put in a template.

There are two kinds of templates:

- **KomputerAgentClusterTemplate** — Cluster-scoped. Shared across all namespaces. This is where you typically define your default agent configuration.
- **KomputerAgentTemplate** — Namespace-scoped. If a namespace-scoped template exists with the same name as a cluster template, the namespace-scoped one takes precedence. This lets teams customize agent configuration without affecting the rest of the cluster.

When an agent is created, it references a template by name (defaulting to `"default"`). The operator resolves the template — checking the agent's namespace first, then falling back to cluster scope — and uses it to build the pod.

**Important:** Every template must set `spec.anthropicKeySecretRef` — a typed reference (`name`, `key`, optional `namespace`) pointing at the Secret holding your Anthropic API key. The operator mirrors that secret into the agent's namespace and injects `ANTHROPIC_API_KEY` into the pod automatically; you must **not** add `ANTHROPIC_API_KEY` to the template's `podSpec.env` yourself (the operator strips it if you do). When `namespace` is empty, the operator reads from the namespace komputer-ai was deployed into.
