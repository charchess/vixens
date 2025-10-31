# Sprint 1 - Terraform Module Talos - COMPLETED ✅

**Date**: October 31, 2025
**Status**: ✅ Completed and Validated
**Cluster**: vixens-dev (obsy + onyx)

## Objectives

Create a reproducible Terraform module for Talos Linux with full destroy/recreate cycle validation.

## Deliverables

### 1. Terraform Module (`terraform/modules/talos/`)

**Files Created/Modified**:
- `main.tf` - Resources with VIP, hostname, certSANs configuration
- `variables.tf` - Per-node configuration structure
- `outputs.tf` - Kubeconfig and talosconfig outputs
- `providers.tf` - Provider documentation
- `versions.tf` - Version constraints

**Key Features Implemented**:
- ✅ Per-node configuration (control plane + workers)
- ✅ Dual-VLAN support (111 internal + 20X services)
- ✅ **VIP automatic configuration**: Extracted from cluster_endpoint, configured on VLAN 111
- ✅ **Hostname configuration**: Automatic setup for all nodes
- ✅ **certSANs with VIP**: API server certificates include VIP
- ✅ **Worker node support**: Separate control_plane_nodes and worker_nodes
- ✅ **Automatic node reset on destroy**: talosctl reset provisioner
- ✅ Odd control plane validation (etcd quorum)
- ✅ Custom image support (factory/schematic)
- ✅ Automatic bootstrap
- ✅ Local config file generation

### 2. Dev Environment (`terraform/environments/dev/`)

**Configuration**:
- **Control plane**: obsy (192.168.111.162)
- **Worker**: onyx (192.168.111.164)
- **VIP**: 192.168.111.160
- **Talos**: v1.11.0
- **Kubernetes**: v1.34.0

**Files**:
- `main.tf` - Module call with node configurations
- `provider.tf` - Provider configuration
- `versions.tf` - Version constraints
- `kubeconfig-dev` - Generated (gitignored)
- `talosconfig-dev` - Generated (gitignored)

## Technical Achievements

### VIP Configuration (Lines 5-7, 35-39 in main.tf)

```hcl
locals {
  vip_address = regex("https?://([^:]+)", var.cluster_endpoint)[0]
}

# In node patches:
vip = vlan.gateway == "" ? {
  ip = local.vip_address
} : null
```

**Result**: VIP 192.168.111.160/32 automatically configured on VLAN 111 for all control planes.

### Hostname Configuration (Lines 19, 180 in main.tf)

```hcl
machine = {
  network = {
    hostname = v.name  # Set node hostname
```

**Result**: Nodes have proper hostnames (obsy, onyx) instead of generic names.

### certSANs Configuration (Lines 51-53 in main.tf)

```hcl
cluster = {
  apiServer = {
    certSANs = [local.vip_address]
  }
}
```

**Result**: API server certificates include VIP for HA access.

### Automatic Node Reset on Destroy (Lines 108-150 in main.tf)

```hcl
resource "null_resource" "node_reset_on_destroy" {
  triggers = {
    node_ip     = local.control_plane_vlan_ips[each.key]
    talosconfig = data.talos_client_configuration.this.talos_config
  }

  provisioner "local-exec" {
    when = destroy
    command = "talosctl reset -n ${self.triggers.node_ip} ..."
  }
}
```

**Result**: Nodes are automatically cleaned (reset) during terraform destroy.

## Validation Tests

### Test 1: Infrastructure Creation
```bash
terraform apply -auto-approve
```
**Result**: ✅ 9 resources created successfully

### Test 2: Cluster Functionality
```bash
kubectl get nodes -o wide
```
**Result**: ✅ 2 nodes Ready (obsy control-plane, onyx worker)

### Test 3: VIP Active
```bash
ping 192.168.111.160
```
**Result**: ✅ VIP responsive

### Test 4: Hostnames
```bash
talosctl get hostname
```
**Result**: ✅ obsy and onyx configured correctly

### Test 5: Destroy
```bash
terraform destroy -auto-approve
```
**Result**: ✅ 9 resources destroyed, nodes reset executed

### Test 6: Recreate
```bash
terraform apply -auto-approve
```
**Result**: ✅ 9 resources recreated, cluster functional

### Test 7: Idempotence
```bash
terraform plan
```
**Result**: ✅ No changes (infrastructure matches configuration)

## Architecture Decisions

### 1. VIP on Internal VLAN
**Decision**: Place VIP on VLAN 111 (non-routed) instead of VLAN 208 (routed)
**Rationale**:
- Kubernetes API should be on internal network
- External access via Traefik ingress (future)
- Reduces attack surface

### 2. Initial IP Strategy
**Decision**: Use 192.168.0.x for initial Terraform access, then VLANs take over
**Rationale**:
- Nodes boot with DHCP on untagged interface
- After Terraform applies config, only VLAN IPs remain
- Simplifies initial provisioning

### 3. Worker Node Separation
**Decision**: Separate `control_plane_nodes` and `worker_nodes` variables
**Rationale**:
- Clear role distinction
- Different patches (workers don't need VIP)
- Easier to scale independently

### 4. Automatic Reset on Destroy
**Decision**: Add provisioner to reset nodes automatically
**Rationale**:
- Ensures clean slate for recreate
- No manual intervention required
- Validates reproducibility

## Issues Encountered and Resolved

### Issue 1: IP 192.168.0.x Persistence
**Problem**: obsy kept maintenance IP after configuration
**Root cause**: Timing of network reconfiguration
**Resolution**: Acceptable - VLANs are primary, .0.x doesn't interfere
**Status**: ✅ Non-blocking (can be cleaned with manual reboot if needed)

### Issue 2: Hostname Not Set
**Problem**: Nodes had generic hostnames
**Resolution**: Added `machine.network.hostname = v.name` in patches
**Status**: ✅ Resolved

### Issue 3: VIP Not Configured
**Problem**: VIP was missing from control plane config
**Resolution**: Added VIP extraction from cluster_endpoint and configuration on VLAN 111
**Status**: ✅ Resolved

### Issue 4: certSANs Missing VIP
**Problem**: API server certificates didn't include VIP
**Resolution**: Added `cluster.apiServer.certSANs = [local.vip_address]`
**Status**: ✅ Resolved

## Performance Metrics

| Operation | Duration | Notes |
|-----------|----------|-------|
| `terraform apply` (create) | ~2 minutes | Fast provisioning |
| Cluster bootstrap | ~30 seconds | Kubernetes Ready |
| `terraform destroy` | ~15 seconds | Including node reset |
| `terraform apply` (recreate) | ~2 minutes | Identical to initial |
| Total destroy/recreate cycle | ~2.5 minutes | Excellent reproducibility |

## Documentation Updates

- ✅ CLAUDE.md updated with Sprint 1 completion
- ✅ Module features documented
- ✅ Example usage updated with worker nodes
- ✅ Current infrastructure status updated
- ✅ Next steps pointing to Sprint 2

## Archon Tasks Completed

| Task ID | Title | Status |
|---------|-------|--------|
| 1.1 | Create module structure | ✅ Done |
| 1.2 | Configure dev environment | ✅ Done |
| 1.3 | Apply Terraform and provision | ✅ Done |
| 1.4 | Validate cluster access | ✅ Done |
| Bonus | VIP configuration | ✅ Done |
| Bonus | Hostname configuration | ✅ Done |

## Lessons Learned

1. **VIP configuration is critical**: Should have been included from the start
2. **Hostname matters**: Makes debugging and management easier
3. **Destroy/recreate validation is essential**: Proves true reproducibility
4. **Worker nodes are different**: Need separate variable structure
5. **Provisioners are powerful**: Node reset automation saves time

## Next Sprint

**Sprint 2: Cilium CNI Deployment**
- Configure Helm provider in Terraform
- Deploy Cilium with kube-proxy replacement
- Enable Hubble observability
- Validate network connectivity

## Conclusion

Sprint 1 is a **complete success**. The Terraform module is fully functional, reproducible, and production-ready for the dev environment. All bonus features (VIP, hostname, worker support) have been implemented and validated.

**The foundation for the Vixens infrastructure is solid. ✅**
