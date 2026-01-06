# Adding a New Talos Node to the Cluster

## Problem Statement

When adding a new worker or control plane node to an existing Talos cluster via Terraform, there's a bootstrap challenge:

**The Issue:**
- Terraform's `talos_machine_configuration_apply` resource tries to connect to the node's VLAN IP address
- But this VLAN IP **doesn't exist yet** on a brand new node in maintenance mode
- New nodes are only accessible on their maintenance IP (192.168.0.x) during initial PXE boot
- This creates a chicken-and-egg problem: can't apply config without VLAN IP, can't get VLAN IP without applying config

**Current Behavior:**
- `terraform apply` fails with: `dial tcp 192.168.X.X:50000: connect: no route to host`
- The node stays in maintenance mode, waiting for configuration

## Current Manual Workaround (Pearl Example)

This is the procedure we successfully used to add pearl as a worker node:

### Step 1: Ensure Node in Maintenance Mode

```bash
# Node should be booted via PXE with Talos maintenance image
# Verify connectivity on maintenance IP
ping -c 3 192.168.0.58  # Replace with your node's maintenance IP

# Test insecure Talos connection
talosctl --nodes 192.168.0.58 --endpoints 192.168.0.58 version -i
```

### Step 2: Add Node to terraform.tfvars

```hcl
worker_nodes = {
  "pearl" = {
    name         = "pearl"
    ip_address   = "192.168.0.58"    # Maintenance IP
    mac_address  = "00:e8:4c:68:83:71"
    install_disk = "sda"
    nameservers  = ["192.168.201.70", "192.168.201.1"]
    network = {
      interface = "enp1s0"
      vlans = [
        {
          vlanId    = 111
          addresses = ["192.168.111.192/24"]
          gateway   = ""
        },
        {
          vlanId    = 201
          addresses = ["192.168.201.192/24"]
          gateway   = "192.168.201.1"
        }
      ]
    }
  }
}
```

### Step 3: Run Targeted Terraform Apply

```bash
cd terraform/environments/prod

# Apply ONLY the new node resources
terraform apply \
  -target='module.environment.module.talos_cluster.data.talos_machine_configuration.worker["pearl"]' \
  -target='module.environment.module.talos_cluster.talos_machine_configuration_apply.worker["pearl"]' \
  -target='module.environment.module.talos_cluster.null_resource.worker_reset_on_destroy["pearl"]' \
  -target='module.environment.module.talos_cluster.null_resource.worker_upgrade["pearl"]'
```

**Expected:** This will fail with connection error (node not reachable on VLAN IP yet)

### Step 4: Extract Machine Configuration

```bash
# Export the generated configuration
terraform show -json | jq -r '
  .values.root_module.child_modules[]
  | select(.address == "module.environment")
  | .child_modules[]
  | select(.address == "module.environment.module.talos_cluster")
  | .resources[]
  | select(.address == "module.environment.module.talos_cluster.data.talos_machine_configuration.worker[\"pearl\"]")
  | .values.machine_configuration
' > /tmp/pearl-worker-config.yaml
```

### Step 5: Apply Configuration Manually (Insecure Mode)

```bash
# Apply config using insecure mode on maintenance IP
talosctl apply-config \
  --nodes 192.168.0.58 \
  --endpoints 192.168.0.58 \
  --insecure \
  --mode=auto \
  --file /tmp/pearl-worker-config.yaml
```

**This triggers:**
- Talos installation to disk
- Network configuration with VLANs
- Node reboot into production mode

### Step 6: Wait for Node Boot

```bash
# Wait ~30-60 seconds, then check VLAN connectivity
ping -c 3 192.168.111.192  # VLAN 111 IP

# Verify node joined cluster
export KUBECONFIG=./kubeconfig-prod
kubectl get nodes -o wide
```

### Step 7: Upgrade to Target Version (if needed)

```bash
export TALOSCONFIG=./talosconfig-prod

# Check current version
talosctl --nodes 192.168.111.192 version

# Upgrade if not on target version
talosctl upgrade \
  --nodes 192.168.111.192 \
  --endpoints 192.168.111.192 \
  --image "factory.talos.dev/metal-installer/249d9135de54962744e917cfe654117000cba369f9152fbab9d055a00aa3664f:v1.12.0" \
  --preserve=true \
  --wait=false

# Wait ~60 seconds for upgrade
sleep 60
kubectl get nodes -o wide | grep pearl
```

### Step 8: Apply Full Terraform to Update State

```bash
# Now that node is accessible on VLAN IP, apply full config
terraform apply

# This should show minimal changes (just endpoint updates)
```

## Architecture Considerations

### Why This Happens

The Terraform Talos provider's `talos_machine_configuration_apply` resource is designed for:
1. **Existing nodes** - Already running Talos, accessible on their configured IPs
2. **Reprovisioning** - Nodes being reconfigured from a known state

It's **not** designed for:
- Bare metal first-time provisioning
- Nodes in maintenance/installer mode

### Maintenance IP vs VLAN IP Lifecycle

```
┌─────────────────┐
│  PXE Boot       │  Maintenance IP (192.168.0.x) - DHCP from UniFi
│  Talos Installer│  Available for insecure connection
└────────┬────────┘
         │ talosctl apply-config --insecure
         ▼
┌─────────────────┐
│  Installing...  │  Writes to disk, configures VLANs
│                 │  Maintenance IP still active
└────────┬────────┘
         │ Reboot
         ▼
┌─────────────────┐
│  Production     │  VLAN IPs only (111, 201)
│  Talos Running  │  Maintenance IP gone
│                 │  Secure connection required
└─────────────────┘
```

### Current Module Limitations

**terraform/modules/talos/main.tf:**
```hcl
resource "talos_machine_configuration_apply" "worker" {
  for_each = var.worker_nodes

  client_configuration        = talos_machine_secrets.cluster.client_configuration
  machine_configuration_input = data.talos_machine_configuration.worker[each.key].machine_configuration

  # Always uses VLAN IP - breaks for new nodes!
  node     = local.worker_vlan_ips[each.key]
  endpoint = local.worker_vlan_ips[each.key]
}
```

## Desired Behavior (Future)

### Requirements

1. **Detect new vs existing nodes** - Terraform should differentiate between:
   - New nodes (not in state, need insecure bootstrap)
   - Existing nodes (in state, use secure VLAN connection)

2. **Automatic bootstrap** - For new nodes:
   - Use maintenance IP (from tfvars `ip_address` field)
   - Apply config in insecure mode
   - Wait for reboot
   - Upgrade to target version
   - Update state to use VLAN IPs

3. **Safe updates** - For existing nodes:
   - Continue using VLAN IPs
   - No unnecessary reconnections
   - Preserve existing behavior

### Proposed Solutions

#### Option 1: Lifecycle-Based Detection

Use Terraform lifecycle to detect resource creation:

```hcl
resource "talos_machine_configuration_apply" "worker" {
  for_each = var.worker_nodes

  # Use maintenance IP during creation, VLAN IP after
  node     = self.id == null ? each.value.ip_address : local.worker_vlan_ips[each.key]
  endpoint = self.id == null ? each.value.ip_address : local.worker_vlan_ips[each.key]

  # ... rest of config
}
```

**Issues:**
- Terraform doesn't expose `self.id` during plan phase
- Would need custom logic or null_resource workarounds

#### Option 2: External Bootstrap Script

Keep Terraform focused on config generation, use external script for bootstrap:

```bash
# scripts/bootstrap-new-node.sh
NODE_NAME=$1
NODE_MAINTENANCE_IP=$2

# 1. Generate config via Terraform
terraform apply -target="...data.talos_machine_configuration.worker[\"$NODE_NAME\"]"

# 2. Extract config
CONFIG=$(terraform output -json | jq -r ".worker_configs.value.$NODE_NAME")

# 3. Apply insecure
talosctl apply-config --nodes $NODE_MAINTENANCE_IP --insecure --file <(echo "$CONFIG")

# 4. Wait for boot
# 5. Import to Terraform state
# 6. Run full terraform apply
```

**Pros:**
- Clear separation of concerns
- More control over bootstrap process
- Easier to debug

**Cons:**
- Manual step outside Terraform
- Not fully declarative

#### Option 3: Two-Phase Terraform

Use separate Terraform resources for bootstrap vs management:

```hcl
# Phase 1: Bootstrap (runs once)
resource "null_resource" "bootstrap_worker" {
  for_each = var.worker_nodes

  triggers = {
    config_hash = sha256(data.talos_machine_configuration.worker[each.key].machine_configuration)
  }

  provisioner "local-exec" {
    command = <<-EOT
      talosctl apply-config \
        --nodes ${each.value.ip_address} \
        --insecure \
        --file <(echo '${data.talos_machine_configuration.worker[each.key].machine_configuration}')
    EOT
  }
}

# Phase 2: Management (ongoing)
resource "talos_machine_configuration_apply" "worker" {
  for_each = var.worker_nodes

  depends_on = [null_resource.bootstrap_worker]

  node     = local.worker_vlan_ips[each.key]
  endpoint = local.worker_vlan_ips[each.key]
  # ...
}
```

**Pros:**
- Fully in Terraform
- Clear bootstrap vs management separation

**Cons:**
- More complex
- Hard to handle failures/retries
- Provisioners are discouraged in Terraform

## Recommended Approach

**For now:** Continue with documented manual procedure (this doc)

**For future:** Implement Option 2 (External Bootstrap Script) with:
- `scripts/add-talos-node.sh <node-name> <environment>`
- Handles full lifecycle automatically
- Updates Terraform state
- Can be enhanced with error handling, retries, validations

## Related Files

- `terraform/modules/talos/main.tf` - Talos cluster module
- `terraform/modules/talos/variables.tf` - Node configuration schema
- `terraform/environments/*/terraform.tfvars` - Per-environment node configs
- `docs/adr/XXX-talos-node-bootstrap.md` - (To be created) ADR on chosen solution

## References

- [Talos apply-config documentation](https://www.talos.dev/v1.12/reference/cli/#talosctl-apply-config)
- [Terraform Talos Provider](https://registry.terraform.io/providers/siderolabs/talos/latest/docs)
- Git history: commit d122d2f (introduced VLAN IP requirement)
- Git history: commit 85ce734 (previous revert with maintenance IPs)
