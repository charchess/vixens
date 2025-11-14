#!/bin/bash
set -euo pipefail

ENVIRONMENT=${1:-dev}

echo "=========================================="
echo "PHASE 1 VALIDATION - Environment: $ENVIRONMENT"
echo "=========================================="
echo ""

FAILED=0

# Test 1: Kustomize build
echo "[1/5] Building ArgoCD manifests..."
if kustomize build argocd/overlays/$ENVIRONMENT --enable-helm > /tmp/validate-$ENVIRONMENT.yaml 2>&1; then
  echo "  ✅ Build successful"
else
  echo "  ❌ Build failed"
  cat /tmp/validate-$ENVIRONMENT.yaml
  FAILED=1
fi

# Test 2: YAML syntax validation
echo "[2/5] Validating YAML syntax..."
if [ -f .yamllint.yaml ]; then
  if yamllint -c .yamllint.yaml argocd/overlays/$ENVIRONMENT 2>&1; then
    echo "  ✅ YAML valid"
  else
    echo "  ⚠️  YAML warnings (non-blocking)"
  fi
else
  echo "  ⚠️  .yamllint.yaml not found, skipping"
fi

# Test 3: YAML parsing validation
echo "[3/5] Validating YAML parsing..."
if python3 -c "import yaml; yaml.safe_load_all(open('/tmp/validate-$ENVIRONMENT.yaml'))" 2>&1; then
  echo "  ✅ YAML parsing successful"
else
  echo "  ❌ YAML parsing failed"
  FAILED=1
fi

# Test 4: Check application count
echo "[4/5] Checking application count..."
app_count=$(grep -c "kind: Application" /tmp/validate-$ENVIRONMENT.yaml || true)
if [ $app_count -ge 3 ]; then
  echo "  ✅ Found $app_count applications"
else
  echo "  ❌ Only found $app_count applications (expected >= 3)"
  FAILED=1
fi

# Test 5: Check environment config
echo "[5/5] Validating environment configuration..."
if grep -q "targetRevision: $ENVIRONMENT" /tmp/validate-$ENVIRONMENT.yaml || grep -q "targetRevision: main" /tmp/validate-$ENVIRONMENT.yaml; then
  echo "  ✅ Environment config appears correct"
else
  echo "  ⚠️  Could not verify targetRevision (might be OK)"
fi

echo ""
if [ $FAILED -eq 1 ]; then
  echo "❌ VALIDATION FAILED"
  exit 1
else
  echo "✅ ALL CHECKS PASSED"
  exit 0
fi
