#!/bin/bash
# validate-yaml.sh

echo "🔍 Validation syntaxique YAML..."
shopt -s globstar nullglob

for file in **/*.yaml **/*.yml; do
  if [[ -f "$file" ]]; then
    if yq eval '.' "$file" > /dev/null 2>&1; then
      echo "✅ $file"
    else
      echo "❌ $file - Erreur de syntaxe"
    fi
  fi
done
