# Local Development

How to build and run komputer.ai from source on a local Kubernetes cluster.

## Prerequisites

- Go 1.22+
- Docker
- A local Kubernetes cluster (Docker Desktop, kind, or minikube)
- An [Anthropic API key](https://console.anthropic.com/)

## Setup

### 1. Install CRDs

```bash
cd komputer-operator
make install
```

### 2. Apply local infrastructure

This creates Redis (as a Deployment), a service pointing to your locally-running API, and a Redis secret:

```bash
kubectl apply -f komputer-operator/config/samples/local-setup.yaml
```

### 3. Create the Anthropic API key secret

```bash
kubectl create secret generic anthropic-api-key \
  --from-literal=api-key=sk-ant-...
```

### 4. Apply platform config and default agent template

```bash
kubectl apply -f komputer-operator/config/samples/komputer_v1alpha1_komputerconfig.yaml
kubectl apply -f komputer-operator/config/samples/komputer_v1alpha1_komputeragentclustertemplate.yaml
```

### 5. Build the agent image

```bash
docker build -t komputer-agent:latest komputer-agent/
```

If you're using kind, load the image into the cluster:

```bash
kind load docker-image komputer-agent:latest --name <cluster-name>
```

### 6. Port-forward Redis

In a dedicated terminal:

```bash
kubectl port-forward svc/redis 6379:6379
```

### 7. Run the API

In a second terminal:

```bash
cd komputer-api
LOCAL=true REDIS_ADDRESS=localhost:6379 go run .
```

> **⚠️ Important:** `LOCAL=true` is required for local development. It disables direct HTTP calls to agent pods (which fail due to pod networking from outside the cluster) and uses `kubectl exec` as a fallback instead. Without it, features like agent history, file downloads, and session reads will time out.

The API starts on `http://localhost:8080`. The Swagger UI is available at `http://localhost:8080/swagger/index.html`.

### 8. Run the operator

In a third terminal:

```bash
cd komputer-operator
make run
```

### 9. Run the UI (optional)

In a fourth terminal:

```bash
cd komputer-ui
npm install
npm run dev
```

Opens at `http://localhost:3000`. Connects to the API on `http://localhost:8080` by default.

### 10. Build and use the CLI

In a fifth terminal:

```bash
cd komputer-cli
go build -o komputer .

./komputer login http://localhost:8080
./komputer run my-agent "Hello world"
```

## What local-setup.yaml creates

| Resource | Purpose |
|----------|---------|
| `Deployment/redis` | In-cluster Redis for event streaming |
| `Service/redis` | Exposes Redis at `redis:6379` inside the cluster |
| `Service/komputer-api` | Points to your locally-running API (via Endpoints) |
| `Endpoints/komputer-api` | Routes `komputer-api:8080` to Docker Desktop host IP (`192.168.65.254`) |
| `Secret/redis-secret` | Empty Redis password for local use |

The `komputer-api` Service + Endpoints trick allows agent pods running inside the cluster to reach your locally-running API on `http://komputer-api:8080`, which is what `KomputerConfig.spec.apiURL` points to.

## Rebuilding after changes

```bash
# After changing agent code
docker build -t komputer-agent:latest komputer-agent/
# kind: kind load docker-image komputer-agent:latest --name <cluster-name>
# Existing agents need to be deleted and recreated to pick up the new image

# After changing CRD types
cd komputer-operator
make generate    # regenerate deepcopy code
make manifests   # regenerate CRD manifests
make install     # apply updated CRDs to cluster

# API and operator pick up changes on restart (Ctrl+C and re-run)
```
