# komputer-ai Helm Chart

Deploys the komputer.ai platform — distributed Claude AI agents on Kubernetes.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.x
- An [Anthropic API key](https://console.anthropic.com/)

## Installation

### 1. Create the namespace and API key secret

```bash
kubectl create namespace komputer-ai
kubectl create secret generic anthropic-api-key \
  --from-literal=api-key=sk-ant-... \
  -n komputer-ai
```

### 2. Install the chart

```bash
helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  --namespace komputer-ai
```

### 3. Access the UI

Port-forward both the UI and API:

```bash
# Terminal 1 — API
kubectl port-forward svc/komputer-ai-api 8080:8080 -n komputer-ai

# Terminal 2 — UI
kubectl port-forward svc/komputer-ai-ui 3000:3000 -n komputer-ai
```

Open [http://localhost:3000](http://localhost:3000) in your browser. The UI connects to the API at `http://localhost:8080` by default.

### 4. Verify

```bash
kubectl get pods -n komputer-ai
```

## Chart Structure

The chart templates are organized by component:

```
templates/
├── _helpers.tpl                    # Shared template helpers
├── validate.yaml                   # Input validation
├── komputer-ui/                    # Web dashboard
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   └── ingress.yaml
├── komputer-api/                   # REST + WebSocket API
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── serviceaccount.yaml
│   └── rbac.yaml
├── komputer-operator/              # Kubernetes operator
│   ├── deployment.yaml
│   ├── serviceaccount.yaml
│   └── rbac.yaml
└── komputer-agent/                 # Agent pod template and config
    ├── clustertemplate.yaml
    ├── komputerconfig.yaml
    ├── serviceaccount.yaml         # Created when agent.serviceAccount.create=true
    └── rbac.yaml                   # Created when agent.serviceAccount.create=true
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `anthropicApiKeySecret.name` | **(Required)** Name of the K8s Secret containing your Anthropic API key | `""` |
| `anthropicApiKeySecret.key` | Key within the secret | `api-key` |
| `operator.replicas` | Operator replica count | `1` |
| `operator.image.repository` | Operator image | `ghcr.io/komputer-ai/komputer-operator` |
| `operator.image.tag` | Operator image tag | `latest` |
| `ui.enabled` | Deploy the web dashboard | `true` |
| `ui.image.repository` | UI image | `ghcr.io/komputer-ai/komputer-ui` |
| `ui.image.tag` | UI image tag | `latest` |
| `ui.replicas` | UI replica count | `1` |
| `ui.apiUrl` | API URL the browser connects to (not in-cluster) | `http://localhost:8080` |
| `ui.service.type` | UI service type | `ClusterIP` |
| `ui.service.port` | UI service port | `3000` |
| `ui.ingress.enabled` | Create an Ingress for the UI | `false` |
| `api.replicas` | API replica count | `1` |
| `api.image.repository` | API image | `ghcr.io/komputer-ai/komputer-api` |
| `api.image.tag` | API image tag | `latest` |
| `api.service.type` | API service type | `ClusterIP` |
| `api.service.port` | API service port | `8080` |
| `api.ingress.enabled` | Create an Ingress for the API | `false` |
| `api.ingress.className` | Ingress class name (e.g. `nginx`, `traefik`, `alb`) | `""` |
| `api.ingress.annotations` | Ingress annotations | `{}` |
| `api.ingress.hosts` | Ingress host rules | See `values.yaml` |
| `api.ingress.tls` | Ingress TLS configuration | `[]` |
| `agent.image.repository` | Agent image used in the default cluster template | `ghcr.io/komputer-ai/komputer-agent` |
| `agent.image.tag` | Agent image tag | `latest` |
| `agent.defaultModel` | Default Claude model for new agents | `claude-sonnet-4-6` |
| `agent.serviceAccount.create` | Create a ServiceAccount, Role, and RoleBinding for agent pods | `false` |
| `agent.serviceAccount.rules` | RBAC rules granted to the agent service account | `[]` |
| `clusterTemplate.storageSize` | PVC size for agent workspaces | `5Gi` |
| `clusterTemplate.resources` | Resource requests/limits for agent pods | See `values.yaml` |
| `redis.enabled` | Deploy Redis HA subchart | `true` |
| `externalRedis.address` | External Redis address (when `redis.enabled=false`) | `""` |
| `externalRedis.passwordSecret.name` | Secret containing external Redis password | `""` |
| `imagePullSecrets` | Image pull secrets for private registries | `[]` |

See [`values.yaml`](values.yaml) for all options.

### External Redis

```bash
helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  --set redis.enabled=false \
  --set externalRedis.address=redis.prod:6379 \
  --set externalRedis.passwordSecret.name=redis-secret \
  --namespace komputer-ai
```

### Ingress

Expose the API and UI externally instead of using `kubectl port-forward`:

```yaml
# values-ingress.yaml
api:
  ingress:
    enabled: true
    className: nginx
    annotations:
      # WebSocket support (required for live streaming)
      nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
      nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: api.komputer.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: komputer-api-tls
        hosts:
          - api.komputer.example.com

ui:
  # Point the UI to the API's external URL (this runs in the browser)
  apiUrl: "https://api.komputer.example.com"
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: komputer.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: komputer-ui-tls
        hosts:
          - komputer.example.com
```

```bash
helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  -f values-ingress.yaml \
  --namespace komputer-ai
```

> **Note:** The API uses WebSockets for live event streaming. Make sure your ingress controller is configured with appropriate timeouts (shown above for nginx).
>
> **Important:** When using ingress, set `ui.apiUrl` to the API's external URL. This value is injected into the browser — it must be reachable from the user's machine, not from within the cluster.

### Private Container Registry

```bash
kubectl create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username=<user> \
  --docker-password=<pat> \
  -n komputer-ai

helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  --set imagePullSecrets[0].name=ghcr-pull-secret \
  --namespace komputer-ai
```

## Granting Kubernetes Access to Agents

By default, agent pods have no Kubernetes API access. If your agents need to use `kubectl` or interact with the Kubernetes API (e.g., to inspect pods, read logs, manage deployments), you must enable the agent service account.

### Enable the agent service account

Set `agent.serviceAccount.create=true` and provide the RBAC rules your agents need in `agent.serviceAccount.rules`.

**Example: read-only access to pods and deployments**

```yaml
# values-with-kubectl.yaml
agent:
  serviceAccount:
    create: true
    rules:
      - apiGroups: [""]
        resources: ["pods", "pods/log"]
        verbs: ["get", "list", "watch"]
      - apiGroups: ["apps"]
        resources: ["deployments", "replicasets"]
        verbs: ["get", "list", "watch"]
```

```bash
helm install komputer-ai oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --set anthropicApiKeySecret.name=anthropic-api-key \
  -f values-with-kubectl.yaml \
  --namespace komputer-ai
```

**Example: full access to a namespace (for DevOps agents)**

```yaml
agent:
  serviceAccount:
    create: true
    rules:
      - apiGroups: ["", "apps", "batch"]
        resources: ["*"]
        verbs: ["*"]
```

When `agent.serviceAccount.create` is `true`, the chart creates:

- A **ServiceAccount** (`<release>-agent`) in the release namespace
- A **Role** with the rules you specify in `agent.serviceAccount.rules`
- A **RoleBinding** binding the role to the service account

The service account is automatically set on the default `KomputerAgentClusterTemplate`, so all agent pods use it.

> **Security note:** Follow the principle of least privilege. Only grant the minimum permissions your agents actually need. The Role is namespace-scoped, so agents cannot access resources outside the release namespace.

## Upgrading

```bash
helm upgrade komputer oci://ghcr.io/komputer-ai/charts/komputer-ai \
  --namespace komputer-ai --reuse-values
```

## Uninstalling

```bash
helm uninstall komputer --namespace komputer-ai
```

This removes all chart-managed resources. Agent PVCs created by the operator are **not** deleted automatically — remove them manually if needed:

```bash
kubectl delete pvc -l app.kubernetes.io/managed-by=komputer-operator -n komputer-ai
```
