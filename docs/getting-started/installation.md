---
title: Installation
description: Install komputer.ai on any Kubernetes cluster with Helm in under 5 minutes.
---

## Prerequisites

- Kubernetes cluster (Docker Desktop, kind, minikube, EKS, GKE, etc.)
- `kubectl` configured
- `helm` 3.x installed
- An [Anthropic API key](https://console.anthropic.com/) — or AWS Bedrock access (install with `--set bedrock.enabled=true --set bedrock.region=<region>` instead; auth is via IRSA on EKS or AWS credentials)
- [cert-manager](https://cert-manager.io/docs/installation/) installed in the cluster — required for the operator's admission webhook (which validates `KomputerSquad` resources). The chart provisions a self-signed `Issuer` + `Certificate` automatically; cert-manager just needs to be present:
  ```bash
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.20.2/cert-manager.yaml
  ```
  > If you genuinely cannot install cert-manager, disable the webhook with `--set webhooks.enabled=false` — but you lose squad-membership validation. See the [Helm chart README](https://github.com/komputer-ai/komputer-ai/tree/main/helm/komputer-ai#squad-admission-webhook) for the trade-off.

## 1. Create the Anthropic API key secret

> Skip this step if you're using AWS Bedrock — agents will authenticate via IRSA or AWS credentials instead, and no Anthropic key is needed.


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

### Using AWS Bedrock instead

```bash
helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set bedrock.enabled=true \
  --set bedrock.region=us-east-1 \
  --set agent.serviceAccount.name=komputer-agent \
  --namespace komputer-ai
```

Agents call Bedrock with the AWS default credential chain. On EKS, use IRSA: annotate an existing ServiceAccount with `eks.amazonaws.com/role-arn` and name it in `agent.serviceAccount.name`, as above. Agent models must be Bedrock IDs such as `us.anthropic.claude-sonnet-4-6`, or application inference profile ARNs.

The role needs these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream",
        "bedrock:ListInferenceProfiles",
        "bedrock:GetInferenceProfile"
      ],
      "Resource": [
        "arn:aws:bedrock:*:*:inference-profile/*",
        "arn:aws:bedrock:*:*:application-inference-profile/*",
        "arn:aws:bedrock:*:*:foundation-model/*"
      ]
    }
  ]
}
```

`bedrock:GetInferenceProfile` matters when agents use application inference profile ARNs: it's how Claude Code finds the model behind the ARN, and so prices each task at that model's rates. Without it, tasks still run, but their cost can be wrong. See [Claude Code on Amazon Bedrock](https://code.claude.com/docs/en/amazon-bedrock#iam-configuration) for the full policy, including the Marketplace permissions for first-time model access.

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

For custom installation options (external Redis, resource limits, etc.), see the Helm Chart's [`README`](https://github.com/komputer-ai/komputer-ai/tree/main/helm/komputer-ai). For building from source, see [Local Development](../contribution/local-development.md).
