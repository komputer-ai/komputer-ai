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

## Minimal cluster template

The simplest usable template. This is roughly what the Helm chart installs as `default`:

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
metadata:
  name: default
spec:
  anthropicKeySecretRef:
    name: anthropic-api-key       # K8s Secret in the komputer-ai install namespace
    key: token                    # key inside that Secret
  storage:
    size: 10Gi
  podSpec:
    containers:
      - name: agent
        image: ghcr.io/komputer-ai/komputer-agent:latest
        resources:
          requests: { cpu: "500m", memory: "1Gi" }
          limits:   { cpu: "2",    memory: "4Gi" }
```

Every agent that doesn't override `spec.templateRef` uses this template.

## Advanced cluster template

A production-grade template with a concurrency cap, a custom storage class, a custom agent image, Bedrock-routed Claude, and a mounted ConfigMap. All standard Kubernetes PodSpec fields are passed through verbatim.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentClusterTemplate
metadata:
  name: production
spec:
  # Route Claude through AWS Bedrock — anthropicKeySecretRef is not required
  # when CLAUDE_CODE_USE_BEDROCK=1 is set on the pod (see env below).

  # Cap how many agents using this template can run concurrently per namespace.
  # 0 (default) = no cap. See concepts/agents.md "Concurrency Control".
  maxConcurrentAgents: 20

  storage:
    size: 50Gi
    storageClassName: fast-ssd

  podSpec:
    serviceAccountName: komputer-agent
    containers:
      - name: agent
        image: ghcr.io/my-org/komputer-agent-custom:v3.2.1
        imagePullPolicy: IfNotPresent
        env:
          - name: CLAUDE_CODE_USE_BEDROCK
            value: "1"
          - name: AWS_REGION
            value: us-east-1
        resources:
          requests: { cpu: "500m", memory: "1Gi" }
          limits:   { cpu: "2",    memory: "4Gi" }
        volumeMounts:
          - name: shared-prompts
            mountPath: /shared/prompts
            readOnly: true
    volumes:
      - name: shared-prompts
        configMap:
          name: prompt-library
```

## Namespace-scoped override

A `KomputerAgentTemplate` (note: no `Cluster`) lets a single team customize a template name without affecting other namespaces. When both exist, the namespace-scoped one wins **for agents in that namespace**.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgentTemplate
metadata:
  name: default                  # same name as a cluster template — overrides it for this ns
  namespace: team-research
spec:
  anthropicKeySecretRef:
    name: anthropic-api-key
    key: token
    namespace: komputer-ai       # explicit ns means the operator pulls from here, then mirrors in
  storage:
    size: 50Gi
  podSpec:
    containers:
      - name: agent
        image: ghcr.io/my-org/komputer-agent-research:latest
        resources:
          requests: { cpu: "1", memory: "2Gi" }
          limits:   { cpu: "2", memory: "4Gi" }
```

## Referencing a template from an agent

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerAgent
metadata:
  name: release-notes-writer
  namespace: team-product
spec:
  instructions: "Draft release notes for v2.4 from the merged PRs since v2.3."
  templateRef: production         # by name; resolution checks ns then cluster scope
  model: claude-sonnet-4-6
```

Via the API:

```bash
curl -X POST http://komputer-api/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "release-notes-writer",
    "instructions": "Draft release notes for v2.4 from the merged PRs since v2.3.",
    "templateRef": "production",
    "namespace": "team-product"
  }'
```

See `concepts/agents.md` for how `spec.podSpec` and `spec.storage` on the **agent** itself further override the template's pod for a single agent (e.g. one-off memory bump).
