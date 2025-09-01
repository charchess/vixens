#!/bin/bash
# verify-structure.sh

echo "🔍 Vérification de la structure..."
for dir in base/*/; do
  if [ ! -f "${dir}kustomization.yaml" ]; then
    echo "⚠️  ${dir}: kustomization.yaml manquant"
  fi
done

echo "✅ Vérification des Applications ArgoCD..."
kubectl apply --dry-run=client -f clusters/vixens/
