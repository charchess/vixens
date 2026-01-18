# ADR-014: Litestream Backup Profiles and Recovery Patterns

**Date:** 2026-01-10
**Status:** Accepted
**Deciders:** Coding Agent
**Tags:** backup, litestream, sqlite

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

We define **four Litestream backup profiles** based on data volatility, recovery requirements, and data criticality. Profiles are applied **per database**, not per application, allowing mixed profiles within a single app (e.g., Hydrus).

### Profile Matrix

| Profile | Snapshot Interval | Sync Interval | Retention | Use Case | Integrity Check |
|---------|------------------|---------------|-----------|----------|-----------------|
| **Critical** | 1h | 1s | 336h (14 days) | Very high activity, data loss <1h unacceptable | Mandatory |
| **Standard** | 6h | 1s | 168h (7 days) | Moderate activity, standard recovery needs | Mandatory |
| **Relaxed** | 24h | 1s | 72h (3 days) | Low activity, configs, recovery within 24h acceptable | Optional |
| **Ephemeral** | - | - | - | Cache/temporary data, skip backup entirely | N/A |

### Critical Profile

**Target Databases:** Hydrus `client.db`, Frigate `frigate.db`, high-traffic media DBs

**Characteristics:**
- Very high write activity (>100 writes/hour)
- Data loss <1 hour unacceptable
- Fastest restore time critical
- Higher storage cost acceptable

**Configuration:**
```yaml
litestream.yml: |
  addr: ":9090"  # Prometheus metrics
  
  dbs:
    - path: /opt/app/critical.db
      replicas:
        - url: s3://$LITESTREAM_BUCKET/app-name/critical.db
          endpoint: $LITESTREAM_ENDPOINT
          sync-interval: 1s
          snapshot-interval: 1h
          retention: 336h  # 14 days
```

### Standard Profile

**Target Databases:** Sonarr, Radarr, Lidarr, Prowlarr, Vaultwarden, most media apps

**Characteristics:**
- Moderate write activity (10-100 writes/hour)
- Standard recovery needs (within 6h acceptable)
- Balance storage cost vs recovery time

**Configuration:**
```yaml
litestream.yml: |
  addr: ":9090"  # Prometheus metrics
  
  dbs:
    - path: /opt/app/database.db
      replicas:
        - url: s3://$LITESTREAM_BUCKET/app-name/database.db
          endpoint: $LITESTREAM_ENDPOINT
          sync-interval: 1s
          snapshot-interval: 6h
          retention: 168h  # 7 days
```

### Relaxed Profile

**Target Databases:** Authentik, Adguard-home, config-only DBs

**Characteristics:**
- Low write activity (<10 writes/hour)
- Recovery within 24h acceptable
- Storage cost optimization priority

**Configuration:**
```yaml
litestream.yml: |
  addr: ":9090"  # Prometheus metrics
  
  dbs:
    - path: /data/config.db
      replicas:
        - url: s3://$LITESTREAM_BUCKET/app-name/config.db
          endpoint: $LITESTREAM_ENDPOINT
          sync-interval: 1s
          snapshot-interval: 24h
          retention: 72h  # 3 days
```

### Ephemeral Profile

**Target Databases:** Hydrus `client.caches.db`, temporary/cache DBs

**Characteristics:**
- Cache/temporary data
- Regenerable or acceptable to lose
- Skip backup entirely to save storage

**Configuration:**
```yaml
# No litestream config - cleanup only in init-container
initContainers:
  - name: cleanup-caches
    command: [sh, -c, "rm -f /data/*.caches.db* /data/*.tmp*"]
```

---

## Retention & Cleanup Strategy (Hybrid Approach)

**Defense in Depth:** Two-layer cleanup to prevent storage explosion.

### Layer 1: Litestream Retention (Primary)

Each profile defines its own retention via `retention` parameter:
- **Critical:** 336h (14 days) - Longest for critical data
- **Standard:** 168h (7 days) - Standard retention
- **Relaxed:** 72h (3 days) - Minimal retention

Litestream sidecar **actively deletes** old WAL files and snapshots from S3/MinIO.

### Layer 2: MinIO Lifecycle Policy (Safety Net)

**Global bucket policy:** 30-day expiration on all objects.

**Purpose:**
- Catches orphaned files if Litestream fails to cleanup
- Prevents indefinite storage growth from bugs
- Safety net if pod crashes for extended period

**Configuration (manual):**
```bash
mc ilm add minio/vixens-litestream --expiry-days 30
```

**Why 30 days?**
- Beyond longest Litestream retention (14d Critical)
- Provides buffer for debugging/recovery
- Prevents runaway storage costs

**Task created:** vixens-s5ch (MinIO policy configuration)

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

## Multi-Database Applications (Profile Mixing)

Apps with multiple databases can **mix profiles** based on each DB's characteristics.

**Example: Hydrus Network (4 databases, 3 profiles)**

```yaml
# litestream-config.yaml
litestream.yml: |
  addr: ":9090"
  
  dbs:
    # Critical: Main database (high activity, critical metadata)
    - path: /opt/hydrus/db/client.db
      replicas:
        - url: s3://$BUCKET/hydrus-client/client.db
          endpoint: $ENDPOINT
          sync-interval: 1s
          snapshot-interval: 1h
          retention: 336h  # 14 days
    
    # Standard: Mappings database (moderate activity)
    - path: /opt/hydrus/db/client.mappings.db
      replicas:
        - url: s3://$BUCKET/hydrus-client/client.mappings.db
          endpoint: $ENDPOINT
          sync-interval: 1s
          snapshot-interval: 6h
          retention: 168h  # 7 days
    
    # Standard: Master database (low-moderate activity)
    - path: /opt/hydrus/db/client.master.db
      replicas:
        - url: s3://$BUCKET/hydrus-client/client.master.db
          endpoint: $ENDPOINT
          sync-interval: 1s
          snapshot-interval: 6h
          retention: 168h  # 7 days
    
    # Ephemeral: Caches (skip backup - handled in init-container)
    # client.caches.db NOT listed - cleanup only

# Init-containers: One per DB
initContainers:
  - name: check-integrity       # Integrity check ALL DBs
  - name: restore-client        # Critical DB restore
  - name: restore-mappings      # Standard DB restore
  - name: restore-master        # Standard DB restore
  - name: cleanup-caches        # Ephemeral: rm -f *.caches.db*
```

**Rationale:**
- `client.db` = Critical (main metadata, tags, file records)
- `mappings.db` / `master.db` = Standard (important but less critical)
- `caches.db` = Ephemeral (regenerable, waste of storage)

**Benefits:**
- Optimized storage cost (no cache backup)
- Fastest restore for critical data (1h snapshots)
- Balanced recovery for secondary DBs (6h snapshots)

---

## Migration Strategy

### Phase 1: Documentation (This ADR)
- ✅ Define Light/Heavy profiles
- ✅ Define standardized init-container pattern

### Phase 2: Observability & Validation

**Before mass migration**, collect production metrics to validate profile choices:

1. **Enable Prometheus Metrics** (vixens-wfxk):
   - Add `addr: ":9090"` to all Litestream configs
   - Expose metrics: WAL count, snapshot count, DB size, sync rate
   
2. **Collect Metrics** (1 week minimum):
   - Monitor WAL generation rate (writes/day)
   - Track snapshot sizes and frequency
   - Observe restore times
   
3. **Analyze & Adjust** (vixens-yx42):
   - Run `scripts/analyze-litestream-metrics.py` (MinIO analysis)
   - Review Prometheus dashboards (if available)
   - Validate/adjust profile assignments
   - Confirm snapshot intervals appropriate

**Deliverable:** Validated profile matrix with production data.

### Phase 3: Create Shared Templates

Create `apps/_shared/litestream/` with:
- `critical-profile-config.yaml` - ConfigMap template (1h snapshots)
- `standard-profile-config.yaml` - ConfigMap template (6h snapshots)
- `relaxed-profile-config.yaml` - ConfigMap template (24h snapshots)
- `ephemeral-cleanup-init.yaml` - Cache cleanup init-container
- `integrity-check-init.yaml` - Reusable integrity check
- `restore-init.yaml` - Reusable restore init-container
- `sidecar-container.yaml` - Reusable Litestream sidecar

### Phase 4: Migrate Existing Apps

**Pilot apps** (1 per profile for validation):
1. **Critical:** Hydrus `client.db` (multi-DB, complex)
2. **Standard:** Vaultwarden (simple, single-DB, critical)
3. **Relaxed:** Authentik (low-activity, config)

**Mass migration** (after pilot validation):
- **Critical DBs:** Frigate events, high-traffic media
- **Standard DBs:** Sonarr, Radarr, Lidarr, Prowlarr, Whisparr, Mylar, Sabnzbd
- **Relaxed DBs:** Adguard-home, config-only apps
- **Ephemeral:** Hydrus caches, temp DBs

### Phase 5: Update Template & Documentation

- Update `apps/template-app/` with profile selection guide
- Document in `docs/guides/adding-new-application.md`
- Create profile selection decision tree
- Add Litestream troubleshooting guide

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

**Phase 2: Observability**
```bash
[ ] 1. Enable Prometheus metrics in all Litestream configs (vixens-wfxk)
[ ] 2. Wait 1 week for metrics collection
[ ] 3. Run scripts/analyze-litestream-metrics.py (MinIO analysis)
[ ] 4. Review metrics and validate profile assignments (vixens-yx42)
[ ] 5. Adjust profiles if needed based on production data
```

**Phase 3: Shared Templates**
```bash
[ ] 6. Create apps/_shared/litestream/ directory structure
[ ] 7. Create critical-profile-config.yaml template (1h snapshots)
[ ] 8. Create standard-profile-config.yaml template (6h snapshots)
[ ] 9. Create relaxed-profile-config.yaml template (24h snapshots)
[ ] 10. Create ephemeral-cleanup-init.yaml snippet
[ ] 11. Create integrity-check-init.yaml snippet
[ ] 12. Create restore-init.yaml snippet
[ ] 13. Create sidecar-container.yaml snippet (with Prometheus)
```

**Phase 4: Pilot Migration**
```bash
[ ] 14. Migrate Hydrus client.db (Critical + multi-DB)
[ ] 15. Migrate Vaultwarden (Standard, simple)
[ ] 16. Migrate Authentik (Relaxed, low-activity)
[ ] 17. Test restore times for each profile
[ ] 18. Validate integrity checks working
```

**Phase 5: Mass Migration & Documentation**
```bash
[ ] 19. Migrate remaining apps per profile
[ ] 20. Configure MinIO lifecycle policy (vixens-s5ch)
[ ] 21. Update template-app with profile selection
[ ] 22. Document in docs/guides/adding-new-application.md
[ ] 23. Create Litestream troubleshooting guide
[ ] 24. Create validation script for configs
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

Use this decision tree to select the appropriate profile **per database**:

1. **Is this cache/temporary data?** → **Ephemeral** (skip backup)
2. **Data loss tolerance:**
   - **< 1 hour** → **Critical** (1h snapshots, 14d retention)
   - **< 6 hours** → **Standard** (6h snapshots, 7d retention)
   - **< 24 hours** → **Relaxed** (24h snapshots, 3d retention)
3. **Write activity (if measurable):**
   - **> 100 writes/hour** → **Critical**
   - **10-100 writes/hour** → **Standard**
   - **< 10 writes/hour** → **Relaxed**
4. **Data criticality:**
   - **Mission-critical** (Vaultwarden passwords, Hydrus metadata) → **Standard** minimum
   - **Important** (Media metadata) → **Standard**
   - **Convenience** (Configs, can rebuild) → **Relaxed**

**Note:** Metrics-driven validation (Phase 2) will provide actual write rates to replace estimates.
