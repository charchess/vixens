#!/bin/bash
echo "☢️ Suppression forcée de traefik-ingress..."

APP_NAME="traefik-ingress"
NAMESPACE="argocd"

# 1. Retirer les finalizers
echo "🔄 Retrait des finalizers..."
kubectl patch application $APP_NAME -n $NAMESPACE \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge \
  --force 2>/dev/null || true

# 2. Suppression forcée
echo "🗑️ Suppression forcée..."
kubectl delete application $APP_NAME -n $NAMESPACE \
  --grace-period=0 \
  --force \
  --timeout=0s 2>/dev/null || true

# 3. Via l'API directement
echo "🎯 API directe..."
if kubectl get application $APP_NAME -n $NAMESPACE -o name 2>/dev/null; then
  kubectl get application $APP_NAME -n $NAMESPACE -o json | \
    jq '.metadata.finalizers = []' | \
    kubectl replace --raw "/apis/argoproj.io/v1alpha1/namespaces/$NAMESPACE/applications/$APP_NAME/finalize" -f - \
    2>/dev/null || true
fi

# 4. Vérifier
if kubectl get application $APP_NAME -n $NAMESPACE 2>/dev/null; then
  echo "❌ Toujours présent..."
else
  echo "✅ Supprimé !"
fi
