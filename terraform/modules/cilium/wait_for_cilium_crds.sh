#!/bin/bash
# This script waits for Cilium CRDs to be registered in the Kubernetes API

# refactor : integrer dans un modules cilium

end_time=$(( $(date +%s) + 600 )) # 10 minutes timeout for CRDs
required_crds=(
  "ciliumnodes.cilium.io"
  "ciliumendpoints.cilium.io"
  "ciliumidentities.cilium.io"
  "ciliumnetworkpolicies.cilium.io"
  "ciliuml2announcementpolicies.cilium.io"
  "ciliumloadbalancerippools.cilium.io"
)

while true; do
  ready_count=0
  for crd in "${required_crds[@]}"; do
    if kubectl get crd "$crd" --kubeconfig "$KUBECONFIG_PATH" &> /dev/null; then
      ready_count=$((ready_count + 1))
    else
      echo "Waiting for CRD: $crd"
      break
    fi
  done

  if [ "$ready_count" -eq "${#required_crds[@]}" ]; then
    echo "All required Cilium CRDs are registered."
    break
  fi

  if [ "$(date +%s)" -gt "$end_time" ]; then
    echo "Timeout: Not all Cilium CRDs were registered after 3 minutes."
    kubectl get crds --kubeconfig "$KUBECONFIG_PATH" # Print available CRDs for debugging
    exit 1
  fi

  sleep 10
done
