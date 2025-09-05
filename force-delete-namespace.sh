#!/bin/bash
NAMESPACE="argocd"

echo "🚨 Suppression forcée du namespace $NAMESPACE..."

# 1. Supprimer toutes les ressources du namespace
echo "📦 Suppression des ressources..."
kubectl get all -n $NAMESPACE -o name | xargs kubectl delete -n $NAMESPACE --grace-period=0 --force 2>/dev/null || true
kubectl get pvc -n $NAMESPACE -o name | xargs kubectl delete -n $NAMESPACE --grace-period=0 --force 2>/dev/null || true
kubectl get configmaps -n $NAMESPACE -o name | xargs kubectl delete -n $NAMESPACE --grace-period=0 --force 2>/dev/null || true
kubectl get secrets -n $NAMESPACE -o name | xargs kubectl delete -n $NAMESPACE --grace-period=0 --force 2>/dev/null || true

# 2. Créer le patch pour retirer les finalizers
echo "🔧 Création du patch..."
cat > namespace-patch.json <<EOL
{
  "metadata": {
    "finalizers": []
  }
}
EOL

# 3. Appliquer le patch via l'API directement
echo "🎯 Patch du namespace..."
APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
TOKEN=$(kubectl create token default -n default 2>/dev/null || echo "")

if [ -n "$TOKEN" ]; then
  # Obtenir le namespace
  kubectl get namespace $NAMESPACE -o json > tmp-namespace.json
  
  # Supprimer les finalizers
  cat tmp-namespace.json | jq '.metadata.finalizers = []' > final-namespace.json
  
  # Mettre à jour via l'API
  curl -k \
    -X PUT \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d @final-namespace.json \
    $APISERVER/api/v1/namespaces/$NAMESPACE/finalize
  
  rm -f tmp-namespace.json final-namespace.json
fi

# 4. Forcer la suppression
echo "🗑️ Suppression finale..."
kubectl delete namespace $NAMESPACE --grace-period=0 --force --timeout=0s 2>/dev/null || true

echo "✅ Opération terminée"
