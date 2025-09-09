#!/bin/bash
# Bootstrap ArgoCD - Premier démarrage cluster
# Usage: ./bootstrap.sh [dev|prod]

cd "$(dirname "$0")/.."
ENVIRONMENT=${1:-dev}

echo "Installing ArgoCD ${ENVIRONMENT} in namespace argocd..."
kubectl --kubeconfig ~/vixens/kubeconfig-${ENVIRONMENT} apply -k ~/vixens/overlays/${ENVIRONMENT}/argocd-bootstrap --validate=false
