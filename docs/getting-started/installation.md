---
title: Installation
description: Install komputer.ai on any Kubernetes cluster with Helm in under 5 minutes.
---

## Prerequisites

- Kubernetes cluster (Docker Desktop, kind, minikube, EKS, GKE, etc.)
- `kubectl` configured
- `helm` 3.x installed
- An [Anthropic API key](https://console.anthropic.com/)

## 1. Create the Anthropic API key secret

```bash
kubectl create namespace komputer-ai
kubectl create secret generic anthropic-api-key \
  --from-literal=api-key=sk-ant-... \
  -n komputer-ai
```

> Create this secret once in the install namespace. The operator mirrors it into every agent namespace automatically — you do **not** need to recreate it for each namespace where agents run, and rotations propagate across mirrors.

## 2. Install with Helm

```bash
helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set anthropicKeySecret.name=anthropic-api-key \
  --namespace komputer-ai
```

This deploys the operator, API, Redis, CRDs, and a default agent template — everything you need.

## 3. Install the CLI

Download from [GitHub Releases](https://github.com/komputer-ai/komputer-ai/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/komputer-ai/komputer-ai/releases/latest/download/komputer-darwin-arm64 -o komputer
chmod +x komputer && sudo mv komputer /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/komputer-ai/komputer-ai/releases/latest/download/komputer-linux-amd64 -o komputer
chmod +x komputer && sudo mv komputer /usr/local/bin/
```

## 4. Connect and run your first agent

```bash
# Port-forward the API and UI (or use an Ingress)
kubectl port-forward svc/komputer-ai-api 8080:8080 -n komputer-ai &
kubectl port-forward svc/komputer-ai-ui 3000:3000 -n komputer-ai &

# Open the dashboard
open http://localhost:3000

# Or use the CLI
komputer login http://localhost:8080
komputer run my-agent "Write a haiku about Kubernetes"
```

For custom installation options (external Redis, resource limits, etc.), see the Helm Chart's [`README`](https://github.com/komputer-ai/komputer-ai/tree/main/helm/komputer-ai). For building from source, see [Local Development](../contribution/local-development).
