# TXO Fabric — Wave 2 native-contract inventory (source checkpoint)

**Scope:** issue #4386; source inventory refreshed on 2026-11-03 after merged #4451, #4453, #4454 and #4455. This is a classification and follow-up gate, **not** evidence of live state, data-copy integrity, or a clean-cluster test. Source-of-truth issues: #4387, #4398, #4406, #4357.

| Surface / source | Observed active contract | Classification | Safe next action / gate |
|---|---|---|---|
| `operator/internal/controller/sharedworkspace_controller.go` | Creates canonical RWX PVCs from declared tenant scopes, using SharedWorkspaceProfile's StorageClass; existing claims require matching ownership labels and access mode | **Native product contract** | Keep. #4398 must prove clean-room provisioning, repeat reconcile and cross-tenant refusal |
| `tenants/hairem/tenantbundle.yaml` | Workspace scopes and `shared-nfs` profile, no `claimSuffix` or workspace migration block | **Native desired state** | Keep. Profile name is historical terminology, not in itself a CSI migration gate |
| `tenants/indiba/tenantbundle.yaml` | Workspace scopes and `shared-nfs` profile, no `claimSuffix` or workspace migration block | **Native desired state** | Keep; same clean-room gates |
| `tenants/hairem/tenantbundle.yaml` / `aiGateway` | Only native `profileRef: litellm-standard`; incident retry, failed generation, transition and historical Job UID removed in merged #4454 | **Native desired state (separate LiteLLM lifecycle)** | Preserve source regression `test_fabric_native_ai_gateway_declarations.py`; retained historic status and operator fail-closed remain #4387 |
| `tenants/indiba/tenantbundle.yaml` / `aiGateway` | Only native `profileRef: litellm-standard`; no historical recovery declaration since #4454 | **Native desired state (separate LiteLLM lifecycle)** | Same #4387 terminal evidence/runtime gate; source merge does not attest old Job completion |
| `operator/internal/controller/tenantbundle_ai_gateway_postgresql.go` | Exact UID-bound transition validation and `MigrationEvidenceMissing` fail-closed paths, terminal migration evidence | **Mixed: durable migration lifecycle + exceptional recovery compatibility** | #4387 classify individual branches, preserve replay prevention and retained status. Do not remove the one-shot schema migration contract |
| `openbao-fabric/overlays/prod/kustomization.yaml` | Renders neutral `storage/retained-pvc.yaml` and `storage/retained-statefulset-volume.yaml`; old `migration/synelia-*` filenames no longer active | **Already retired source naming** | #4406 must update stale description; do not redo PR #4417 |
| `openbao-fabric/overlays/prod/storage/retained-pvc.yaml` | Physical PVC `data-openbao-fabric-0-synelia`, class `storage-rwo-retain`, `Prune=false,Delete=false`; migration-source provenance labels already retired in #4429 | **Stateful physical identity + migration audit metadata** | Keep PVC name, class, retention and pruning safeguards. Do not remove protective annotations or rename the physical claim. The native base StatefulSet uses an independently verified volumeClaimTemplate (#4436); switching the existing production overlay to the base would change the claim name and may mount empty storage (#4406) |
| Historical issue/runbook records of old CSI, copy Jobs, PV, snapshots, zvols | Historical provenance rather than ordinary reconciliation authority | **Audit history / rollback evidence** | Preserve in issues/runbooks; do not revive retired controller gates or delete live storage |

## Merged native source cleanup checkpoint

| Area | Verified source result | Still not proven |
|---|---|---|
| CSI migration inputs | #4434 removed ignored runtime/workspace migration Go bindings and CRD spec schemas; #4433 guards new tenant YAML declarations | Kubernetes stored-object compatibility under future CR updates beyond schema pruning |
| Hermes bootstrap | #4431 removed destructive profile cleanup; merged #4451 retired legacy `profiles/<agentKey>` subPath routing and retained `adoptLegacyProfile` as fail-closed, immutable API sentinel | Any **stored** old CR still carrying `adoptLegacyProfile: true` will fail reconcile; no inference of safe physical data move |
| Native Hermes mount | #4437 / #4451: normal runtime uses flat `/opt/data` and canonical PVC; the legacy subPath renderer is gone | Source regression is not a guarantee that an old retained PVC contains a flat Hermes home |
| OpenBao native-vs-retained | #4436 tested a new-cluster native volumeClaimTemplate and explicit retained production claim identity; #4429 removed historical provenance labels | A production overlay transition without empty-data substitution; the physical claim `data-openbao-fabric-0-synelia` is deliberately still active |

**Explicit stored-CR compatibility:** `adoptLegacyProfile` is now an immutable,
fail-closed sentinel in API/CRD, **not an active subPath feature** (merged
#4451). It refuses provisioning and Deployment mutation for stored `true`
rather than redirecting a historical profile to an empty directory. That may
block a live old CR's reconciliation; only a separately reviewed data
transition can resolve it. Legacy audit `status.runtimeStorageMigration` and
`status.workspaceMigration` remain stored history, not native reconcile
inputs. They must not be pruned merely to make the YAML look clean.

## API/CRD and CI classification after native cutover (2026-11-04)

This source classification is checked against the current main API/CRD and
workflow files, not against live Kubernetes objects. It deliberately separates
the archived wire contract from current reconciliation authority.

| Surface | Current consumer / proof | Classification | Disposition |
|---|---|---|---|
| `AgentIdentity.spec.runtime.storage.retentionPolicy` and `TenantBundle.spec.agentStorage` | Native agent PVC provisioning and retention (#4470, #4384) | **Permanent product contract** | Keep; tenant policy required for a new PVC |
| `SharedWorkspaceProfile.spec.storageClassName`, `TenantBundle.spec.workspace` scopes | Canonical organization/group/user shared/skills provisioning, exact mount ownership (#4385, #4413) | **Permanent product contract** | Keep; no source/copy Job dependency |
| `AgentIdentity.spec.runtime.storage.adoptLegacyProfile` (CRD + Go) | Stored CR immutable sentinel; `true` rejected by normal reconcile (#4451) | **Stored-object compatibility** | Keep until verified stored-CR migration; never treat as provisioner opt-in |
| `AgentIdentity.status.runtimeStorageMigration` (CRD + Go) | Archived source/target claim+UID, copy Job and phase; no active controller input | **Historical audit status** | Preserve wire schema, required UID fields, enum and read-only provenance; never treat `LegacyTargetBinding` as byte proof |
| `TenantBundle.status.workspaceMigration` (CRD + Go) | Archived list keyed by `sourceClaimName`, source/target UID, phase and Job evidence; no active controller input | **Historical audit status** | Preserve exact list-map key and historical phases pending stored-CR review |
| `TenantBundle.spec.aiGateway.migration*`, `status.aiGatewayMigration`, LiteLLM PostgreSQL migration Job | Separate one-shot **database schema** lifecycle, not CSI/PV movement (#4387) | **Permanent independent schema migration/incident guard** | Keep under its own fail-closed Job/UID tests; CSI cutover assertions must not delete it |
| `scripts/validation/tests/test_fabric_native_storage_declarations.py` + `.github/workflows/validate.yaml` unconditional offline job | Rejects old CSI tenant fields, foreign profile StorageClasses and retired CSI status as controller routing input | **Permanent source safety guard** | Keep; extend with exact Kubernetes CRD archive contract to prevent accidental status pruning |
| `apps/60-services/txo-fabric/operator/api/v1alpha1/storage_native_contract_test.go` | Checks Go JSON archive shape and absence of runtime spec migration fields | **Permanent API safety regression** | Keep alongside *generated CRD* test; Go JSON alone cannot prove Kubernetes does not prune fields |
| `.github/workflows/synelia-csi-ci.yaml` and `scripts/validation/check_synelia_prod_render.py` | Immutable chart, exact rendered CSI provider and six non-default StorageClasses (#4479) | **Permanent release/source contract** | Keep; render is not a physical copy or recovery attestation |
| `scripts/validation/fabric_rwx_acceptance.py` | Read-only observations for specific retained RWX data/PV identities (#4357) | **Physical acceptance ledger**, not reconciliation | Keep separately; do not claim passing CI proves tenant bytes or live mounts |
| PostgreSQL recovery staging #4467; OpenBao retained PV binding #4406; Hermes UMI manual sync #4405 | Still-selected desired-state resource or stateful safety gate | **Physical-gated operational exceptions** | Keep unchanged until specific acceptance and approved cleanup |

**Evidence semantics:** `Bound` and matching PV `claimRef` prove binding
identity; a completed copy Job can prove its terminal Kubernetes condition;
a status `LegacyRecovered` or `LegacyTargetBinding` proves only the recorded
historical classification, **not** content parity. Byte-for-byte copy or
restoration evidence must be obtained separately. No stored CRD/status schema
pruning, storage deletion, snapshot/zvol retirement or direct cluster action is
authorized by this source audit.

## Required closure evidence

1. **#4387 (LiteLLM, independent):** merged #4453 enables atomic retirement
   of incident requests after verified terminal success; #4454 removed
   hAIrem/Indiba historical requests from portable GitOps. Neither commit
   proves physical terminal history or runtime convergence. Their dedicated
   source guard is wired in CI.
2. **#4398 (clean room):** source tests exercise canonical RWX PVC allocation
   and multi-tenant repeated reconciliation; controller tests also exercise
   LiteLLM attempt-zero creation. A full physical fresh installation and
   data content tests remain distinct evidence, not a prerequisite for the
   generic declarative contract.
3. **#4406 (OpenBao):** native `base/statefulset.yaml` has a retained
   `volumeClaimTemplates` for fresh installs. The **production overlay**
   intentionally selects the *different* retained physical claim
   `data-openbao-fabric-0-synelia`, with `Prune=false,Delete=false` and
   no new volume template. Its physical cutover/restore proof remains
   separate; changing prod to the base would select another PVC.
4. **#4386 (final source sweep):** the existing
   `scripts/validation/tests/test_fabric_native_storage_declarations.py`
   refuses old workspace `claimSuffix`/`migration` and AgentIdentity
   `runtime.storage.migration`/`adoptLegacyProfile:true` across active
   GitOps tenants; PR `test/4386-enforce-native-storage-declarations-ci`
   adds it to the always-running Kustomize validation. This prevents
   accidental reintroduction of CSI incident inputs in future tenant
   declarations without relying on a cluster.
5. **#4357 (physical acceptance):** independently verify retained PVC/PV
   identities, user data, OpenBao health, Argo image and running revision.
   Source/CI success is deliberately not represented as this evidence.

## Surface classification / disposal constraints

| Source surface | Class | Explicit policy |
|---|---|---|
| Agent RWO `ensurePVC`, tenant workspace RWX `SharedWorkspaceReconciler`, provider-neutral StorageClasses | Permanent product contract | Keep native dynamic PVC provisioning and fail-closed ownership, multi-tenant and retention checks |
| Tenant `spec.workspace.migration`, `claimSuffix`, Agent `runtime.storage.migration` | Retired migration-only inputs | Absent from active Go spec and active tenant YAML after #4434; reject their return in GitOps |
| Agent `adoptLegacyProfile` API/CEL | Stored-object compatibility sentinel | Keep readable and immutable; `true` is rejected by runtime, not silently remounted |
| `status.runtimeStorageMigration`, `status.workspaceMigration` | Historical audit status | Preserve existing object/API schema until independently verified safe retirement; not a normal reconciliation gate |
| OpenBao production PVC `data-openbao-fabric-0-synelia`, patch, Retain and anti-prune options | Immutable physical production binding | Keep unchanged until separately authorized, data-preserving transition #4406 |
| Historical CSI Jobs, snapshots, zvols, PVs and bridge records | One-shot operational artifacts / audit and rollback records | Do not infer obsolete physical resources can be deleted from Git source alone; require separate authorized cleanup |

**Non-goals:** No migration API revival, no provider-specific Fabric special case, no hAIrem-specific generic controller behavior, no physical PV/PVC deletion, no direct Argo synchronization, no production promotion. This document is a bounded first inventory, not a declaration that #4386 or Wave 2 is complete.

## Retired Hermes subPath reconciliation (merged #4451)

The legacy subPath, bootstrap full-PVC mount, and special routing were
removed in merged #4451. The API sentinel remains deliberately fail-closed.
The physical state of old CRs is a separate deployment-compatibility gate;
source-only tests do not authorize deleting, copying or remounting data.
