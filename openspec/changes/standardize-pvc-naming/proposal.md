# Standardize PVC Naming Convention

## Why

**Current State:**
- PVC names vary: `homeassistant-config`, `media-pvc`
- No documented naming convention
- Synology CSI creates LUNs with names like: `k8s-csi-pvc-b5176cdb-a4d8-426c-a733-61654b571789`
- Hard to identify which LUN belongs to which app in Synology DSM

**Problems:**
- ❌ **Poor LUN traceability**: UUID-based LUN names don't indicate application
- ❌ **No naming standard**: Inconsistent PVC naming across applications
- ❌ **Manual mapping required**: Must check Kubernetes to find which LUN is which

**Vision:**
Establish a clear PVC naming convention `<application>-<volume-purpose>` that makes it easy to trace LUNs back to their applications through Kubernetes metadata (namespace/name visible in DSM description field).

**Note:** We CANNOT include `<env>` in PVC names because they are defined in `base/` and shared across all environments via Kustomize overlays.

## What Changes

### Naming Convention

**Format:** `<application>-<volume-purpose>`

**Examples:**
- `homeassistant-config` ✅ (already compliant)
- `jellyfin-config`, `jellyfin-cache`
- `nextcloud-data`, `nextcloud-db`
- `postgresql-data`
- `media-pvc` → `media-shared` (rename for clarity)

### Rules

1. **Lowercase kebab-case**: All lowercase, words separated by hyphens
2. **Application prefix**: Start with application name (matches namespace ideally)
3. **Purpose suffix**: Describe what the volume stores (config, data, cache, media, db)
4. **No environment suffix**: Since PVC is in base/, no -dev/-test/-staging/-prod
5. **Max 63 characters**: Kubernetes name length limit

### Documentation

**CLAUDE.md section:**
```markdown
## PVC Naming Convention

Format: `<application>-<volume-purpose>`

Examples:
- homeassistant-config  # Home Assistant configuration
- jellyfin-config       # Jellyfin server config
- jellyfin-cache        # Jellyfin transcoding cache
- nextcloud-data        # Nextcloud user files
- postgresql-data       # PostgreSQL database
- media-shared          # Shared media library (NFS)

Rules:
- Lowercase kebab-case
- Application name prefix (matches namespace)
- Purpose describes content (config/data/cache/media/db)
- No environment suffix (use namespace for isolation)
- Max 63 characters
```

### Synology LUN Naming

**What we get:**
- LUN name: `k8s-csi-pvc-b5176cdb-a4d8-426c-a733-61654b571789` (driver-generated, unchangeable)
- LUN description: `homeassistant/homeassistant-config` (namespace/pvcname, automatically populated)

**Traceability:**
- In Synology DSM, search LUN description for application name
- Description follows pattern: `<namespace>/<pvc-name>`
- Our PVC naming makes descriptions searchable

## Non-Goals

- **Not renaming existing PVCs**: Risky, requires data migration
- **Not customizing LUN names**: Synology CSI driver doesn't support it
- **Not adding environment to PVC names**: Breaks Kustomize base/overlay pattern
- **Not implementing LUN name templates**: Would require forking driver

## Testing Strategy

### Phase 1: Document Current State
1. List all existing PVCs: `kubectl get pvc -A`
2. Check names against convention
3. Identify non-compliant names (media-pvc)

### Phase 2: Apply Convention to New PVCs
1. All new applications must follow convention
2. Review PRs for PVC naming compliance
3. Add yamllint custom rule (future enhancement)

### Phase 3: Optional Rename (Non-Critical)
1. Rename `media-pvc` → `media-shared` (if time permits)
2. Requires:
   - Create new PVC with new name
   - Copy data (or recreate volume)
   - Update references in deployments
   - Delete old PVC

## Success Criteria

- ✅ Naming convention documented in CLAUDE.md
- ✅ All future PVCs follow `<application>-<volume-purpose>` format
- ✅ Existing PVCs catalogued (compliant vs non-compliant)
- ✅ LUN descriptions in Synology DSM are searchable by application name
- ✅ Convention enforced in PR reviews

## Rollback Plan

Documentation change only - no rollback needed. Existing PVCs remain unchanged.
