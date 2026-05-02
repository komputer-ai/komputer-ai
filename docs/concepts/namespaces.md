---
title: Namespaces
description: Namespace-aware isolation for agents, templates, secrets, and config.
---

komputer.ai is fully namespace-aware. Namespaces provide isolation boundaries for agents and their resources:

- **Agents** are namespace-scoped — two teams can each have an agent named `researcher` without conflict.
- **Templates** can be namespace-scoped (per-team overrides) or cluster-scoped (shared defaults).
- **Config** is cluster-scoped — one Redis and API configuration for the whole platform.
- **Secrets** live in the same namespace as their agent.

When the operator reconciles an agent it creates the agent's namespace if missing, then mirrors the Secrets referenced by the template's `anthropicKeySecretRef` and the cluster `KomputerConfig`'s `redis.passwordSecret` into that namespace before the pod is created. The mirrors track the source — rotating the source secret in the install namespace propagates to every agent namespace within a reconcile cycle.

> You only create the source Secrets **once**, in the namespace komputer-ai was deployed into (e.g. `komputer-ai`). Agents in other namespaces don't need their own copies. Mirrors are labelled `komputer.ai/mirrored-from-ns=<source-ns>` so you can tell them apart from secrets you manage yourself.

All API endpoints and CLI commands support namespace selection. If no namespace is specified, the server's default namespace is used.
