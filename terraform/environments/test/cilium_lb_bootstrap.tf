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
    command = "bash ${path.module}/wait_for_cilium_crds.sh"

    environment = {
      KUBECONFIG_PATH = "${path.module}/kubeconfig-test"
    }
  }
}

# Apply the IPAddressPool for the 'test' environment
resource "kubectl_manifest" "cilium_ip_pool" {
  yaml_body = file("${path.module}/../../../apps/cilium-lb/overlays/test/ippool.yaml")

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