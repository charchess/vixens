# ArgoCD Environment Overlays

Each environment has its own overlay with environment-specific configuration and applications.

## Directory Structure

```
overlays/
├── dev/
│   ├── apps/              # Application definitions
│   ├── env-config.yaml    # Environment configuration (build-time only)
│   └── kustomization.yaml # Kustomize overlay configuration
├── test/
│   ├── apps/
│   ├── env-config.yaml
│   └── kustomization.yaml
├── staging/
│   ├── apps/
│   ├── env-config.yaml
│   └── kustomization.yaml
└── prod/
    ├── apps/
    ├── env-config.yaml
    └── kustomization.yaml
```

## Environment Configuration

Each environment has an `env-config.yaml` ConfigMap with environment-specific values. This ConfigMap is **NOT deployed** to the cluster - it's only used at build time by Kustomize for variable substitution.

### Environment Variables Reference

| Variable | Description | Dev | Test | Staging | Prod |
|----------|-------------|-----|------|---------|------|
| **Environment Identification** |
| `environment` | Environment name | `dev` | `test` | `staging` | `prod` |
| `git_branch` | Git branch for ArgoCD apps | `dev` | `test` | `staging` | `main` |
| `cluster_name` | Cluster identifier | `vixens-dev` | `vixens-test` | `vixens-staging` | `vixens-prod` |
| **Domain Configuration** |
| `domain_suffix` | Domain prefix | `dev` | `test` | `stg` | `""` |
| `base_domain` | Base domain | `truxonline.com` | `truxonline.com` | `truxonline.com` | `truxonline.com` |
| `full_domain` | Complete domain | `dev.truxonline.com` | `test.truxonline.com` | `stg.truxonline.com` | `truxonline.com` |
| **Network Configuration** |
| `vlan_id` | Services VLAN ID | `208` | `209` | `210` | `201` |
| `vlan_subnet` | Services subnet | `192.168.208.0/24` | `192.168.209.0/24` | `192.168.210.0/24` | `192.168.201.0/24` |
| `vlan_gateway` | Services gateway | `192.168.208.1` | `192.168.209.1` | `192.168.210.1` | `192.168.201.1` |
| `vlan_internal` | Internal VLAN (same for all) | `111` | `111` | `111` | `111` |
| **LoadBalancer IP Pools** |
| `lb_pool_start` | Pool start IP | `192.168.208.70` | `192.168.209.70` | `192.168.210.70` | `192.168.201.70` |
| `lb_pool_end` | Pool end IP | `192.168.208.89` | `192.168.209.89` | `192.168.210.89` | `192.168.201.89` |
| `lb_pool_assigned_start` | Static IPs start | `.70` | `.70` | `.70` | `.70` |
| `lb_pool_assigned_end` | Static IPs end | `.79` | `.79` | `.79` | `.79` |
| `lb_pool_auto_start` | Dynamic IPs start | `.80` | `.80` | `.80` | `.80` |
| `lb_pool_auto_end` | Dynamic IPs end | `.89` | `.89` | `.89` | `.89` |
| **Service IPs** |
| `traefik_lb_ip` | Traefik LoadBalancer IP | `192.168.208.70` | `192.168.209.70` | `192.168.210.70` | `192.168.201.70` |
| `argocd_server_ip` | ArgoCD Server IP | `192.168.208.71` | `192.168.209.71` | `192.168.210.71` | `192.168.201.71` |
| **Cluster Configuration** |
| `cluster_vip` | Kubernetes API VIP | `192.168.111.160` | `192.168.111.180` | `192.168.111.190` | `192.168.111.200` |
| `cluster_endpoint` | API endpoint | `https://192.168.111.160:6443` | `https://192.168.111.180:6443` | `https://192.168.111.190:6443` | `https://192.168.111.200:6443` |
| **Versions** |
| `talos_version` | Talos Linux version | `v1.11.5` | `v1.11.5` | `v1.11.5` | `v1.11.5` |
| `kubernetes_version` | Kubernetes version | `v1.34.0` | `v1.34.0` | `v1.34.0` | `v1.34.0` |
| `cilium_version` | Cilium CNI version | `v1.18.3` | `v1.18.3` | `v1.18.3` | `v1.18.3` |
| `argocd_version` | ArgoCD version | `v7.7.7` | `v7.7.7` | `v7.7.7` | `v7.7.7` |
| **Resource Configuration** |
| `resource_tier` | Resource sizing tier | `development` | `testing` | `staging` | `production` |

## Usage

These values are injected into application manifests using Kustomize replacements (future enhancement in later phases).

Currently, applications in `apps/` directories reference these values manually via patches.

### Example: Using Environment Variables

In future phases, we'll use Kustomize replacements like this:

```yaml
# kustomization.yaml
replacements:
  - source:
      kind: ConfigMap
      name: env-config
      fieldPath: data.environment
    targets:
      - select:
          kind: Application
        fieldPaths:
          - spec.source.targetRevision
```

This would automatically inject the correct `git_branch` into all ArgoCD Applications.

## Network Architecture

### VLAN Segmentation

Each environment uses **dual-VLAN architecture**:

1. **VLAN 111** (Internal) - Non-routed
   - Inter-node communication (etcd, kubelet, CNI)
   - Storage access (Synology NAS: 192.168.111.69)
   - Kubernetes API VIP
   - Management host: grenat (192.168.111.64)

2. **VLAN 20X** (Services) - Routed
   - External service exposure (Ingress, LoadBalancer)
   - Cilium LB IPAM pools: .70-.79 (assigned), .80-.89 (auto)
   - Internet gateway configured on this VLAN

### IP Pool Strategy

Each environment has **20 IPs** in LoadBalancer pool:

- **.70-.79** (10 IPs): Assigned/Static pool for known services
  - `.70`: Traefik Ingress Controller
  - `.71`: ArgoCD Server
  - `.72-.79`: Reserved for future services (cert-manager, monitoring, etc.)

- **.80-.89** (10 IPs): Auto/Dynamic pool for Cilium automatic allocation
  - Used for services without explicit IP annotation
  - Allocated by Cilium LB IPAM automatically

## Environment Lifecycle

| Environment | Purpose | Stability | Destroy/Recreate |
|-------------|---------|-----------|------------------|
| **dev** | Active development, testing | Low | ✅ Safe to destroy anytime |
| **test** | Integration testing, validation | Medium | ✅ Safe to destroy for testing |
| **staging** | Pre-production validation | High | ⚠️ Planned destroy only |
| **prod** | Production services | Critical | ❌ Never destroy |

## Adding a New Environment

1. Copy an existing overlay (e.g., `dev`)
2. Update `env-config.yaml` with new environment values
3. Update VLAN IDs, IP ranges, cluster VIP
4. Update domain suffix
5. Update `kustomization.yaml` resources
6. Test with `kustomize build`

## See Also

- [Application Templates](../base/app-templates/README.md)
- [CLAUDE.md](../../CLAUDE.md) - Complete project documentation
- [Network Diagram](../../docs/architecture/network-diagram.md)
