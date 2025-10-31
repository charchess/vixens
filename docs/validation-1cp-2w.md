# Validation: 1 Control Plane + 2 Workers Configuration

**Date**: 2025-10-31
**Status**: ✅ VALIDATED
**Cluster**: vixens-dev

## Configuration Tested

| Node | Role | Maintenance IP | VLAN 111 IP | VLAN 208 IP | Interface |
|------|------|----------------|-------------|-------------|-----------|
| obsy | control-plane | 192.168.0.162 | 192.168.111.162 | 192.168.208.162 | enx00155d00cb10 |
| onyx | worker | 192.168.0.164 | 192.168.111.164 | 192.168.208.164 | enx00155d00cb11 |
| opale | worker | 192.168.0.163 | 192.168.111.163 | 192.168.208.163 | enx00155d00cb0b |

## Topology
- **Control Planes**: 1 (obsy)
- **Workers**: 2 (onyx, opale)
- **Talos Version**: v1.11.3
- **Kubernetes Version**: v1.34.0
- **Cilium Version**: v1.18.3
- **Network**: Dual-VLAN (111 internal, 208 services)

## Validation Results

### ✅ Cluster Formation
```
talosctl get members
- obsy: controlplane, version 2
- onyx: worker, version 1
- opale: worker, version 1
```

### ✅ Kubernetes Nodes
```
kubectl get nodes
NAME    STATUS   ROLES           AGE   VERSION
obsy    Ready    control-plane   12m   v1.34.0
onyx    Ready    <none>          11m   v1.34.0
opale   Ready    <none>          11m   v1.34.0
```

### ✅ Cilium CNI Deployment
```
kubectl get pods -n kube-system -l k8s-app=cilium
- 3 Cilium agents (1 per node): Running
- 3 Envoy proxies: Running
- 1 Cilium operator: Running
- Hubble relay: Running
- Hubble UI: Running
```

### ✅ Network Configuration
- All nodes have dual-VLAN IPs configured correctly
- VIP 192.168.111.160 assigned to control plane
- Default gateway on VLAN 208 (192.168.208.1)
- Inter-node communication on VLAN 111

### ✅ Core Components
- kube-apiserver: Running
- kube-controller-manager: Running
- kube-scheduler: Running
- coredns: 2 replicas Running

## Key Fixes Applied

### Interface Name Correction
**Issue**: opale node had incorrect network interface configured
- **Wrong**: `enx00155d00cb12` (MAC 00:15:5D:00:CB:12)
- **Correct**: `enx00155d00cb0b` (MAC 00:15:5D:00:CB:0B)

**Fix**: Corrected in `terraform/environments/dev/main.tf`

### IP Addressing Strategy
**Finding**: Nodes initially accessible on maintenance IPs (192.168.0.x), then transition to VLAN IPs after configuration
- Control plane accessible on both maintenance and VLAN IPs during deployment
- Workers transition fully to VLAN IPs after configuration

## Testing Gaps Filled

This test validated the **missing configuration** from the scalability testing matrix:
- ✅ 1 CP + 2 W deployment → **WORKS** (now validated)
- ✅ 2 CP validation → FAILS (as expected - etcd requires odd numbers)
- ✅ 3 CP deployment → WORKS (previously validated)

## Deployment Timeline

1. **terraform apply initiated**: Cluster provisioning started
2. **Configuration applied**: All 3 nodes configured with network settings
3. **Bootstrap**: Control plane bootstrapped successfully
4. **Workers joined**: Both workers joined cluster automatically
5. **Cilium deployed**: CNI installed via Helm (49 seconds)
6. **Nodes Ready**: All nodes Ready after Cilium initialization (~90 seconds)

**Total time**: ~12 minutes from terraform apply to fully operational cluster

## Conclusion

The 1 control plane + 2 workers configuration is **fully functional** and production-ready for dev/test environments. All components deployed successfully, network configuration correct, and Cilium CNI operational.

This configuration is suitable for:
- Development environments requiring worker isolation
- Testing workloads across multiple nodes
- Non-HA scenarios where single control plane is acceptable

**Next Steps**: Scale to 3 control planes for HA (Sprint 3)
