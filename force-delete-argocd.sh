#!/bin/bash
echo "🧹 Nettoyage forcé d'ArgoCD..."

# Supprimer toutes les applications
kubectl get applications -A -o name | xargs kubectl delete --grace-period=0 --force 2>/dev/null || true

# Supprimer tous les appprojects  
kubectl get appprojects -A -o name | xargs kubectl delete --grace-period=0 --force 2>/dev/null || true

# Supprimer les finalizers du namespace
kubectl get namespace argocd -o json | jq '.metadata.finalizers = []' | kubectl replace --raw "/api/v1/namespaces/argocd/finalize" -f -

# Forcer la suppression
kubectl delete namespace argocd --grace-period=0 --force --timeout=0s 2>/dev/null || true

echo "✅ Nettoyage terminé"
