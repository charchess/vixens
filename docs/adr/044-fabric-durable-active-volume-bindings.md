# ADR-044: native workspace provisioning

**Date:** 2026-10-31 (revised after maintainer clarification)
**Status:** Accepted — replaces the previous per-volume binding proposal
**Scope:** AIaaS
**Related issues:** #4375, #4384, #4385, #4386, #4357

## Context

The retired CSI migration left ordinary reconciles dependent on copy Jobs,
source PVCs, historical statuses and incidental provenance labels. That logic
was useful during a one-time operation, not for permanent operation.

The maintainer explicitly clarified that **AgentIdentity runtimes are
disposable**. This does **not** mean tenant-wide shared business workspaces
or physical PVC/PV resources may be deleted without separate authorization.

## Decision

1. **Private agent RWO:** use the original, canonical
   `hermes-<agentKey>-data` PVC and normal `ensurePVC` provisioning.
   Do not carry RWO migration/attestation/UID-pin state into the permanent
   controller. A disposable agent may restart with a new home; no automatic
   deletion of retained legacy storage is part of the refactor.
2. **Tenant shared RWX:** keep already existing PVC **names**. For historical
   physically suffixed claims, `spec.workspace.claimSuffix` selects their
   literal name suffix, e.g. `synelia`. Empty is the normal, unsuffixed
   provisioning path. A nonempty suffix is **read-only adoption**: verify
   that each claimed tenant/group/user/organization scope has an existing
   Bound RWX PVC, matching labels/class/retention and a PV bound to its exact
   live PVC UID. Never create or relabel an absent selected PVC.
3. **Migration/copy:** an actual future physical data migration is a
   separately reviewed operational operation with its own byte-verification
   evidence. No copy Job, historical source or migration status is consulted
   to mount steady-state agent or workspace claims. A Bound PVC or a source
   provenance label is **not** evidence that bytes were copied.
4. **Audit and compatibility:** historical status/CRD fields may remain
   readable temporarily to avoid pruning legacy objects. They are not
   active runtime authorization. Removal from stored API schemas and
   physical retirement require their own compatibility/rollback review.

No product contract contains a CSI driver name, zvol, NAS address or
transport-specific policy. An existing PVC name with a `-synelia` suffix
is a stable physical resource name and is **not renamed**.

## Tradeoffs and risks

- Private agent runtime state may be reset when switching back to a
  canonical claim. This is accepted for disposable AgentIdentity instances;
  other services, workspace data and retained PVCs are not declared
  disposable by implication.
- A simple RWX claim-name selector is intentionally not a new permanent
  UID registry. Controllers verify the **current** PVC/PV UID binding and
  tenant ownership on each reconcile. Independent restoration/replacement
  of the real physical volume must still be reviewed operationally.
- Existing RWX scopes cannot safely start without the chosen target PVC.
  This is a deliberate fail-closed condition, not an invitation to create
  an empty replacement claim.
- Migration evidence retained in GitHub issues / read-only audits must not
  be used to synthesize a successful copy from a target label or Bound phase.

## Gates

Rollout needs GitOps renders, Go regression/race/vet, schema generation and
real read-only PVC/PV inventory of shared workspace targets (#4357). CI
cannot attest to cluster bytes or Argo image convergence. Neither a merge,
a dev tag nor this ADR authorizes production promotion, PV deletion, or
physical reconfiguration.

## References

- [ADR-043](043-provider-neutral-storage-contracts.md)
- #4384 — disposable RWO simplification
- #4385 — existing RWX claim selection
- #4386 — migration-only source cleanup
- #4357 — physical verification still outstanding
