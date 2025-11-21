# ADR 006: Terraform 3-Level Architecture for Enhanced DRY Refactoring

**Date**: 2025-11-14 (Revised: 2025-11-17)
**Status**: Accepted
**Deciders**: Infrastructure Team

## Context

The initial Terraform architecture used a 3-level hierarchy (`environments/ → base/ → modules/`) that introduced unnecessary abstraction and complexity. A refactoring effort aimed to simplify this to a 2-level architecture. However, during implementation, it became apparent that a carefully designed 3-level architecture could offer **enhanced DRY principles and better encapsulation** for common environment-specific logic.

Key issues identified in the *original* 3-level architecture included:

- **27 scattered variables** across multiple files, making configuration difficult to track
- **Hardcoded values** in module files (chart versions, capabilities, timeouts)
- **`base/` abstraction layer** that often acted as a simple pass-through, adding complexity without significant value.
- **Duplicated configurations** (tolerations, capabilities, network settings)
- **Poor maintainability** requiring changes in 4+ files for simple updates

## Decision

We have adopted a **3-level architecture** with a dedicated `modules/environment` layer to centralize and orchestrate common infrastructure components for each environment, thereby enhancing DRY principles.

### Architecture Changes

**Original (3-level):**
```
environments/dev/main.tf → base/main.tf → modules/{talos,cilium,argocd}
```

**Refactored (Current 3-level for Enhanced DRY):**
```
environments/dev/main.tf → modules/environment/ → modules/{shared,talos,cilium,argocd}
```
The `terraform/base/` directory is now considered **deprecated and unused** in favor of `modules/environment`.

### Key Improvements

#### 1. Environment Orchestration Module (`modules/environment`)
A new module, `terraform/modules/environment/`, was introduced to act as a central orchestration layer for each environment. This module encapsulates the common logic for deploying core infrastructure components (Talos cluster, Cilium CNI, ArgoCD GitOps) in a DRY manner.

#### 2. Shared Module (`modules/shared`)
The `terraform/modules/shared/` module continues to serve as a single source of truth for global, reusable configurations such as:
- **Chart versions**: Cilium 1.18.3, ArgoCD 7.7.7, Traefik 25.0.0, cert-manager v1.14.4
- **Control plane tolerations**: Reusable across Cilium, ArgoCD, Hubble
- **Cilium capabilities**: Validated set of 11 Linux capabilities for Talos
- **Network defaults**: Pod subnet, service subnet
- **Security defaults**: Common security contexts
- **Timeouts**: Helm install (20min), upgrade (15min), API wait (5min)

#### 3. Typed Variable Objects
The refactoring successfully reduced 27 scattered variables to **8 typed objects** within each environment's `variables.tf` (e.g., `environments/dev/variables.tf`).

```hcl
# Before: 27+ individual variables
variable "cluster_name" { ... }
variable "cluster_endpoint" { ... }
variable "talos_version" { ... }
# ... 24+ more variables

# After: 8 typed objects
variable "cluster" {
  type = object({
    name               = string
    endpoint           = string
    talos_version      = string
    talos_image        = string
    kubernetes_version = string
  })
}

variable "control_plane_nodes" { ... }
variable "worker_nodes" { ... }
variable "paths" { ... }
variable "argocd" { ... }
variable "environment" { ... }
variable "git_branch" { ... }
```

Benefits:
- **Logical grouping**: Related configs together
- **Type safety**: Terraform validation at plan time
- **Clear contracts**: Explicit module interfaces
- **Easy discovery**: All cluster settings in one object

#### 4. Optimized Module Interfaces
Module inputs were simplified by leveraging both the `shared` module and the new `environment` orchestration module.

```hcl
# Example: Cilium module input referencing shared config
module "cilium" {
  chart_version = module.shared.chart_versions.cilium
  cilium_agent_capabilities = module.shared.cilium_config.agent_capabilities
  control_plane_tolerations = module.shared.control_plane_tolerations
  timeout = module.shared.timeouts.helm_install
}
```

#### 5. Robust wait_for_k8s_api Script
Enhanced cluster readiness checks with **two-phase validation** are now integrated within the `modules/environment` layer:

**Phase 1: API Server Response**
- 90s initial delay for Talos bootstrap
- 60 attempts × 10s (10 min timeout)
- Checks `/healthz` endpoint

**Phase 2: Control Plane Readiness**
- Validates kube-apiserver, kube-controller-manager, kube-scheduler pods
- Requires **3 consecutive successful checks** (stability verification)
- 120 attempts × 10s (20 min timeout)
- **Note**: etcd runs as Talos system service (not K8s pod), so not checked

**Key Learnings:**
- Static pods on Talos take 8-9 minutes to start on fresh cluster
- Single successful check is insufficient (API can be intermittently available)
- Control plane pods use `hostNetwork: true` (don't need Cilium to start)
- Bash brace expansion `{1..40}` doesn't work in Terraform heredoc

#### 6. Validated Configuration Values

Through destroy/recreate testing, we validated optimal values:

**Cilium Capabilities** (11 required for Talos):
```hcl
add = [
  "CHOWN", "KILL", "NET_ADMIN", "NET_RAW",
  "IPC_LOCK", "SYS_ADMIN", "SYS_RESOURCE",
  "DAC_OVERRIDE", "FOWNER", "SETGID", "SETUID"
]
```
- Reducing to 5 capabilities caused pod crashes
- All 11 capabilities validated as necessary

**Timeouts**:
- Helm install: **1200s (20 min)** - Fresh cluster Cilium installation can take 15-17 min
- Helm upgrade: **900s (15 min)** - Sufficient for upgrades
- API wait: Embedded in wait script (10 min Phase 1 + 20 min Phase 2)

#### 7. Backend Configuration
S3 backend credentials handling:
- Created `.envrc` with `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
- Added `.envrc` to `.gitignore`
- Temporarily kept inline credentials in `backend.tf` for stability testing
- Future: Complete migration to environment variables (TFREF-006 TODO)

## Consequences

### Positive

✅ **Enhanced DRY principle**: Centralization of common environment logic in `modules/environment` further reduces repetition.
✅ **Clearer Orchestration**: The `modules/environment` provides a single, clear entry point for defining an environment's core infrastructure.
✅ **Reduced complexity (per environment)**: Each `environments/{env}/main.tf` becomes simpler, primarily calling the `modules/environment`.
✅ **Type safety**: Structured objects with validation.
✅ **Maintainability**: Updates to common environment logic are centralized.
✅ **Validated robustness**: Destroy/recreate workflow tested and working
✅ **Clear interfaces**: Module contracts are explicit.
✅ **Better readability**: Logical grouping of related configs.
✅ **Reproducibility**: Fresh cluster deployment validated (~30 min total).

### Negative

⚠️ **Increased abstraction layer**: Adds another layer of indirection compared to a pure 2-level architecture.
⚠️ **Learning curve**: Developers must understand the role of the new `modules/environment` layer.
⚠️ **Backend credentials**: Still temporarily inline (TODO for complete migration).

### Neutral

ℹ️ **State migration**: Required for existing environments (one-time operation).
ℹ️ **Documentation updates**: All docs reflect new architecture.
ℹ️ **Longer initial apply**: 20-30 min (but validated as reliable).

## Implementation

### Migration Path

1. **Create `modules/shared`** with all DRY configs.
2. **Transform variables** to typed objects in environments.
3. **Create `modules/environment`** to orchestrate core infrastructure components.
4. **Update `environments/{env}/main.tf`** to call `modules/environment`.
5. **Deprecate `terraform/base/`** (no longer actively used).
6. **Test destroy/recreate** workflow thoroughly.
7. **Update documentation** (CLAUDE.md, ADRs, README.md).

### Validation

Completed 5+ destroy/recreate cycles in dev environment:
- ✅ Cluster provisions in ~30 minutes
- ✅ All 3 control plane nodes operational
- ✅ Cilium CNI deploys successfully
- ✅ ArgoCD bootstraps automatically
- ✅ All services become healthy
- ✅ No manual intervention required

### Files Modified

**Created:**
- `terraform/modules/shared/` (locals.tf, outputs.tf, variables.tf, versions.tf)
- `terraform/modules/environment/` (main.tf, argocd.tf, cilium.tf, outputs.tf, providers.tf, talos.tf, variables.tf)
- `docs/adr/006-terraform-3-level-architecture.md` (this file)

**Updated:**
- `terraform/environments/dev/main.tf` (now calls `modules/environment`)
- `terraform/environments/dev/variables.tf` (8 typed objects)
- `terraform/environments/dev/terraform.tfvars` (restructured values)
- `terraform/modules/{talos,cilium,argocd}/` (now called by `modules/environment`)
- `CLAUDE.md` (architecture documentation)
- `.gitignore` (added .envrc)

**Deprecated:**
- `terraform/base/` (entire directory is no longer actively used)

## Related

- ADR 001: Talos Linux (establishes immutable infrastructure foundation)
- ADR 002: ArgoCD GitOps (automated application deployment)
- ADR 004: Cilium CNI (network plugin with specific capability requirements)
- Sprint 1-6 completion (validated end-to-end workflow)

## References

- Terraform Best Practices: https://www.terraform-best-practices.com/
- Talos Linux: https://www.talos.dev/
- DRY Principle: https://en.wikipedia.org/wiki/Don%27t_repeat_yourself
- Internal: Terraform refactoring task tracking (TFREF-000 through TFREF-008)
