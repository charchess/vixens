#!/bin/bash
# ============================================================================
# IMPORT TERRAFORM REFACTORING TASKS TO ARCHON
# ============================================================================
# Script pour créer le projet et les tâches dans Archon
#
# Usage:
#   ./scripts/import-archon-tasks.sh

set -euo pipefail

TASKS_FILE="/root/vixens/docs/terraform-refactoring-archon-tasks.json"

echo "🚀 Import des tâches Terraform Refactoring dans Archon"
echo ""

# Vérifier que le fichier existe
if [ ! -f "$TASKS_FILE" ]; then
  echo "❌ Fichier de tâches non trouvé: $TASKS_FILE"
  exit 1
fi

echo "📋 Fichier de tâches trouvé: $TASKS_FILE"
echo ""

# TODO: Utiliser l'API Archon ou les outils MCP pour créer le projet et les tâches
# Pour l'instant, afficher les tâches

echo "📦 Projet à créer:"
jq -r '.project.name' "$TASKS_FILE"
echo ""

echo "📝 Tâches à créer:"
jq -r '.tasks[] | "\(.id): \(.title) (\(.estimated_duration))"' "$TASKS_FILE"
echo ""

echo "⏱️  Timeline totale: $(jq -r '.timeline.estimated_total' "$TASKS_FILE")"
echo ""

echo "💡 Utiliser Claude Code avec MCP Archon pour importer ces tâches"
echo "   ou importer manuellement depuis: $TASKS_FILE"
