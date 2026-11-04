# ADR-043: Provider-neutral Kubernetes storage contracts

**Date:** 2026-10-25
**Status:** Accepted
**Scope:** Core
**Related Project:** vixens core
**Related Issues:** #3959, #4205
**Deciders:** Vixens maintainers
**Tags:** storage, csi, kubernetes, contracts, retention

## Context

The first official TrueNAS CSI staging deployment exposed StorageClasses named after
its implementation (`synelia-nfs-*`, `synelia-iscsi-*`). Those names leak a driver
and transport into workload manifests, proliferate backend-specific variants, and
make a future provider change appear to require application changes.

Issue #3959 established the required platform direction: upper layers consume a
small explicit catalogue of capabilities rather than NAS paths, protocols, provider
names, or historical CSI identities. The initial rollout validated the official
CSI in parallel with the retired provisioner. That rollout is historical, not
an ongoing prerequisite for the current provider-neutral classes. Existing
bound PVs retain their original CSI identity and cannot be adopted by
changing a class name.

## Decision

Core publishes exactly these non-default StorageClass contracts wherever the
platform can satisfy them:

| Contract | Access guarantee | Reclaim policy |
|---|---|---|
| `storage-rwo-retain` | single-writer RWO | `Retain` |
| `storage-rwo-delete` | single-writer RWO | `Delete` |
| `storage-rwx-retain` | shared RWX | `Retain` |
| `storage-rwx-delete` | shared RWX | `Delete` |
| `storage-rwo-xfs-retain` | single-writer RWO, XFS filesystem | `Retain` |
| `storage-rwo-xfs-delete` | single-writer RWO, XFS filesystem | `Delete` |

The XFS pair is a deliberate filesystem-qualified RWO extension, not a replacement
for the ext4 default RWO contracts. It remains published because active consumers
(Home Assistant, Trilium, AdGuard Home, Actual Budget, mail storage, Music Assistant
and Frigate) explicitly request it. A later removal requires migration of every
consumer; it must not be silently aliased to ext4.

For the current official TrueNAS CSI implementation, `storage-rwo-*` is realised
through iSCSI with `WaitForFirstConsumer`; `storage-rwx-*` is realised through NFS
with immediate binding. The XFS pair is also iSCSI with `WaitForFirstConsumer`.
Those are implementation bindings, not application-facing names. All six classes must remain non-default, support expansion only when the
underlying driver configuration does, and use isolated driver-managed parents.

`Retain` is the contract for durable state. `Delete` is limited to explicitly
reprovisionable/disposable data. A workload must not encode `synelia`, `truenas`,
`iscsi`, `nfs`, NAS endpoints or backing ZFS paths in its storage class selection.

The Fabric operator's compatibility with an **already bound** retained claim
whose immutable StorageClass reflects its earlier runtime profile is not a
fresh provisioning path. Its RWO reconcile must verify exact namespace,
claim name, PVC UID, PV `claimRef` UID and StorageClass before it can maintain
that existing data. A foreign, unbound or incompletely identified PV fails
closed without metadata mutation or retroactive adoption. All tenants
provisioning **new** Fabric agent PVCs must declare `spec.agentStorage` with
explicit provider-neutral retained and disposable RWO classes. Stored tenant
CRs lacking this field can continue reconciling verified *existing* claims.
A Bound Retain PVC with an exact tenant/agent identity, PVC UID, PV claimRef
and matching immutable PVC/PV StorageClass remains valid even if a later
Hermes runtime profile declares a different StorageClass. An unbound claim
or an incomplete/foreign PV identity fails closed; the current runtime
profile cannot authorize replacement of preserved state. An absent PVC
must fail closed rather than inherit the Hermes runtime profile's StorageClass. This protects API-created tenants as well as GitOps
manifests, without recreating historical stateful data.

No existing PV, PVC, CSI `volumeHandle`, volume attachment, zvol, dataset, target,
extent, NFS export, or workload is renamed, rebound, adopted, or migrated by this
decision. Historical `retired-csi` and Synelia staging classes remain for the
PVs already bound to them. Each consumer migration creates a separately reviewed
replacement claim and follows an exclusive, verified data-copy and rollback gate.

## Source validation and operational boundary

The production CSI Helm release must render exactly one official `CSIDriver`
and the six non-default StorageClasses above. Each class must bind to the
declared production CSI provisioner, matching access protocol, filesystem,
binding mode and reclaim policy. This is checked on the **rendered objects**,
not by finding class names inside a YAML string. A credential `Secret` must
not be materialized by the chart.

These offline checks verify GitOps declarations only. Copying, restoring or
retiring pre-existing PVC/PV data is bounded maintenance and needs its own
reviewed procedure and explicitly approved desired-state change. No source
test, completed Job or PVC `Bound` phase is evidence of byte integrity or
permission to delete a stateful volume.

## Consequences

### Positive

- Applications consume stable access/reclamation guarantees instead of a storage
  provider implementation.
- The class catalogue has six observable, testable entries rather than classes per
  transport, application, dataset, environment, or recovery scenario.
- The official CSI driver can evolve behind the contract without rewriting new
  workload manifests.
- Historical storage remains safe because class creation does not alter bound PVs.

### Negative

- The provider-neutral name does not itself prove performance, snapshots,
  `ReadWriteOncePod`, encryption, topology, or availability; those require separate
  documented guarantees before being claimed.
- Existing workloads continue to carry legacy class names until individually
  migrated; this is intentional compatibility debt, not a reason to mutate bound
  volumes.
- A cluster that cannot honour one of the contracts must fail provisioning
  explicitly; it must not silently substitute another class or backend.

## Alternatives considered

### Keep `synelia-*` as the public contract

Rejected. `synelia` is a deployment/driver identity, not a capability requested by
an application. It would force provider-specific vocabulary into every consumer.

### Create one class per backend, dataset, application, or stateful use case

Rejected. Backend and data-instance details belong to Core bindings and PV/PVC
instances. They do not define a reusable Kubernetes storage guarantee.

### Rename or rebind legacy PVs in place

Rejected. A StorageClass is not a textual alias for a bound PV. Changing CSI
identity, handles, IQNs, or claim bindings risks data loss and has no supported
adoption contract between `retired-csi` and the official driver.

## References

- #3959 — platform layers and provider-neutral contract direction
- #4205 — official TrueNAS CSI parallel staging
- `apps/01-storage/synelia-csi/` — current official driver binding
- `docs/reference/platform-layer-contracts-proposal.md` — originating proposal
