#!/usr/bin/env bash
# ============================================================================
# WAIT FOR KUBERNETES API - Robust cluster readiness check
# ============================================================================
# This script validates Kubernetes control plane readiness after Talos bootstrap.
# It performs two-phase validation to ensure the cluster is truly operational.
#
# Usage: ./wait-for-k8s-api.sh <kubeconfig_path>
#
# Phase 1: API server response (10 min timeout)
# Phase 2: Control plane components ready (20 min timeout)
#
# Exit codes:
#   0 - Cluster ready
#   1 - Timeout or error
# ============================================================================

set -euo pipefail

# ----------------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------------
KUBECONFIG_PATH="${1:-}"

if [ -z "$KUBECONFIG_PATH" ]; then
  echo "❌ Error: kubeconfig path required"
  echo "Usage: $0 <kubeconfig_path>"
  exit 1
fi

if [ ! -f "$KUBECONFIG_PATH" ]; then
  echo "❌ Error: kubeconfig not found at: $KUBECONFIG_PATH"
  exit 1
fi

# Timeouts and delays
INITIAL_DELAY=90          # Wait for Talos bootstrap
PHASE1_MAX_ATTEMPTS=60    # 60 × 10s = 10 minutes
PHASE2_MAX_ATTEMPTS=120   # 120 × 10s = 20 minutes
REQUIRED_CONSECUTIVE=3    # Number of consecutive successful checks

# ----------------------------------------------------------------------------
# Phase 0: Initial delay
# ----------------------------------------------------------------------------
echo "⏳ Waiting for Kubernetes API to be ready..."
echo "⏸️  Initial delay: waiting ${INITIAL_DELAY} seconds for Talos bootstrap..."
echo "📁 Kubeconfig path: $KUBECONFIG_PATH"
sleep $INITIAL_DELAY

# ----------------------------------------------------------------------------
# Phase 1: Wait for API server to respond
# ----------------------------------------------------------------------------
echo "📡 Phase 1: Waiting for API server to respond..."
i=1
while [ $i -le $PHASE1_MAX_ATTEMPTS ]; do
  if kubectl --kubeconfig="$KUBECONFIG_PATH" get --raw /healthz &>/dev/null; then
    echo "✅ API server responded on attempt $i"
    break
  fi
  echo "⏳ Attempt $i/$PHASE1_MAX_ATTEMPTS - API not ready yet (waiting 10s)..."
  sleep 10
  i=$((i + 1))
done

if [ $i -gt $PHASE1_MAX_ATTEMPTS ]; then
  echo "❌ Timeout: API never responded after $((PHASE1_MAX_ATTEMPTS * 10 / 60)) minutes"
  exit 1
fi

# ----------------------------------------------------------------------------
# Phase 2: Wait for control plane components to be ready
# ----------------------------------------------------------------------------
echo "🔍 Phase 2: Waiting for control plane components to be ready..."
READY=0
ATTEMPT=1

while [ $ATTEMPT -le $PHASE2_MAX_ATTEMPTS ]; do
  # Check control plane pods in kube-system
  # Note: On Talos, etcd runs as a system service (not a Kubernetes pod)
  APISERVER_COUNT=$(kubectl --kubeconfig="$KUBECONFIG_PATH" get pods -n kube-system -l component=kube-apiserver --no-headers 2>/dev/null | grep -c Running | tr -d '\n' || echo "0")
  CONTROLLER_COUNT=$(kubectl --kubeconfig="$KUBECONFIG_PATH" get pods -n kube-system -l component=kube-controller-manager --no-headers 2>/dev/null | grep -c Running | tr -d '\n' || echo "0")
  SCHEDULER_COUNT=$(kubectl --kubeconfig="$KUBECONFIG_PATH" get pods -n kube-system -l component=kube-scheduler --no-headers 2>/dev/null | grep -c Running | tr -d '\n' || echo "0")

  echo "📊 Control plane status: kube-apiserver=$APISERVER_COUNT kube-controller=$CONTROLLER_COUNT kube-scheduler=$SCHEDULER_COUNT"

  # We need at least 1 of each component
  if [ "$APISERVER_COUNT" -ge 1 ] && [ "$CONTROLLER_COUNT" -ge 1 ] && [ "$SCHEDULER_COUNT" -ge 1 ]; then
    READY=$((READY + 1))
    echo "✅ Control plane ready ($READY/$REQUIRED_CONSECUTIVE consecutive checks)"

    if [ $READY -ge $REQUIRED_CONSECUTIVE ]; then
      echo "🎉 Kubernetes control plane is STABLE and ready!"
      echo "📋 Final status:"
      kubectl --kubeconfig="$KUBECONFIG_PATH" get nodes
      kubectl --kubeconfig="$KUBECONFIG_PATH" get pods -n kube-system | grep "kube-"
      exit 0
    fi
    sleep 5
  else
    if [ $READY -gt 0 ]; then
      echo "⚠️  Control plane became unstable (was at $READY/$REQUIRED_CONSECUTIVE)"
    fi
    READY=0
    echo "⏳ Waiting for control plane... (10s)"
    sleep 10
  fi

  ATTEMPT=$((ATTEMPT + 1))
done

echo "❌ Control plane not ready after $((PHASE2_MAX_ATTEMPTS * 10 / 60)) minutes"
exit 1
