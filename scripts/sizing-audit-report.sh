#!/bin/bash
# Legacy sizing audit report helper.
#
# This script remains useful only for read-only inspection of the historical
# sizing-audit PolicyReports. It must not mutate ClusterPolicy state.
# Current sizing policy/standards live in Git; see:
#   docs/reference/RESOURCE_STANDARDS.md
#   docs/adr/029-align-maturity-with-current-platform.md

set -e

KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
NAMESPACE=${1:-kyverno}

echo "=== Sizing Audit Report (legacy/read-only) ==="
echo "Date: $(date)"
echo ""

echo "Checking Kyverno policy reports..."
REPORTS=$(kubectl get policyreports -n "$NAMESPACE" 2>/dev/null | grep sizing-audit || true)

if [ -z "$REPORTS" ]; then
    echo "No sizing-audit policy reports found."
    echo "This helper targets the historical sizing-audit policy and may no longer"
    echo "represent the current sizing contract."
    exit 0
fi

echo "Pods reported by historical sizing-audit:"
echo "------------------------------------------"
kubectl get policyreports -n "$NAMESPACE" -o json | \
    jq -r '.items[]
        | select(.metadata.name | contains("sizing-audit"))
        | .results[]
        | select(.policy == "sizing-audit")
        | select(.result == "fail")
        | "  Namespace: \(.resources[0].namespace) | Resource: \(.resources[0].name) | Message: \(.message)"' 2>/dev/null || \
    echo "  (No non-compliant pods found or report unavailable)"

echo ""
echo "=== Next Steps ==="
echo "1. Read docs/reference/RESOURCE_STANDARDS.md and ADR-029."
echo "2. Inspect current apps/00-infra/kyverno/base/policies/ manifests."
echo "3. Encode any persistent sizing/policy change in Git on a branch."
echo "4. Validate through PR CI and ArgoCD; do not kubectl patch ClusterPolicy state."
