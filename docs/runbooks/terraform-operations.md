# Terraform Operations Runbook

## Overview

This runbook covers common Terraform operations for managing Talos Kubernetes clusters in the Vixens infrastructure.

## Prerequisites

```bash
# Install required tools
# - Terraform >= 1.5.0
# - talosctl >= 1.10.0
# - kubectl >= 1.30.0

# Working directory
cd /root/vixens/terraform/environments/dev
```

## Standard Operations

### 1. Initialize Environment

```bash
# First time or after changing providers
terraform init

# Upgrade providers to latest matching version constraints
terraform init -upgrade
```

### 2. Plan Changes

```bash
# Review changes before applying
terraform plan

# Save plan to file for later apply
terraform plan -out=tfplan

# Plan with specific variables
terraform plan -var="cluster_name=vixens-test"
```

### 3. Apply Changes

```bash
# Apply with confirmation prompt
terraform apply

# Apply saved plan
terraform apply tfplan

# Apply without confirmation (CI/CD)
terraform apply -auto-approve
```

### 4. Check State

```bash
# Show current state
terraform show

# List all resources
terraform state list

# Show specific resource
terraform state show 'module.talos_cluster.talos_machine_secrets.cluster'

# Inspect module outputs
terraform output
terraform output -json kubeconfig
```

### 5. Validate Configuration

```bash
# Format code
terraform fmt -recursive

# Validate syntax
terraform validate

# Check for outdated providers
terraform providers
```

## Destroy/Recreate Lifecycle

### ⚠️ CRITICAL LIMITATION

**terraform destroy does NOT reset physical nodes** - it only removes terraform state tracking.

After `terraform destroy`:
- Nodes continue running with their old cluster configuration and certificates
- Terraform can't connect to nodes with new credentials (certificate mismatch)
- Recreate requires manual node reset via console or hypervisor access

**For true clean recreate, you MUST:**
1. Use VM console to manually reset nodes (talosctl reset via console), OR
2. Use hypervisor (Proxmox) to reset/reinstall VMs, OR
3. Keep working talosconfig before destroy to reset nodes manually

### When to Destroy

✅ **Safe to destroy:**
- Dev environment after sprint completion
- Test environment for validation testing
- Before major refactoring to validate clean build

❌ **DO NOT destroy:**
- Prod environment (physical infrastructure)
- Staging without explicit approval
- During normal development (use `apply` to update)

### Destroy Workflow

```bash
cd terraform/environments/dev

# 0. IMPORTANT: Save working talosconfig before destroy (for node cleanup)
cp talosconfig-dev talosconfig-predestroy-backup

# 1. Review what will be destroyed
terraform plan -destroy

# 2. Destroy all resources
terraform destroy

# Confirm with 'yes' when prompted
# OR use -auto-approve for scripted operations:
# terraform destroy -auto-approve

# 3. REQUIRED: Manually reset nodes using saved config
# (Otherwise terraform apply will fail with certificate errors)
export TALOSCONFIG=$PWD/talosconfig-predestroy-backup
talosctl reset \
  -n 192.168.111.162 \
  -e 192.168.111.162 \
  --system-labels-to-wipe STATE \
  --system-labels-to-wipe EPHEMERAL \
  --graceful=true \
  --reboot \
  --wait=false

# Wait ~2-3 minutes for node reboot, then you can terraform apply
```

**What gets destroyed:**
- Talos machine configurations on nodes
- Bootstrap state
- Kubernetes cluster
- Generated kubeconfig/talosconfig files

**What persists:**
- Terraform state file (terraform.tfstate)
- VM infrastructure (nodes still exist, just reconfigured)
- Git repository

### Recreate from Scratch

**IMPORTANT: Two-step bootstrap process required**

When nodes are in maintenance mode (after manual reset), they boot with a temporary IP (192.168.0.162). You must apply configuration in two steps:

```bash
cd terraform/environments/dev

# Step 1: Apply with maintenance IP
# Edit main.tf and temporarily change ip_address to maintenance IP
sed -i 's/ip_address   = "192.168.111.162"/ip_address   = "192.168.0.162"/' main.tf

terraform apply -auto-approve
# This applies configuration to node on maintenance IP
# Node will reboot and come up with VLAN configuration

# Wait for node to reboot and come up on configured VLANs (~2 minutes)
sleep 120

# Verify node is accessible on VLAN 111
ping -c 2 192.168.111.162

# Step 2: Apply with configured IP
# Restore the correct IP in main.tf
sed -i 's/ip_address   = "192.168.0.162"/ip_address   = "192.168.111.162"/' main.tf

terraform apply -auto-approve
# This updates state to use the configured IP

# Validate new cluster
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
export TALOSCONFIG=$PWD/talosconfig-dev

talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 version
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 services
```

### ⚠️ Network Configuration Limitation: Native VLAN DHCP

**Expected behavior:** Nodes will have an additional IP on the native/untagged VLAN (192.168.0.162) even after VLAN configuration.

**Why this occurs:**
- Hypervisor (Hyper-V) assigns the virtual NIC to a native VLAN for basic connectivity
- Native VLAN provides DHCP IP (192.168.0.162) required for maintenance mode bootstrap
- Talos configures tagged VLANs (111, 208) on top of the physical interface
- Both native and tagged VLANs coexist on the same physical interface

**Why we keep this configuration:**
- ✅ **Maintenance mode accessibility**: When nodes reset to maintenance mode, they need a DHCP IP to receive initial configuration via network
- ✅ **Bootstrap process**: Two-step bootstrap (maintenance IP → VLAN IPs) requires native VLAN connectivity
- ✅ **No VLAN conflict**: Native VLAN (0/1) is different from tagged VLANs (111, 208), no double-tagging issue

**Network state after full configuration:**
```
Node IPs:
- 192.168.0.162   (native VLAN, DHCP, used only for maintenance/bootstrap)
- 192.168.111.162 (VLAN 111, static, internal cluster communication)
- 192.168.208.162 (VLAN 208, static, services network with gateway)
```

**Operational notes:**
- Primary cluster operations use VLAN 111 (192.168.111.162)
- Services and external access use VLAN 208 (192.168.208.162)
- Native VLAN IP (192.168.0.162) can be ignored during normal operations
- Native VLAN IP is essential for node recovery and maintenance procedures

**Alternative rejected:** Configuring Hyper-V in pure trunk mode (all VLANs tagged, no native VLAN) would eliminate the native IP but make nodes unreachable in maintenance mode, preventing remote configuration.

### Validation After Recreate

```bash
# 1. Terraform should show no changes
terraform plan
# Expected: "No changes. Your infrastructure matches the configuration."

# 2. Talos node is accessible
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 health

# 3. Kubernetes API is accessible
kubectl get nodes
kubectl get pods -A

# 4. Network configuration is correct
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 get addresses | grep "VLAN\|192.168"
```

## Troubleshooting

### Issue: Terraform State Locked

```bash
# If apply/destroy fails with "state locked"
# Check for stuck processes
ps aux | grep terraform

# Force unlock (USE WITH CAUTION)
terraform force-unlock <lock-id>
```

### Issue: Provider Configuration Changed

```bash
# Re-initialize with new provider config
terraform init -reconfigure
```

### Issue: Node Not Accessible

```bash
# Check if node is on expected IP
talosctl --nodes 192.168.208.162 --endpoints 192.168.208.162 version  # External VLAN
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 version  # Internal VLAN

# Check VLAN configuration
talosctl --nodes <ip> --endpoints <ip> get links
talosctl --nodes <ip> --endpoints <ip> get addresses
```

### Issue: Kubeconfig Not Working

```bash
# Regenerate configs
terraform apply -refresh-only

# Verify VIP is accessible
ping 192.168.111.160  # From management host (grenat)

# Check Kubernetes API
talosctl --nodes 192.168.111.160 service kubelet
```

## Advanced Operations

### Moving Resources Between States

```bash
# Export resource from state
terraform state pull > backup.tfstate

# Remove resource from state (without destroying)
terraform state rm 'module.talos_cluster.talos_machine_bootstrap.this'

# Import existing resource
terraform import 'module.talos_cluster.talos_machine_secrets.cluster' machine_secrets
```

### Targeting Specific Resources

```bash
# Apply changes to specific resource only
terraform apply -target='module.talos_cluster.talos_machine_configuration_apply.control_plane["obsy"]'

# Destroy specific resource only
terraform destroy -target='local_file.kubeconfig'
```

### Working with Multiple Environments

```bash
# Switch between environments
cd terraform/environments/dev
terraform apply

cd ../test
terraform apply

# Use workspaces (alternative approach)
terraform workspace list
terraform workspace select dev
terraform apply
```

## Best Practices

1. **Always run `terraform plan` before `apply`**
2. **Commit code before destroying infrastructure**
3. **Use `-auto-approve` only in CI/CD or scripts**
4. **Keep state files secure** (contain sensitive data)
5. **Document destroy operations** in git commits
6. **Validate after recreate** (terraform plan = no changes)
7. **Use descriptive commit messages** for infrastructure changes

## Emergency Procedures

### Complete State Reset

```bash
# EXTREME CAUTION: This deletes all state tracking
rm terraform.tfstate*
rm -rf .terraform/

# Reinitialize
terraform init

# Import existing resources (if any survive)
# OR destroy VMs manually and start fresh
```

### Node Stuck in Bad State

```bash
# Reset Talos machine (correct command)
# This wipes STATE and EPHEMERAL partitions, reboots, and prepares for reconfiguration
talosctl --talosconfig $TALOSCONFIG reset \
  -n $NODE_IP \
  -e $NODE_IP \
  --system-labels-to-wipe STATE \
  --system-labels-to-wipe EPHEMERAL \
  --graceful=true \
  --reboot \
  --wait=false

# Example for obsy on VLAN 208 (initial access):
export TALOSCONFIG=/root/vixens/terraform/environments/dev/talosconfig-dev
talosctl reset \
  -n 192.168.208.162 \
  -e 192.168.208.162 \
  --system-labels-to-wipe STATE \
  --system-labels-to-wipe EPHEMERAL \
  --graceful=true \
  --reboot \
  --wait=false

# Wait for node to reboot (~2-3 minutes)
# Then reapply Terraform
terraform apply
```

## Changelog

| Date | Version | Change |
|------|---------|--------|
| 2025-10-30 | 1.0 | Initial runbook for Phase 1 |
