# Bootstrap Cilium Load Balancer resources before ArgoCD needs them
# This breaks the chicken-and-egg dependency cycle during initial apply.

# Readiness Probe for Cilium CRDs
# This resource waits for the Cilium CRDs to be registered in the Kubernetes API
# before allowing dependent resources (like CiliumLoadBalancerIPPool) to be created.
resource "null_resource" "wait_for_cilium_crds" {
  depends_on = [
    helm_release.cilium,
    null_resource.wait_for_k8s_api
  ]

  provisioner "local-exec" {
    command = <<-EOT
      end_time=$(( $(date +%s) + 180 )) # 3 minutes timeout for CRDs
      while true; do
        # Check for CiliumLoadBalancerIPPool CRD
        kubectl --kubeconfig "$KUBECONFIG" get crd ciliumloadbalancerippools.cilium.io >/dev/null 2>&1
        CRD1_STATUS=$?

        # Check for CiliumL2AnnouncementPolicy CRD
        kubectl --kubeconfig "$KUBECONFIG" get crd ciliuml2announcementpolicies.cilium.io >/dev/null 2>&1
        CRD2_STATUS=$?

        if [ "$CRD1_STATUS" -eq 0 ] && [ "$CRD2_STATUS" -eq 0 ]; then
          echo "Cilium CRDs are ready."
          break
        fi

        if [ "$(date +%s)" -gt "$end_time" ]; then
          echo "Timeout: Cilium CRDs did not become ready after 3 minutes."
          exit 1
        fi

        echo "Waiting for Cilium CRDs to be ready..."
        sleep 10
      done
    EOT

    environment = {
      KUBECONFIG = "${path.module}/kubeconfig-dev"
    }
  }
}

# Apply the IPAddressPool for the 'dev' environment
resource "kubectl_manifest" "cilium_ip_pool" {
  yaml_body = file("${path.module}/../../../apps/cilium-lb/overlays/dev/ippool.yaml")

  depends_on = [
    null_resource.wait_for_cilium_crds
  ]
}

# Apply the L2 announcement policy
resource "kubectl_manifest" "cilium_l2_policy" {
  yaml_body = file("${path.module}/../../../apps/cilium-lb/base/l2policy.yaml")

  depends_on = [
    null_resource.wait_for_cilium_crds
  ]
}