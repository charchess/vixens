    
#!/bin/bash
# ./bootstrap.sh [dev|prod]
set -euo pipefail

ENVIRONMENT=${1:-dev}
ROOT_DIR="$(dirname "$0")/.."
# Assurez-vous que ce chemin est correct
KUBECONFIG="${ROOT_DIR}/talos/${ENVIRONMENT}/kubeconfig" 

echo "🚀 Bootstrapping ArgoCD for environment: ${ENVIRONMENT}"

# 1️⃣ ÉTAPE 1 : Appliquer la base minimale d'ArgoCD (SANS patchs de config)
#    (Assurez-vous que le kustomization.yaml ne contient que install.yaml + namespace)
echo "🔧 Installing minimal ArgoCD from raw manifests..."
kubectl --kubeconfig "$KUBECONFIG" apply -k "${ROOT_DIR}/overlays/${ENVIRONMENT}/argocd-bootstrap"

# 2️⃣ ÉTAPE 2 : Attendre que le serveur soit prêt
echo "⏳ Waiting for ArgoCD server deployment..."
kubectl --kubeconfig "$KUBECONFIG" -n argocd wait --for=condition=available --timeout=300s deployment/argocd-server

# 3️⃣ ÉTAPE 3 : Créer l'application "argocd" qui va se gérer elle-même
#    C'est cette application qui contient la VRAIE configuration (via Helm + patchs)
echo "🌱 Applying self-management ArgoCD Application..."
kubectl --kubeconfig "$KUBECONFIG" apply -f "${ROOT_DIR}/clusters/vixens/argocd/apps/argocd.yaml"

# 4️⃣ ÉTAPE 4 : Appliquer le reste (App of Apps)
echo "🌍 Installing the main App of Apps (vixens-root)..."
kubectl --kubeconfig "$KUBECONFIG" apply -f "${ROOT_DIR}/clusters/vixens/argocd/02-vixens-dev-root.yaml"

echo "⏳ ArgoCD is now synchronizing itself. This may take a moment."
echo "   Check status with: kubectl --kubeconfig ${KUBECONFIG} -n argocd get app argocd"

# 5️⃣ Retrieve admin password
echo "🔑 ArgoCD Admin Password (may become invalid if auth is disabled by GitOps sync):"
kubectl --kubeconfig "$KUBECONFIG" -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo ""

echo "✅ Bootstrap complete. ArgoCD will now manage its own configuration from Git."

