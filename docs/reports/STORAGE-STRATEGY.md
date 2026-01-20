# Storage & Backup Strategy

This document formalizes the strategy for persistent storage (PVC) and data protection across the Vixens infrastructure.

---

## 1. Storage Class Strategy

We use different StorageClasses based on the environment to balance data safety and cluster cleanliness.

| Environment | StorageClass | Reclaim Policy | Goal |
| :--- | :--- | :--- | :--- |
| **Production** | `synelia-iscsi-retain` | **Retain** | Prevent data loss if PVC is accidentally deleted. |
| **Development** | `synelia-iscsi-delete` | **Delete** | Ensure automatic cleanup of volumes after tests. |
| **Shared** | `nfs-storage` | **Retain** | High-availability access for media and logs. |

---

## 2. PVC Naming Convention

To ensure clarity and maintainability, PVCs must follow this naming pattern:

**Pattern:** `<app-name>-<purpose>-pvc`

*Examples:*
- `homeassistant-config-pvc`
- `postgresql-data-pvc`
- `frigate-clips-pvc`

---

## 3. Backup Requirements (Standard "Elite")

All applications with persistent state must implement the following backup tiers:

### Tier 1: Real-time DB Backup (Litestream)
- **Applicability:** All SQLite-based applications.
- **Requirement:** A `litestream` sidecar must be configured to stream changes to S3 storage.
- **Integrity:** Use the "Fail-Safe Integrity" pattern (check-integrity initContainer).

### Tier 2: Configuration Backup (Rclone)
- **Applicability:** Applications with files in `/config`.
- **Requirement:** An `rclone` sidecar must perform periodic sync (every 5-15 min) to S3.

### Tier 3: Disaster Recovery (Velero)
- **Applicability:** All Namespaces.
- **Requirement:** Scheduled daily snapshots of all cluster resources and PVs (to be implemented via `vixens-i7xx`).

---

## 4. Implementation Snippet

```yaml
# Example PVC for a production app
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myapp-config-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: synelia-iscsi-retain # For Prod
```

---

## 5. Storage Sizing Guidelines

- **Config-only:** 1Gi - 2Gi
- **Databases:** 10Gi - 20Gi (Shared PG uses 50Gi)
- **Media Cache:** 50Gi - 100Gi
- **Media Library:** Direct NFS mount (No PVC)
