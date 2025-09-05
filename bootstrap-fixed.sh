#!/bin/bash
set -euo pipefail

echo "🚀 Bootstrap ArgoCD - App-of-Apps (FIXED)"

# 1. Vérifier si ArgoCD est déjà installé
if kubectl get namespace argocd >/dev/null 2>&1 && kubectl get deployment argocd-server -n argocd >/dev/null 2>&1; then
    echo "✅ ArgoCD déjà installé"
else
    echo "📦 Installation d'ArgoCD via Helm..."
    
    # Créer le namespace
    kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
    
    # Installation Helm complète (CRDs inclus)
    helm repo add argo https://argoproj.github.io/argo-helm
    helm repo update
    helm install argocd argo/argo-cd -n argocd --create-namespace \
        --set server.service.type=LoadBalancer \
        --set server.service.loadBalancerIP=192.168.200.71 \
        --set crds.install=true
fi

# 2. Attendre qu'ArgoCD soit prêt
echo "⏳ Attente d'ArgoCD..."
kubectl wait --for=condition=ready pod --all -n argocd --timeout=600s || {
    echo "⚠️  Attente prolongée, mais on continue..."
}

# 3. Appliquer l'App-of-Apps root
echo "🎯 Application du Root App..."
kubectl apply -f clusters/bootstrap/root-app.yaml

# 4. Info de connexion
echo ""
echo "✅ Bootstrap terminé !"
echo "📍 ArgoCD URL: http://192.168.200.71:8080"
echo "🔑 Admin password:"
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo ""
