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
| Workload / namespace | Name future instance `openbao-fabric`, in a distinct Fabric-controlled management namespace; platform and tenant read clients are not bootstrap administrators | Namespace owner, Pod Security, SCC/Kyverno and versioned chart/image unknown |
| Server identity | Dedicated minimal ServiceAccount, `automountServiceAccountToken: false` by default; any auth helper token is separately projected with exact verified audience | Real issuer/audience/token reviewer, bound SA and precise auth method unverified |
| Stateful data | Dedicated PVC/PV selected by inspected StorageClass with **Retain** reclaim policy, immutable snapshot/restore/backup contract; **no delete reclaim fallback** | Physical CSI class, backup target, capacity, RPO/RTO and successful restore drill absent |
| Process topology | Private ClusterIP-only HTTP/TLS listener; locked init/unseal, audited admin boundary, securityContext nonroot/read-only where image supports it | OpenBao version/Helm configuration, exact TLS, HA/raft storage and initialization method undecided |
| Network ingress | Namespace/workload default-deny; platform-approved AuthN clients/ESO per namespace and explicit health/metrics sources only | Actual ingress labels, pods, ports, CA chain and endpoints require observation |
| Network egress | Default-deny: only **exact** DNS (if needed), storage/backup/Kubernetes TokenReview/telemetry endpoints separately justified; no broad Internet, Authentik, tenant-to-tenant network or admin ingress | Exact bootstrap versus runtime links not yet established; **do not allow 0.0.0.0/0** |
| ESO | New **`openbao-fabric-store`** separate from `openbao`; prefer namespace-scoped `SecretStore` identities for tenant consumers; explicit `remoteRef` ACL per immutable tenant ID | ESO auth backend/token reviewer path, name/issuer/audience/CA, role binding and cross-namespace policy |
| Secret content | Never commit Secret values, password, root/unseal/recovery material or example base64 blobs; use explicitly approved external bootstrap channel | Real platform custody/rotation mechanism not yet approved |
| Argo | Separate `openbao-fabric` non-prod activation app/overlay **only after** these choices; later human-approved prod composition | No production Argo target changes in #4046 |
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
