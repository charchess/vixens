# #4193 — Synelia official TrueNAS CSI (UMI, preflight + rollback)

**Status:** live UMI physical validation remains incomplete. The first manual Argo Sync failed due to missing Cilium API, corrected by merged PR #4199. The subsequent manual Sync partially created the new cluster-scoped CSI driver and four StorageClasses, but failed because namespace `synelia-csi` was not found. This increment explicitly declares a first-wave Synelia namespace. Review, merge, refresh Argo and verify the desired resources **before** another separately initiated manual Sync. No TrueNAS volume provisioning or legacy PV adoption is established.

## Contract and pinned upstream

- UMI-only ArgoCD Application: `argocd/overlays/dev/apps/synelia-csi-umi.yaml`, manual sync (no `automated`).
- Chart: `truenas/truenas-csi` **1.4.0**, driver `ghcr.io/truenas/truenas-csi:v1.4.0`. Chart index SHA256: `3a1b9f326447d54beb3fec20e3b026a4a07f68cb7f3685c7a71f7e9d70589367` (verify downloaded chart against this digest in release preparation). The chart pins provisioner v6.1.1, attacher v4.11.0, snapshotter v8.5.0, resizer v2.1.0, registrar v2.16.0, livenessprobe v2.18.0. Source: https://github.com/truenas/truenas-csi/tree/v1.4.0/charts/truenas-csi
- New CSIDriver `synelia.csi.truenas.io`, namespace/release `synelia-csi`. Existing democratic-csi instances, storage classes and volumes are **not changed**.
- Separate **non-default** classes `synelia-nfs-retain`, `synelia-nfs-delete`, `synelia-iscsi-retain`, `synelia-iscsi-delete`. `Delete` classes are for **new disposable test PVCs only**. Nothing points an existing workload to these names.
- New ZFS test parents: `tank/k8s/vixens/synelia-test/nfs` and `tank/k8s/vixens/synelia-test/iscsi`, distinct from democratic-csi's `tank/k8s/vixens/{nfs,iscsi}/vols`. Owner observed `tank` ONLINE with ~58T free and confirmed the new Synelia parent tree absent on 2026-10-25. Official CSI v1.4.0 sets `CreateAncestors` when `datasetPath` is specified for NFS and iSCSI; do **not** manually create parents for this preflight. The driver will create them when a separately authorized new PVC is provisioned, subject to its TrueNAS API permissions.

## GitOps App-of-Apps wiring (desired state, not live evidence)

The root `vixens-app-of-apps` template in `argocd/base/root-app.yaml.tpl`
renders Applications from an externally supplied `overlay_path`. The **UMI**
preflight must confirm that the live root `spec.source.path` really resolves
to `argocd/overlays/dev` at the expected Git revision; checking that files exist
in Git or that CI renders them does not prove UMI's root consumes that path.

Expected desired-state chain:

```text
vixens-app-of-apps (root, overlay_path = argocd/overlays/dev)
  -> argocd/overlays/dev/kustomization.yaml
    -> apps/synelia-csi-umi.yaml (child Application, manual sync)
      -> apps/01-storage/synelia-csi/overlays/dev/kustomization.yaml
        -> apps/01-storage/synelia-csi/umi (ESO + Cilium)
      -> pinned upstream Helm 1.4.0 + umi/values.yaml (non-default StorageClasses)
```

**Storage identity is `synelia`, not `umi`:** `umi` appears only in GitOps
deployment location/Application identity. None of the StorageClass names or ZFS
test dataset parent paths contains `umi`. If an earlier Synelia volume was
already provisioned under the previous `synelia-umi-test` parent, **stop**:
the path change needs an explicit per-volume data/integrity assessment and must
not silently move or recreate a PV or zvol.

Read-only root/child inspection before deployment:

```sh
kubectl -n argocd get application vixens-app-of-apps -o jsonpath='{.spec.source.path}{"\\n"}{.spec.source.targetRevision}{"\\n"}'
kubectl -n argocd get application synelia-csi-umi -o jsonpath='{.spec.sources[2].path}{"\\n"}'
kubectl get pv -o custom-columns=NAME:.metadata.name,SC:.spec.storageClassName,CSI:.spec.csi.driver,HANDLE:.spec.csi.volumeHandle
```

The parent App-of-Apps may create/update the **child Application** automatically,
but it does not authorize a manual sync of Synelia's CSI resources. Keep the
child's `spec.syncPolicy.automated` absent; if the live root points elsewhere,
resolve root ownership in a separately reviewed GitOps change before rollout.

## Hard physical gate before ANY Argo manual sync

1. Confirm latest main, reviewed PR, approval for **UMI-only** change and no overlapping sync/prune. Record the old CSIDrivers, StorageClasses, all PV/PVC/VolumeAttachments (including Released/Retain), TrueNAS zvol/dataset, portal, target, IQN, LUN, consumer, backups and snapshot identifiers. Specifically preserve Hermes 80Gi PVC `services/hermes-data-truenas` and its ext4 EIO/`emergency_ro` incident under #4185.
2. Confirm **TrueNAS 27** serves the WebSocket API to `wss://nas.truxonline.com/api/current`, and its certificate chains to a trusted CA **and has DNS SAN for `nas.truxonline.com`**. TLS verification stays enabled; if a private CA is required, commit the **public CA certificate only** in a dedicated reviewed ConfigMap and set `truenas.existingCABundleConfigMap`. Do not use `insecureSkipTLS`. Keep API WSS interface separate from iSCSI `192.168.111.69:3260` and NFS `nas.truxonline.com:2049`; verify routes and TrueNAS binding before sync.
3. In **OpenBao Infra**, provision a **new**, minimal-authority, non-shared CSI API key at `vixens/dev/apps/01-storage/synelia-csi/api`, property `api-key`. Grant ESO read only to that path; verify `ClusterSecretStore/openbao` is Ready, token policy and refresh. Never use Fabric runtime secrets or copy key values to CI/logs/Git. Rotate/revoke the exposed historical bearer key from #4185 separately after dependency checks.
4. Verify UMI k3s/Kubernetes version and compatibility with the pinned CSI sidecars, presence of `open-iscsi` and active `iscsid` on the host, `/var/lib/kubelet` mount propagation, required privileged host mounts, node plugin registration socket. The node DaemonSet uses `hostNetwork`; **Kubernetes Pod NetworkPolicy does not constrain node host traffic**. Explicitly check host firewall egress to API 443, NFS 2049, iSCSI 3260 and cluster DNS/API. The UMI `NetworkPolicy` selects CSI controller/node pods; CoreDNS has `k8s-app=kube-dns` in `kube-system`. Verify the built-in K3s kube-router NetworkPolicy controller is actually enabled and enforcing rules; Flannel alone is not an enforcer.
5. Confirm TrueNAS ACL/API key scope, ZFS test parent paths, available capacity, iSCSI initiator/target policies, NFS exports, no conflicting CSIDriver registrations or controller ownership. No snapshot controller/CRD installation from this PR; `volumeSnapshotClass.enabled=false` even if separate snapshot-controller app is present.

## UMI k3s/Flannel networking correction (#4193; 2026-10-25)

**Physical cause observed:** a manual Argo Sync of `synelia-csi-umi` failed with
`failed to discover server resources for group version cilium.io/v2` before
creating resources. UMI is a single-node k3s installation with
`10-flannel.conflist` and no Cilium CRDs/DaemonSet; the Talos+Cilium ADR is
not the observed UMI runtime. The original UMI overlay contained two unsupported
`CiliumNetworkPolicy` resources. Neither deleting those policies nor using
`SkipDryRunOnMissingResource` is an acceptable remedy.

**GitOps correction:** the UMI overlay uses two namespaced
`networking.k8s.io/v1` NetworkPolicies, each selecting only its corresponding
Synelia CSI pod labels. Egress is limited to observed CoreDNS Pod labels
(`kube-system/k8s-app=kube-dns`, TCP/UDP 53), Kubernetes API Service
`10.43.0.1:443`, its observed host endpoint `172.30.208.64:6443`,
and TrueNAS `192.168.111.69` (controller: WSS TCP 443;
node: WSS/NFS/iSCSI TCP 443/2049/3260).
The endpoint address `172.30.208.64` is from UMI's **current WSL2 boot**,
may change after restart, and must be rechecked and updated via GitOps
before re-syncing if it changes. Both API addresses are needed because Service
IP destination NAT can be applied before Pod policy evaluation.

**Enforcement caveat:** Flannel itself does not enforce NetworkPolicy.
K3s ordinarily includes a kube-router network-policy controller, but its
effective activation must be checked in the live k3s service flags/systemd
config and with controlled egress-positive/negative tests. These YAML policies
are *not* evidence of runtime enforcement. The node DaemonSet currently uses
`hostNetwork`; Kubernetes Pod NetworkPolicy does not restrict host traffic
or host `iscsiadm`. Keep host firewall rules and port reachability as separate
checks. Any missing authorization/API egress must be diagnosed, not fixed by
opening `0.0.0.0/0` or disabling TLS verification.

Read-only checks on UMI before retrying an approved sync:

```sh
kubectl get crd ciliumnetworkpolicies.cilium.io --ignore-not-found
kubectl -n kube-system get pods -l k8s-app=kube-dns --show-labels
kubectl -n default get service kubernetes -o wide
kubectl -n default get endpointslice -l kubernetes.io/service-name=kubernetes -o wide
systemctl cat k3s | grep -E 'flannel|network-policy|ExecStart' || true
sudo iptables-save 2>/dev/null | grep -m 5 KUBE-ROUTER || true
kubectl -n argocd get application synelia-csi-umi -o jsonpath='{.spec.syncPolicy.automated}{"\\n"}{.status.sync.status}{"\\n"}'
```

**Do not directly `kubectl apply` the rendered CSI resources.** UMI already
has Argo self-heal disabled for this Application (`automated=null`).
Direct apply would create untracked drift and bypass the now reviewable GitOps
policy fix. Once this PR is reviewed and merged, first verify that Argo has
refreshed to the new `main` revision and the desired resource list has two
`NetworkPolicy` resources and **zero** `CiliumNetworkPolicy` resources.
Use a one-shot **Argo operation.sync** request scoped to this Application
(no prune/force/replace) as documented in the issue; inspect
`.status.operationState.phase` and ESO readiness before any PVC test.

## Explicit namespace bootstrap after second failed UMI Sync (#4193; 2026-10-25)

**Actual partial first-install result:** after #4199 cleared the missing
`cilium.io/v2` API, the next manual Argo operation reported
`phase=Failed`, message `namespaces "synelia-csi" not found`.
The cluster-scoped `CSIDriver/synelia.csi.truenas.io` and all four **new**
Synelia StorageClasses were successfully installed, but CSI controller/node
resources and the OpenBao-backed ExternalSecret were not reported installed.
This is **partial sync**, not a Healthy/ready CSI: the Application health
field may still say `Healthy` despite the failed operation. Historical
Hindsight VolumeAttachments remained `ATTACHED=true`, and Hermes remained
`ATTACHED=false` as before.

The existing Application requested `CreateNamespace=true` together with
`ServerSideApply=true`; some Argo CD versions have a documented interaction
causing namespace creation failures. This is a plausible explanation, not
a proven Argo bug/version diagnosis for UMI. See upstream
https://github.com/argoproj/argo-cd/issues/16829 and
https://argo-cd.readthedocs.io/en/latest/user-guide/sync-options/.

**GitOps remediation:** `apps/01-storage/synelia-csi/umi/namespace.yaml`
declares only `Namespace/synelia-csi` with Argo sync wave `-1`, before
the chart, ESO and NetworkPolicies at default wave 0. Namespace local
sync option `Prune=false` protects it against a future prune. The
UMI-only Kustomize layer includes this file; it is not deployed to prod.
The existing four StorageClasses and CSIDriver retain their identities.
**No** generic `kubectl create namespace` bootstrap, no root App-of-Apps
sync, no cluster-wide CSI adoption or existing PVC mutation.

**Before retry:** inspect exact live namespace presence and operation tasks:

```sh
kubectl get namespace synelia-csi --ignore-not-found -o name
kubectl -n argocd get application synelia-csi-umi -o json | jq '{
  options: .spec.syncPolicy.syncOptions,
  phase: .status.operationState.phase,
  message: .status.operationState.message,
  tasks: [.status.operationState.syncResult.resources[]? |
    {kind, namespace, name, status, message}]
}'
```

After the reviewed fix is merged, refresh only `synelia-csi-umi`
and verify its desired Argo resources include exactly one
`Namespace/synelia-csi`, plus two native `NetworkPolicy`, and
**zero** CiliumNetworkPolicy. Confirm no unrelated namespace with the
same name is already owned by another application; stop and inspect if
one exists. Recheck the WSL2 API endpoint against the native egress policy.
Only then request a normal manual Argo `operation.sync`, `prune=false`,
no force/replace. Observe the per-resource sync result, namespace, ESO
Ready condition, pods/DaemonSet and existing iSCSI attachments; do not
provision any PVC until the CSI is actually ready and independently tested.

## Deployment + read-only diagnostics (AFTER signoff and Argo manual sync)

Use read-only commands (no secrets, no header dumps):

```sh
kubectl -n argocd get application synelia-csi-umi -o jsonpath='{.status.sync.status} {.status.health.status}{"\n"}'
kubectl -n synelia-csi get externalsecret synelia-csi-credentials
kubectl -n synelia-csi get deployment,daemonset,pod
kubectl get csidriver synelia.csi.truenas.io
kubectl get storageclass synelia-nfs-retain synelia-nfs-delete synelia-iscsi-retain synelia-iscsi-delete
kubectl -n synelia-csi logs deployment/synelia-csi-controller -c csi-controller --tail=80
kubectl get pv,pvc --all-namespaces
kubectl get volumeattachments.storage.k8s.io
```

Expect ESO Ready, controller and node Ready, CSI registered, **no pre-existing PV ownership/identity changes**. Do not interpret a Ready pod as filesystem integrity or successful legacy volume adoption. Do not print decoded Secret, authorization header, or high-verbosity API request traces.

## Disposable test protocol (separately approved)

On explicitly new test PVCs in an isolated namespace, test NFS RWX with two safe writer/readers and iSCSI RWO with a single writer. Use a file with a generated checksum, verify it from the remount, restart pods, unmount/remount and inspect attach/detach/VolumeAttachment. Test expansion only after documenting capacity and driver support; verify negative cases (API down/403, network blocked, unreachable iSCSI target) fail closed **without any freshly formatted fallback or deletion of retained volumes**. Verify idempotence across reconciles. For Delete classes, deletion of disposable PVCs is explicitly destructive; record separate authorization and exact test PVC names. Never point tests to old Hermes or Hindsight PV/PVC.

## Legacy CSI adoption is NOT established

The official driver uses `csi.truenas.io` semantics with its own volume identity, while democratic-csi PVs use `org.democratic-csi.*` and embed `volumeHandle`, portal, target/IQN/LUN, CSI attributes and finalizers. Upstream's v1.4.0 chart/README describes dynamic provisioning and snapshots/clones, **not a documented democratic-csi in-place adoption contract**. No code here alters PV attributes; forbid hand-relabeling `spec.csi.driver`, `volumeHandle`, claimRef, finalizers or reusing the same filesystem RW through two drivers. Treat direct static adoption as **unsupported until independently proven**.

For Hermes, #4185 must first identify and preserve the original 80Gi LUN, backup/snapshot, IQN/extent/target, and establish offline ext4 integrity under explicit human authorization. Preferred future cutover: isolated ZFS snapshot/clone + independently verified single-writer copy/restore into a **new** volume, file-level checksums and application-specific consistency gates, with originals retained and a separately reviewed rollback. If feasible static import is later proven against the exact v1.4.0 driver, record an independent proof on disposable volumes before even considering a stateful volume.

## Existing Hermes LUN recovery: isolated retained destination (#4185)

**Operator evidence (2026-10-25):** the old 80Gi Bound/Retain Hermes PV
and PVC still reference `org.democratic-csi.truenas-iscsi`; the old
controller is failing and its VolumeAttachment remains `attached=false`.
The original TrueNAS ZVOL, target #2, extent #2 and LUN0 were identified,
then its iSCSI session was recovered on UMI. An independently cloned
pre-incident snapshot was prepared. The original filesystem was mounted on
UMI at `/mnt/hermes-4185-source-ro` using `ro,noload,nodev,nosuid,noexec`;
the kernel confirmed `ro,norecovery`, and all 13 existing `/profiles`
directories are visible. This is **not** proof of file-level integrity or
ext4 journal safety. A read-only fsck of the recovery clone reported
uncorrected bitmap errors, and the old live ext4 had earlier EIO / JBD2
journal-abort events.

**Observed Synelia success:** a separate 1Gi disposable Synelia iSCSI
test PVC was attached with `synelia.csi.truenas.io` and a 1MiB write +
SHA256 readback passed; the two existing Hindsight attachments stayed true.
This proves provisioning for new volumes, **not import compatibility**.

**Compatibility reasoning:** the pinned upstream driver's
`ControllerPublishVolume` can look up an existing TrueNAS ZVOL by dataset
path and reconstruct its iSCSI target/extent/LUN. See
https://github.com/truenas/truenas-csi/blob/v1.4.0/pkg/driver/controller.go
and `pkg/driver/driver.go`. However the historical PV uses a
`democratic-csi` identity/handle, so it cannot simply be re-labeled to
Synelia. Import would require a new static PV/PVC with the ZFS dataset
path as the Synelia CSI handle, strict RWO exclusivity, and a safe plan for
the damaged original ext4. **Static import has not been tested and is not
the chosen recovery method.**

**New opt-in generic StorageClass:** `synelia-iscsi-stateful-retain`
uses `Retain`, `WaitForFirstConsumer`, ext4, `pool=tank`,
`datasetPath=k8s/vixens/synelia/stateful/iscsi`. Unlike the existing
`synelia-{nfs,iscsi}-{retain,delete}` classes, this parent is **not**
under the disposable `synelia-test` tree and does not overlap any old
`democratic-csi` managed path. It is not default, and no PVC is
automatically created by installing the class. No special hAIrem behavior.

**Physical gate for the separately authorized data copy:**

1. Confirm Argo desired state contains the new class and UMI/manual-only sync
   remains in effect. Sync only `synelia-csi-umi` after PR review/merge,
   without prune or force. Verify all old PVs/attachments remain unchanged.
2. Record original source `findmnt` options: **ro + norecovery**; do not
   unmount/remount it RW or run `fsck` on the original. If the original
   experiences read errors, stop, investigate, and consider the recovery
   snapshot clone. Do not blindly skip unreadable files.
3. Provision a separate 80Gi-or-greater **Retain** PVC in `services`
   using this new class, with an explicitly named new destination PVC.
   Require its PV to have `spec.csi.driver=synelia.csi.truenas.io` and
   `Retain`. Never reuse the old PV name, handle or ZVOL.
4. Transfer **files**, not the raw 80Gi ext4 disk image, from the
   already-mounted read-only source via a tightly scoped, temporary
   recovery Pod on UMI. Preserve numerical UIDs/GIDs, modes, hardlinks,
   symlinks, timestamps, ACLs and xattrs using an appropriately tested
   file-copy tool. Avoid following symlinks or copying external mounts.
   Never grant the copy Pod a Kubernetes API token.
5. Verify a complete zero-error copy, sampled/full checksums, inode/file
   counts, directory identity and 13 profiles; report unreadable entries
   explicitly, not as success. Do not run Hermes on the new PVC until
   recovery copy acceptance is established.
6. Only then switch the Hermes Deployment's PVC through separately reviewed
   GitOps, with automated reconciles still guarded. Keep the old PV/PVC,
   original ZVOL, clone and snapshots under Retain for explicit rollback.

**Non-goals:** no old driver deletion, no raw volumeHandle rewrite, no
retroactive `VolumeAttachment` status patch, no ext4 journal repair on
the original, no prod deployment, no automatic cleanup of the test PVC.

### Hermes UMI manual-only Argo gate before data cutover (#4185; 2026-10-25)

**Operator's explicit integrity decision:** the existing 80Gi historical Hermes
source is mounted on UMI as ext4 `ro,norecovery`; 13 profiles were observed.
The owner reports a successful `cp -a -x` / `sync -f` with exit 0 and has
accepted **sampling-based content validation**, declining a full-tree diff for
time reasons. Full bitwise verification and completeness of copied xattrs/ACLs
remain **unproven**, and this risk must be carried into runtime acceptance.
Original zvol, retained legacy PV/PVC, snapshot/clone and Hindsight iSCSI
attachments must stay untouched for rollback.

**Important race discovered on source inspection:** the current dev Argo
Application `hermes-umi` has `syncPolicy.automated.prune=true,selfHeal=true`
and follows `main`. Merging a new Deployment claim reference without a
prior gate could immediately restart Hermes on a volume still mounted by
the temporary `services/hermes-4185-copy` recovery Pod. RWO does not
mean exclusive access between Pods on the same node, so competing writers
cannot be treated as safe.

1. In a **separate first PR**, omit `automated` from only the
   `argocd/overlays/dev/apps/hermes-umi.yaml` Argo Application.
   Root App-of-Apps applies that child Application change without altering
   the Hermes Deployment or any existing PVC. After review/merge verify
   **on UMI live**: `kubectl -n argocd get application hermes-umi -o
   jsonpath='{.spec.syncPolicy.automated}{"\\n"}'` is empty and no sync operation is
   active. Stop if the real app remains automated. Avoid forced sync/prune.
2. In a **separate, reviewed dev-only cutover PR**, declare existing
   `services/hermes-data-synelia-recovery` PVC at 80Gi using
   `synelia-iscsi-stateful-retain` and explicit
   `volumeName=pvc-66d68670-a146-4ae3-87f6-421860f4554e`, to
   **fail closed** if this actual recovered PV identity is absent.
   Add `Prune=false,Delete=false` on the destination PVC. Change only
   the UMI/dev Deployment `data` claim reference to this PVC, leaving
   the base/old `hermes-data-truenas` 80Gi PVC and any prod source untouched.
   **Never** patch or replace the legacy CSI PV, or allow creation of a
   second empty fallback volume under the same logical recovery claim.
3. **Before manual one-shot `hermes-umi` sync**, verify old/new PVC UID,
   bound PV name, new PV driver/path/Retain, actual content sampled, and
   healthy needed secrets (e.g. `hermes-peer-api-keys`). Finish the recovery
   verification Pod and ensure it is not writing/mounting the RWO target
   simultaneously with Hermes. Keep historical source mounted strictly RO;
   perform no ext4 repair on it.
4. After a **no-prune, no-force manual sync**, verify pod volumes,
   init-container completion and Hermes gateway readiness with the intended
   profiles and workspace intact. Do not announce recovery solely because
   Deployment reports Available; validate intended Hermes user operations.
   Only after signoff consider restoring App auto-sync in another reviewed PR.
5. If this cutover fails, keep both old and new PVC/PV and ZVOL protected;
   pause reconciliation, inspect read-only evidence before any rollback.
   Do not revert to old PV blindly while its CSI external-attacher is failing.
   Explicitly plan a safe data-recovery rollback.

## Rollback boundary

Before physical cutover (this PR's intended scope), stop using new test PVCs, disable/delete only the **new** Argo Application via reviewed Git change; preserve any already created Retain PVs, inspect all CSI finalizers and ZFS resources first. Do not delete old CSI or touch legacy PV/PVCs. Do not uninstall/delete a CSI while *any* PVC it provisioned remains needed; export a fresh inventory first. Keep ESO key until dependent test volumes have been safely detached, then rotate/revoke using OpenBao Infra and documented operator approval. Restoring `hairem-hub-umi` Argo auto-sync and Hermes repair belongs to #4185, not this deployment.

### Evidence status

2026-10-24: source-only preflight against `main` and `truenas/truenas-csi v1.4.0`; the #4194 GitHub CI rendered the pinned Helm chart and Kustomize manifests and passed source safety tests. That is **not** physical evidence: the live UMI App-of-Apps path, Synelia PVC inventory, TrueNAS 27 WSS/API, TLS SAN, host iSCSI, OpenBao token and filesystem checksum tests remain **pending**. Re-run the equivalent CI jobs for each new revision before review.
