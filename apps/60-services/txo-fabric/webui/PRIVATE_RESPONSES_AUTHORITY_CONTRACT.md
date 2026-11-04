# Private Responses authority contract

## Purpose

This is the production-only authority contract between the Fabric WebUI BFF and
TXO Fabric operator for the stateless Hermes Responses path. It is deliberately
separate from `browser-authority/resolve`: that endpoint is a presentation
contract and must never disclose an endpoint, credential reference, store/model
binding, namespace, workload identity, or runtime status.

The contract exists so the BFF can serve an authenticated human while keeping
all browser input non-authoritative and preserving the same generic
`TenantBundle`/`AgentIdentity` lifecycle for hAIrem and any later agent lot.
It does not initiate a data migration and must not contain source/cutover/copy
state.

## Boundary

- Listener: the existing operator private listener only; no public Service,
  Ingress, tenant route, or browser-callable endpoint.
- Caller: exactly the `txo-fabric-webui-bff` ServiceAccount, authenticated by
  the existing projected-token, audience, HMAC assertion and single-use JTI
  pattern.
- Requests: canonical tenant and agent *keys* selected by the BFF only after
  its verified global OIDC session and fresh tenant access check. A request
  never contains a Kubernetes URL, service address, namespace, secret name,
  credential, user-supplied Fabric ID, agent UID, generation, model, store or
  conversation key.
- Responses: fixed JSON shapes, bounded sizes, no raw Kubernetes errors,
  Secret data, bearer tokens, OpenFGA identifiers, pod IPs, tool payloads or
  topology outside the BFF-to-operator private boundary.
- Every failure is a generic deny. A bad assertion, replay, TokenReview issue,
  stale identity, ownership drift, unavailable policy check or API read error
  performs no Hermes request.

## Required operations

### `resolve-tenant`

Input: `{ tenantKey, fabricTenantId }` from the server-owned tenant catalogue.

The operator returns only the immutable TenantBundle UID and readiness facts
needed by the Responses adapter. It must verify that the requested key and
`TEN…` ID are the same live TenantBundle, that it is not deleting, and that its
ready/observed state is current.

### `resolve-agent`

Input: `{ tenantKey, fabricTenantId, agentKey }`.

The operator resolves exactly one live AgentIdentity and returns the canonical
agent UID, tenant owner UID, generation/observedGeneration, readiness and the
fixed private API capability facts. It must reject an AgentIdentity that moved
tenant, is deleting, stale, not ready, has private API disabled, or whose
controller-derived API credential reference does not exactly equal
`hermes-api-<agentKey>`.

### `authorize-chat`

Input: canonical Fabric user ID already obtained from the verified tenant
identity bridge plus `{ tenantKey, fabricTenantId, agentKey }`.

The operator rechecks the effective OpenFGA `agent.chat` decision against the
canonical resolved AgentIdentity. Tenant membership is not an agent grant.
This operation returns a boolean allow only; it does not return store/model IDs
or tuples.

### `verify-runtime`

Input: the exact canonical tenant/agent identity obtained above.

The operator independently verifies the controller-owned Service,
EndpointSlice and ready Pod identity for the expected private Hermes API port.
It returns only `verified: true`; it must not return an address or endpoint.
The BFF derives the fixed in-cluster service DNS name locally from canonical
keys only after this operation succeeds.

## Credential and conversation-key boundaries

The private authority endpoint never returns Hermes API credential values.
The BFF may read only a controller-issued per-AgentIdentity credential through
a separately reviewed, namespace-scoped mechanism after `verify-runtime` and
before one Responses POST. Likewise, the versioned HMAC conversation-key
provider is BFF-owned and receives only immutable `{fabricTenantId,
fabricUserId,agentUID,agentGeneration}` scope; no browser value can choose or
read it.

## Activation acceptance

Before either BFF gate is enabled, tests must show: canonical allow; revoked
`agent.chat` deny; cross-tenant/key collision deny; AgentIdentity replacement
invalidates prior continuity; stale/deleted runtime deny; endpoint mismatch
deny; missing/wrong credential deny before Hermes I/O; and a two-agent A/B/A
conversation proof through the authenticated global WebUI. The same proof must
run against a second generic tenant fixture. No hAIrem data migration is part
of this contract or its acceptance criteria.
