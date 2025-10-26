# 1. Generate the cluster's certificates and base client configuration
resource "talos_client_configuration" "vixens" {
  cluster_name = var.cluster_name
  client_name  = "administrator"
  endpoints    = [for node in var.control_plane_nodes : node.ip_address_int]
  nodes        = [for node in var.control_plane_nodes : node.ip_address_int]
}

# 2. Generate machine configurations for each control plane node
resource "talos_machine_configuration" "control_plane" {
  for_each = var.control_plane_nodes

  cluster_name               = var.cluster_name
  cluster_endpoint           = var.cluster_endpoint
  machine_name               = each.value.hostname
  machine_type               = "controlplane"
  talos_version              = "v1.7.1" # Use a known, recent version
  kubernetes_version         = "1.29.5" # Use a known, recent version
  client_configuration       = talos_client_configuration.vixens.client_configuration
  machine_network_interfaces = [
    {
      device_selectors = { pci_bus_id = "0000:00:10.0" }
      vlans = [
        {
          vlan_id = 111
          dhcp = false
          addresses = ["${each.value.ip_address_int}/24"]
          routes = [
            { network = "0.0.0.0/0", gateway = each.value.gateway_int }
          ]
          nameservers = each.value.dns_servers
        },
        {
          vlan_id = each.value.vlan_id_ext
          dhcp = false
          addresses = ["${each.value.ip_address_ext}/24"]
          routes = []
        }
      ]
    }
  ]
  machine_install = {
    disk = each.value.install_disk
  }
  machine_patch = {
    "cluster.network.cni.name" = "none"
  }
}


# --- TÂCHE SUIVANTE : NOUS REMPLIRONS CETTE PARTIE ENSEMBLE ---
# 1. Générer la configuration de base du cluster (talos_client_configuration)
# 2. Générer les machineconfigs pour chaque nœud (talos_machine_configuration)
# 3. Appliquer la configuration aux nœuds (talos_machine_bootstrap)
# 4. Récupérer le kubeconfig (data.talos_kubeconfig)