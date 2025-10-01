#!/usr/bin/env bash
# run-bootstrap.sh <env>
# Render ArgoCD chart (helm template), apply core manifests, then apply bootstrap kustomize for env.
set -euo pipefail

ENV="${1:-dev}"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VALUES_FILE="${REPO_ROOT}/bootstrap/argocd-values.yaml"
CORE_MANIFEST="${REPO_ROOT}/bootstrap/argocd-core-manifests.yaml"
ARGO_CHART_VERSION="3.7.1"

command -v helm >/dev/null || { echo "helm is required"; exit 1; }
command -v kubectl >/dev/null || { echo "kubectl is required"; exit 1; }

echo "[INFO] Rendering ArgoCD manifests (chart v${ARGO_CHART_VERSION})..."
helm repo add argo https://argoproj.github.io/argo-helm >/dev/null 2>&1 || true
helm repo update >/dev/null 2>&1

helm template argocd argo/argo-cd --version "${ARGO_CHART_VERSION}" -n argocd -f "${VALUES_FILE}" --include-crds > "${CORE_MANIFEST}"

echo "[INFO] Applying ArgoCD core manifests..."
kubectl apply -f "${CORE_MANIFEST}"

echo "[INFO] Waiting for argocd-server deployment (up to 180s)..."
if kubectl -n argocd get deployment argocd-server >/dev/null 2>&1; then
  kubectl -n argocd wait --for=condition=available deployment/argocd-server --timeout=180s || echo "[WARN] argocd-server not yet available"
else
  echo "[INFO] argocd-server deployment not yet present; controller will reconcile."
fi

echo "[INFO] Applying bootstrap kustomize for env '${ENV}'..."
kubectl apply -k "${REPO_ROOT}/bootstrap/${ENV}"

echo "[DONE] Bootstrap applied for ${ENV}."
echo " - Check: kubectl -n argocd get pods"
echo " - Check Argo Apps: kubectl -n argocd get applications.argoproj.io"
