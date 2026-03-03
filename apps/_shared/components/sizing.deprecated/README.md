# ⚠️ DEPRECATED: Kustomize Sizing Components

**Status**: DEPRECATED as of 2026-03-03  
**Removal**: Scheduled for 2026-04-03 (30 days)

## What Happened?

These Kustomize components have been **replaced by Kyverno mutation policies** using labels.

## Migration Required

### Old Approach (DEPRECATED - DO NOT USE):
```yaml
# kustomization.yaml
components:
  - ../../../../_shared/components/sizing/medium
```

### New Approach (CURRENT - USE THIS):
```yaml
# kustomization.yaml
metadata:
  labels:
    vixens.io/sizing.<container-name>: medium
```

## Why Deprecated?

1. **Dynamic mutation**: Kyverno policies mutate pods at admission time
2. **Centralized management**: One policy file vs scattered components
3. **Per-container granularity**: Support for multi-container pods (main + sidecars)
4. **Guaranteed QoS support**: G-sizing profiles (G-small, G-medium, G-large, G-xl)
5. **Better defaults**: Fallback to micro instead of unset

## Available Sizing Profiles

### Standard (Burstable QoS) - Bronze to Diamond Tier

| Label | CPU (req/lim) | Memory (req/lim) | Use Case |
|-------|---------------|------------------|----------|
| `micro` | 10m / 100m | 64Mi / 128Mi | Sidecars, exporters |
| `small` | 50m / 500m | 256Mi / 512Mi | Go/Rust optimized apps |
| `medium` | 200m / 1000m | 512Mi / 1Gi | Standard web apps |
| `large` | 1000m / 2000m | 2Gi / 4Gi | Databases, heavy apps |
| `xlarge` | 2000m / 4000m | 4Gi / 8Gi | AI processing, indexing |

### Guaranteed QoS (G-Sizing) - Orichalcum Tier Only

| Label | CPU | Memory | QoS Class |
|-------|-----|--------|-----------|
| `G-small` | 50m / 50m | 128Mi / 128Mi | Guaranteed |
| `G-medium` | 200m / 200m | 512Mi / 512Mi | Guaranteed |
| `G-large` | 1000m / 1000m | 2Gi / 2Gi | Guaranteed |
| `G-xl` | 2000m / 2000m | 4Gi / 4Gi | Guaranteed |

## Migration Examples

### Single Container App

**Before**:
```yaml
# apps/myapp/overlays/prod/kustomization.yaml
components:
  - ../../../../_shared/components/sizing/medium
```

**After**:
```yaml
# apps/myapp/overlays/prod/kustomization.yaml
metadata:
  labels:
    vixens.io/sizing.myapp: medium
```

### Multi-Container App (with Sidecar)

**Before**:
```yaml
components:
  - ../../../../_shared/components/sizing/medium  # Applied to ALL containers
```

**After**:
```yaml
metadata:
  labels:
    vixens.io/sizing.myapp: medium        # Main container
    vixens.io/sizing.litestream: small    # Sidecar gets different sizing!
```

### Orichalcum Tier (Guaranteed QoS)

```yaml
metadata:
  labels:
    vixens.io/sizing.critical-app: G-large  # Guaranteed QoS: 2Gi/2Gi
```

## Migration Guide

See full guide: `docs/guides/sizing-migration.md`

## References

- **ADR-022**: 7-Tier Goldification System
- **Kyverno Policy**: `apps/00-infra/kyverno/base/policies/sizing-mutate.yaml`
- **Documentation**: `docs/reference/guaranteed-qos-sizing.md`
- **Standards**: `docs/reference/RESOURCE_STANDARDS.md`

## Timeline

- **2026-03-03**: Components deprecated, directory renamed
- **2026-03-10**: Validation that zero apps use components
- **2026-03-17**: Warning emails to team
- **2026-03-24**: Final check before removal
- **2026-04-03**: Directory deleted permanently

## Support

Questions? See `docs/guides/sizing-migration.md` or ask in #vixens-platform channel.
