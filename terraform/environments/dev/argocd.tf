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
      extraArgs = ["--insecure"] # HTTP mode (Traefik will terminate TLS later)

      # Service configuration (parameterized per environment)
      service = {
        type = var.argocd_service_type
        # LoadBalancer IP only used when type is LoadBalancer
        loadBalancerIP = var.argocd_service_type == "LoadBalancer" ? var.argocd_loadbalancer_ip : null
        annotations = {
          "environment" = var.environment
        }
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

    # Config
    configs = {
      params = {
        "server.insecure" = true
      }
    }
  })]

  # Deploy after Cilium is ready
  depends_on = [
    helm_release.cilium
  ]
}
