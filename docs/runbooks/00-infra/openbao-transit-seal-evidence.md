# Infra Transit seal evidence gate

**Issue:** #4071
**Status:** evidence-only; this document does not authorize installation,
selection, rollout, or migration.

## Purpose and boundary

This gate records the non-secret evidence required before a future, separately
approved startup seal-unwrapping change can be considered. It applies only to
Infra custody for OpenBao Fabric startup. It does not create a Fabric runtime
secret authority and does not authorize a Fabric `SecretStore`,
`ExternalSecret`, tenant path, client, or fallback to Infra.

The canonical legacy ESO binding remains
`apps/00-infra/openbao/base/cluster-secret-store.yaml` and is intentionally
outside this contract. Its current plaintext address is not evidence for this
gate and must not be repurposed.

## Evidence record requirements

Store evidence in the approved private operational record. Commit only the
record identifier, review date, responsible Infra custodian, and PASS/FAIL
outcomes; never copy credential, JWT, certificate, key, request, response, or
encoded material into Git, CI, or chat.

### 1. Exact private HTTPS trust record

The operational record must identify one approved private Infra HTTPS endpoint
by its non-secret reference, including its exact hostname/SNI, port, CA trust
reference, certificate identity result, and private route result. The record
must show strict certificate validation succeeded for that exact hostname.

Record PASS only when all are true:

- the route is private and the approved destination and port are exact;
- the server certificate chains to the approved CA reference and matches the
  exact SNI/hostname;
- availability and failure behavior were observed without an alternate
  plaintext, public, or broad-egress route;
- the legacy ESO endpoint was not treated as the startup seal endpoint.

### 2. Kubernetes-auth identity record

Record the exact Infra auth role as
`auth/kubernetes/role/txo-fabric-seal-bootstrap` and show its only bound
subject is
`system:serviceaccount:txo-fabric-system:openbao-fabric-seal-bootstrap`.
Record the approved audience `txo-fabric-infra-seal` and projected-token TTL
`600` seconds.

Evidence must show TokenReview and login acceptance for that exact projected,
Pod-bound identity. It must also show denial for a wrong service account,
namespace, issuer, audience, expired token, and revoked identity. Ordinary ESO
identity must not obtain this authority.

### 3. Transit authority record

Record the approved Transit key reference `transit/txo-fabric-seal` and the
policy-review identifier. Capability evidence must show the only allowed
operations are:

- `update transit/encrypt/txo-fabric-seal`
- `update transit/decrypt/txo-fabric-seal`

Record denial outcomes for another Transit key, KV access, list, policy or auth
management, token management, and administrative operations. The source example
in `apps/00-infra/openbao/transit-seal-contract/` is a review artifact, not a
policy installer.

### 4. Renewal and outage record

Record non-secret results proving the issued periodic or orphan startup token
renews through its intended lifecycle on OpenBao 2.6.2. Demonstrate that expiry,
renewal failure, Infra unavailability, and reauthentication failure fail closed;
no retained server state, source file, environment literal, Kubernetes Secret,
or Fabric runtime path may receive the credential.

Record a controlled Infra restart/unavailability exercise and recovery result:
the startup dependency must either recover through the approved reauthentication
path or remain sealed. It must never use a runtime KV or ESO fallback.

## Evidence decision

The approving Infra custodian signs one PASS/FAIL result per section and records
open failures. A PASS here is evidence only: it does not select this example,
install an auth role or policy, change the legacy ESO binding, create a network
rule, or authorize a Fabric migration. Any later implementation requires a new,
explicitly approved change and its own source and physical validation.
