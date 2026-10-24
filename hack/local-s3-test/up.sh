#!/usr/bin/env bash
# Brings up an isolated local test environment for the S3 session store:
# a kind cluster (in Docker Desktop's Docker), MinIO as S3, and komputer-ai
# built from this checkout. It never touches your current kubectl context —
# everything uses its own kubeconfig.
#
#   hack/local-s3-test/up.sh            # build images + create/upgrade everything
#   SKIP_BUILD=1 hack/local-s3-test/up.sh
#
# Then: API http://localhost:28080, UI http://localhost:23000,
#       MinIO console http://localhost:29001 (minioadmin / minioadmin-local).
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
CLUSTER=komputer-s3-test
export KUBECONFIG="${KUBECONFIG_S3_TEST:-$HOME/.kube/komputer-s3-test.yaml}"
# Bedrock credentials for agents are copied from this context (secret
# default/komputer-aws-creds). Override with SOURCE_CONTEXT=… or set
# SKIP_AWS_COPY=1 and create the secret yourself.
SOURCE_CONTEXT="${SOURCE_CONTEXT:-docker-desktop}"

log() { printf '\n==> %s\n' "$*"; }

if ! kind get clusters | grep -qx "$CLUSTER"; then
  log "Creating kind cluster $CLUSTER (kubeconfig: $KUBECONFIG)"
  kind create cluster --name "$CLUSTER" --config "$HERE/kind.yaml" --kubeconfig "$KUBECONFIG"
fi

if [ -z "${SKIP_BUILD:-}" ]; then
  log "Building images from $ROOT"
  docker build -t komputer-operator:s3-test "$ROOT/komputer-operator"
  docker build -t komputer-api:s3-test -f "$ROOT/komputer-api/Dockerfile" "$ROOT"
  docker build -t komputer-agent:s3-test "$ROOT/komputer-agent"
  docker build -t komputer-ui:s3-test "$ROOT/komputer-ui"
fi
log "Loading images into kind"
for img in komputer-operator komputer-api komputer-agent komputer-ui; do
  kind load docker-image "$img:s3-test" --name "$CLUSTER"
done

log "MinIO"
# MinIO no longer publishes community images to Docker Hub; use the local copies,
# re-tagged as single-platform images (kind can't import a multi-arch index
# whose other platforms aren't in the local cache).
for img in minio mc; do
  docker image inspect "minio/$img:latest" >/dev/null 2>&1 || { echo "missing local image minio/$img:latest (docker pull it from a mirror first)"; exit 1; }
  echo "FROM minio/$img:latest" | docker build -q -t "local-$img:s3-test" - >/dev/null
  kind load docker-image "local-$img:s3-test" --name "$CLUSTER"
done
kubectl apply -f "$HERE/minio.yaml"
kubectl -n minio rollout status deploy/minio --timeout=180s
kubectl -n minio wait --for=condition=complete job/make-bucket --timeout=180s

log "Secrets"
kubectl create namespace komputer-system --dry-run=client -o yaml | kubectl apply -f -
kubectl -n komputer-system create secret generic komputer-s3 \
  --from-literal=AWS_ACCESS_KEY_ID=minioadmin --from-literal=AWS_SECRET_ACCESS_KEY=minioadmin-local \
  --dry-run=client -o yaml | kubectl apply -f -
if [ -z "${SKIP_AWS_COPY:-}" ]; then
  # Copy without ever printing the values: strip cluster metadata, re-apply.
  kubectl --kubeconfig "$HOME/.kube/config" --context "$SOURCE_CONTEXT" -n default get secret komputer-aws-creds -o json \
    | python3 -c 'import json,sys; s=json.load(sys.stdin); print(json.dumps({"apiVersion":"v1","kind":"Secret","type":s.get("type","Opaque"),"metadata":{"name":"komputer-aws-creds","namespace":"default"},"data":s["data"]}))' \
    | kubectl apply -f -
fi

log "komputer-ai (helm)"
helm dependency build "$ROOT/helm/komputer-ai" >/dev/null
helm upgrade --install komputer-ai "$ROOT/helm/komputer-ai" -n komputer-system -f "$HERE/values.yaml" --wait --timeout 5m

log "Agent template: Bedrock credentials via envFrom (as in your Docker Desktop install)"
kubectl patch komputeragentclustertemplate default --type=json \
  -p '[{"op":"add","path":"/spec/podSpec/containers/0/envFrom","value":[{"secretRef":{"name":"komputer-aws-creds"}}]}]'

log "Host access (NodePorts)"
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Service
metadata: { name: komputer-ai-api-nodeport, namespace: komputer-system }
spec:
  type: NodePort
  selector: { app.kubernetes.io/name: komputer, app.kubernetes.io/instance: komputer-ai, app.kubernetes.io/component: api }
  ports: [{ port: 8080, targetPort: 8080, nodePort: 30080 }]
---
apiVersion: v1
kind: Service
metadata: { name: komputer-ai-ui-nodeport, namespace: komputer-system }
spec:
  type: NodePort
  selector: { app.kubernetes.io/name: komputer, app.kubernetes.io/instance: komputer-ai, app.kubernetes.io/component: ui }
  ports: [{ port: 3000, targetPort: 3000, nodePort: 30300 }]
EOF

log "Ready"
echo "  API:   http://localhost:28080/api/v1/config"
echo "  UI:    http://localhost:23000"
echo "  MinIO: http://localhost:29001  (minioadmin / minioadmin-local)"
echo "  kubectl: export KUBECONFIG=$KUBECONFIG"
