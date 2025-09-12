#!/bin/bash
# Bootstrap ArgoCD Self-Managed
# Usage: ./bootstrap.sh [dev|prod]

set -euo pipefail

cd "$(dirname "$0")/.."
ENVIRONMENT=${1:-dev}

echo "🚀 Installing ArgoCD ${ENVIRONMENT} (self-managed mode)..."

# 1. Installation INITIALE d'ArgoCD (hors GitOps)
kubectl --kubeconfig ~/vixens/environments/${ENVIRONMENT}/kubeconfig apply -k ~/vixens/overlays/${ENVIRONMENT}/argocd-bootstrap --validate=false

# 2. Attendre qu'ArgoCD soit prêt
echo "⏳ Waiting for ArgoCD to be ready..."
kubectl --kubeconfig ~/vixens/environments/${ENVIRONMENT}/kubeconfig -n argocd wait --for=condition=available --timeout=300s deployment/argocd-server

# 3. Créer le project et l'app ArgoCD (première fois uniquement)
echo "🔧 Creating ArgoCD self-managed project and app..."
kubectl --kubeconfig ~/vixens/environments/${ENVIRONMENT}/kubeconfig apply -f ~/vixens/clusters/vixens/argocd/00-vixens-project.yaml
kubectl --kubeconfig ~/vixens/environments/${ENVIRONMENT}/kubeconfig apply -f ~/vixens/clusters/vixens/argocd/apps/argocd.yaml

# 4. Mot de passe admin
echo "🔑 ArgoCD Admin Password:"
kubectl --kubeconfig ~/vixens/environments/${ENVIRONMENT}/kubeconfig -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo ""

echo "✅ ArgoCD is now self-managed via GitOps!"
echo "📋 Next: Commit argocd/ files and watch self-management begin"
