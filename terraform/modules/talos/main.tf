# 1. Generate the cluster's certificates and base client configuration
resource "talos_client_configuration" "vixens" {
  cluster_name = var.cluster_name
  client_name  = "administrator"
  endpoints    = [for node in var.control_plane_nodes : node.ip_address_int]
  nodes        = [for node in var.control_plane_nodes : node.ip_address_int]
}

# --- TÂCHE SUIVANTE : NOUS REMPLIRONS CETTE PARTIE ENSEMBLE ---
# 1. Générer la configuration de base du cluster (talos_client_configuration)
# 2. Générer les machineconfigs pour chaque nœud (talos_machine_configuration)
# 3. Appliquer la configuration aux nœuds (talos_machine_bootstrap)
# 4. Récupérer le kubeconfig (data.talos_kubeconfig)