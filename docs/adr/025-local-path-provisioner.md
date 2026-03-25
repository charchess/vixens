# ADR-025: Local Path Provisioner for Node-Local Storage

**Date:** 2026-03-25
**Status:** Accepted
**Tags:** storage, local, performance

## Context

The cluster uses Synology iSCSI (via CSI) as the sole block storage backend. All PVCs — from large media libraries to tiny SQLite databases — traverse the network to the NAS. This creates I/O pressure on the NAS for workloads that don't need network-attached storage.

~17 apps use small SQLite databases (1-4Gi) backed up continuously by DataAngel to S3 (MinIO). For these apps, node-local NVMe/SSD storage would provide faster I/O without NAS dependency, while DataAngel ensures durability via S3 backup.

## Decision

Deploy **Rancher Local Path Provisioner v0.0.35** with two StorageClasses:

| StorageClass | Reclaim Policy | Usage |
|---|---|---|
| `local-path-delete` | Delete | **Default.** Auto-cleanup on PVC deletion. DataAngel handles recovery. |
| `local-path-retain` | Retain | Special cases needing manual data preservation. |

### Why Local Path Provisioner

- **Officially documented by Sidero Labs** for Talos Linux
- **Native Talos v1.12 UserVolumeConfig** integration (no extraMounts hack)
- **Battle-tested** — default StorageClass in K3s (millions of installs)
- **Minimal footprint** — single controller pod, ephemeral busybox helpers
- **Functionally sufficient** — dynamic provisioning, per-PV directory isolation, node affinity

### Alternatives Evaluated

| Option | Verdict | Reason |
|---|---|---|
| OpenEBS LocalPV Hostpath | Runner-up | Similar features but requires legacy extraMounts on Talos, CNCF status uncertain |
| TopoLVM | Rejected | LVM complexity overkill for small SQLite PVs |
| Longhorn | Rejected | Adds its own iSCSI layer — defeats the purpose |
| Kubernetes native Local PV | Rejected | No dynamic provisioning, manual PV management |
| Piraeus/LINSTOR | Rejected | DRBD kernel module overkill |

## Trade-offs

### Accepted

- **No inter-node failover.** Local PVs are bound to the node. If the node dies, data is lost locally. DataAngel S3 backup is the recovery mechanism.
- **No capacity tracking.** The scheduler doesn't know remaining disk space. Mitigated by monitoring node filesystem usage via Prometheus.
- **Node drain = pod Pending.** During maintenance, the pod waits for its node to return (same as iSCSI with node affinity).

### Mitigations

- **DataAngel S3 backup** — continuous replication to MinIO. Recovery from node loss: delete PV, recreate PVC on new node, DataAngel restores from S3.
- **Prometheus alerts** — `node_filesystem_avail_bytes` alert at 80% on the local-path partition.
- **`strategy: Recreate`** — already required for RWO PVCs. No change needed.

## When to Use

| Scenario | StorageClass |
|---|---|
| SQLite apps with DataAngel backup | `local-path-delete` |
| Apps needing fast local I/O (cache, tmp) | `local-path-delete` |
| Apps needing network-attached persistence | `synelia-iscsi-retain` (existing) |
| Databases (PostgreSQL, MariaDB) | `synelia-iscsi-retain` (existing) |
| Media libraries (large, shared NFS) | NFS (existing) |

## Implementation

### Prerequisites (terravixens)

Talos UserVolumeConfig on each node:
```yaml
apiVersion: v1alpha1
kind: UserVolumeConfig
name: local-path-provisioner
provisioning:
  diskSelector:
    match: disk.transport == 'nvme'
  minSize: 200GB
  maxSize: 200GB
```

### Deployment (vixens)

- Local Path Provisioner v0.0.35 in `apps/01-storage/local-path-provisioner/`
- Base path: `/var/mnt/local-path-provisioner`
- Namespace: `local-path-storage` with `pod-security.kubernetes.io/enforce: privileged`
- Two StorageClasses with `volumeBindingMode: WaitForFirstConsumer`

### Security

- **CVE-2025-62878 (CVSS 10.0):** Path traversal in `pathPattern` parameter. Fixed in v0.0.34. Must use v0.0.34+. Never set `allowUnsafePathPattern: true`.

## References

- [Sidero Labs: Local Storage on Talos](https://docs.siderolabs.com/kubernetes-guides/csi/local-storage)
- [Talos v1.12 User Volumes](https://docs.siderolabs.com/talos/v1.12/configure-your-talos-cluster/storage-and-disk-management/disk-management/user)
- [Rancher Local Path Provisioner](https://github.com/rancher/local-path-provisioner)
- [CVE-2025-62878](https://github.com/advisories/GHSA-jr3w-9vfr-c746)
- Issue: #2283, #2435
