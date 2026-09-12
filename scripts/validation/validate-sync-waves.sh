#!/bin/bash
# validate-sync-waves.sh
#
# Valide la syntaxe YAML et l'ordre des sync waves
# Usage: ./scripts/validate-sync-waves.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

echo "🔍 Validation des Sync Waves"
echo "=============================="
echo ""

# Check dependencies
if ! command -v yq &> /dev/null; then
  echo "❌ Erreur: yq n'est pas installé"
  exit 1
fi

if ! command -v yamllint &> /dev/null; then
  echo "⚠️  yamllint n'est pas installé, skip validation syntax"
  SKIP_YAMLLINT=true
fi

errors=0

# 1. Validate YAML syntax
if [ "$SKIP_YAMLLINT" != "true" ]; then
  echo "📝 Validation syntax YAML..."
  if yamllint argocd/overlays/dev/apps/*.yaml 2>&1 | grep -v "warning"; then
    echo "✅ Syntax YAML valide"
  else
    echo "❌ Erreurs de syntax YAML détectées"
    errors=$((errors + 1))
  fi
  echo ""
fi

# 2. Check all files are valid YAML with yq
echo "📝 Validation structure YAML avec yq..."
invalid_files=0
for file in argocd/overlays/dev/apps/*.yaml; do
  if ! yq eval '.' "$file" > /dev/null 2>&1; then
    echo "❌ Fichier invalide: $file"
    invalid_files=$((invalid_files + 1))
  fi
done

if [ $invalid_files -eq 0 ]; then
  echo "✅ Tous les fichiers sont du YAML valide"
else
  echo "❌ $invalid_files fichier(s) invalide(s)"
  errors=$((errors + 1))
fi
echo ""

# 3. List apps by sync wave
echo "📊 Distribution des sync waves:"
echo ""
printf "%-10s %-40s %s\n" "WAVE" "APPLICATION" "STATUS"
printf "%-10s %-40s %s\n" "----" "-----------" "------"

for file in argocd/overlays/dev/apps/*.yaml; do
  app_name=$(basename "$file" .yaml)
  wave=$(yq eval '.metadata.annotations."argocd.argoproj.io/sync-wave" // "none"' "$file")

  # Check for critical apps without waves
  case "$app_name" in
    cilium-lb|synology-csi-secrets)
      if [ "$wave" = "none" ]; then
        printf "%-10s %-40s %s\n" "$wave" "$app_name" "⚠️  MISSING"
        errors=$((errors + 1))
      else
        printf "%-10s %-40s %s\n" "$wave" "$app_name" "✅"
      fi
      ;;
    *)
      printf "%-10s %-40s %s\n" "$wave" "$app_name" "✓"
      ;;
  esac
done | sort -n

echo ""

# 4. Summary by wave
echo "📈 Résumé par wave:"
for wave in -5 -4 -3 -2 -1 0 1 2 3 4 5 6 7 8 9 10 none; do
  count=$(grep -r "sync-wave.*\"$wave\"" argocd/overlays/dev/apps/ 2>/dev/null | wc -l)
  if [ "$wave" = "none" ]; then
    count=$(grep -rL "sync-wave" argocd/overlays/dev/apps/ 2>/dev/null | wc -l)
  fi

  if [ $count -gt 0 ]; then
    printf "   Wave %5s: %2d apps\n" "$wave" "$count"
  fi
done
echo ""

# 5. Check dependencies order
echo "🔗 Vérification ordre des dépendances:"

# PostgreSQL apps should be >= wave 5 (postgresql-shared is wave 4)
pgsql_apps=("linkwarden" "netbox" "docspell")
for app in "${pgsql_apps[@]}"; do
  if [ -f "argocd/overlays/dev/apps/${app}.yaml" ]; then
    wave=$(yq eval '.metadata.annotations."argocd.argoproj.io/sync-wave" // "0"' "argocd/overlays/dev/apps/${app}.yaml")
    if [ "$wave" -lt 5 ] 2>/dev/null; then
      echo "❌ $app (wave $wave) devrait être >= 5 (après postgresql-shared wave 4)"
      errors=$((errors + 1))
    else
      echo "✅ $app (wave $wave) correctement après PostgreSQL"
    fi
  fi
done

echo ""

# 6. Final result
echo "=============================="
if [ $errors -eq 0 ]; then
  echo "✅ Validation réussie! Aucune erreur détectée."
  echo ""
  echo "Prochaines étapes:"
  echo "  1. Commit: git add -A && git commit -m 'feat(argocd): add sync waves'"
  echo "  2. Push: git push origin main (via PR)"
  echo "  3. Observer ArgoCD auto-sync"
  exit 0
else
  echo "❌ Validation échouée: $errors erreur(s) détectée(s)"
  echo ""
  echo "Corrigez les erreurs avant de commiter."
  exit 1
fi
