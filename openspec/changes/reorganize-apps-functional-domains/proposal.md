# Reorganize Apps into Functional Domains with Numeric Prefixes

## Why

The current flat `apps/` structure mixes infrastructure, storage, applications, and test utilities without clear organization. As the homelab grows, this creates:

**Problems:**
- **Poor discoverability**: Hard to find related applications
- **Unclear deployment order**: No indication which apps depend on others
- **Mixed concerns**: Critical infrastructure (cert-manager) sits next to test utilities (whoami)
- **Maintenance burden**: Difficult to apply bulk operations to related apps
- **Obsolete code**: Dead code like `synology-csi-talos` lingers without clear deprecation

**Current State (12 apps, flat structure):**
```
apps/
├── argocd
├── cert-manager
├── cert-manager-webhook-gandi
├── cilium-lb
├── homeassistant
├── mail-gateway
├── nfs-storage
├── synology-csi              ← ACTIVE (zebernst fork for Talos)
├── synology-csi-talos        ← OBSOLETE (official image, incompatible)
├── traefik
├── traefik-dashboard
└── whoami
```

**Vision:**
Organize applications into **functional domains** with **numeric prefixes** indicating deployment order and criticality. Separate infrastructure from applications, group by purpose, and remove obsolete code.

## What Changes

### Proposed Structure

```
apps/
├── 00-infra/              # Core Kubernetes infrastructure (Wave 1-2)
│   ├── argocd             # GitOps controller
│   ├── cert-manager       # TLS certificate management
│   ├── cert-manager-webhook-gandi
│   ├── cilium-lb          # LoadBalancer (L2 Announcements + LB IPAM)
│   ├── traefik            # Ingress controller
│   └── traefik-dashboard  # Traefik observability UI
│
├── 01-storage/            # Persistent storage providers (Wave 3)
│   ├── synology-csi       # iSCSI CSI driver (Talos-compatible)
│   └── nfs-storage        # NFS PV/PVC for shared storage
│
├── 02-monitoring/         # Observability stack (Wave 4 - Future)
│   ├── prometheus         # Metrics collection
│   ├── grafana            # Metrics visualization
│   ├── loki               # Log aggregation
│   └── alertmanager       # Alert management
│
├── 03-security/           # Security and authentication (Wave 4 - Future)
│   ├── authentik          # SSO/OIDC provider
│   ├── vaultwarden        # Password manager
│   └── oauth2-proxy       # OAuth2 authentication proxy
│
├── 10-home/               # Home automation domain (Wave 5)
│   └── homeassistant      # Smart home automation
│
├── 20-media/              # Media management domain (Wave 5 - Future)
│   ├── jellyfin           # Media server
│   ├── sonarr             # TV show management
│   ├── radarr             # Movie management
│   ├── prowlarr           # Indexer manager
│   └── transmission       # BitTorrent client
│
├── 30-productivity/       # Productivity domain (Wave 6 - Future)
│   ├── nextcloud          # File sync and collaboration
│   ├── paperless-ngx      # Document management
│   └── bookstack          # Wiki/documentation
│
├── 40-network/            # Network services domain (Wave 6 - Future)
│   ├── mail-gateway       # Mail relay/gateway
│   ├── pihole             # DNS ad-blocking
│   └── wireguard          # VPN server
│
├── 50-backup/             # Backup and disaster recovery (Wave 7 - Future)
│   ├── velero             # Kubernetes backup
│   └── restic             # Application data backup
│
└── 99-test/               # Test and debug utilities (Wave 99)
    └── whoami             # Ingress/TLS test application
```

### Numeric Prefix Conventions

| Prefix | Category | Criticality | Deployment Order |
|--------|----------|-------------|------------------|
| **00-** | Infrastructure | Critical | Wave 1-2 (must deploy first) |
| **01-** | Storage | High | Wave 3 (depends on 00-infra) |
| **02-** | Monitoring | High | Wave 4 (observes everything) |
| **03-** | Security | High | Wave 4 (enables auth for apps) |
| **10-19** | Home Domain | Medium | Wave 5+ (applications) |
| **20-29** | Media Domain | Medium | Wave 5+ (applications) |
| **30-39** | Productivity Domain | Medium | Wave 6+ (applications) |
| **40-49** | Network Domain | Medium | Wave 6+ (applications) |
| **50-59** | Backup Domain | Medium | Wave 7+ (operational) |
| **99-** | Test/Debug | Low | Wave 99 (can break anytime) |

### Key Principles

1. **Numeric prefixes indicate criticality**: Lower numbers = higher priority
2. **Functional domains group related apps**: home, media, productivity, network
3. **Clear deployment order**: Can deploy in prefix order (00 → 01 → 02 → ...)
4. **Extensible**: Easy to add 11-home-sensors, 21-media-requests, etc.
5. **ArgoCD compatible**: Apps sync in order, infrastructure before applications

### Cleanup Actions

**Remove obsolete code:**
- ❌ **Delete `apps/synology-csi-talos/`** - Obsolete Synology CSI (official image, incompatible with Talos)
  - Replaced by: `apps/synology-csi/` (zebernst fork, working)
  - Not deployed in ArgoCD
  - Uses outdated sidecars (v3.0.0, v3.3.0, v1.3.0)

**Relocate applications:**
- ✅ Move all existing apps to new structure
- ✅ Update ArgoCD Application manifests with new paths
- ✅ Update documentation references

## Non-Goals

- **Not changing app internal structure**: Only moving directories, no refactoring inside apps
- **Not renaming apps**: App names stay the same (e.g., `homeassistant` not renamed)
- **Not changing namespaces**: Kubernetes namespaces remain unchanged
- **Not reordering ArgoCD waves**: Existing wave annotations stay the same initially

## Testing Strategy

### Phase 1: Validation (Pre-Migration)
1. Document all ArgoCD Application paths
2. Verify no hardcoded paths in scripts or documentation
3. Create migration script to update all references

### Phase 2: Migration (Dev Environment)
1. Create new directory structure in dev branch
2. Move apps to new locations
3. Update ArgoCD Application manifests
4. Test ArgoCD sync (should detect moved apps)
5. Verify all applications remain Healthy after sync

### Phase 3: Validation (Post-Migration)
1. Verify all apps still accessible via Ingress
2. Check ArgoCD Application health status
3. Validate documentation updated
4. Run smoke tests on critical apps (homeassistant, traefik)

### Phase 4: Multi-Environment Rollout
1. Apply to test environment
2. Apply to staging environment
3. Apply to prod environment (after validation)

## Success Criteria

- ✅ All apps organized into functional domains with numeric prefixes
- ✅ Obsolete `synology-csi-talos` removed from repository
- ✅ ArgoCD Applications updated with new paths
- ✅ All apps remain Healthy in ArgoCD after migration
- ✅ Documentation (CLAUDE.md, README.md) updated with new structure
- ✅ No broken links or references to old paths
- ✅ Deployment order clearly indicated by prefixes

## Rollback Plan

If migration causes issues:
1. Revert commit with directory moves
2. Revert ArgoCD Application path updates
3. Force ArgoCD resync
4. All apps should return to previous state (Kustomize paths deterministic)

Git makes rollback trivial - just revert the commit moving directories.
