# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Vixens is a multi-cluster Kubernetes homelab infrastructure following GitOps best practices. The project is built in phases: **Phase 1** provisions infrastructure with Terraform, **Phase 2** deploys services via ArgoCD, and **Phase 3** runs applications.

**Core Stack:**
- **OS**: Talos Linux v1.11.3 (immutable, API-driven)
- **Kubernetes**: v1.30.0
- **Infrastructure**: Terraform + Talos provider
- **GitOps**: ArgoCD (App-of-Apps pattern) - Phase 2
- **CNI**: Cilium v1.16.5 (eBPF, kube-proxy replacement, Hubble observability)
- **LoadBalancer**: MetalLB (Layer 2) - Phase 2
- **Ingress**: Traefik v3.x - Phase 2
- **Storage**: Synology CSI (iSCSI) - Phase 2

## Current Phase: Phase 1 (Infrastructure as Code)

**Status**: Sprint 1 - Terraform module Talos created and validated ✅

The project is iterative with a **destroy/recreate** strategy for dev/test environments to ensure reproducibility.

## Architecture

### Multi-Cluster Strategy

The infrastructure consists of 4 independent clusters, each on a dedicated VLAN:

| Environment | Nodes                     | VLAN Internal | VLAN Services | VIP              | Status |
|-------------|---------------------------|---------------|---------------|------------------|--------|
| **Dev**     | obsy (cp), onyx (worker)  | 111           | 208           | 192.168.111.160  | ✅ Active |
| **Test**    | carny, celesty, citrine   | 111           | 209           | 192.168.111.180  | ⏳ Sprint 9 |
| **Staging** | TBD (3 nodes)             | 111           | 210           | 192.168.111.190  | 📅 Future |
| **Prod**    | Physical nodes (3)        | 111           | 201           | 192.168.111.200  | 📅 Phase 3 |

### Dual-VLAN Network Architecture

Each node has **two VLANs** configured on a single physical interface:

- **VLAN 111** (192.168.111.0/24) - **Non-routed, internal**
  - Inter-node communication (etcd, kubelet, CNI)
  - Storage access (Synology NAS: 192.168.111.69)
  - Kubernetes API VIP
  - Management host: grenat (192.168.111.64)

- **VLAN 20X** (192.168.20X.0/24) - **Routed, services**
  - External service exposure (Ingress, LoadBalancer)
  - MetalLB IP pools: .70-.79 (assigned), .80-.89 (auto)
  - Internet gateway configured on this VLAN

### Repository Structure

```
vixens/
├── terraform/                      # Phase 1: Infrastructure as Code
│   ├── modules/talos/             # Reusable Talos cluster module ✅
│   │   ├── main.tf                # Resources + per-node patches
│   │   ├── variables.tf           # Per-node config (disk, network, etc.)
│   │   ├── outputs.tf             # kubeconfig, talosconfig
│   │   ├── providers.tf           # Provider documentation
│   │   └── versions.tf            # Terraform >= 1.5.0, Talos ~> 0.9
│   └── environments/
│       ├── dev/                   # Dev cluster (obsy + onyx) ✅
│       │   ├── main.tf            # Module call with node configs
│       │   ├── versions.tf        # Provider versions
│       │   ├── provider.tf        # Provider config
│       │   ├── kubeconfig-dev     # Generated (gitignored)
│       │   └── talosconfig-dev    # Generated (gitignored)
│       ├── test/                  # Test cluster ⏳ Sprint 9
│       ├── staging/               # Staging cluster 📅 Future
│       └── prod/                  # Prod cluster 📅 Future
│
├── argocd/                        # Phase 2: ArgoCD self-management
│   ├── base/
│   │   ├── argocd-install.yaml   # ArgoCD Helm Application
│   │   └── root-app.yaml         # App-of-Apps root
│   └── overlays/
│       ├── dev/
│       ├── test/
│       ├── staging/
│       └── prod/
│
├── apps/                          # Phase 2: Infrastructure apps
│   ├── metallb/
│   │   ├── base/
│   │   │   ├── helm-release.yaml
│   │   │   └── ipaddresspool.yaml
│   │   └── overlays/
│   │       ├── dev/               # VLAN 208 pools
│   │       ├── test/              # VLAN 209 pools
│   │       └── prod/              # VLAN 201 pools
│   ├── traefik/
│   ├── cert-manager/
│   ├── synology-csi/
│   ├── authelia/
│   └── monitoring/
│
├── docs/                          # Documentation
│   ├── architecture/
│   │   ├── network-diagram.md
│   │   ├── storage-strategy.md
│   │   └── gitops-workflow.md
│   ├── adr/                       # Architecture Decision Records
│   │   ├── 001-talos-linux.md
│   │   ├── 002-argocd-gitops.md
│   │   ├── 003-vlan-segmentation.md
│   │   └── 004-cilium-cni.md
│   └── ROADMAP.md                 # Sprint-based roadmap
│
├── .github/workflows/
│   ├── validate-terraform.yaml
│   ├── validate-kustomize.yaml
│   └── promote.yaml
│
├── CLAUDE.md                      # This file
└── README.md                      # Quick start guide
```

## Development Commands

### Terraform (Phase 1)

```bash
# Working directory for dev environment
cd /root/vixens/terraform/environments/dev

# Format all Terraform files
terraform fmt -recursive

# Initialize Terraform
terraform init

# Validate configuration
terraform validate

# Plan infrastructure changes
terraform plan

# Apply infrastructure changes (creates cluster)
terraform apply

# Check state
terraform show

# DESTROY infrastructure (see lifecycle section below)
terraform destroy
```

### Terraform Destroy/Recreate Lifecycle

The project follows an **iterative destroy/recreate strategy** for dev/test environments:

**When to destroy:**
- After completing a major sprint to validate reproducibility
- Before significant refactoring
- To test clean slate provisioning
- NOT during normal development (just apply changes)

**Destroy/Recreate workflow:**
```bash
cd terraform/environments/dev

# 1. Destroy cluster
terraform destroy -auto-approve

# 2. Recreate from scratch
terraform apply -auto-approve

# 3. Validate cluster is functional
talosctl --talosconfig=talosconfig-dev --nodes 192.168.111.162 --endpoints 192.168.111.162 version
kubectl --kubeconfig=kubeconfig-dev get nodes
```

**Important notes:**
- Destroy is **safe** for dev/test (virtualized nodes)
- Destroy is **dangerous** for prod (physical infrastructure)
- Always commit Terraform code before destroying
- Validation: `terraform plan` should show "no changes" after recreate

### Talos Node Management

```bash
# Set config files as env vars (recommended)
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
export TALOSCONFIG=/root/vixens/terraform/environments/dev/talosconfig-dev

# Check Talos version
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 version

# Check node health
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 health

# Check services status
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 services

# Check etcd members (cluster)
talosctl --nodes 192.168.111.160 etcd members

# Check network configuration
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 get addresses
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 get links

# View machine configuration
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 get machineconfig -o yaml

# Reboot node
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 reboot
```

### Kubernetes (via kubectl)

```bash
# Check nodes
kubectl get nodes -o wide

# Check system pods
kubectl get pods -n kube-system

# Check all resources
kubectl get all -A

# View node details
kubectl describe node obsy

# Access logs
kubectl logs -n kube-system <pod-name>
```

### Validation & Testing

```bash
# Terraform validation
cd terraform/environments/dev
terraform validate
terraform fmt -check -recursive

# Check Cilium status (after Phase 2)
cilium status
cilium connectivity test

# Network connectivity tests
ping 192.168.111.162  # VLAN 111 (internal)
ping 192.168.208.162  # VLAN 208 (services)
```

## Terraform Module: talos

Location: `terraform/modules/talos/`

The Talos module provisions Kubernetes control plane clusters on Talos Linux with **per-node configuration**.

### Key Features ✅

- **Per-node configuration**: Each node has its own disk, network, and patches
- **Dual-VLAN support**: Automatic configuration of internal (111) + services (20X) VLANs
- **VIP automatic configuration**: VIP extracted from cluster_endpoint and configured on VLAN 111
- **Hostname configuration**: Automatic hostname setup for all nodes (control plane + workers)
- **certSANs with VIP**: API server certificates automatically include VIP
- **Worker node support**: Support for both control plane and worker nodes
- **Automatic node reset on destroy**: talosctl reset provisioner cleans nodes on destroy
- **Odd control plane validation**: Enforces etcd quorum requirements (1, 3, 5 nodes)
- **Custom image support**: Optional factory/schematic images for extensions (iSCSI, etc.)
- **Automatic bootstrap**: First node is bootstrapped automatically
- **Config file generation**: Creates kubeconfig and talosconfig locally
- **Destroy/recreate safe**: Fully tested and reproducible infrastructure

### Module Interface

**Required Variables:**

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `cluster_name` | string | Cluster name | `"vixens-dev"` |
| `cluster_endpoint` | string | Kubernetes API VIP | `"https://192.168.111.160:6443"` |
| `control_plane_nodes` | map(object) | **Per-node config** (see below) | `{ "obsy" = {...} }` |

**control_plane_nodes object structure:**
```hcl
{
  name         = string           # Node hostname
  ip_address   = string           # IP for Talos API access (initially external VLAN)
  mac_address  = string           # MAC address
  install_disk = string           # Install disk path (e.g., "/dev/sda")
  network = object({
    interface = string            # Physical interface name
    vlans = list(object({
      vlanId    = number          # VLAN ID (111, 208, etc.)
      addresses = list(string)    # IP addresses with CIDR
      gateway   = string          # Gateway (empty for non-routed VLANs)
    }))
  })
}
```

**Optional Variables:**

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `talos_version` | string | `"v1.11.3"` | Talos Linux version |
| `kubernetes_version` | string | `"v1.30.0"` | Kubernetes version |
| `talos_image` | string | `""` | Custom factory image URL for extensions |
| `pod_subnet` | string | `"10.244.0.0/16"` | Pod CIDR |
| `service_subnet` | string | `"10.96.0.0/12"` | Service CIDR |

**Outputs:**

| Output | Description | Sensitive |
|--------|-------------|-----------|
| `kubeconfig` | Kubernetes config file content | Yes |
| `talosconfig` | Talos config file content | Yes |
| `control_plane_configs` | Per-node machine configs | Yes |
| `cluster_endpoint` | Kubernetes API endpoint | No |

### Example Usage

```hcl
module "talos_cluster" {
  source = "../../modules/talos"

  cluster_name     = "vixens-dev"
  talos_version    = "v1.11.3"
  cluster_endpoint = "https://192.168.111.160:6443"

  # Optional: Custom image with iSCSI extension
  # talos_image = "factory.talos.dev/installer/<schematic_id>:v1.11.3"

  control_plane_nodes = {
    "obsy" = {
      name         = "obsy"
      ip_address   = "192.168.0.162"  # Initial maintenance IP for Terraform access
      mac_address  = "00:15:5D:00:CB:10"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb10"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.162/24"]
            gateway   = ""
          },
          {
            vlanId    = 208
            addresses = ["192.168.208.162/24"]
            gateway   = "192.168.208.1"
          }
        ]
      }
    }
  }

  worker_nodes = {
    "onyx" = {
      name         = "onyx"
      ip_address   = "192.168.0.164"  # Initial maintenance IP for Terraform access
      mac_address  = "00:15:5D:00:CB:11"
      install_disk = "/dev/sda"
      network = {
        interface = "enx00155d00cb11"
        vlans = [
          {
            vlanId    = 111
            addresses = ["192.168.111.164/24"]
            gateway   = ""
          },
          {
            vlanId    = 208
            addresses = ["192.168.208.164/24"]
            gateway   = "192.168.208.1"
          }
        ]
      }
    }
  }
}

# Generate config files locally
resource "local_file" "kubeconfig" {
  content         = module.talos_cluster.kubeconfig
  filename        = "${path.module}/kubeconfig-dev"
  file_permission = "0600"
}
```

## Current Infrastructure Status (Sprint 1 COMPLETED ✅)

### Dev Cluster ✅

| Component | Status | Details |
|-----------|--------|---------|
| **Terraform module** | ✅ Done | VIP, hostname, certSANs, destroy/recreate validated |
| **Node obsy** | ✅ Deployed | Control plane - 192.168.111.162 + VIP 192.168.111.160 |
| **Node onyx** | ✅ Deployed | Worker - 192.168.111.164 |
| **Talos** | ✅ Running | v1.11.0, Kubernetes v1.34.0 |
| **VIP HA** | ✅ Active | 192.168.111.160/32 on VLAN 111 |
| **Hostnames** | ✅ Configured | obsy, onyx (automatic) |
| **VLANs** | ✅ Configured | VLAN 111 (internal) + VLAN 208 (services) |
| **Config files** | ✅ Generated | kubeconfig-dev, talosconfig-dev (local) |
| **Destroy/Recreate** | ✅ Validated | Fully reproducible infrastructure |

### Pending (Future Sprints)

| Sprint | Component | Status |
|--------|-----------|--------|
| **1** | **Terraform module + dev 2 nodes** | **✅ DONE** |
| 2 | Cilium CNI | ⏳ Next |
| 3 | Scale to 3 control planes | 📅 Sprint 3 |
| 4 | ArgoCD bootstrap | 📅 Sprint 4 |
| 5-11 | Phase 2 services | 📅 Sprints 5-11 |

## Important Notes

### Phase 1 (Current)
- **Terraform-managed**: All infrastructure is code
- **Immutable**: Talos nodes are disposable and reproducible
- **Destroy/recreate safe**: Dev/test can be destroyed anytime
- **Per-node config**: Each node has its own disk, network, patches
- **Dual-VLAN required**: Internal (111) + Services (20X)

### Phase 2 (Future - GitOps)
- **ArgoCD App-of-Apps**: One root application manages all services
- **Kustomize overlays**: Base + per-environment patches
- **Branch per environment**: dev/test/staging/main branches
- **Auto-sync**: Git push = automatic deployment

### Archon Task Management
- Tasks tracked in Archon MCP server
- Sprint-based workflow (Sprints 1-11)
- Each task has validation criteria
- Mark tasks: todo → doing → review → done

## Next Steps

**Sprint 1** (✅ COMPLETED):
1. ✅ Task 1.1: Module structure - DONE
2. ✅ Task 1.2: Configure dev environment - DONE
3. ✅ Task 1.3: Apply Terraform and provision - DONE
4. ✅ Task 1.4: Validate cluster access - DONE
5. ✅ Bonus: VIP configuration - DONE
6. ✅ Bonus: Hostname configuration - DONE
7. ✅ Bonus: Worker node support - DONE
8. ✅ Bonus: Destroy/recreate validation - DONE

**Immediate** (Sprint 2 - Cilium CNI):
- Task 2.1: Configure Helm provider in Terraform
- Task 2.2: Deploy Cilium via Terraform Helm
- Task 2.3: Apply Terraform and deploy Cilium
- Task 2.4: Validate Cilium operational and network connectivity

**After Sprint 2**:
- Sprint 3: Scale to 3 control planes HA
- Sprint 4: Bootstrap ArgoCD
- Sprints 5-11: Phase 2 services

See `docs/ROADMAP.md` for complete sprint breakdown.
