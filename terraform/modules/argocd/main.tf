resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.chart_version
  namespace        = var.namespace
  create_namespace = true

  wait          = true
  wait_for_jobs = true
  timeout       = 600

  values = [yamlencode({
    server = {
      config = {
        url = "http://${var.argocd_loadbalancer_ip}"
      }
      extraArgs = concat(
        var.argocd_insecure ? ["--insecure"] : [],
        var.argocd_disable_auth ? ["--disable-auth"] : []
      )
      service = {
        type           = var.argocd_service_type
        loadBalancerIP = var.argocd_service_type == "LoadBalancer" ? var.argocd_loadbalancer_ip : null
        annotations = {
          "environment" = var.environment
        }
      }
      ingress = {
        enabled          = false
        ingressClassName = "traefik"
        hosts            = [var.argocd_hostname]
        paths            = ["/"]
        annotations = {
          "traefik.ingress.kubernetes.io/router.entrypoints" = "web"
        }
      }
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    repoServer = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    controller = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    redis = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    applicationSet = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    notifications = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    dex = {
      enabled = false
    }
    redisSecretInit = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }
    configs = {
      params = {
        "server.insecure" = var.argocd_insecure
      }
      cm = {
        "users.anonymous.enabled" = var.argocd_anonymous_enabled ? "true" : "false"
        "url"                     = "http://${var.argocd_loadbalancer_ip}"
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

resource "kubectl_manifest" "argocd_root_app" {
  yaml_body = templatefile(var.root_app_template_path, {
    environment     = var.environment
    target_revision = var.git_branch
    overlay_path    = "argocd/overlays/${var.environment}"
  })

  depends_on = [
    helm_release.argocd
  ]
}


