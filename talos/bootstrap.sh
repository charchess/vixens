# bootstrap.sh (remplace l'ancien contenu helm)
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(dirname "$(realpath "$0")")
ENV="$1"

if [[ -z "$ENV" ]]; then
  echo "Usage: $0 <dev|prod>"
  exit 1
fi
if [[ "$ENV" != "dev" && "$ENV" != "prod" ]]; then
  echo "Erreur : l'argument doit être 'dev' ou 'prod'"
  exit 1
fi

export KUBECONFIG=~/vixens/kubeconfig-"$ENV"

# 1) Namespace
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -

# 2) Installer les manifests officiels (controller + CRDs) - version "stable"
# Ceci démarre ArgoCD (controller) pour permettre ensuite la gestion GitOps.
kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 3) Appliquer (optionnel) tes settings bootstrap locaux (ex: pre settings)
# # kubectl apply -f "$SCRIPT_DIR/00-pre-argo-settings.yaml" || true
kubectl apply -f "$SCRIPT_DIR/00-pre-argo-settings.yaml" || true
kubectl apply -f "$SCRIPT_DIR/00-pre-argo-settings.yaml" || true

echo "Bootstrap minimal ArgoCD lancé. Attends quelques instants que les pods démarrent..."
kubectl -n argocd get pods --watch --timeout=120s || true

echo "Ensuite, sync la racine vixens-root dans ArgoCD pour que le repo prenne le relais."
