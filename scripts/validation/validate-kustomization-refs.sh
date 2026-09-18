#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

errors=0

while IFS= read -r -d '' file; do
  dir="$(dirname "$file")"

  while IFS= read -r ref; do
    [ -z "$ref" ] && continue
    case "$ref" in
      http://*|https://*|git@*) continue ;;
    esac
    if [ ! -e "$dir/$ref" ]; then
      echo "ERROR: missing resource '$ref' referenced by $file"
      errors=$((errors + 1))
    fi
  done < <(yq eval '.resources[]? // empty' "$file" 2>/dev/null || true)

  while IFS= read -r ref; do
    [ -z "$ref" ] && continue
    if [ ! -e "$dir/$ref" ]; then
      echo "ERROR: missing patch '$ref' referenced by $file"
      errors=$((errors + 1))
    fi
  done < <(yq eval '(.patches[]?.path // empty), (.patchesStrategicMerge[]? // empty), (.patchesJson6902[]?.path // empty)' "$file" 2>/dev/null || true)
done < <(find apps argocd -name kustomization.yaml -type f -print0)

if [ "$errors" -ne 0 ]; then
  echo "Kustomization reference validation failed: $errors error(s)"
  exit 1
fi

echo "Kustomization references OK"
