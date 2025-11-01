# Dev Environment Variables

# Environment
environment = "dev"
vlan_services_subnet = "192.168.208.0/24"

# ArgoCD Configuration
argocd_service_type    = "LoadBalancer"  # Upgraded in Sprint 5 with MetalLB
argocd_loadbalancer_ip = "192.168.208.71"  # Reserved IP for future LoadBalancer
argocd_hostname        = "argocd.dev.vixens.lab"

argocd_disable_auth = true  # Future Ingress hostname (Sprint 6)
