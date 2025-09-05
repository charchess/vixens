#!/bin/bash
set -euo pipefail

echo "🚀 Bootstrap ArgoCD - App-of-Apps"

# 1. Vérifier si ArgoCD est déjà installé
if kubectl get namespace argocd >/dev/null 2>&1; then
    echo "✅ Namespace argocd existe déjà"
    
    # Vérifier si les CRDs sont présents
    if kubectl get crd applications.argoproj.io >/dev/null 2>&1; then
        echo "✅ CRDs ArgoCD déjà installés"
    else
        echo "📦 Installation des CRDs..."
        kubectl apply -f clusters/bootstrap/argocd-install.yaml
    fi
else
    echo "📦 Installation complète d'ArgoCD..."
    kubectl apply -f clusters/bootstrap/argocd-install.yaml
fi

# 2. Attendre que les CRDs soient prêts
echo "⏳ Attente des CRDs..."
timeout 60 bash -c 'until kubectl get crd applications.argoproj.io >/dev/null 2>&1; do sleep 2; done' || {
    echo "❌ Timeout attente CRDs"
    exit 1
}

# 3. Installer ArgoCD via Helm (si pas déjà fait)
if ! kubectl get deployment argocd-server -n argocd >/dev/null 2>&1; then
    echo "🎯 Installation ArgoCD..."
    helm repo add argo https://argoproj.github.io/argo-helm
    helm repo update
    helm install argocd argo/argo-cd -n argocd --create-namespace \
        --set server.service.type=LoadBalancer \
        --set server.service.loadBalancerIP=192.168.200.71
fi

# 4. Attendre qu'ArgoCD soit prêt
echo "⏳ Attente d'ArgoCD..."
kubectl wait --for=condition=ready pod --all -n argocd --timeout=300s || {
    echo "⚠️  Pods pas encore prêts, mais on continue..."
}

# 5. Appliquer l'App-of-Apps root
echo "🎯 Application du Root App..."
kubectl apply -f clusters/bootstrap/root-app.yaml

# 6. Info de connexion
echo ""
echo "✅ Bootstrap terminé !"
echo "📍 ArgoCD URL: http://192.168.200.71:8080"
echo "🔑 Admin password:"
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo ""
