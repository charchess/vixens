# ============================================================================
# PROVIDERS CONFIGURATION
# ============================================================================

provider "helm" {
  kubernetes = {
    host                   = module.environment.talos_cluster.kubernetes_host
    client_certificate     = base64decode(module.environment.talos_cluster.kubernetes_client_certificate)
    client_key             = base64decode(module.environment.talos_cluster.kubernetes_client_key)
    cluster_ca_certificate = base64decode(module.environment.talos_cluster.kubernetes_ca_certificate)
  }
}

provider "kubectl" {
  host                   = module.environment.talos_cluster.kubernetes_host
  client_certificate     = base64decode(module.environment.talos_cluster.kubernetes_client_certificate)
  client_key             = base64decode(module.environment.talos_cluster.kubernetes_client_key)
  cluster_ca_certificate = base64decode(module.environment.talos_cluster.kubernetes_ca_certificate)
  load_config_file       = false
}

provider "kubernetes" {
  host                   = module.environment.talos_cluster.kubernetes_host
  client_certificate     = base64decode(module.environment.talos_cluster.kubernetes_client_certificate)
  client_key             = base64decode(module.environment.talos_cluster.kubernetes_client_key)
  cluster_ca_certificate = base64decode(module.environment.talos_cluster.kubernetes_ca_certificate)
}
