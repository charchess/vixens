# OpenBao Fabric retained volume — compatibility and transition gate

**Tracking:** #4406, #4357, #4386, #3728. **Status:** source-only plan; NOT a production procedure authorized for execution.

## Two distinct storage contracts

| Scenario | Desired-state source | Effective data claim | Safety property |
|---|---|---|---|
| New cluster built from `openbao-fabric/base` | `base/statefulset.yaml` | StatefulSet `volumeClaimTemplates` produces `data-openbao-fabric-0` | Kubernetes/CSI provisions canonical RWO Retain; no historical snapshot, copy Job or source UID required |
| Existing production overlay | `overlays/prod/storage/retained-statefulset-volume.yaml` and `retained-pvc.yaml` | Explicit `data-openbao-fabric-0-synelia` | Preserves a physical data identity with `Prune=false,Delete=false` and avoids silent volume substitution |

The base and overlay **must not be interchanged solely to clean up a name**. A manifest is not evidence that the backend contains the correct OpenBao Raft data, that a snapshot can be restored, or that the secret plane can auto-unseal.

## Preflight evidence — read-only only

The maintainer/operator carrying out future acceptance must record *values that identify resources*, not secrets:

1. Exact Git `prod-stable` revision, Argo application sync/health and deployed OpenBao Fabric StatefulSet image and pod status. Do not infer current production state from `main`.
2. StatefulSet `spec.volumeClaimTemplates` or explicit `volumes[].persistentVolumeClaim.claimName`, actual pod `/openbao/data` volume binding, namespace and claim name.
3. PVC UID, StorageClass, status, PV name; PV `spec.claimRef` namespace/name/UID, volume handle, reclaim policy and CSI driver. Check that both observed and desired resources select the *same intended* claim.
4. Read-only health/seal condition, availability and safe restore/backup evidence. Never print tokens, root keys, unseal shares, lease payloads or secret values; a healthy pod alone is not a data-integrity proof.
5. Snapshot/backup evidence with retention and a tested independent restore target. A retained PV is not a backup. If any evidence is missing, the transition is blocked, not implicitly approved.
6. Inventory any pending canonical-name PVC `data-openbao-fabric-0`; distinguish a new empty or pending claim from the active retained one. Do not delete, bind, patch or overwrite either resource.

This guide deliberately contains no mutation commands and no assertion that the inventory or restore rehearsal has happened.

## Non-destructive decision table

| Observed state | Allowed GitOps decision |
|---|---|
| Fresh cluster, no existing data/claim | Use native base and its `volumeClaimTemplates`; independently test bootstrap/unseal path |
| Production uses retained `data-openbao-fabric-0-synelia` | Keep existing prod overlay and prune safeguards; record identity and backup evidence |
| Canonical `data-openbao-fabric-0` exists but contains no proven restored data | **Stop.** Never switch the production StatefulSet just because the native template exists |
| Verified independent restore/data transition with exact PV/PVC identities, rehearsed rollback, explicit human authorization | Draft a separate stateful GitOps transition candidate and review its manifest diff; still no automatic promotion |
| Any mismatch in claim UID, PV claimRef, health, unseal or restore evidence | **Stop, preserve existing data and observed bindings.** Investigate under #4406/#4357 |

## Candidate and rollback requirements

A future candidate needs: (a) before/after rendered Kustomize manifest diff, showing exact claim identity, retention and workload changes; (b) test of fresh-cluster native provisioning and of restart with existing retained data; (c) data integrity and restore evidence that does not disclose secret values; (d) a separately authorized downtime/cutover strategy; (e) verified rollback that accounts for both desired-state Git and physical storage. **Git revert alone cannot reverse data written to a new volume.**

No automatic deletion of the historical PV/PVC/snapshot, no Argo forced sync/prune, no production promotion and no namespace-specific operator exception are authorized here.

## CI guardrails and scope

Source regressions in `scripts/validation/test_openbao_fabric_source.py` verify the base native StatefulSet template and the retained overlay's exact claim binding (#4436). They cannot verify live backend bytes, current pod mounts, backups, seal state, or external storage array metadata. `ADR-044` governs native Fabric tenant/agent provisioning separately; this OpenBao physical identity is not a general product storage policy.
