# #4185 — UMI TrueNAS iSCSI cutover after offline ext4 recovery

**Status:** operator-assisted maintenance; not a Kubernetes manifest or Argo application. **Never auto-sync this procedure.**

## Physically verified on 2026-10-24

- TrueNAS `synellia` has the intended interface `192.168.111.69:3260`; all four original target IQNs, extents and LUN 0 were restored against the **same pre-existing zvols**. The old address `192.168.200.244` cannot return.
- Original Hindsight cache PV `pvc-c5b1130c-d400-4f45-9cc8-bd1a25cf8e43` was switched from `Delete` to `Retain`, then locally rebound to `nas.truxonline.com:3260`. Owner-run script output verified PV/PVC Bound, source portal and same CSI volumeHandle, with **new PVC UID** `8c563062-639a-4555-9bc0-1b35c7b696a5`. The live mount is **not yet validated**.
- Hermes `pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77`, ext4 UUID `79009fd9-9dad-4817-a252-2939205715ac`, and Hindsight PostgreSQL `pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf`, ext4 UUID `2d6b531d-d11d-4535-a136-447c33997c06`, were repaired offline. Both final `e2fsck -f -n` checks had five passes without remaining errors and exit code 0. This **does not verify application data, PostgreSQL WAL, or successful writable mounts**.
- `services/hermes`, `hindsight/hindsight-api` and `hindsight/hindsight-postgresql` were scaled to zero. Argo `hermes-umi` and `hindsight-umi` auto-sync were disabled for maintenance. Their original desired states MUST be restored later.
- **HomeAssistant is deprecated on UMI** per owner; its Released/Retain PV and TrueNAS zvol are intentionally left unchanged.
- After the owner performed the Hermes and PostgreSQL API-object cutovers (date: 2026-10-24), a read-only Kubernetes PV listing reported all **three** in-scope PVs **Bound / Retain**, with unchanged original PV names and `spec.csi.volumeAttributes.portal=nas.truxonline.com:3260`:
  - `pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77` → `services/hermes-data-truenas`
  - `pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf` → `hindsight/data-hindsight-postgresql-0`
  - `pvc-c5b1130c-d400-4f45-9cc8-bd1a25cf8e43` → `hindsight/hindsight-api-model-cache`
  This verifies API binding/portal **only**. Post-cutover CSI mount, writable volume state, PostgreSQL WAL/database and Hermes profile/bootstrap checks remain unverified. No storage deletion was reported.

## Remaining two original Bound/Retain PVs

| PVC | Existing PV and CSI handle | Size | StorageClass | Original TrueNAS LUN |
| --- | --- | --- | --- | --- |
| `services/hermes-data-truenas` | `pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77` | 80Gi | `truenas-iscsi-retain` | IQN `iqn.2005-10.org.freenas.ctl:csi-pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77-umi`, 0 |
| `hindsight/data-hindsight-postgresql-0` | `pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf` | 8Gi | `truenas-iscsi-retain` | IQN `iqn.2005-10.org.freenas.ctl:csi-pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf-umi`, 0 |

Each PV's immutable CSI source still records the retired portal. Modifying only the CSI configuration Secret is insufficient.

## Post-cutover CSI credentials/configuration evidence (read-only)

On 2026-10-24 owner verified `truenas-csi-iscsi-driver-config` ExternalSecret `Ready=True` / `SecretSynced` and read only the **non-secret endpoint lines** from the active driver Secret: API/SSH hosts `nas.truxonline.com` and `targetPortal=nas.truxonline.com:3260`. All three restored PVCs were `Bound` with the original PV names. Thus the former retired portal is no longer present in the **observed PV and CSI endpoint configuration fields**. Do not infer that the CSI node-plugin has staged the real disks or that any application data is sound.

Argo applications `hermes-umi` and `hindsight-umi` remain explicitly `automated.enabled=false`. `hindsight-umi.spec.source` was null in an owner projection, but **`spec.sources` was not inspected**: it may be multi-source. Before any automatic sync, inspect the source(s), Helm desired PVCs and name/UID semantics without printing Helm values or secret contents.

## GitOps resumption guard (unverified)

The current `main` at `ac4d76cc933eae3a052c703bad722e8beb2a555c` declares `apps/60-services/hermes/base/pvc.yaml` as a normal RWO 80Gi `truenas-iscsi-retain` PVC **without** an explicit `spec.volumeName`. The live re-created PVC has an explicit `volumeName` (and a new UID). Before resuming `hermes-umi` selfHeal, compare Argo rendered/desired PVC against the live spec and ensure the restore binding will **not** be removed or trigger deletion/reprovisioning. Do not hardcode UMI's PV name in the shared Hermes base (it is not portable).

The running `hindsight-umi` Application used a Helm-tracked model-cache PVC, while `main` documents retirement of the standalone chart. Inspect the live Argo `hindsight-umi` source(s), Helm chart values and all desired PVC templates before any sync or automatic self-heal. A valid PVC `Bound` status does not prove the controller CSI configuration points at the new TrueNAS portal; confirm the non-secret API/SSH/iSCSI endpoints from the active CSI Secret, with the remaining auth material redacted.

Read-only owner check, use K3s kubeconfig explicitly (bare root `kubectl` on UMI returned `NotFound`):
```bash
K=/etc/rancher/k3s/k3s.yaml
sudo kubectl --kubeconfig="$K" -n argocd get application hermes-umi hindsight-umi -o json \
  | jq '.items[] | {name:.metadata.name, source:.spec.source, sources:.spec.sources, automated:.spec.syncPolicy.automated}'
sudo kubectl --kubeconfig="$K" -n services get pvc hermes-data-truenas -o json \
  | jq '{phase:.status.phase,volumeName:.spec.volumeName,uid:.metadata.uid}'
sudo kubectl --kubeconfig="$K" -n hindsight get pvc hindsight-api-model-cache data-hindsight-postgresql-0 -o json \
  | jq '.items[] | {name:.metadata.name,phase:.status.phase,volumeName:.spec.volumeName,uid:.metadata.uid}'
```
Do not proceed to scale pods or reenable Argo until actual mount and application data safety plan is reviewed.

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
