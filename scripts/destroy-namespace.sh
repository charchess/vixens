#!/usr/bin/env bash
# destroy-namespace.sh  <namespace>

set -euo pipefail

NS=${1:-argocd}

echo "=== DESTROY EVERYTHING in namespace $NS ==="

# 1. Retirer tous les finalizers sur tous les objets
echo "[1/4] Removing finalizers on all resources …"
kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n 1 -I {} bash -c "
      kubectl get {} -n $NS -o name 2>/dev/null \
      | xargs -n 1 -I res kubectl patch res -n $NS -p '{\"metadata\":{\"finalizers\":null}}' --type=merge 2>/dev/null || true"

# 2. Supprimer violemment les objets restants
echo "[2/4] Force-deleting all remaining objects …"
kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n 1 -I {} bash -c "
      kubectl delete {} --all -n $NS --grace-period=0 --force --ignore-not-found=true 2>/dev/null || true"

# 3. Supprimer le namespace
echo "[3/4] Deleting the namespace …"
kubectl delete namespace "$NS" --ignore-not-found=true --grace-period=0 --force 2>/dev/null || true

# 4. Si le namespace reste en Terminating, on le "déblocke" via l'API
echo "[4/4] Final cleanup if namespace is stuck in Terminating …"
TEMPFILE=$(mktemp)
kubectl get namespace "$NS" -o json > "$TEMPFILE"
jq '.spec.finalizers = []' "$TEMPFILE" > "${TEMPFILE}.patched"
kubectl proxy &
PROXY_PID=$!
sleep 1
curl -X PUT \
  --data-binary @"${TEMPFILE}.patched" \
  http://localhost:8001/api/v1/namespaces/$NS/finalize
kill $PROXY_PID
rm -f "$TEMPFILE" "${TEMPFILE}.patched"

echo "=== Namespace $NS (argocd) should now be GONE ==="
