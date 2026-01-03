# Talos Machine Secrets - generated once per cluster
resource "talos_machine_secrets" "cluster" {}

# Extract VIP address from cluster_endpoint
locals {
  vip_address = regex("https?://([^:]+)", var.cluster_endpoint)[0]
}

# Generate per-node configuration patches
locals {
  node_patches = {
    for k, v in var.control_plane_nodes : k => yamlencode({
      machine = {
        install = merge(
          {
            disk = v.install_disk
          },
          # Add custom image only if provided (non-empty string)
          var.talos_image != "" ? { image = var.talos_image } : {}
        )
        network = merge(
          {
            hostname = v.name # Set node hostname
            interfaces = [{
              interface = v.network.interface
              dhcp      = false # Disable DHCP on physical interface
              addresses = []    # No IP on untagged interface
              vlans = [
                for vlan in v.network.vlans : merge(
                  {
                    vlanId    = vlan.vlanId
                    addresses = vlan.addresses
                    routes = vlan.gateway != "" ? [{
                      network = "0.0.0.0/0"
                      gateway = vlan.gateway
                    }] : []
                  },
                  # Add VIP on internal VLAN (no gateway = VLAN 111)
                  vlan.gateway == "" ? {
                    vip = {
                      ip = local.vip_address
                    }
                  } : {}
                )
              ]
            }]
          },
          # Add nameservers only if provided (non-empty list)
          length(v.nameservers) > 0 ? { nameservers = v.nameservers } : {}
        )
      }
      cluster = {
        network = {
          podSubnets     = [var.pod_subnet]
          serviceSubnets = [var.service_subnet]
          # Disable default CNI (Flannel) to use Cilium instead
          cni = {
            name = "none"
          }
        }
        # Disable kube-proxy (Cilium replaces it)
        proxy = {
          disabled = true
        }
        # Add VIP to API server certificates
        apiServer = {
          certSANs = [local.vip_address]
        }
      }
    })
  }

  # Extract VLAN IP with gateway (routable IP) for each control plane node
  # This is used for talosctl reset during destroy, as nodes are not reachable on maintenance IPs
  control_plane_vlan_ips = {
    for k, v in var.control_plane_nodes : k => [
      for vlan in v.network.vlans :
      split("/", vlan.addresses[0])[0]
      if vlan.gateway != ""
    ][0]
  }

  control_plane_vlan111_ips = {
    for k, v in var.control_plane_nodes : k => [
      for vlan in v.network.vlans :
      split("/", vlan.addresses[0])[0]
      if vlan.vlanId == 111
    ][0]
  }

  # Extract VLAN IP with gateway for each worker node
  worker_vlan_ips = {
    for k, v in var.worker_nodes : k => [
      for vlan in v.network.vlans :
      split("/", vlan.addresses[0])[0]
      if vlan.gateway != ""
    ][0]
  }
}

data "talos_machine_configuration" "control_plane" {
  for_each = var.control_plane_nodes

  cluster_name     = var.cluster_name
  machine_type     = "controlplane"
  cluster_endpoint = var.cluster_endpoint
  machine_secrets  = talos_machine_secrets.cluster.machine_secrets
  talos_version    = var.talos_version

  config_patches = [
    local.node_patches[each.key]
  ]
}

data "talos_client_configuration" "this" {
  cluster_name         = var.cluster_name
  client_configuration = talos_machine_secrets.cluster.client_configuration
  endpoints            = [local.vip_address]
  nodes = concat(
    [for k, v in var.control_plane_nodes : local.control_plane_vlan111_ips[k]],
    [for k, v in var.worker_nodes : local.worker_vlan_ips[k]]
  )
}

resource "talos_machine_configuration_apply" "control_plane" {
  for_each = var.control_plane_nodes

  client_configuration        = talos_machine_secrets.cluster.client_configuration
  machine_configuration_input = data.talos_machine_configuration.control_plane[each.key].machine_configuration
  # Use VLAN IP (routable) instead of maintenance IP for configuration apply
  # Maintenance IPs are only accessible during initial PXE boot, not after Talos config
  node     = local.control_plane_vlan_ips[each.key]
  endpoint = local.control_plane_vlan_ips[each.key]
}

resource "talos_machine_bootstrap" "this" {
  depends_on = [
    talos_machine_configuration_apply.control_plane
  ]
  node                 = [for k, v in var.control_plane_nodes : local.control_plane_vlan_ips[k]][0]
  client_configuration = talos_machine_secrets.cluster.client_configuration
}

resource "talos_cluster_kubeconfig" "this" {
  depends_on = [
    talos_machine_bootstrap.this
  ]
  client_configuration = talos_machine_secrets.cluster.client_configuration
  node                 = [for k, v in var.control_plane_nodes : local.control_plane_vlan_ips[k]][0]
}

# Automatic node reset on destroy - Control Plane
resource "null_resource" "node_reset_on_destroy" {
  for_each = var.control_plane_nodes

  # This resource depends on the cluster being configured
  depends_on = [
    talos_machine_bootstrap.this
  ]

  # Store talosconfig content and VLAN IP in triggers
  # Using VLAN IP with gateway (routable IP) because nodes are not reachable on maintenance IPs after config is applied
  # This ensures the talosconfig is available even after local_file is destroyed
  triggers = {
    node_ip     = local.control_plane_vlan_ips[each.key]
    talosconfig = data.talos_client_configuration.this.talos_config
  }

  # Reset node before destroying terraform state
  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      # Create temporary talosconfig file with stored content
      TEMP_TALOSCONFIG=$(mktemp)
      cat > $TEMP_TALOSCONFIG <<'EOF'
${self.triggers.talosconfig}
EOF

      export TALOSCONFIG=$TEMP_TALOSCONFIG
      echo "Resetting node ${self.triggers.node_ip} before destroy..."
      talosctl reset \
        -n ${self.triggers.node_ip} \
        -e ${self.triggers.node_ip} \
        --system-labels-to-wipe STATE \
        --system-labels-to-wipe EPHEMERAL \
        --graceful=false \
        --reboot \
        --wait=false || echo "Warning: Node reset failed, continuing destroy"

      # Cleanup temp file
      rm -f $TEMP_TALOSCONFIG
      echo "Node ${self.triggers.node_ip} reset initiated"
    EOT
  }
}

# Worker nodes configuration
locals {
  worker_patches = {
    for k, v in var.worker_nodes : k => yamlencode({
      machine = {
        install = merge(
          {
            disk = v.install_disk
          },
          # Add custom image only if provided (non-empty string)
          var.talos_image != "" ? { image = var.talos_image } : {}
        )
        network = merge(
          {
            hostname = v.name # Set node hostname
            interfaces = [{
              interface = v.network.interface
              dhcp      = false # Disable DHCP on physical interface
              addresses = []    # No IP on untagged interface
              vlans = [
                for vlan in v.network.vlans : {
                  vlanId    = vlan.vlanId
                  addresses = vlan.addresses
                  routes = vlan.gateway != "" ? [{
                    network = "0.0.0.0/0"
                    gateway = vlan.gateway
                  }] : []
                }
              ]
            }]
          },
          # Add nameservers only if provided (non-empty list)
          length(v.nameservers) > 0 ? { nameservers = v.nameservers } : {}
        )
      }
      cluster = {
        network = {
          podSubnets     = [var.pod_subnet]
          serviceSubnets = [var.service_subnet]
        }
      }
    })
  }
}

data "talos_machine_configuration" "worker" {
  for_each = var.worker_nodes

  cluster_name     = var.cluster_name
  machine_type     = "worker"
  cluster_endpoint = var.cluster_endpoint
  machine_secrets  = talos_machine_secrets.cluster.machine_secrets
  talos_version    = var.talos_version

  config_patches = [
    local.worker_patches[each.key]
  ]
}

resource "talos_machine_configuration_apply" "worker" {
  for_each = var.worker_nodes

  client_configuration        = talos_machine_secrets.cluster.client_configuration
  machine_configuration_input = data.talos_machine_configuration.worker[each.key].machine_configuration
  # Use VLAN IP (routable) instead of maintenance IP for configuration apply
  node     = local.worker_vlan_ips[each.key]
  endpoint = local.worker_vlan_ips[each.key]
}

# Automatic node reset on destroy - Workers
resource "null_resource" "worker_reset_on_destroy" {
  for_each = var.worker_nodes

  # This resource depends on the worker being configured
  depends_on = [
    talos_machine_configuration_apply.worker
  ]

  # Store talosconfig content and VLAN IP in triggers
  # Using VLAN IP with gateway (routable IP) because nodes are not reachable on maintenance IPs after config is applied
  # This ensures the talosconfig is available even after local_file is destroyed
  triggers = {
    node_ip     = local.worker_vlan_ips[each.key]
    talosconfig = data.talos_client_configuration.this.talos_config
  }

  # Reset node before destroying terraform state
  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      # Create temporary talosconfig file with stored content
      TEMP_TALOSCONFIG=$(mktemp)
      cat > $TEMP_TALOSCONFIG <<'EOF'
${self.triggers.talosconfig}
EOF

      export TALOSCONFIG=$TEMP_TALOSCONFIG
      echo "Resetting worker node ${self.triggers.node_ip} before destroy..."
      talosctl reset \
        -n ${self.triggers.node_ip} \
        -e ${self.triggers.node_ip} \
        --system-labels-to-wipe STATE \
        --system-labels-to-wipe EPHEMERAL \
        --graceful=false \
        --reboot \
        --wait=false || echo "Warning: Worker node reset failed, continuing destroy"

      # Cleanup temp file
      rm -f $TEMP_TALOSCONFIG
      echo "Worker node ${self.triggers.node_ip} reset initiated"
    EOT
  }
}

# Automatic Talos upgrade when talos_version or talos_image changes - Control Plane
resource "null_resource" "control_plane_upgrade" {
  for_each = var.control_plane_nodes

  # Trigger upgrade when version or image changes
  triggers = {
    talos_version = var.talos_version
    talos_image   = var.talos_image
    node_ip       = local.control_plane_vlan_ips[each.key]
  }

  # Upgrade must happen after node is configured and bootstrapped
  depends_on = [
    talos_machine_bootstrap.this,
    talos_machine_configuration_apply.control_plane
  ]

  # Upgrade node when triggers change
  provisioner "local-exec" {
    command = <<-EOT
      # Create temporary talosconfig file
      TEMP_TALOSCONFIG=$(mktemp)
      cat > $TEMP_TALOSCONFIG <<'EOF'
${data.talos_client_configuration.this.talos_config}
EOF

      export TALOSCONFIG=$TEMP_TALOSCONFIG

      # Determine image to use
      IMAGE="${var.talos_image != "" ? var.talos_image : format("ghcr.io/siderolabs/installer:%s", var.talos_version)}"

      echo "Upgrading node ${self.triggers.node_ip} to $IMAGE..."
      talosctl upgrade \
        --nodes ${self.triggers.node_ip} \
        --endpoints ${self.triggers.node_ip} \
        --image "$IMAGE" \
        --preserve=true \
        --wait=false

      # Cleanup temp file
      rm -f $TEMP_TALOSCONFIG
      echo "Node ${self.triggers.node_ip} upgrade initiated"
    EOT
  }
}

# Automatic Talos upgrade when talos_version or talos_image changes - Workers
resource "null_resource" "worker_upgrade" {
  for_each = var.worker_nodes

  # Trigger upgrade when version or image changes
  triggers = {
    talos_version = var.talos_version
    talos_image   = var.talos_image
    node_ip       = local.worker_vlan_ips[each.key]
  }

  # Upgrade must happen after node is configured
  depends_on = [
    talos_machine_configuration_apply.worker
  ]

  # Upgrade node when triggers change
  provisioner "local-exec" {
    command = <<-EOT
      # Create temporary talosconfig file
      TEMP_TALOSCONFIG=$(mktemp)
      cat > $TEMP_TALOSCONFIG <<'EOF'
${data.talos_client_configuration.this.talos_config}
EOF

      export TALOSCONFIG=$TEMP_TALOSCONFIG

      # Determine image to use
      IMAGE="${var.talos_image != "" ? var.talos_image : format("ghcr.io/siderolabs/installer:%s", var.talos_version)}"

      echo "Upgrading worker node ${self.triggers.node_ip} to $IMAGE..."
      talosctl upgrade \
        --nodes ${self.triggers.node_ip} \
        --endpoints ${self.triggers.node_ip} \
        --image "$IMAGE" \
        --preserve=true \
        --wait=false

      # Cleanup temp file
      rm -f $TEMP_TALOSCONFIG
      echo "Worker node ${self.triggers.node_ip} upgrade initiated"
    EOT
  }
}
