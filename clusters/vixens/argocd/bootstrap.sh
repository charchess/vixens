#!/bin/bash
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

helm upgrade --install argocd argo/argo-cd --version 8.3.1 --namespace argocd --create-namespace
kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-cd/v3.1.1/manifests/crds/appproject-crd.yaml

kubectl apply -f $SCRIPT_DIR/../../../clusters/vixens/argocd/00-vixens-project.yaml
kubectl apply -f $SCRIPT_DIR/../../../clusters/vixens/argocd/01-vixens-$ENV-root.yaml

kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo
