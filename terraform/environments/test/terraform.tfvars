# Test Environment Variables

# Environment
environment = "test"
vlan_services_subnet = "192.168.209.0/24"

# ArgoCD Configuration
argocd_service_type    = "LoadBalancer"
argocd_loadbalancer_ip = "192.168.209.81"
argocd_hostname        = "argocd.test.vixens.lab"

argocd_disable_auth = true
