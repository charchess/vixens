# ADR 006: Terraform 2-Level Architecture and DRY Refactoring

**Date**: 2025-11-14
**Status**: Accepted
**Deciders**: Infrastructure Team

## Context

The initial Terraform architecture used a 3-level hierarchy (`environments/ → base/ → modules/`) that introduced unnecessary abstraction and complexity. Key issues included:

- **27 scattered variables** across multiple files, making configuration difficult to track
- **Hardcoded values** in module files (chart versions, capabilities, timeouts)
- **base/ abstraction layer** that added no value (simple pass-through)
- **Duplicated configurations** (tolerations, capabilities, network settings)
- **Poor maintainability** requiring changes in 4+ files for simple updates

## Decision

We migrated to a **2-level architecture** with centralized DRY (Don't Repeat Yourself) configurations:

### Architecture Changes

**Before (3-level):**
```
environments/dev/main.tf → base/main.tf → modules/{talos,cilium,argocd}
```

**After (2-level):**
```
environments/dev/main.tf → modules/{shared,talos,cilium,argocd}
```

### Key Improvements

#### 1. Shared Module (DRY)
Created `terraform/modules/shared/` as single source of truth for:
- **Chart versions**: Cilium 1.18.3, ArgoCD 7.7.7, Traefik 25.0.0, cert-manager v1.14.4
- **Control plane tolerations**: Reusable across Cilium, ArgoCD, Hubble
- **Cilium capabilities**: Validated set of 11 Linux capabilities for Talos
- **Network defaults**: Pod subnet, service subnet
- **Security defaults**: Common security contexts
- **Timeouts**: Helm install (20min), upgrade (15min), API wait (5min)

#### 2. Typed Variable Objects
Reduced 27 scattered variables to **8 typed objects**:

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

#### 3. Removed base/ Abstraction
The `base/` module was eliminated as it only added complexity:
- Environments now call `modules/` directly
- No intermediate pass-through layer
- Clearer dependency graph
- Faster Terraform execution

#### 4. Optimized Module Interfaces
Simplified module inputs by using shared module:

```hcl
# Before: Each module received individual values
module "cilium" {
  chart_version = "1.18.3"
  cilium_agent_capabilities = { add = [...11 capabilities...] }
  control_plane_tolerations = [...repeated config...]
  timeout = 600
}

# After: Modules reference shared config
module "cilium" {
  chart_version = module.shared.chart_versions.cilium
  cilium_agent_capabilities = module.shared.cilium_config.agent_capabilities
  control_plane_tolerations = module.shared.control_plane_tolerations
  timeout = module.shared.timeouts.helm_install
}
```

#### 5. Robust wait_for_k8s_api Script
Enhanced cluster readiness checks with **two-phase validation**:

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

✅ **Reduced complexity**: 3-level → 2-level architecture
✅ **DRY principle**: Single source of truth for all shared configs
✅ **Type safety**: Structured objects with validation
✅ **Maintainability**: Updates in 1-2 files instead of 4+
✅ **Validated robustness**: Destroy/recreate workflow tested and working
✅ **Clear interfaces**: Module contracts are explicit
✅ **Better readability**: Logical grouping of related configs
✅ **Reproducibility**: Fresh cluster deployment validated (~30 min total)

### Negative

⚠️ **Breaking change**: Existing environments require migration
⚠️ **Learning curve**: Developers must understand new structure
⚠️ **Backend credentials**: Still temporarily inline (TODO for complete migration)

### Neutral

ℹ️ **State migration**: Required for existing environments (one-time operation)
ℹ️ **Documentation updates**: All docs reflect new architecture
ℹ️ **Longer initial apply**: 20-30 min (but validated as reliable)

## Implementation

### Migration Path

1. **Create shared module** with all DRY configs
2. **Transform variables** to typed objects in environments
3. **Update module calls** to use shared module references
4. **Remove base/ module** and update environment main.tf
5. **Test destroy/recreate** workflow thoroughly
6. **Update documentation** (CLAUDE.md, ADRs, README.md)

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
- `docs/adr/006-terraform-2-level-architecture.md` (this file)

**Updated:**
- `terraform/environments/dev/main.tf` (2-level module calls, wait script)
- `terraform/environments/dev/variables.tf` (8 typed objects)
- `terraform/environments/dev/terraform.tfvars` (restructured values)
- `terraform/modules/{talos,cilium,argocd}/` (use shared module)
- `CLAUDE.md` (architecture documentation)
- `.gitignore` (added .envrc)

**Removed:**
- `terraform/base/` (entire directory eliminated)

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
