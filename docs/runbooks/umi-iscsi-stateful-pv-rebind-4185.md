# #4185 — UMI TrueNAS iSCSI cutover after offline ext4 recovery

**Status:** operator-assisted maintenance; not a Kubernetes manifest or Argo application. **Never auto-sync this procedure.**

## Physically verified on 2026-10-24

- TrueNAS `synellia` has the intended interface `192.168.111.69:3260`; all four original target IQNs, extents and LUN 0 were restored against the **same pre-existing zvols**. The old address `192.168.200.244` cannot return.
- Original Hindsight cache PV `pvc-c5b1130c-d400-4f45-9cc8-bd1a25cf8e43` was switched from `Delete` to `Retain`, then locally rebound to `nas.truxonline.com:3260`. Owner-run script output verified PV/PVC Bound, source portal and same CSI volumeHandle, with **new PVC UID** `8c563062-639a-4555-9bc0-1b35c7b696a5`. The live mount is **not yet validated**.
- Hermes `pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77`, ext4 UUID `79009fd9-9dad-4817-a252-2939205715ac`, and Hindsight PostgreSQL `pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf`, ext4 UUID `2d6b531d-d11d-4535-a136-447c33997c06`, were repaired offline. Both final `e2fsck -f -n` checks had five passes without remaining errors and exit code 0. This **does not verify application data, PostgreSQL WAL, or successful writable mounts**.
- `services/hermes`, `hindsight/hindsight-api` and `hindsight/hindsight-postgresql` were scaled to zero. Argo `hermes-umi` and `hindsight-umi` auto-sync were disabled for maintenance. Their original desired states MUST be restored later.
- **HomeAssistant is deprecated on UMI** per owner; its Released/Retain PV and TrueNAS zvol are intentionally left unchanged.

## Remaining two original Bound/Retain PVs

| PVC | Existing PV and CSI handle | Size | StorageClass | Original TrueNAS LUN |
| --- | --- | --- | --- | --- |
| `services/hermes-data-truenas` | `pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77` | 80Gi | `truenas-iscsi-retain` | IQN `iqn.2005-10.org.freenas.ctl:csi-pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77-umi`, 0 |
| `hindsight/data-hindsight-postgresql-0` | `pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf` | 8Gi | `truenas-iscsi-retain` | IQN `iqn.2005-10.org.freenas.ctl:csi-pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf-umi`, 0 |

Each PV's immutable CSI source still records the retired portal. Modifying only the CSI configuration Secret is insufficient.

## Deliberate single-PV cutover (human execution only)

1. Confirm Argo remains paused, owning workload replicas are 0, no Pod consumes the PVC, no VolumeAttachment and no CSI globalmount remain. Manual iSCSI sessions for Hermes/PostgreSQL were logged in for `fsck`; verify exact block UUIDs and unmounted state, then log these **manual** sessions out before any CSI stage attempt. Never logout a mounted LUN.
2. Export full original PV/PVC JSON locally with restrictive permissions. Assert exact current `Bound/Retain` phase, claim namespace/name/UID, CSI driver `org.democratic-csi.truenas-iscsi`, original volumeHandle, target IQN, LUN 0, capacity and storageClass, and sole `pv-protection`/`pvc-protection` finalizers.
3. Prepare new static PV: **same PV name, volumeHandle, target IQN, LUN and entire CSI source except** `volumeAttributes.portal=nas.truxonline.com:3260`; `persistentVolumeReclaimPolicy=Retain`; `claimRef` reserves only old PVC namespace/name (no obsolete UID). Do not transfer old PV metadata UID, resourceVersion, finalizers, provisioner annotations or status.
4. Prepare replacement PVC: same namespace/name, class, request size, mode and accessModes with **explicit** `spec.volumeName` matching existing PV name. Do not transfer old PVC UID, resourceVersion, dynamic bound/provisioner annotations or status. A new UID is **expected**.
5. With a fresh stop-on-error guard and human confirmation for that specific PV: delete only old **PVC API object**, wait until old PV is `Released` and remains `Retain`; delete only that old PV **API object** (not TrueNAS storage); create static replacement PV first and then prebound PVC. Validate both `Bound`, matching new PVC UID / PV claimRef UID, unchanged `volumeHandle`, correct portal and `Retain`. Never use `kubectl replace --force`, strip finalizers, or auto-retry a partial cutover.
6. Validate CSI staging to the same zvol after migrating **one** stateful volume at a time; ensure no re-provision or formatting and perform data/application checks. PostgreSQL needs WAL/database consistency proof; clean ext4 alone is insufficient.
7. Reconcile the Argo desired-state PVC contract and ownership before restoring replica counts and auto-sync. Do not assume reverting a Git commit can restore the retired NAS address.

**Rollback boundary:** original PV/PVC JSON backups are recovery evidence, not an automated rollback. Once an old object is deleted its UID changes; returning to a previous state requires an explicit binding/provisioner review. If any check fails, **stop with applications down and data retained** and record the exact phase to issue #4185.

This document does not authorize deletion, fsck, mount, iSCSI logout, Argo sync or promotion by an automated agent. No secret values should be copied into issue/PR.

## Traceability: intent → implementation → evidence → remaining derivation

#4185 preserve original stateful volumes and correct immutable CSI source → controlled static API-object rebind per PV → cache owner-reported success and two final offline ext4 checks → Hermes/PG cutovers, full app/WAL recovery, reviewed GitOps reconciliation and UMI live validation remain outstanding.
