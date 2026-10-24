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

## Rollback boundary

Before physical cutover (this PR's intended scope), stop using new test PVCs, disable/delete only the **new** Argo Application via reviewed Git change; preserve any already created Retain PVs, inspect all CSI finalizers and ZFS resources first. Do not delete old CSI or touch legacy PV/PVCs. Do not uninstall/delete a CSI while *any* PVC it provisioned remains needed; export a fresh inventory first. Keep ESO key until dependent test volumes have been safely detached, then rotate/revoke using OpenBao Infra and documented operator approval. Restoring `hairem-hub-umi` Argo auto-sync and Hermes repair belongs to #4185, not this deployment.

### Evidence status

2026-10-24: source-only preflight against `main` and `truenas/truenas-csi v1.4.0`; the #4194 GitHub CI rendered the pinned Helm chart and Kustomize manifests and passed source safety tests. That is **not** physical evidence: the live UMI App-of-Apps path, Synelia PVC inventory, TrueNAS 27 WSS/API, TLS SAN, host iSCSI, OpenBao token and filesystem checksum tests remain **pending**. Re-run the equivalent CI jobs for each new revision before review.
