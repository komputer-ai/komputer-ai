---
title: Config
description: Cluster-scoped singleton that holds platform-wide settings.
---

**KomputerConfig** is a cluster-scoped singleton that holds platform-wide settings:

- **Redis connection** — Address, database number, stream prefix, and optional password secret. Redis is the event bus that connects agents to the API.
- **API URL** — The internal cluster URL of the komputer-api service. Manager agents use this to create and manage sub-agents via HTTP.

The operator auto-discovers this resource — agents and templates don't need to reference it explicitly.

## Minimal config

The Helm chart installs this for you. You only need to author one by hand if you're managing the install via raw manifests or pointing at an externally-managed Redis.

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerConfig
metadata:
  name: komputer
spec:
  redis:
    address: komputer-redis.komputer-ai.svc.cluster.local:6379
  apiURL: http://komputer-api.komputer-ai.svc.cluster.local:8080
```

## Config with auth + custom DB

When Redis is shared with other workloads, isolate by `db` and `streamPrefix` and authenticate via a Secret:

```yaml
apiVersion: komputer.komputer.ai/v1alpha1
kind: KomputerConfig
metadata:
  name: komputer
spec:
  redis:
    address: redis.shared-infra.svc.cluster.local:6379
    db: 3
    streamPrefix: komputer-events-prod      # default: "komputer-events"
    passwordSecret:
      name: shared-redis-creds              # K8s Secret
      key: password
      # namespace: shared-infra             # optional; defaults to install namespace
  apiURL: http://komputer-api.komputer-ai.svc.cluster.local:8080
```

## Verifying

```bash
kubectl get komputerconfig
# NAME       AGE
# komputer   17d

kubectl describe komputerconfig komputer
```

The API and operator log a `redis worker started` message at boot with the resolved address — if you see that, the config is loaded.
