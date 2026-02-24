#!/bin/bash
# Sizing Audit Report Script
# Generates a report of pods missing the vixens.io/sizing label
# Run this after the sizing-audit policy has been active for a few minutes

set -e

KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
NAMESPACE=${1:-kyverno}

echo "=== Sizing Audit Report ==="
echo "Date: $(date)"
echo ""

# Check if Kyverno policy reports exist
echo "Checking Kyverno policy reports..."
REPORTS=$(kubectl get policyreports -n $NAMESPACE 2>/dev/null | grep sizing-audit || true)

if [ -z "$REPORTS" ]; then
    echo "No sizing-audit policy reports found yet."
    echo "The policy needs time to audit all pods (usually 5-10 minutes)."
    echo ""
    echo "To check later, run:"
    echo "  kubectl get policyreports -n $NAMESPACE -o yaml | grep -A10 sizing-audit"
    exit 0
fi

# Get all pods missing the label
echo "Pods missing 'vixens.io/sizing' label:"
echo "----------------------------------------"
kubectl get policyreports -n $NAMESPACE -o json | \
    jq -r '.items[] | 
        select(.metadata.name | contains("sizing-audit")) |
        .results[] |
        select(.policy == "sizing-audit") |
        select(.result == "fail") |
        "  Namespace: \(.resources[0].namespace) | Resource: \(.resources[0].name) | Message: \(.message)"' 2>/dev/null || \
    echo "  (No non-compliant pods found or report not ready yet)"

echo ""
echo "=== Next Steps ==="
echo "1. Add vixens.io/sizing labels to critical apps before enabling mutation"
echo "2. Example for an overlay kustomization.yaml:"
echo ""
echo "   metadata:"
echo "     labels:"
echo "       vixens.io/sizing: small"
echo ""
echo "3. Available sizing values: micro, small, medium, large, xlarge"
echo ""
echo "4. When ready to enable automatic sizing:"
echo "   kubectl patch clusterpolicy sizing-mutate -p '{\"spec\":{\"validationFailureAction\":\"Enforce\"}}'"
