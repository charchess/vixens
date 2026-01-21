#!/bin/bash
set -e

# Simple script to deploy secrets after cluster creation
# Usage: ./scripts/bootstrap-secrets.sh <environment>

ENVIRONMENT=${1:-dev}
KUBECONFIG="$HOME/vixens/.secrets/${ENVIRONMENT}/kubeconfig-${ENVIRONMENT}"

if [[ ! -f "$KUBECONFIG" ]]; then
  echo "❌ Kubeconfig not found: ${KUBECONFIG}"
  echo "💡 Run 'terraform apply' first"
  exit 1
fi

echo "🔐 Deploying secrets for ${ENVIRONMENT}..."
for file in "$HOME/vixens/.secrets/${ENVIRONMENT}/*.yaml"
do
	echo "deploying $file"
	kubectl --kubeconfig="$KUBECONFIG" apply -f "$file"
done
echo "✅ Done"
