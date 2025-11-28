# ============================================================================
# ARGOCD MODULE - OPTIMIZED WITH DRY PRINCIPLE
# ============================================================================
# Eliminates duplication by using shared tolerations and typed configurations

resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.chart_version
  namespace        = var.namespace
  create_namespace = true

  wait          = true
  wait_for_jobs = true
  timeout       = var.timeout

  values = [yamlencode({
    server = {
      config = {
        url = "http://${var.argocd_config.loadbalancer_ip}"
      }
      extraArgs = concat(
        var.argocd_config.insecure ? ["--insecure"] : [],
        var.argocd_config.disable_auth ? ["--disable-auth"] : []
      )
      service = {
        type           = var.argocd_config.service_type
        loadBalancerIP = var.argocd_config.service_type == "LoadBalancer" ? var.argocd_config.loadbalancer_ip : null
        annotations = {
          "environment"           = var.environment
          "io.cilium/lb-ipam-ips" = var.argocd_config.loadbalancer_ip
        }
      }
      ingress = {
        enabled          = false
        ingressClassName = "traefik"
        hosts            = [var.argocd_config.hostname]
        paths            = ["/"]
        annotations = {
          "traefik.ingress.kubernetes.io/router.entrypoints" = "web"
        }
      }
      tolerations = var.control_plane_tolerations
    }

    # DRY: Apply same tolerations to all components
    repoServer      = { tolerations = var.control_plane_tolerations }
    controller      = { tolerations = var.control_plane_tolerations }
    redis           = { tolerations = var.control_plane_tolerations }
    applicationSet  = { tolerations = var.control_plane_tolerations }
    notifications   = { tolerations = var.control_plane_tolerations }
    redisSecretInit = { tolerations = var.control_plane_tolerations }

    dex = {
      enabled = false
    }

    configs = {
      params = {
        "server.insecure" = var.argocd_config.insecure
      }
      cm = {
        "users.anonymous.enabled" = var.argocd_config.anonymous_enabled ? "true" : "false"
        "url"                     = "http://${var.argocd_config.loadbalancer_ip}"
        "policy.csv"              = <<-EOT
          p, role:readonly, applications, get, */*, allow
          p, role:readonly, applications, list, */*, allow
          p, role:readonly, clusters, get, *, allow
          p, role:readonly, repositories, get, *, allow
          p, role:readonly, repositories, list, *, allow
          p, role:readonly, projects, get, *, allow
          p, role:readonly, projects, list, *, allow
          g, anonymous, role:readonly
          g, default, role:readonly
        EOT
      }
      rbac = {
        create        = true
        policyDefault = "role:readonly"
      }
    }
  })]

  depends_on = [
    var.cilium_module
  ]
}

# --------------------------------------------------------------------------
# INFISICAL UNIVERSAL AUTH SECRET (BOOTSTRAP)
# --------------------------------------------------------------------------
# Deploy Infisical credentials secret before root-app to enable InfisicalSecret CRDs
resource "kubernetes_secret_v1" "infisical_universal_auth" {
  count = var.infisical_secret_path != "" ? 1 : 0

  metadata {
    name      = "infisical-universal-auth"
    namespace = var.namespace

    labels = {
      "app"        = "infisical-operator"
      "managed-by" = "terraform"
    }
  }

  data = {
    clientId     = yamldecode(file(var.infisical_secret_path)).stringData.clientId
    clientSecret = yamldecode(file(var.infisical_secret_path)).stringData.clientSecret
  }

  type = "Opaque"

  depends_on = [
    helm_release.argocd,
    var.cilium_module
  ]
}

# --------------------------------------------------------------------------
# ROOT APPLICATION (App-of-Apps)
# --------------------------------------------------------------------------
resource "kubectl_manifest" "argocd_root_app" {
  yaml_body = templatefile(var.root_app_template_path, {
    environment     = var.environment
    target_revision = var.git_branch
    overlay_path    = "argocd/overlays/${var.environment}"
  })

  depends_on = [
    helm_release.argocd,
    kubernetes_secret_v1.infisical_universal_auth
  ]
}
