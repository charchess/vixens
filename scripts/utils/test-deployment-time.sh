#!/bin/bash
# test-deployment-time.sh
#
# Mesure le temps de déploiement complet du cluster
# Usage: ./scripts/test-deployment-time.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
KUBECONFIG="${KUBECONFIG:-/root/vixens/terraform/environments/dev/kubeconfig-dev}"
export KUBECONFIG

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "⏱️  Test de Temps de Déploiement"
echo "================================"
echo ""

# Wait for ArgoCD to be available
echo "⏳ Attente qu'ArgoCD soit disponible..."
while ! kubectl get namespace argocd > /dev/null 2>&1; do
  echo "   ArgoCD namespace pas encore créé..."
  sleep 5
done

while ! kubectl get application -n argocd vixens-app-of-apps > /dev/null 2>&1; do
  echo "   App-of-Apps pas encore déployée..."
  sleep 5
done

echo "✅ ArgoCD disponible"
echo ""

# Start timing
START_TIME=$(date +%s)
echo "🚀 Début du déploiement: $(date)"
echo ""

# Monitor deployment
total_apps=0
wave_current=""
wave_start_time=$START_TIME

while true; do
  # Get current stats
  stats=$(kubectl get applications -n argocd -o json 2>/dev/null | jq -r '
    .items |
    {
      total: length,
      healthy: [.[] | select(.status.health.status == "Healthy")] | length,
      synced: [.[] | select(.status.sync.status == "Synced")] | length,
      progressing: [.[] | select(.status.health.status == "Progressing")] | length,
      degraded: [.[] | select(.status.health.status == "Degraded")] | length,
      outOfSync: [.[] | select(.status.sync.status == "OutOfSync")] | length
    } |
    "TOTAL=\(.total) HEALTHY=\(.healthy) SYNCED=\(.synced) PROGRESSING=\(.progressing) DEGRADED=\(.degraded) OUTOFSYNC=\(.outOfSync)"
  ')

  if [ -z "$stats" ]; then
    echo "⏳ Attente des applications..."
    sleep 10
    continue
  fi

  eval "$stats"
  total_apps=$TOTAL

  # Get current wave being deployed
  wave_deploying=$(kubectl get applications -n argocd -o json 2>/dev/null | jq -r '
    [.items[] | select(.status.sync.status == "Synced" and .status.health.status != "Healthy") | .metadata.annotations."argocd.argoproj.io/sync-wave" // "0"] |
    unique |
    sort |
    .[0] // "none"
  ')

  # Detect wave change
  if [ "$wave_deploying" != "$wave_current" ] && [ "$wave_deploying" != "none" ]; then
    if [ -n "$wave_current" ]; then
      wave_duration=$(($(date +%s) - wave_start_time))
      echo ""
      echo -e "${GREEN}✅ Wave $wave_current terminée en ${wave_duration}s${NC}"
      echo ""
    fi
    wave_current=$wave_deploying
    wave_start_time=$(date +%s)
    echo -e "${YELLOW}🔄 Déploiement Wave $wave_current...${NC}"
  fi

  # Calculate elapsed time
  current_time=$(date +%s)
  elapsed=$((current_time - START_TIME))
  elapsed_min=$((elapsed / 60))
  elapsed_sec=$((elapsed % 60))

  # Display progress
  printf "\r⏱️  %02d:%02d | Apps: %2d/%2d Healthy | %2d Synced | %2d Progressing | %2d Degraded | %2d OutOfSync" \
    $elapsed_min $elapsed_sec $HEALTHY $TOTAL $SYNCED $PROGRESSING $DEGRADED $OUTOFSYNC

  # Check if all apps are healthy and synced
  if [ "$HEALTHY" -eq "$TOTAL" ] && [ "$SYNCED" -eq "$TOTAL" ]; then
    echo ""
    echo ""
    break
  fi

  # Safety timeout (30 minutes)
  if [ $elapsed -gt 1800 ]; then
    echo ""
    echo ""
    echo -e "${RED}⏰ Timeout après 30 minutes${NC}"
    echo ""
    echo "Applications non Healthy:"
    kubectl get applications -n argocd -o json | jq -r '
      .items[] |
      select(.status.health.status != "Healthy") |
      "\(.metadata.name): \(.status.sync.status) / \(.status.health.status)"
    '
    exit 1
  fi

  sleep 5
done

# Final wave timing
if [ -n "$wave_current" ]; then
  wave_duration=$(($(date +%s) - wave_start_time))
  echo -e "${GREEN}✅ Wave $wave_current terminée en ${wave_duration}s${NC}"
  echo ""
fi

# Calculate total time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
DURATION_MIN=$((DURATION / 60))
DURATION_SEC=$((DURATION % 60))

# Display results
echo "================================"
echo -e "${GREEN}✅ Déploiement terminé avec succès!${NC}"
echo ""
echo "📊 Statistiques:"
echo "   Temps total: ${DURATION_MIN}m ${DURATION_SEC}s"
echo "   Applications: $total_apps"
echo "   Toutes Healthy: $HEALTHY"
echo "   Toutes Synced: $SYNCED"
echo ""

# Check for any pod restarts
echo "🔄 Vérification des redémarrages:"
restarts=$(kubectl get pods -A -o json | jq -r '
  [.items[] |
   select(.status.containerStatuses != null) |
   .status.containerStatuses[] |
   select(.restartCount > 0)] |
  length
')

if [ "$restarts" -eq 0 ]; then
  echo -e "   ${GREEN}✅ Aucun redémarrage de pod${NC}"
else
  echo -e "   ${YELLOW}⚠️  $restarts container(s) ont redémarré${NC}"
  kubectl get pods -A -o json | jq -r '
    .items[] |
    select(.status.containerStatuses != null) |
    select(.status.containerStatuses[] | .restartCount > 0) |
    "\(.metadata.namespace)/\(.metadata.name): \(.status.containerStatuses[0].restartCount) restarts"
  ' | head -10
fi

echo ""

# Check for CrashLoopBackOff events
echo "💥 Vérification CrashLoopBackOff:"
crashes=$(kubectl get events -A --field-selector reason=BackOff 2>/dev/null | grep -v "LAST SEEN" | wc -l || echo "0")

if [ "$crashes" -eq 0 ]; then
  echo -e "   ${GREEN}✅ Aucun CrashLoopBackOff${NC}"
else
  echo -e "   ${RED}❌ $crashes événements CrashLoopBackOff${NC}"
fi

echo ""
echo "================================"
echo "📈 Objectifs:"
echo "   Temps: 30-45 minutes (actuel: ${DURATION_MIN}m)"
echo "   CrashLoopBackOff: 0 (actuel: $crashes)"
echo "   Redémarrages: 0 (actuel: $restarts)"
echo ""

# Success criteria
if [ "$DURATION_MIN" -le 45 ] && [ "$crashes" -eq 0 ]; then
  echo -e "${GREEN}🎉 Tous les objectifs atteints!${NC}"
  exit 0
else
  echo -e "${YELLOW}⚠️  Certains objectifs non atteints${NC}"
  exit 0
fi
