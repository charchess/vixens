# OpenBao Fabric — POC phase 1 and future privilege envelope

The operator is not the secrets administrator. See
[controlled tenant provisioning](tenant-provisioning-boundary.md)
for a canonical TenantBundle → intent → independent boundary → limited
role/ESO sequence. This source-only contract cannot create a real role.

Phase 1 is now explicitly approved and executable: one empty tagged OpenBao
2.6.2 StatefulSet, retained 5Gi Raft PVC, private ClusterIP and deny-all
NetworkPolicy, selected by TXO dev/prod Kustomize. This document continues
to govern **future privileged credential consumers**, not this empty POC.

## Future privileged/runtime object envelope (POC exists)

| Boundary | Contract | Activation blocker |
|---|---|---|
| Workload / namespace | Existing restricted `txo-fabric-system` and the named `openbao-fabric` pod; no tenant clients | Future precise platform/tenant ServiceAccount bindings require separate review |
| Server identity | Official image non-root user, `automountServiceAccountToken: false`, no K8s auth client in phase 1 | Exact future issuer/audience/reviewer binding deferred |
| Stateful data | 5Gi `truenas-iscsi-retain` RWO claim template, Retain/Retain lifecycle, Raft on volume; no Delete fallback | Bind/provisioning observations and #4049 native snapshot/restore still pending |
| Process topology | One OpenBao 2.6.2 server, Raft single-node, ClusterIP 8200, HTTP only while zero network clients, health readiness sealed-aware | TLS, audit, bootstrap and any HA remain separate |
| Network ingress | Exact-selector zero-client policy plus namespace default-deny, no public routing | Future TLS client allowlist separately reviewed |
| Network egress | Zero egress for local Raft one-node POC; no DNS/Internet allow | Future TokenReview/backup/telemetry allowlist based on verified identities only |
| ESO | New **`openbao-fabric-store`** separate from `openbao`; prefer namespace-scoped `SecretStore` identities for tenant consumers; explicit `remoteRef` ACL per immutable tenant ID | ESO auth backend/token reviewer path, name/issuer/audience/CA, role binding and cross-namespace policy |
| Secret content | Never commit Secret values, password, root/unseal/recovery material or example base64 blobs; use explicitly approved external bootstrap channel | Real platform custody/rotation mechanism not yet approved |
| Argo | Selected in existing TXO dev/prod Kustomize desired-state; dev tracks main, prod tracks prod-stable | Physical dev acceptance and explicit prod candidate approval separate |
| Failure / rollback | Startup fails closed without quorum/unseal/credential; remove newly activated overlay with GitOps and retain physical data | Prove storage retention and no data/secret loss on controlled rollback |

### Physical preconditions for trusting real credentials

An authorized operator must supply evidence **without credential
values**: chosen chart/image release and digest; actual CSI StorageClass
and PV reclaim policy; actual TokenReview issuer/audience and exact
ServiceAccount subjects; Fabric OpenBao listening address and private
certificate/CA; network allowlist limited by label + port; backup/snapshot
destination and restore drill; system policy owner and ESO reconciliation
semantics. Re-review ADR-041 and
[runbook](../../../../../docs/troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md)
before making any file renderable.

The empty POC may converge in dev through normal GitOps; a human-authorized
prod-stable promotion is separate. A dedicated reviewed **credential
activation** PR is still required for TLS, audit, auth, ESO and consumers.
Backup/isolated-restore acceptance in #4049 blocks real durable credentials.

### GitOps precedence

The historical `ClusterSecretStore/openbao` is **Infra**, and unchanged.
Existing TXO consumers referencing it remain operationally as-is during
the migration decision, despite their classification debt. A new Fabric
consumer may **not** introduce an Infra store reference. A future migration
must verify access parity + tenant denial and then change the consumer
to `openbao-fabric-store` in a separate reversible GitOps PR; do not
transitively grant Infra token privileges to Fabric workloads.

**No physical bootstrap or activation is approved by this document.**
