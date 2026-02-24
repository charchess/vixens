# PodDisruptionBudget Components

Standardized Kustomize components for managing PodDisruptionBudget (PDB) configurations across applications.

## Overview

PodDisruptionBudget ensures application availability during voluntary disruptions (node drains, cluster upgrades). These components provide a clean, reusable way to apply PDB configurations via Kustomize.

## Current Architecture

```
base/
  ├── deployment.yaml          # Your app deployment
  ├── service.yaml             # Service
  └── pdb.yaml                 # PDB with NO minAvailable (or default)

overlays/prod/
  ├── kustomization.yaml
  │   resources:
  │     - ../../base
  │   components:
  │     - ../../../../_shared/components/poddisruptionbudget/0  # ← Component sets minAvailable
  └── ingress.yaml
```

## Available Components

| Component | minAvailable | Use Case | Example Apps |
|-----------|--------------|----------|--------------|
| `0/` | `0` | Test apps, non-critical jobs | whoami, batch jobs |
| `1/` | `1` | Single-replica apps | HomeAssistant, databases |
| `50percent/` | `50%` | HA apps, multi-replica services | Jellyfin, web services |

## Usage Guide

### 1. Add PDB Resource to Base

Create a basic PDB in `base/pdb.yaml` (without or with placeholder minAvailable):

```yaml
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: myapp
spec:
  # minAvailable will be set by the component!
  selector:
    matchLabels:
      app: myapp
```

### 2. Include PDB in Base Kustomization

```yaml
# base/kustomization.yaml
resources:
  - deployment.yaml
  - service.yaml
  - pdb.yaml  # ← Add this
```

### 3. Apply Component in Overlay

```yaml
# overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base
  - ingress.yaml

components:
  - ../../../../_shared/components/default
  - ../../../../_shared/components/poddisruptionbudget/1  # ← Pick your level
```

## Component Selection Guide

### `0/` - Zero Pod Budget
**When to use:**
- Test/experimental applications
- Batch jobs or cronjobs
- Apps where downtime is acceptable
- Single-node dev environments

**Example:**
```yaml
components:
  - ../../../../_shared/components/poddisruptionbudget/0
```

**Result:**
```yaml
spec:
  minAvailable: 0  # All pods can be disrupted
```

### `1/` - Single Pod Budget
**When to use:**
- Single-replica stateful apps
- Applications that must stay partially available
- StatefulSets where one pod must survive

**Example:**
```yaml
components:
  - ../../../../_shared/components/poddisruptionbudget/1
```

**Result:**
```yaml
spec:
  minAvailable: 1  # At least 1 pod must remain
```

### `50percent/` - High Availability Budget
**When to use:**
- Production web services
- Multi-replica stateless apps
- Apps requiring quorum
- Critical infrastructure

**Example:**
```yaml
components:
  - ../../../../_shared/components/poddisruptionbudget/50percent
```

**Result:**
```yaml
spec:
  minAvailable: 50%  # Half of pods must remain
```

## Comparison: Old vs New Approach

### ❌ Old: Patch File
```yaml
# overlays/prod/pdb-patch.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: myapp
spec:
  minAvailable: 1

# overlays/prod/kustomization.yaml
patches:
  - path: pdb-patch.yaml
```

**Problems:**
- Duplication across apps
- Inconsistent values
- No standardization

### ✅ New: Component
```yaml
# overlays/prod/kustomization.yaml
components:
  - ../../../../_shared/components/poddisruptionbudget/1
```

**Benefits:**
- Reusable
- Consistent
- Self-documenting
- Easy to change globally

## Migration Guide

### Step 1: Check Current PDB
```bash
# Find apps with PDB patches
grep -r "minAvailable" apps/*/overlays/*/
```

### Step 2: Move PDB to Base
If PDB is only in overlay, move to base:
```bash
mv apps/myapp/overlays/prod/pdb.yaml apps/myapp/base/
```

### Step 3: Update Base Kustomization
Add PDB to base resources:
```yaml
# base/kustomization.yaml
resources:
  - pdb.yaml  # ← Add
```

### Step 4: Replace Patch with Component
```yaml
# overlays/prod/kustomization.yaml
# ❌ REMOVE:
# patches:
#   - path: pdb-patch.yaml

# ✅ ADD:
components:
  - ../../../../_shared/components/poddisruptionbudget/1
```

### Step 5: Test
```bash
kustomize build apps/myapp/overlays/prod | grep -A5 "PodDisruptionBudget"
```

## Best Practices

1. **Always define PDB in base** - Ensures every environment has PDB
2. **Use components for minAvailable** - Standardizes values
3. **Match to your replicas:**
   - 1 replica → use `0/` or `1/`
   - 2+ replicas → use `50percent/`
4. **Document your choice** - Add comment in kustomization.yaml

## Troubleshooting

### PDB not applied
Check that PDB resource exists in base:
```bash
kubectl get pdb -n myns
```

### Component not working
Verify kustomize build:
```bash
kustomize build apps/myapp/overlays/prod | grep -A10 "kind: PodDisruptionBudget"
```

### Wrong minAvailable
Check which component is applied:
```bash
grep -r "poddisruptionbudget" apps/myapp/overlays/*/
```

## References

- [Kubernetes PDB Documentation](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- [Kustomize Components](https://kubectl.docs.kubernetes.io/guides/config_management/components/)
- [ADR-022: 7-Tier Goldification System](../../adr/022-7-tier-goldification-system.md) - Platinum tier requires PDB

## Related

- `docs/guides/sizing-migration.md` - Resource sizing with Kyverno
- `docs/reference/quality-standards.md` - Quality tier requirements
- `apps/_shared/components/` - Other reusable components
