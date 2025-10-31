# Multi-Environment Configuration Pattern

## Overview

This document describes how Vixens infrastructure supports environment-specific configurations for services like ArgoCD, MetalLB, Traefik, and other components deployed across dev/test/staging/prod clusters.

## Architecture Principles

### 1. Environment Isolation

Each environment is **completely isolated** at the infrastructure level:

| Environment | Directory | VLANs | Cluster | State |
|-------------|-----------|-------|---------|-------|
| **dev** | `terraform/environments/dev/` | 111 (internal) + 208 (services) | vixens-dev | ✅ Active |
| **test** | `terraform/environments/test/` | 111 (internal) + 209 (services) | vixens-test | 📅 Sprint 9 |
| **staging** | `terraform/environments/staging/` | 111 (internal) + 210 (services) | vixens-staging | 📅 Future |
| **prod** | `terraform/environments/prod/` | 111 (internal) + 201 (services) | vixens-prod | 📅 Phase 3 |

### 2. Configuration Pattern

**Each environment directory contains:**

```
environments/<env>/
├── main.tf              # Infrastructure (Talos cluster)
├── cilium.tf            # CNI deployment
├── argocd.tf            # GitOps controller (uses variables)
├── variables.tf         # Variable definitions with validations
├── terraform.tfvars     # Environment-specific VALUES
├── versions.tf          # Terraform/provider versions
└── provider.tf          # Provider configuration
```

**Key principle:**
- `.tf` files define **structure** and use `var.*` references
- `.tfvars` files contain **environment-specific values**
- Same code structure across all environments, different values

## ArgoCD Multi-Environment Configuration

### Variable Architecture

**variables.tf** (same across all environments):
```hcl
variable "argocd_service_type" {
  description = "ArgoCD server service type (ClusterIP, LoadBalancer)"
  type        = string
  default     = "ClusterIP"

  validation {
    condition     = contains(["ClusterIP", "LoadBalancer"], var.argocd_service_type)
    error_message = "Service type must be either ClusterIP or LoadBalancer."
  }
}

variable "argocd_loadbalancer_ip" {
  description = "ArgoCD server LoadBalancer IP (used when service_type is LoadBalancer)"
  type        = string
  default     = "192.168.208.71"
}

variable "argocd_hostname" {
  description = "ArgoCD server hostname for Ingress (Sprint 6+)"
  type        = string
  default     = "argocd.dev.vixens.lab"
}

variable "environment" {
  description = "Environment name (dev, test, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "test", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, test, staging, prod."
  }
}

variable "vlan_services_subnet" {
  description = "VLAN services subnet (208 for dev, 209 for test, etc.)"
  type        = string
  default     = "192.168.208.0/24"
}
```

### Environment-Specific Values

**dev/terraform.tfvars:**
```hcl
environment = "dev"
vlan_services_subnet = "192.168.208.0/24"

argocd_service_type    = "ClusterIP"  # Development: internal access only
argocd_loadbalancer_ip = "192.168.208.71"  # Reserved for future use
argocd_hostname        = "argocd.dev.vixens.lab"
```

**test/terraform.tfvars** (future):
```hcl
environment = "test"
vlan_services_subnet = "192.168.209.0/24"

argocd_service_type    = "LoadBalancer"  # Test: LoadBalancer for team access
argocd_loadbalancer_ip = "192.168.209.71"
argocd_hostname        = "argocd.test.vixens.lab"
```

**staging/terraform.tfvars** (future):
```hcl
environment = "staging"
vlan_services_subnet = "192.168.210.0/24"

argocd_service_type    = "LoadBalancer"  # Staging: pre-production testing
argocd_loadbalancer_ip = "192.168.210.71"
argocd_hostname        = "argocd.staging.vixens.lab"
```

**prod/terraform.tfvars** (future):
```hcl
environment = "prod"
vlan_services_subnet = "192.168.201.0/24"

argocd_service_type    = "LoadBalancer"  # Production: Ingress via Traefik (Sprint 6)
argocd_loadbalancer_ip = "192.168.201.71"  # Temporary LoadBalancer until Ingress ready
argocd_hostname        = "argocd.vixens.lab"  # Production domain
```

### Usage in argocd.tf

**argocd.tf** (identical across all environments):
```hcl
resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = "7.7.7"
  namespace        = "argocd"
  create_namespace = true

  values = [yamlencode({
    server = {
      extraArgs = ["--insecure"]

      service = {
        type = var.argocd_service_type  # 👈 Variable reference
        loadBalancerIP = var.argocd_service_type == "LoadBalancer" ? var.argocd_loadbalancer_ip : null
        annotations = {
          "environment" = var.environment  # 👈 Tags for observability
        }
      }

      # ... tolerations and other config
    }
  })]
}
```

## Configuration Evolution by Sprint

### Current (Sprint 4)
- **dev**: ClusterIP (kubectl port-forward access)
- **test/staging/prod**: Not deployed yet

### Sprint 5 (MetalLB)
- **dev**: Upgrade to LoadBalancer (192.168.208.71)
- **test**: LoadBalancer on VLAN 209 when cluster is created

### Sprint 6 (Traefik Ingress)
- **dev/test/staging**: Ingress via Traefik
- **prod**: Ingress with TLS termination

## Extending Pattern to Other Services

This pattern applies to **all environment-specific services**:

### MetalLB (Sprint 5)
```hcl
# variables.tf
variable "metallb_ip_pool_start" {
  type = string
}

variable "metallb_ip_pool_end" {
  type = string
}

# dev/terraform.tfvars
metallb_ip_pool_start = "192.168.208.70"
metallb_ip_pool_end   = "192.168.208.79"

# test/terraform.tfvars
metallb_ip_pool_start = "192.168.209.70"
metallb_ip_pool_end   = "192.168.209.79"
```

### Traefik (Sprint 6)
```hcl
# variables.tf
variable "traefik_loadbalancer_ip" {
  type = string
}

variable "traefik_hostname" {
  type = string
}

# dev/terraform.tfvars
traefik_loadbalancer_ip = "192.168.208.70"
traefik_hostname        = "*.dev.vixens.lab"

# prod/terraform.tfvars
traefik_loadbalancer_ip = "192.168.201.70"
traefik_hostname        = "*.vixens.lab"
```

### Synology CSI (Sprint 7)
```hcl
# variables.tf
variable "synology_nas_ip" {
  type = string
}

variable "synology_volume_prefix" {
  type = string
}

# All environments share same NAS, but different volumes:
# dev/terraform.tfvars
synology_nas_ip       = "192.168.111.69"
synology_volume_prefix = "vixens-dev"

# prod/terraform.tfvars
synology_nas_ip       = "192.168.111.69"
synology_volume_prefix = "vixens-prod"
```

## Validation Checklist

When adding a new environment-specific configuration:

- [ ] Define variables in `variables.tf` with:
  - Clear description
  - Type constraint
  - Validation rules
  - Sensible default (usually for dev)

- [ ] Use `var.*` references in `.tf` files (never hardcode)

- [ ] Set environment-specific values in `terraform.tfvars`

- [ ] Add conditional logic for environment-specific behavior when needed:
  ```hcl
  loadBalancerIP = var.service_type == "LoadBalancer" ? var.loadbalancer_ip : null
  ```

- [ ] Test with `terraform plan` to verify variables are correctly interpolated

- [ ] Document in this file if introducing a new pattern

## Benefits

1. **Single Source of Truth**: One `.tf` file per component, shared across environments
2. **Type Safety**: Terraform validates types and constraints
3. **Environment Isolation**: No risk of cross-environment configuration leaks
4. **Easy Replication**: Copy `terraform.tfvars` to create new environment
5. **GitOps Ready**: Each environment can be managed by separate ArgoCD instance
6. **Auditability**: Git history shows exactly what changed per environment

## Related Documentation

- [GitOps Workflow](./gitops-workflow.md) - How ArgoCD manages applications
- [Network Diagram](./network-diagram.md) - VLAN segmentation per environment
- [ROADMAP](../../docs/ROADMAP.md) - Sprint-based deployment plan
- ADR 002: ArgoCD GitOps (coming soon)

## Examples

### Adding a New Environment

To create the **test** environment:

1. Copy variable definitions from dev (already done):
   ```bash
   cp environments/dev/variables.tf environments/test/
   cp environments/dev/versions.tf environments/test/
   cp environments/dev/provider.tf environments/test/
   ```

2. Create test-specific `main.tf`, `cilium.tf`, `argocd.tf` (same structure as dev)

3. Create `test/terraform.tfvars` with test-specific values:
   ```hcl
   environment = "test"
   vlan_services_subnet = "192.168.209.0/24"
   argocd_service_type = "LoadBalancer"
   argocd_loadbalancer_ip = "192.168.209.71"
   argocd_hostname = "argocd.test.vixens.lab"
   ```

4. Initialize and deploy:
   ```bash
   cd environments/test
   terraform init
   terraform plan
   terraform apply
   ```

### Changing Configuration for One Environment

To upgrade **dev** ArgoCD from ClusterIP to LoadBalancer:

1. Edit `environments/dev/terraform.tfvars`:
   ```hcl
   argocd_service_type = "LoadBalancer"  # Changed from "ClusterIP"
   ```

2. Apply change:
   ```bash
   cd environments/dev
   terraform plan  # Review changes
   terraform apply
   ```

No code changes needed in `.tf` files - only value change in `.tfvars`.

---

**Last Updated:** 2025-11-01 (Sprint 4 - ArgoCD deployment)
**Status:** ✅ Pattern validated in dev environment
