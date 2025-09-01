#!/bin/bash
# validate-helm.sh

CHARTS=(
  "argo-cd:7.3.11"
  "traefik:29.0.1"
  "longhorn:1.7.1"
)

for chart in "${CHARTS[@]}"; do
  IFS=':' read -r name version <<< "$chart"
  echo "🔍 Validation $name:$version..."
  helm template test "$name" --repo "https://$(echo $name | cut -d- -f1).github.io/helm-charts" --version "$version" > /dev/null && echo "✅" || echo "❌"
done
