# UMI TrueNAS iSCSI re-IP — PV cutover preflight (#4185)

**Status:** source-only preparation; **NOT an executable recovery or a deployment authorization**.
**Applies only to:** UMI, former portal `192.168.200.244:3260`, desired
`nas.truxonline.com:3260` (dedicated Kubernetes TrueNAS interface observed at
`192.168.111.69`). The old address cannot be restored. Production is out of scope.

## What is already fixed

Merged #4186 and #4187 repaired the UMI OpenBao/ESO auth and TrueNAS
CSI driver Secret. The operator physically verified `ClusterSecretStore/openbao
Ready=True`, both CSI ExternalSecrets `Ready=True` and the refreshed driver
`targetPortal: nas.truxonline.com:3260`.

This **does not** update immutable `PersistentVolume.spec.csi.volumeAttributes.portal`
in existing dynamically created PVs, or the already-established node iSCSI
sessions. Existing PV source records and sessions still use `.200.244`.

| Namespace/PVC | PV suffix | State | Reclaim | Last CSI attachment | Last filesystem evidence |
| --- | --- | --- | --- | --- | --- |
| services/hermes-data-truenas | 5cb237dc… | Bound | Retain | umi / true | /dev/sde ext4 emergency_ro, journal aborted |
| hindsight/data-hindsight-postgresql-0 | 71395a3d… | Bound | Retain | umi / true | /dev/sdf ext4 emergency_ro, journal aborted |
| homeassistant/homeassistant-config | c09f3044… | Released | Retain | none | unmounted; preserve historical volume |
| hindsight/hindsight-api-model-cache | c5b1130c… | Bound | **Delete** | umi / true | /dev/sdd mounted rw, not proven healthy |

Each has its *own* IQN, LUN `0`, CSI `volumeHandle` equal to its PV name.
Do not infer target/zvol identity from `lun=0` alone.

## 1. Read-only evidence and offline candidate validation

On **UMI**, verify the context and collect PV metadata into a private **local**
file. The file must not be committed or uploaded: it can contain operational
annotations and identity metadata.

```bash
kubectl config current-context
umask 077
kubectl get pv -o json > /root/umi-pv-before-4185.json

python3 scripts/storage/umi_iscsi_pv_cutover.py \
  --pv-json /root/umi-pv-before-4185.json
```

The script makes **zero Kubernetes, TrueNAS or iSCSI calls**. It checks all
four exact names and claimRef namespaces, names and UIDs, driver, target IQN,
LUN, volumeHandle, portal, accessModes, capacity, StorageClass and reclaim
policy. It rejects cross-claim references, omitted or unexpected old-portal
volumes, source drift and possible inline CSI credential fields. The expected
input is a **pre-migration** full PV List; after starting a phased cutover it
deliberately fails closed on portal/status drift rather than silently adapting.

To produce **review-only proposals**, in a private local workspace:

```bash
python3 scripts/storage/umi_iscsi_pv_cutover.py \
  --pv-json /root/umi-pv-before-4185.json \
  --emit-review-yaml > /root/umi-pv-review-only.yaml
```

**DO NOT apply this file.** It does not include Kubernetes metadata UID,
resourceVersion, finalizers, provisioning/controller annotations or status.
It retains the original `spec.claimRef.uid` for human comparison, which may
not match a future PVC after an explicitly approved rebind. Ordinary
`kubectl apply` cannot change a PV's immutable source; attempting to apply
these records directly is neither a supported migration nor safe.

The only intended changes in proposed `spec` are the portal to
`nas.truxonline.com:3260` and the *candidate* cache reclaim policy from
`Delete` to `Retain`. **The real cache PV remains Delete until a separately
approved live protection action**; a candidate manifest does not protect it.

## 2. Physical gate before any data-plane change

All are mandatory, with non-secret evidence recorded in #4185:

1. Map each *exact* existing TrueNAS target IQN + LUN 0 to its current extent
   and surviving zvol. Verify target exposure on `192.168.111.69:3260`;
   a successful TCP handshake is not evidence of target/LUN mapping.
2. Verify existing backups and preserve crash-consistent snapshots / independent
   copies of the actual zvols, especially Hermes and **PostgreSQL Hindsight**.
   A snapshot taken after aborted writes cannot retroactively restore lost
   data or guarantee transactional consistency.
3. Agree a maintenance window and a GitOps-controlled desired-state plan to
   quiesce each actual data owner. Account for the already-failed PostgreSQL
   and Hermes writes; no assumption of a successful database shutdown.
4. Capture original PV and PVC definitions including claim UID, bindings,
   annotations/finalizers, and `VolumeAttachment` and node session states.
   Confirm new target is the **same** backend zvol and nothing will re-provision
   or format it. Verify cache `Delete` policy is protected before *any*
   possible deletion/rebind.
5. Separately review and approve an ordered replacement/rebind plan. The CSI
   PersistentVolumeSource is immutable: changing this field requires a
   controlled object lifecycle, not a patch. No delete/recreate commands are
   provided here precisely because an active bound PVC and a CSI provisioner
   can cause accidental backend destruction or accidental blank provisioning.

## 3. Controlled cutover design — a further authorized increment

- Keep original zvol and target IQN/LUN, `volumeHandle`, exact PVC ownership,
  capacity and StorageClass. Preserve **all four** volumes, including the
  Released/Retain HomeAssistant PV and the active Delete-policy cache.
- During an approved, single-volume window, protect against auto-reconciliation
  and writers using a reviewed GitOps change. Verify no Pod continues to use
  the PVC and that the attachment is actually released before manipulating the
  CSI volume representation; do not trust a stale `attached=true` as I/O health.
- Stage a replacement PV/source with the corrected portal only after all
  preservation gates, with an explicit original-object and claimant-UID
  reconciliation/rollback procedure. A dynamic PV may have deletion finalizers
  and provisioner annotations whose handling requires separate review.
  **Do not remove finalizers to force a deletion.**
- Dry-review the PV/PVC binding before re-staging. Verify the new node session
  actually connects to the intended target at `192.168.111.69:3260`,
  including stable block identity. Validate on a non-critical protected volume
  first; never mass-relogin the three active targets.
- Only once device identity, transport and verified protected copy are
  established, evaluate **offline** filesystem repair on a safe clone or
  offline protected LUN. Existing `k3s_iscsi_fsck_recover_v2.0.0.sh` is **not
  re-IP safe**: it resolves the old by-path/PV portal and can scale to zero
  before checking connectivity. Do not run its `run` or `resume` commands
  for this incident as-is.
- Validate ext4 mount and application behavior separately. For PostgreSQL,
  filesystem checks **do not** certify WAL or database integrity; follow
  PostgreSQL consistency/recovery steps after a verified backup.

**Rollback:** Hold workloads quiesced and use the preserved original source
record, zvol snapshot/clone and UID/binding evidence under a separately
approved rebind plan. Merely reverting a GitOps commit cannot re-enable the
retired `.200.244` IP or undo journal corruption. Never delete or format
the underlying LUN as a rollback.

## Forbidden in this PR

No Argo Application includes candidate PVs. No `kubectl apply/edit/delete`,
PV source patch, manual CSI/Argo sync, token disclosure, `iscsiadm` login or
logout, node restart, forced mount or `fsck`. Do not merge an executable
cutover into auto-synced GitOps trees without a separately reviewed physical
go/no-go. Prod remains pinned to `prod-stable`.

Related: [TrueNAS CSI README](../../apps/01-storage/truenas-csi/README.md),
[#4185](https://github.com/charchess/vixens/issues/4185), historical
iSCSI ext4 incidents in `docs/post-mortems/`.
