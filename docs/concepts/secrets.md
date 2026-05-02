---
title: Secrets
description: How komputer.ai handles credentials — agent-specific Kubernetes Secrets injected as SECRET_* env vars.
---

Agents often need credentials to do their work — API keys, tokens, passwords. komputer.ai handles this through Kubernetes Secrets:

- When creating an agent, you can pass key-value secrets (e.g. `GITHUB=ghp_xxx`).
- The API creates a Kubernetes Secret and links it to the agent CR.
- The operator injects each key as a `SECRET_*` environment variable into the agent pod (e.g. `SECRET_GITHUB`).
- The agent's system prompt instructs Claude to check `SECRET_*` env vars when credentials are needed.
- When the agent is deleted, its secrets are automatically cleaned up via Kubernetes owner references.

Two platform secrets — the Anthropic API key (referenced by the template's `anthropicKeySecretRef`) and the Redis password (referenced by `KomputerConfig.spec.redis.passwordSecret`) — are managed separately by the operator. You create them once in the install namespace; the operator mirrors them into every agent namespace and injects `ANTHROPIC_API_KEY` into the pod automatically. Don't add `ANTHROPIC_API_KEY` to the template's `podSpec.env` yourself — the operator strips it.

Agent-specific secrets and the operator-managed mirrors are merged at pod creation time. If there's a conflict, agent secrets take precedence.
