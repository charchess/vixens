# ArgoCD GitOps Deployment
# ArgoCD for continuous deployment and GitOps workflow

resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = "7.7.7"
  namespace        = "argocd"
  create_namespace = true

  # Wait for ArgoCD to be ready
  wait          = true
  wait_for_jobs = true
  timeout       = 600 # 10 minutes

  values = [yamlencode({
    # Server configuration
    server = {
      # Insecure mode (HTTP, no TLS) - environment-specific
      # dev/test: true (Traefik will terminate TLS later)
      # staging/prod: false (TLS at ArgoCD level)
      config = {
        url = "http://${var.argocd_loadbalancer_ip}"
      }
      extraArgs = concat(
        var.argocd_insecure ? ["--insecure"] : [],
        var.argocd_disable_auth ? ["--disable-auth"] : []
      )

      # Service configuration (parameterized per environment)
      service = {
        type = var.argocd_service_type
        annotations = merge(
          {
            "environment" = var.environment
          },
          # Use Cilium IPAM annotation for LoadBalancer IP assignment
          var.argocd_service_type == "LoadBalancer" && var.argocd_loadbalancer_ip != null ? {
            "io.cilium/lb-ipam-ips" = var.argocd_loadbalancer_ip
          } : {}
        )
      }

      # Tolerate control-plane taint for full control-plane cluster
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # Repo server tolerations
    repoServer = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # Application controller tolerations
    controller = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # Redis tolerations
    redis = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # ApplicationSet controller tolerations
    applicationSet = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # Notifications controller tolerations
    notifications = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # Dex (SSO) tolerations
    dex = {
      enabled = false # Disable Dex for now, will configure later if needed
    }

    # Redis secret init Job tolerations
    redisSecretInit = {
      tolerations = [
        {
          key      = "node-role.kubernetes.io/control-plane"
          operator = "Exists"
          effect   = "NoSchedule"
        }
      ]
    }

    # Config (environment-specific)
    configs = {
      params = {
        "server.insecure" = var.argocd_insecure
      }
      cm = {
        "users.anonymous.enabled" = var.argocd_anonymous_enabled ? "true" : "false"

        "url"        = "http://${var.argocd_loadbalancer_ip}"
        "policy.csv" = <<-EOT
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

      # Désactiver le secret admin


      # Configuration RBAC
      rbac = {
        create        = true
        policyDefault = "role:readonly" # TODO: a corrigé pour utiliser une variable
      }
    }
  })]

  # Deploy after Cilium is ready and the LB pool has been created
  depends_on = [
    module.cilium
  ]
}

# Bootstrap root-app automatically (App-of-Apps pattern)
# This enables full GitOps automation - after this, all deployments are via Git
# Template is rendered with environment-specific values
resource "kubectl_manifest" "argocd_root_app" {
  yaml_body = templatefile("../../../argocd/base/root-app.yaml.tpl", {
    environment     = var.environment
    target_revision = var.git_branch
    overlay_path    = "argocd/overlays/${var.environment}"
  })

  # Wait for ArgoCD to be fully deployed and healthy
  depends_on = [
    helm_release.argocd
  ]
}
