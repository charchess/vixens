# ADR-044: native workspace provisioning

**Date:** 2026-11-01 (restored native provisioning)
**Status:** Accepted — replaces the previous per-volume binding proposal
**Scope:** AIaaS
**Related issues:** #4375, #4384, #4385, #4386, #4413, #4357

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
2. **Tenant shared RWX:** the tenant declares workspace scopes, and the
   operator creates canonical PVCs using the declared RWX storage profile.
   New scopes use the same provisioning path on old and new tenants.
   No claim suffix or manual PVC mapping is part of the product contract.
3. **Migration/copy:** an actual future physical data migration is a
   separately reviewed operational operation with its own byte-verification
   evidence. No copy Job, historical source or migration status is consulted
   to mount steady-state agent or workspace claims. A Bound PVC or a source
   provenance label is **not** evidence that bytes were copied.
4. **Runtime mount admission:** an AgentIdentity may mount only its
   declared canonical workspace scopes after checking the actual Bound PVC
   tenant/scope identity, profile StorageClass/RWX and the PV's exact
   namespace/name/UID claimRef plus reclaim policy. Pending or foreign claims
   are not alternate sources; they block the mount until normal dynamic
   provisioning converges. If an existing claim becomes invalid, the serving
   Deployment and human entry are withdrawn without deleting durable PVCs.
   PVC/PV/profile changes requeue the affected tenant agents for validation.
   These checks do not prove any file-copy or data-integrity outcome.
5. **Audit and compatibility:** historical status/CRD fields may remain
   readable temporarily to avoid pruning legacy objects. They are not
   active runtime authorization. Removal from stored API schemas and
   physical retirement require their own compatibility/rollback review.

No product contract contains CSI driver names, NAS addresses, physical PVC
suffixes or transport-specific policies. Historical physical volume identities
are managed separately from the permanent product contract.

## Tradeoffs and risks

- Private agent runtime state may be reset when switching back to a
  canonical claim. This is accepted for disposable AgentIdentity instances;
  other services, workspace data and retained PVCs are not declared
  disposable by implication.
- The historical RWX PVCs still carry data under physical names with a
  suffix. #4400 tracks the separate operational transition to the canonical
  names. This is not a reason to introduce a permanent controller exception.
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
- #4385 — native RWX scope and ownership protection
- #4413 — fail-closed agent PVC/PV verification before mounts
- #4398 — fresh tenant bootstrap
- #4400 — one-time canonical-name transition
- #4386 — migration-only source cleanup
- #4357 — physical verification still outstanding
