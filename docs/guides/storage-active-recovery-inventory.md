# Active storage recovery declarations — source inventory

Scope: bounded source review for [#4386](https://github.com/charchess/vixens/issues/4386), at `main` commit `f439013e730a77a20b3078a974651b5220ea608f` (2026-11-03). This is **not** the complete CSI migration inventory and is **not** runtime acceptance. It records verified GitOps references that cannot be removed merely to make source history look native.

## Classification and safe next action

| Surface / exact source evidence | Classification | Active source consumer / contract | Retirement gate |
| --- | --- | --- | --- |
| `apps/04-databases/postgresql-shared/overlays/prod/kustomization.yaml` includes `recovery-staging/postgresql-shared-recovery.yaml` and `recovery-staging/postgresql-shared-recovery-egress.yaml` | One-shot recovery staging, **still selected in prod render** | CNPG `postgresql-shared-recovery` with `storage-rwo-retain`, 200Gi and backup ID `20261023T020005`; separate from canonical `postgresql-shared` | [#4467](https://github.com/charchess/vixens/issues/4467): confirm live Cluster, consumers, retained PVC/PV identity, backup and restore usability; explicit retire/retain decision and no automatic destructive prune |
| `apps/60-services/txo-fabric/openbao-fabric/overlays/prod/kustomization.yaml` selects `storage/retained-pvc.yaml` and `storage/retained-statefulset-volume.yaml` | Retained **physical binding**, not a generic product interface | Prod StatefulSet patch removes `volumeClaimTemplates` and mounts existing `data-openbao-fabric-0-synelia` instead of newly provisioned `data-openbao-fabric-0` | [#4406](https://github.com/charchess/vixens/issues/4406): verify live claim/PV/UID, recoverability, backup and rollback before any new canonical binding |
| `argocd/overlays/dev/apps/hermes-umi.yaml` omits automated `syncPolicy.automated` intentionally | Stateful operational safety gate (not historical CSI migration code) | Existing Hermes UMI Application targets `apps/60-services/hermes/overlays/dev`; manual Argo sync protects retained data | [#4405](https://github.com/charchess/vixens/issues/4405): physically verify retained data/workload, PVC identity and safe sync policy; do not silently enable prune or auto-sync |
| `apps/60-services/mail/overlays/prod/kustomization.yaml` lists `pvc-canonical-targets.yaml`, but not `canonical-target-bind-jobs.yaml` | Active canonical PVC declaration; named historical Job **not selected by this Kustomization** | Mail prod overlay still selects PVC declarations, which are not equivalent to copy Jobs | [#4407](https://github.com/charchess/vixens/issues/4407): inspect other references and retained credentials/egress before deleting inactive job artifacts; do not delete declared active PVCs |

## Native versus historical

- **Native provisioning policy:** storage classes and declarative workload claims for new installs must not require an old CSI migration Job, recovery provenance or historical status to reconcile. The OpenBao *base* StatefulSet has native `volumeClaimTemplates`; its existing production binding is a separate physical exception.
- **One-shot operational artifact:** PostgreSQL recovery staging is **rendered today**, irrespective of whether a migration label is merely historical. A clean-looking repository cannot be achieved by removing it without the safety proof and approval in #4467.
- **Retained physical identity:** names ending in `-synelia` or `-recovery` can be immutable live references, not evidence that the product is still migrating. Renaming a manifest does not rename the claim or preserve its data.
- **Audit history:** previous migration explanations and tests may remain in dedicated incident/ADR history as history; they must not act as implicit runtime authorization.
- **Safe-to-retire:** this table establishes **no** safe-to-retire active workload or claim. The mail Job's absence from the one Kustomization above proves only non-selection **there**, not repository-wide absence of references.

## Evidence limits and remaining inventory

These source files prove selection and declared names, **not** Argo convergence, pod mounts, PV claimRefs/UIDs, bytes copied, backup restoration, actual network reachability or production state. A `Bound` PVC, label, completed Job or backup ID is not copy-integrity evidence.

Before marking [#4386](https://github.com/charchess/vixens/issues/4386) complete, extend the inventory to every controller/CRD `migration` field, stored-CR compatibility path, AgentIdentity/TenantBundle (including prepare-only agents), RWX/RWO binding, active Kustomize overlay, CI test, workflow, temporary image/Job, snapshot and rollback artifact. Cross-reference [#4384](https://github.com/charchess/vixens/issues/4384), [#4385](https://github.com/charchess/vixens/issues/4385), [#4400](https://github.com/charchess/vixens/issues/4400), [#4357](https://github.com/charchess/vixens/issues/4357) and [#4271](https://github.com/charchess/vixens/issues/4271). No removal or physical validation is authorized by this document.

**Source-only rollback:** revert this documentation. **Data rollback:** not applicable; no stateful resource, reconciliation logic, CRD, manifest, workload or volume is changed.
