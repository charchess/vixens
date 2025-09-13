    
#!/bin/bash
# ./bootstrap.sh [dev|prod]
# Bootstrap ArgoCD using a 2-stage hand-off to GitOps self-management.

set -euo pipefail

ENVIRONMENT=${1:-dev}
ROOT_DIR="$(dirname "$0")/.."
KUBECONFIG="${ROOT_DIR}/talos/vixens-${ENVIRONMENT}/kubeconfig"

echo "🚀 Bootstrapping ArgoCD for environment: ${ENVIRONMENT}"

# --- TEMPS 1 : INSTALLATION DE LA GRAINE ARGO CD ---

# 1. Créez le namespace au cas où il n'existerait pas.
echo "🔧 [1/5] Creating namespace 'argocd'..."
kubectl --kubeconfig "$KUBECONFIG" create namespace argocd --dry-run=client -o yaml | kubectl apply -f -

# 2. Installez ArgoCD depuis le manifeste officiel. C'est la source la plus fiable pour les CRDs et les composants de base.
echo "🔧 [2/5] Installing ArgoCD core components and CRDs from official manifest..."
kubectl --kubeconfig "$KUBECONFIG" apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.11.0/manifests/install.yaml

# 3. Attendez que les CRDs soient enregistrées par Kubernetes. C'est l'étape qui corrige votre erreur actuelle.
echo "⏳ [3/5] Waiting for ArgoCD CRDs to be established..."
kubectl --kubeconfig "$KUBECONFIG" wait --for condition=established --timeout=90s crd/applications.argoproj.io
kubectl --kubeconfig "$KUBECONFIG" wait --for condition=established --timeout=90s crd/appprojects.argoproj.io
echo "✅ CRDs are ready."

# 4. Attendez que l'ArgoCD "graine" soit pleinement opérationnel.
echo "⏳ [4/5] Waiting for initial ArgoCD server to be available..."
untaint-control-plane.sh
kubectl --kubeconfig "$KUBECONFIG" -n argocd wait --for=condition=available --timeout=300s deployment/argocd-server
echo "✅ Initial ArgoCD server is running."

# --- TEMPS 2 : PASSAGE DE CONTRÔLE À GITOPS ---

# 5. Appliquez vos configurations (AppProject et l'Application Racine qui démarre tout).
#    C'est le "hand-off". On donne les clés du camion à ArgoCD lui-même.
echo "🌱 [5/5] Handing off control to GitOps by applying the Root Application..."
kubectl --kubeconfig "$KUBECONFIG" apply -k "${ROOT_DIR}/overlays/${ENVIRONMENT}/argocd-bootstrap"

# --- FIN ---

# Récupérez le mot de passe pour le premier accès.
echo "🔑 ArgoCD Admin Password:"
kubectl --kubeconfig "$KUBECONFIG" -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo ""
echo ""
echo "✅ Bootstrap hand-off complete!"
echo "ArgoCD is now running and will begin managing itself from Git."
echo "Le serveur ArgoCD peut redémarrer pendant qu'il se synchronise. C'est normal."
echo "Pour suivre le statut : kubectl -n argocd get applications"

  