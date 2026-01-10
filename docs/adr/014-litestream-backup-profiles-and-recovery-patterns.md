# ADR-014: Litestream Backup Profiles and Recovery Patterns

**Status:** Accepted  
**Date:** 2026-01-10  
**Authors:** Coding Agent  
**Tags:** backup, disaster-recovery, litestream, sqlite, architecture  

---

## Context

Vixens infrastructure uses Litestream for SQLite database replication across 15+ applications (Vaultwarden, Hydrus-client, Sonarr, Radarr, Lidarr, Prowlarr, Whisparr, Mylar, Sabnzbd, Frigate, HomeAssistant, Adguard-home, Authentik, and template-app). Current implementations vary significantly in backup frequency, recovery strategies, and integrity checking approaches.

**Current Pain Points:**
- No standardized backup configuration (all apps use default settings)
- Inconsistent recovery patterns (Vaultwarden: simple restore, Hydrus: integrity checking + multi-DB restore)
- No differentiation based on data volatility
- Missing standardized "Fail-Safe Integrity" init-container pattern
- Cache databases (e.g., Hydrus caches) backed up unnecessarily

**Observed Implementations:**

**Simple Pattern (Vaultwarden):**
```yaml
initContainers:
  - name: restore-db
    args: [restore, -config, /etc/litestream.yml, -if-db-not-exists, -if-replica-exists, /data/db.sqlite3]
```

**Sophisticated Pattern (Hydrus):**
```yaml
initContainers:
  - name: check-integrity  # PRAGMA integrity_check
  - name: restore-db       # Per-database restore
  - name: restore-mappings
  - name: restore-master
  - name: restore-caches   # Skip restore, cleanup only
```

---

## Decision

We define **two Litestream backup profiles** based on data volatility and recovery requirements: **Light** and **Heavy**.

### Profile Matrix

| Profile  | Snapshot Interval | Sync Interval | Retention | Use Case | Integrity Check |
|----------|------------------|---------------|-----------|----------|-----------------|
| **Light** | 24h | 10s | 72h (3 days) | Low-volatility: configs, metadata | Optional |
| **Heavy** | 6h | 1s | 168h (7 days) | High-volatility: media metadata, logs | Mandatory |

### Light Profile

**Target Applications:** Vaultwarden, Adguard-home, Authentik

**Characteristics:**
- Data changes infrequently (configs, user accounts)
- Recovery within 24h acceptable
- Low storage cost priority

**Configuration:**
```yaml
litestream.yml: |
  dbs:
    - path: /data/db.sqlite3
      replicas:
        - url: s3://$LITESTREAM_BUCKET/app-name/db.sqlite3
          endpoint: $LITESTREAM_ENDPOINT
          sync-interval: 10s
          snapshot-interval: 24h
          retention: 72h
```

### Heavy Profile

**Target Applications:** Hydrus-client, Sonarr, Radarr, Lidarr, Prowlarr, Whisparr, Mylar, Sabnzbd, Frigate, HomeAssistant

**Characteristics:**
- Data changes frequently (media libraries, downloads, recordings)
- Point-in-time recovery critical
- Higher storage cost acceptable

**Configuration:**
```yaml
litestream.yml: |
  dbs:
    - path: /opt/app/database.db
      replicas:
        - url: s3://$LITESTREAM_BUCKET/app-name/database.db
          endpoint: $LITESTREAM_ENDPOINT
          sync-interval: 1s
          snapshot-interval: 6h
          retention: 168h  # 7 days
```

**Special Case: Cache Databases**
- **Do NOT backup** ephemeral cache DBs (e.g., `hydrus client.caches.db`)
- **Do cleanup** in init-container: `rm -f /path/*.caches.db*`

---

## Standardized Recovery Pattern: "Fail-Safe Integrity"

All SQLite apps MUST implement the following init-container sequence:

### 1. Fix Permissions (if needed)
```yaml
- name: fix-permissions
  image: busybox:latest
  command: [sh, -c, "chown -R 1000:1000 /data"]
  volumeMounts:
    - name: data
      mountPath: /data
```

### 2. Integrity Check (Heavy profile MANDATORY, Light optional)
```yaml
- name: check-integrity
  image: keinos/sqlite3:latest
  securityContext:
    runAsUser: 1000
  command: ["/bin/sh", "-c"]
  args:
    - |
      for db in /data/*.db; do
        [ -e "$db" ] || continue
        echo "🔍 Checking $db..."
        if ! sqlite3 "$db" "PRAGMA integrity_check;" | grep -q "ok"; then
          echo "❌ Corruption detected in $db! Deleting for restore."
          rm -f "$db" "$db-wal" "$db-shm"
        else
          echo "✅ $db is healthy."
        fi
      done
  volumeMounts:
    - name: data
      mountPath: /data
```

### 3. Restore from Replica
```yaml
- name: restore-db
  image: litestream/litestream:0.3.13
  securityContext:
    runAsUser: 1000
  command: ["/bin/sh", "-c"]
  args:
    - |
      rm -f /data/*.tmp* /data/*-wal
      DB_PATH=/data/db.sqlite3
      if [ -f "$DB_PATH" ]; then
        echo "✅ Local healthy DB found. Skipping restore."
        exit 0
      fi
      echo "⚠️ DB missing or corrupted. Restoring from S3..."
      litestream restore -config /etc/litestream.yml -if-db-not-exists -if-replica-exists "$DB_PATH"
  envFrom:
    - secretRef:
        name: litestream-shared-secrets
  volumeMounts:
    - name: data
      mountPath: /data
    - name: litestream-config
      mountPath: /etc/litestream.yml
      subPath: litestream.yml
```

### 4. Sidecar Container
```yaml
containers:
  - name: litestream
    image: litestream/litestream:0.3.13
    args: [replicate, -config, /etc/litestream.yml]
    resources:
      requests:
        cpu: 10m
        memory: 64Mi
      limits:
        cpu: 100m
        memory: 128Mi
    envFrom:
      - secretRef:
          name: litestream-shared-secrets
    volumeMounts:
      - name: data
        mountPath: /data
      - name: litestream-config
        mountPath: /etc/litestream.yml
        subPath: litestream.yml
```

---

## Multi-Database Applications

For apps with multiple databases (e.g., Hydrus: client.db, client.mappings.db, client.master.db, client.caches.db):

1. **Separate restore init-containers** per database
2. **Skip caches** - only cleanup: `rm -f /data/*.caches.db*`
3. **Shared litestream.yml** with all non-cache DBs listed

**Example (Hydrus):**
```yaml
initContainers:
  - name: check-integrity       # Check all DBs
  - name: restore-client        # Main DB
  - name: restore-mappings      # Mappings DB
  - name: restore-master        # Master DB
  - name: cleanup-caches        # Skip restore, rm only
```

---

## Migration Strategy

### Phase 1: Documentation (This ADR)
- ✅ Define Light/Heavy profiles
- ✅ Define standardized init-container pattern

### Phase 2: Create Shared Template
Create `apps/_shared/litestream/` with:
- `light-profile-config.yaml` - ConfigMap template
- `heavy-profile-config.yaml` - ConfigMap template
- `integrity-check-init.yaml` - Reusable init-container
- `restore-init.yaml` - Reusable init-container
- `sidecar-container.yaml` - Reusable sidecar

### Phase 3: Migrate Existing Apps
1. **Light Profile Apps** (3 apps):
   - Vaultwarden
   - Adguard-home
   - Authentik

2. **Heavy Profile Apps** (12+ apps):
   - Media stack (Sonarr, Radarr, Lidarr, Prowlarr, Whisparr, Mylar, Sabnzbd)
   - Hydrus-client (special: multi-DB + integrity check)
   - Frigate
   - HomeAssistant

### Phase 4: Update Template
Update `apps/template-app/` to use standardized pattern with profile selection.

---

## Consequences

### Positive
✅ **Standardization** - All SQLite apps follow same recovery pattern  
✅ **Data Protection** - Integrity checks prevent silent corruption  
✅ **Cost Optimization** - Light profile reduces storage costs for low-volatility data  
✅ **Recovery Confidence** - Fail-safe pattern guarantees restoration on corruption  
✅ **DRY Compliance** - Shared templates in `apps/_shared/litestream/`  

### Negative
⚠️ **Storage Costs** - Heavy profile with 7-day retention increases S3 costs  
⚠️ **Startup Time** - Integrity checks add 10-30s to pod startup (Heavy profile)  
⚠️ **Migration Effort** - 15+ apps to update manually  

### Mitigation
- **Costs:** Monitor S3 usage, adjust retention based on actual needs
- **Startup:** Acceptable trade-off for data integrity
- **Migration:** Gradual rollout, prioritize critical apps first (Vaultwarden, Hydrus)

---

## Implementation Checklist

```bash
[ ] 1. Create apps/_shared/litestream/ directory structure
[ ] 2. Create light-profile-config.yaml template
[ ] 3. Create heavy-profile-config.yaml template
[ ] 4. Create integrity-check-init.yaml snippet
[ ] 5. Create restore-init.yaml snippet
[ ] 6. Create sidecar-container.yaml snippet
[ ] 7. Update template-app to use profiles
[ ] 8. Migrate Vaultwarden (Light)
[ ] 9. Migrate Hydrus-client (Heavy + multi-DB)
[ ] 10. Migrate media stack apps (Heavy)
[ ] 11. Document usage in docs/guides/adding-new-application.md
[ ] 12. Create validation script for Litestream configs
```

---

## References

- **Litestream Documentation:** [context7.com/benbjohnson/litestream](https://context7.com/benbjohnson/litestream/llms.txt)
- **Current Implementation:** `apps/60-services/vaultwarden/` (Light), `apps/20-media/hydrus-client/` (Heavy)
- **Related ADRs:**
  - [ADR-013: Layered Configuration Disaster Recovery](013-layered-configuration-disaster-recovery.md)

---

## Alternatives Considered

### Alternative 1: Single Universal Profile
- **Rejected:** Wastes storage on low-volatility apps, insufficient for high-volatility apps

### Alternative 2: Per-App Custom Configuration
- **Rejected:** Violates DRY, difficult to maintain consistency

### Alternative 3: No Integrity Checking
- **Rejected:** Silent corruption risk too high (experienced in Hydrus production)

---

## Notes

**MinIO Endpoint:** `http://192.168.111.69:9001` (Synology NAS)  
**Bucket:** `vixens-litestream` (configured per environment)  
**Secrets:** `litestream-shared-secrets` (Infisical: `/shared/litestream`)

**Profile Selection Criteria:**
- **Writes per hour < 10** → Light
- **Writes per hour > 100** → Heavy
- **Data loss tolerance > 1 day** → Light
- **Data loss tolerance < 1 hour** → Heavy
- **Cache/ephemeral data** → No backup (cleanup only)
