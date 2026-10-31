# ADR-044: Fabric durable active-volume bindings

**Date:** 2026-10-31
**Status:** Accepted
**Scope:** AIaaS
**Related Project:** vixens roadmap
**Related Issues:** #4375, #4384, #4385, #4386, #4357
**Deciders:** Vixens maintainers
**Tags:** txo-fabric, storage, lifecycle, stateful, csi

## Context

ADR-043 defines provider-neutral Kubernetes storage classes. It intentionally does
not define how Fabric identifies an already-existing stateful volume after a
one-time storage migration.

Fabric currently derives selected claims from CSI migration input (`cutover`, a
suffix and historical copy-job/source evidence). That makes ordinary steady-state
reconciliation depend on an operation that is no longer running. It also risks
confusing a Bound claim and provenance labels with copy-integrity proof.

At the `224462a46df2` baseline, source inventory shows thirteen hAIrem RWO
migration declarations, of which three declare `cutover: true`; all four Indiba
RWO declarations declare `cutover: true`. Both hAIrem and Indiba declare RWX
workspace cutover; Indiba additionally retains a copy-job prefix. This is a
source inventory, not runtime proof of bound PVC/PV identities or copy integrity.
Physical inventory remains a read-only acceptance gate in #4357.

## Decision

Fabric has three separate storage domains:

1. **Permanent active binding.** A finalized AgentIdentity or workspace scope
   must declare the exact active PVC identity contract in durable desired state:
   claim name, observed PVC UID, owning Fabric identity/scope, storage class,
   access mode and retention policy. Reconciliation selects only this binding.
   It validates each invariant and fails closed on absence, UID replacement,
   class/access/retention mismatch, ownership mismatch or tenant/scope mismatch.
   It never creates a replacement claim while a finalized active binding is
   missing or invalid.
2. **One-time migration/copy.** Source claims, replacement suffixes, copy Jobs,
   bridge claims and byte-integrity evidence authorize only a new migration.
   They remain fail-closed until genuine UID-bound copy evidence exists. A Bound
   target or `LegacyTargetBinding` proves neither copied bytes nor integrity.
3. **Historical audit.** Immutable source/target provenance and verified copy
   records remain available for audit and rollback analysis, but are not an
   input required by normal finalized reconciliation.

The durable contract is provider-neutral: it contains no CSI driver, transport,
NAS path, dataset, volume handle or implementation name. Existing physical PVC
names, including names ending in `-synelia`, remain stable identities and are not
renamed by this decision.

The contract is implemented in the AgentIdentity and TenantBundle APIs in the
sequential #4384 (RWO) and #4385 (RWX) increments. It must be expressed in the
specification rather than status alone so a lost status record cannot authorize
adoption of a same-named replacement. A controller may populate only verified
status/audit observations; it must never silently rewrite the declared binding.
Prepare-only agents and scopes continue to use the existing migration gate until
they are separately finalized through verified copy evidence.

## Consequences

### Positive

- Normal reconciliation becomes independent of expired Jobs, retired source PVCs
  and migration status while preserving the exact current stateful claim.
- Lost status, a recreated claim and forged historical provenance all fail closed.
- RWO and RWX share one identity/ownership/retention model without collapsing
  their distinct access and multi-consumer rules.
- The audit record remains meaningful without being misrepresented as runtime
  copy authorization.

### Negative

- Finalizing existing volumes requires a read-only runtime inventory and an
  explicit desired-state binding update; source manifests alone cannot invent a
  PVC UID.
- A missing or replaced active claim intentionally degrades the workload rather
  than provisioning an empty substitute. Recovery requires an explicit, reviewed
  restore or migration action.
- Transitional migration fields remain until #4384 and #4385 have proved their
  respective finalization contracts; #4386 may only remove paths after that.

## Alternatives considered

### Keep selecting the suffix target when `cutover: true`

Rejected. The selector remains coupled to migration history and cannot distinguish
an intended active claim from a same-named or newly provisioned empty claim after
status/source evidence is lost.

### Use status as the only active identity record

Rejected. Status can be absent or lost; it cannot be the sole authorization to
adopt a stateful replacement. Desired state must pin the intended identity.

### Treat a Bound target with migration labels as copied data

Rejected. Labels establish provenance only. Copy integrity is a separate,
one-time evidence contract and cannot be inferred from claim binding.

### Rename migrated claims to remove implementation-shaped suffixes

Rejected. PVC names are live physical identities. A cosmetic rename would create
a new storage lifecycle and violate the preservation requirement.

## References

- [ADR-043](043-provider-neutral-storage-contracts.md) — provider-neutral class guarantees
- GitHub #4375 — parent contract and source/runtime acceptance requirements
- GitHub #4384 — RWO implementation increment
- GitHub #4385 — RWX implementation increment
- GitHub #4386 — deferred transition cleanup
- GitHub #4357 — physical acceptance ledger
