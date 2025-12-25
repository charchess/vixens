#!/bin/bash
# sync-waves-batch-update.sh
#
# Ajoute les sync waves aux applications ArgoCD
# Usage: ./scripts/sync-waves-batch-update.sh [--dry-run]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DRY_RUN=false

# Parse arguments
if [[ "$1" == "--dry-run" ]]; then
  DRY_RUN=true
  echo "🔍 DRY-RUN MODE: Aucune modification ne sera appliquée"
  echo ""
fi

cd "$REPO_ROOT"

# Check dependencies
if ! command -v yq &> /dev/null; then
  echo "❌ Erreur: yq n'est pas installé"
  echo "Installation: snap install yq ou brew install yq"
  exit 1
fi

echo "📋 Ajout des sync waves aux applications ArgoCD"
echo "================================================"
echo ""

# Function to add or update sync-wave annotation
add_sync_wave() {
  local file="$1"
  local wave="$2"
  local app_name=$(basename "$file" .yaml)

  if [ ! -f "$file" ]; then
    echo "⚠️  $app_name: Fichier n'existe pas, skip"
    return
  fi

  # Get current wave if exists
  current_wave=$(yq eval '.metadata.annotations."argocd.argoproj.io/sync-wave" // "none"' "$file")

  if [ "$current_wave" = "$wave" ]; then
    echo "✓  $app_name: Wave $wave déjà configurée"
    return
  fi

  if [ "$DRY_RUN" = true ]; then
    echo "🔍 $app_name: Wave $current_wave → $wave (DRY-RUN)"
    return
  fi

  # Add annotations section if doesn't exist
  if ! yq eval '.metadata.annotations' "$file" > /dev/null 2>&1; then
    yq eval -i '.metadata.annotations = {}' "$file"
  fi

  # Set sync-wave
  yq eval -i ".metadata.annotations.\"argocd.argoproj.io/sync-wave\" = \"$wave\"" "$file"

  echo "✅ $app_name: Wave $current_wave → $wave"
}

# Phase 1: Wave -1 (Services Partagés Storage/Cache)
echo "=== Phase 1: Wave -1 (Services Partagés) ==="
add_sync_wave "argocd/overlays/dev/apps/nfs-storage.yaml" "-1"
add_sync_wave "argocd/overlays/dev/apps/redis-shared.yaml" "-1"
echo ""

# Phase 2: Wave 0 (Infrastructure Réseau)
echo "=== Phase 2: Wave 0 (Infrastructure Réseau) ==="
add_sync_wave "argocd/overlays/dev/apps/synology-csi.yaml" "0"
add_sync_wave "argocd/overlays/dev/apps/traefik.yaml" "0"
echo ""

# Phase 2b: Wave 1 (Dépend de l'infrastructure)
echo "=== Phase 2b: Wave 1 (Post-Infrastructure) ==="
add_sync_wave "argocd/overlays/dev/apps/traefik-dashboard.yaml" "1"
echo ""

# Phase 3: Wave 5 (Apps avec PostgreSQL - après wave 4 postgresql-shared)
echo "=== Phase 3: Wave 5 (Apps avec PostgreSQL) ==="
add_sync_wave "argocd/overlays/dev/apps/linkwarden.yaml" "5"
add_sync_wave "argocd/overlays/dev/apps/netbox.yaml" "5"
add_sync_wave "argocd/overlays/dev/apps/docspell.yaml" "5"
echo ""

# Phase 4: Wave 10 (Applications sans dépendances critiques)
echo "=== Phase 4: Wave 10 (Applications Métier - défaut) ==="
# Liste des apps qui peuvent rester sans annotation (wave implicite 0)
# ou qu'on met explicitement à 10 pour clarté
apps_wave_10=(
  "argocd-image-updater"
  "authentik"
  "booklore"
  "changedetection"
  "frigate"
  "gluetun"
  "headlamp"
  "homeassistant"
  "homepage"
  "hydrus-client"
  "jellyfin"
  "jellyseerr"
  "lazylibrarian"
  "lidarr"
  "mail-gateway"
  "mosquitto"
  "music-assistant"
  "mylar"
  "prowlarr"
  "radarr"
  "sabnzbd"
  "sonarr"
  "vaultwarden"
  "whisparr"
  "whoami"
  "adguard-home"
  "netvisor"
  "birdnet-go"
  "gitops-revision-controller"
  "reloader"
  "metrics-server"
  "hubble-ui"
  "loki"
  "promtail"
)

for app in "${apps_wave_10[@]}"; do
  add_sync_wave "argocd/overlays/dev/apps/${app}.yaml" "10"
done
echo ""

echo "================================================"
if [ "$DRY_RUN" = true ]; then
  echo "🔍 DRY-RUN terminé. Aucune modification appliquée."
  echo ""
  echo "Pour appliquer les changements, exécutez:"
  echo "  $0"
else
  echo "✅ Sync waves ajoutées avec succès!"
  echo ""
  echo "📊 Résumé des modifications:"
  git status --short argocd/overlays/dev/apps/ | wc -l | xargs echo "   Fichiers modifiés:"
  echo ""
  echo "📋 Vérification:"
  echo "   Wave -1: $(grep -r 'sync-wave.*"-1"' argocd/overlays/dev/apps/ 2>/dev/null | wc -l) apps"
  echo "   Wave  0: $(grep -r 'sync-wave.*"0"' argocd/overlays/dev/apps/ 2>/dev/null | wc -l) apps"
  echo "   Wave  1: $(grep -r 'sync-wave.*"1"' argocd/overlays/dev/apps/ 2>/dev/null | wc -l) apps"
  echo "   Wave  5: $(grep -r 'sync-wave.*"5"' argocd/overlays/dev/apps/ 2>/dev/null | wc -l) apps"
  echo "   Wave 10: $(grep -r 'sync-wave.*"10"' argocd/overlays/dev/apps/ 2>/dev/null | wc -l) apps"
  echo ""
  echo "Prochaines étapes:"
  echo "  1. Vérifier les modifications: git diff argocd/overlays/dev/apps/"
  echo "  2. Tester syntax: ./scripts/validate-sync-waves.sh"
  echo "  3. Commit: git add argocd/overlays/dev/apps/ && git commit -m 'feat(argocd): add sync waves'"
fi
