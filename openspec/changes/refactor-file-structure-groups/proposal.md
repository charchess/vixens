# Refactor Application File Structure with Logical Groups

## Why

The current `apps/` directory structure organizes applications alphabetically without semantic grouping. As the infrastructure grows, this creates:
- **Poor discoverability**: Hard to understand which apps belong to which layer/purpose
- **Unclear dependencies**: Not obvious which apps must be deployed first
- **Maintenance burden**: No clear separation between infrastructure, platform, and application layers

**Current Pain Points:**
- Mixed infrastructure (cert-manager, traefik) with applications (authentik, homeassistant)
- No clear indication of deployment order or dependencies
- Difficult to identify which apps are "core infrastructure" vs "optional services"

**Proposed Grouping Strategy:**
Organize applications into logical stacks following the **Infrastructure → Platform → Applications** pattern:

1. **infrastructure/** - Core K8s infrastructure (must deploy first)
   - cilium-lb, cert-manager, cert-manager-webhook-gandi, traefik

2. **storage/** - Persistent storage providers
   - synology-csi

3. **platform/** - Platform services enabling applications
   - authentik (SSO/Auth), monitoring (Prometheus/Grafana)

4. **applications/** - End-user applications
   - homeassistant, media-stack, etc.

## What Changes

### Directory Restructure

**Before:**
```
apps/
├── argocd/
├── cert-manager/
├── cert-manager-webhook-gandi/
├── cilium-lb/
├── synology-csi/
├── traefik/
├── authentik/           (future)
└── homeassistant/       (future)
```

**After:**
```
apps/
├── infrastructure/
│   ├── cilium-lb/
│   ├── cert-manager/
│   ├── cert-manager-webhook-gandi/
│   └── traefik/
├── storage/
│   └── synology-csi/
├── platform/
│   ├── authentik/
│   └── monitoring/      (future)
└── applications/
    ├── homeassistant/
    └── media-stack/     (future)
```

**Note:** `argocd/` directory stays at root level as it's self-management, not a deployed app.

### ArgoCD Path Updates

All ArgoCD Application manifests must update `spec.source.path` to reflect new structure:

**Example for traefik (dev environment):**
```yaml
# Before
spec:
  source:
    path: apps/traefik/overlays/dev

# After
spec:
  source:
    path: apps/infrastructure/traefik/overlays/dev
```

This applies to all environments (dev, test, staging, prod).

### Sync Wave Adjustment

Group-based sync waves ensure correct deployment order:

- **Wave -1**: Storage providers (synology-csi secrets)
- **Wave 0**: Infrastructure secrets (cert-manager, traefik)
- **Wave 1**: Infrastructure applications (cert-manager, traefik)
- **Wave 2**: Infrastructure configuration (ClusterIssuers, IngressRoutes)
- **Wave 3**: Platform services (authentik, monitoring)
- **Wave 4**: Applications (homeassistant, media-stack)

### Documentation Updates

- Update CLAUDE.md repository structure diagram
- Update README.md with new directory layout
- Create `apps/infrastructure/README.md` explaining group purpose
- Create `apps/storage/README.md` explaining group purpose
- Create `apps/platform/README.md` explaining group purpose
- Create `apps/applications/README.md` explaining group purpose

## Impact

**Developer Experience:**
- ✅ **Discoverability**: Immediately understand application purpose from location
- ✅ **Clarity**: Clear separation of concerns (infra vs platform vs apps)
- ✅ **Onboarding**: New team members understand structure faster

**Operations:**
- ✅ **Deployment Order**: Groups naturally encode deployment dependencies
- ✅ **Troubleshooting**: Easier to isolate issues to specific layer
- ✅ **Scaling**: Clear where to add new applications

**Maintenance:**
- ✅ **Refactoring**: Can upgrade entire groups (e.g., all infrastructure)
- ✅ **Testing**: Can test groups in isolation
- ✅ **Documentation**: Each group can have its own README

**Risk:**
- ⚠️ Large Git change with many file moves (use `git mv` for history preservation)
- ⚠️ Requires updating ALL ArgoCD Application manifests across all environments
- ⚠️ Potential for missed path references in documentation
- Mitigation: Use search/replace, validate with `openspec validate`, test in dev first

## Non-Goals

- Not changing application configurations (only directory structure)
- Not modifying Kustomize bases (only moving directories)
- Not creating new applications (just organizing existing ones)
- Not changing ArgoCD App-of-Apps pattern (only updating paths)
