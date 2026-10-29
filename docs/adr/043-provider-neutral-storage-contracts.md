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
names, or historical CSI identities. The official CSI driver is now physically
validated in parallel with `retired-csi`; existing PVs retain their original CSI
identity and cannot be adopted by changing a class name.

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

No existing PV, PVC, CSI `volumeHandle`, volume attachment, zvol, dataset, target,
extent, NFS export, or workload is renamed, rebound, adopted, or migrated by this
decision. Historical `retired-csi` and Synelia staging classes remain for the
PVs already bound to them. Each consumer migration creates a separately reviewed
replacement claim and follows an exclusive, verified data-copy and rollback gate.

## Consequences

### Positive

- Applications consume stable access/reclamation guarantees instead of a storage
  provider implementation.
- The class catalogue has four observable, testable entries rather than classes per
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
