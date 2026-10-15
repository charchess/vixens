# OpenBao Fabric → Infra Transit seal (preflight only)

**BLOCKED: no runnable seal migration is selected.** This standalone
Kustomization renders only `ServiceAccount/openbao-fabric-seal-bootstrap`.
It is not referenced by any Fabric/TXO base, dev or prod overlay and
has no role, TokenReview permission, projected JWT, Pod, egress or secret.
Existing Shamir/Raft remains selected. No credentials are present.

## Trust/identity contract — source names, never values

- Namespace: `txo-fabric-system`; dedicated SA:
  `openbao-fabric-seal-bootstrap`, exclusive to bootstrap (never BFF,
  operator, ESO or tenant), `automountServiceAccountToken: false`.
- Future **init-only** bound projection: file path `token`,
  `audience: txo-fabric-infra-seal`, `expirationSeconds: 600`;
  no projected token mounted to the running Fabric server.
- Infra `auth/kubernetes/role/txo-fabric-seal-bootstrap` must be bound
  only to that one namespace/SA/audience and a single Transit policy,
  with TokenReview reviewer configured and tested against wrong issuer,
  subject, namespace and audience.
- Exact Infra Transit mount/key:
  `transit/` / `txo-fabric-seal`.
  The ONLY policy grants are:
  `update transit/encrypt/txo-fabric-seal` and
  `update transit/decrypt/txo-fabric-seal`.
  No wildcard, KV read/list, `sys/*`, policy, auth or token creation,
  root/recovery authority or unrelated Infra path.
- Infra Transit is a **seal-unwrapping dependency**, never a Fabric
  runtime KV, ESO store, ExternalSecret or tenant fallback.

## Blocker and future opt-in architecture

The current canonical Infra `ClusterSecretStore/openbao` uses
`http://nas.truxonline.com:8200`. That is NOT a verified TLS endpoint.
This source cannot establish an exact trusted HTTPS endpoint, CA/SNI
or private destination for Cilium egress. No `tls_skip_verify`, HTTP
fallback, broad CIDR/DNS egress or guessed TLS address is allowed.

After **physical Infra HTTPS/CA/hostname/TokenReview evidence** and
reviewed policy/key lifecycle, a separate authorized PR must add the
runtime Pod implementation: hardened init container gets only the bound
JWT, exchanges it over verified Infra HTTPS and writes its narrowly
scoped periodic/orphan Transit token to mode-0600 **memory-backed**
`emptyDir` for Fabric startup. The token must never reach a Kubernetes
Secret, ConfigMap, source HCL, CI, logs, Raft PVC or normal env literal.
The server uses a TLS-verified `seal "transit"` and its
`disable_renewal=false` internal token renewal. Prove real OpenBao
2.6.2 renewal and failure/re-auth lifecycle; **a file sink alone does
not prove the server re-reads renewed tokens**. If periodic renewal
cannot be established, NO-GO rather than a static token.

Kubernetes NetworkPolicy limits a **Pod, not an individual container**.
Only a proved private Infra HTTPS IP/port may be allowed for the
dedicated Fabric Pod, combined with init-only JWT mount and strict
Transit ACL; no BFF/tenant egress and no Internet catch-all.

## Seal migration is not initialization

See [the OpenBao Fabric operational runbook](../../../../../docs/troubleshooting/txo-fabric-openbao-bootstrap-and-restore.md)
for the separate approved migration ceremony, Raft/PVC retention,
downtime, old Shamir shares with `unseal -migrate`, automatic unseal
post-restart acceptance, recovery-key limitation and rollback stop points.

References:
- https://openbao.org/docs/configuration/seal/transit/
- https://openbao.org/docs/concepts/seal/#seal-migration
- https://openbao.org/docs/auth/kubernetes/
