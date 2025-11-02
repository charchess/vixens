# Bootstrap Cilium Load Balancer resources before ArgoCD needs them
# This breaks the chicken-and-egg dependency cycle during initial apply.

# Apply the IPAddressPool for the 'dev' environment
resource "kubectl_manifest" "cilium_ip_pool" {
  yaml_body = file("${path.module}/../../../apps/cilium-lb/overlays/dev/ippool.yaml")

  depends_on = [
    helm_release.cilium
  ]
}

# Apply the L2 announcement policy
resource "kubectl_manifest" "cilium_l2_policy" {
  yaml_body = file("${path.module}/../../../apps/cilium-lb/base/l2policy.yaml")

  depends_on = [
    helm_release.cilium
  ]
}