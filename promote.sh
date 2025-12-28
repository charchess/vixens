#!/bin/bash
#
# Promote dev → main (trunk-based workflow)
#
# Usage: ./promote.sh
#
# Note: This script is being deprecated in favor of GitHub Actions workflow.
# Use instead: gh workflow run promote-prod.yaml -f version=v1.2.3
#

set -e

echo "🚀 Promotion dev → main"
echo ""

# Create PR dev → main
echo "📝 Creating PR dev → main..."
id=$(gh pr create -B main -H dev -t "promote dev to main" -b "" | sed 's/^.*pull\/\(.*\)$/\1/')

echo "✅ PR #$id created"
echo ""

# Auto-merge
echo "🔀 Auto-merging PR #$id..."
gh pr merge $id -m --auto

echo ""
echo "✅ Promotion completed!"
echo ""
echo "⚠️  NOTE: This script is deprecated."
echo "    Use GitHub Actions instead:"
echo "    gh workflow run promote-prod.yaml -f version=v1.2.3"

