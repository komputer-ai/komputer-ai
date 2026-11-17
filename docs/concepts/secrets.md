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

## Limiting which secrets an agent can get

By default an agent can be given any Secret in its namespace: through `spec.secrets`, through the API's `secretRefs`, or by a manager agent's `attach_secret` tool. Sub-agents also inherit their office manager's secrets.

To limit that, set `allowedSecrets` on the agent template (`KomputerAgentTemplate` or `KomputerAgentClusterTemplate`):

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
metadata:
  name: default
spec:
  allowedSecrets:
    - github-*        # prefix match
    - slack-token     # exact name
```

For the chart's default template, set `clusterTemplate.allowedSecrets`.

With a list set:

- The API rejects `secretRefs` outside the list (`400`) on create and on `PATCH`.
- The operator injects only allowed Secrets, both the agent's own and those inherited from its manager. Anything else is skipped and logged, so a Secret that slips into `spec.secrets` some other way still isn't handed out.
- An empty list (the default) keeps today's behavior.

Pick the template per trust level. For example, give manager agents (which can create sub-agents and attach secrets) a template that only allows read-only credentials.

### Env vars vs. files

Secrets are injected as `SECRET_*` environment variables. Mounting them as files instead wouldn't keep them from the agent: the agent runs Bash and can read anything its process can, file or env. What protects a secret is not giving it to an agent that doesn't need it, which is what `allowedSecrets` controls. Pair it with `allowedTools` / `allowedCommands` if an agent should hold a credential but not run arbitrary commands with it.
