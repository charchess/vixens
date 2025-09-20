    
# clusters/vixens/argocd/bootstrap.sh
#!/bin/bash
# Ce script unifié installe ArgoCD depuis le manifeste officiel et
# passe ensuite le contrôle à la structure GitOps App-of-Apps.

set -euo pipefail

ENVIRONMENT=${1:-dev}

echo "🚀 Bootstrapping ArgoCD for environment: ${ENVIRONMENT}"

# --- TEMPS 1 : INSTALLATION DE LA GRAINE ARGO CD ---

# 1. Créez le namespace (idempotent)
echo "🔧 [1/4] Creating namespace 'argocd'..."
untaint-control-plane.sh
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -

# 2. Installez ArgoCD depuis le manifeste officiel.
echo "🔧 [2/4] Installing ArgoCD core components from official manifest..."
untaint-control-plane.sh
#kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.11.0/manifests/install.yaml
helm repo add argo https://argoproj.github.io/argo-helm || echo "Repo 'argo' already exists."
helm repo update
helm upgrade --install argocd argo/argo-cd \
  --version 5.51.5 \
  --namespace argocd \
  --create-namespace \
  --wait \
  --set configs.cm."server\.insecure"=true \
  --set configs.cm."server\.disable\.auth"=true \
  --set configs.cm."users\.anonymous\.enabled"=true \
  --set server.insecure=true \
  --set server.disableAuth=true




# 3. Attendez que les CRDs et le déploiement principal soient prêts.
echo "⏳ [3/4] Waiting for ArgoCD CRDs and Deployment to be ready..."
untaint-control-plane.sh
kubectl wait --for condition=established --timeout=90s crd/applications.argoproj.io
kubectl wait --for condition=established --timeout=90s crd/appprojects.argoproj.io
kubectl -n argocd wait --for=condition=available --timeout=300s deployment/argocd-server
echo "✅ Initial ArgoCD server is running."

# --- TEMPS 2 : PASSAGE DE CONTRÔLE À GITOPS (LE HAND-OFF PROPRE) ---

# 4. Appliquez UNIQUEMENT le projet et l'Application Racine.
#    C'est le "hand-off" clair. On ne déploie rien d'autre manuellement.
#    ArgoCD prendra le relais à partir de ce point.
echo "🌱 [4/4] Handing off control to GitOps by applying the Project and Root Application..."
untaint-control-plane.sh
kubectl apply -f clusters/vixens/argocd/00-vixens-project.yaml
kubectl apply -f clusters/vixens/argocd/01-vixens-${ENVIRONMENT}-root.yaml

# --- FIN ---

echo ""
echo "✅ Bootstrap hand-off complete!"
echo "ArgoCD is now running and will begin managing itself from Git."
echo "Pour suivre le statut : kubectl get applications -n argocd -w"
echo ""
echo "🔑 Initial Admin Password:"
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo ""
