#!/usr/bin/env bash
# Deletes the local S3 session store test cluster (nothing else is touched).
set -euo pipefail
kind delete cluster --name komputer-s3-test --kubeconfig "${KUBECONFIG_S3_TEST:-$HOME/.kube/komputer-s3-test.yaml}"
rm -f "${KUBECONFIG_S3_TEST:-$HOME/.kube/komputer-s3-test.yaml}"
