# Issue #3981 — private Hermes API chat vertical slice (SOURCE ONLY)

## Scope and non-activation

This is a **Node source/test contract**, not a runnable Hermes API Server
activation. The existing global BFF runtime serves **health only** while
`TXO_WEBUI_BFF_BROWSER_ENABLED=false`. The operator private identity bridge
remains `TXO_FABRIC_PRIVATE_BRIDGE_ENABLED=false`. OpenFGA store rollout
remains disabled. Neither `apps/60-services/txo-fabric/overlays/prod` nor
`webui/runtime/networkpolicy.yaml` is widened, and no Kubernetes Secret,
ExternalSecret, Job, Service, port, Ingress or API Server environment variable
is created in this increment.

The actual current AgentIdentity generator in
`operator/internal/controller/agentidentity_resources.go` starts
`hermes gateway run --replace`. It does **not** launch a credentialed
Hermes API Server on 8642 nor generate a ClusterIP Service for it. The
runtime image's pinned Hermes source is
`nousresearch/hermes-agent:v2026.9.24`; its documented
`POST /api/sessions` and
`POST /api/sessions/{id}/chat/stream` / SSE API are the **proposed**
adapter surface, not proof of the live runtime, exact event order,
cancellation behavior, credential parser, or API semantics.
The source contract does not assert actual Hermes identity attestation.

## Private BFF request chain

```text
Authenticated global Authentik OIDC session from Fabric BFF SQL
    -> issuer-safe signed BFF bridge (existing #4037, OFF)
    -> immutable tenant Fabric user ID (not OIDC sub, cookie or browser)
    -> current tenant catalog + active membership/FGA store/model/tuple freshness
    -> current per-agent OpenFGA can_chat + effective-policy Check
    -> canonical TenantBundle.spec.tenantId and AgentIdentity UID/generation
    -> exact private service and endpoint identity attestation
    -> one tenant-scoped API key *reference* resolved by trusted server
    -> opaque Fabric user+tenant+agent thread / durable lease (repo injected)
    -> Hermes session owned by server (never client session ID)
    -> allowlisted SSE text/status/tool-name projection to authenticated caller
```

`server/private-hermes-chat.mjs` is the new **strict** composition
factory. It requires both boolean gates to be true, authenticated +
verified Fabric user identity, a trusted per-operation authorization
callback, canonical tenant and agent resolvers, an **independent**
`verifyEndpoint` attestation callback, and a credential reader.
**No runtime production entrypoint instantiates it.** Tests are the
only current providers of those callbacks. Never allow a browser
`X-TXO-Principal`, claimed `verified` flag, `Authorization` bearer,
Hermes URL, Session ID, Pod name, API key or AgentIdentity ID to be an
authoritative value. The current `createGlobalWebUIHandler` already
supplies verified immutable per-tenant `fabricUserId` from the private
ledger; it is a request *seam*, not a deployed IAM backend.

The BFF factory pins the operator's existing naming contract:

- namespace: `tenant-<canonical-TenantBundle-name>` (not browser host);
- API Service **future-only**:
  `hermes-<canonical-AgentIdentity.spec.agentKey>` in that namespace;
- port: TCP `8642`, API root
  `http://hermes-<agentKey>.tenant-<tenantKey>.svc:8642`;
- credential reference **future-only**:
  `hermes-api-<agentKey>` in the **same** namespace; never embed the
  credential in a manifest, call, query parameter or URL;
- exact AgentIdentity UID, tenant bundle UID, observedGeneration
  equals generation, non-deleting `Ready` state, exact tenant ID,
  exact expected service/namespace and `apiServerEnabled=true`
  all required **before reading a credential or making an upstream call**.
- `verifyEndpoint` must verify actual EndpointSlice/Service selectors,
  owner UID, Pod identity and namespace via the trusted Kubernetes
  control-plane path (or approved equivalent workload mTLS), **not**
  DNS string matching alone. It is a required injected dependency, not
  a fake `true` in production. No implementation is installed yet.
- current model does **not** add a field to AgentIdentity's CRD: these
  properties are the future privileged resolver's observed output,
  not client-controlled CR fields.

Future Go operator increment (SEPARATE, requires Go regression and
`go test -race`) must **default API_SERVER_ENABLED=false**, start no
additional process until an **existing, scoped per-AgentIdentity**
credential ref and backend readiness are verified, and create only a
ClusterIP Service and narrowly selected NetworkPolicies for
`txo-fabric-webui-bff` Pod label + dedicated ServiceAccount → port
8642. Operator-owned tenant pod ingress policy must deny direct
browser, other tenant pods, unapproved workload, CPA/OpenFGA and
global network access. The BFF's own current NetworkPolicy has NO
Hermes egress; a later reviewed activation must add only the
explicit agent/tenant port mapping, never a blanket tenant/Internet
rule. The future private credential should come from the **Fabric
OpenBao plane** (#4046), not add another consumer to Infra
`ClusterSecretStore/openbao`. A missing Fabric OpenBao store/key
must mean the API stays OFF and Pods/ExternalSecrets are not generated
from the unactivated source. Do not use a fake key to pass readiness.

## Thread isolation and streaming

`createChatSessionService(...strictBinding:true)` retains the existing
server-assigned remote session design, now additionally persisting
`fabricTenantId`, canonical `agentUID` and `tenantBundleUID`
on its **server-side-only** repo record. The global Fabric principal
holds the canonical tenant-specific Fabric user ID, not an OIDC subject.
Creation, list, get, turn, post-lease revalidation and the *last*
pre-stream target resolve all enforce this binding and a fresh
OpenFGA/per-agent grant. Old non-strict usage remains available for
existing source tests but is **not** an acceptable production BFF path.
A replaced/deleted/moved AgentIdentity is not auto-adopted into a
historical thread. Missing/ambiguous identity or unavailable key or
attestation denies. There is no client-controlled `sessionId`
endpoint in the Fabric chat handler.

The repo dependency `insert/get/list/claim` MUST be implemented with
**durable atomic, cross-replica** thread storage and leases before
any browser activation; today's test fixture is an in-memory fake,
not an installable backend. Cancellation uses downstream abort
propagated to Hermes; cancellation of an *already-running server-side*
Hermes tool/run needs physical API behavior verification. **Reconnect
means retrieving the SAME opaque Fabric thread after fresh grant
verification, not replaying POST /turns and never inventing a second
remote Hermes session.** A future explicit idempotency key and stored
turn outcome is required before automatic POST replay or resumable
SSE cursor semantics; automatic replay is currently **forbidden**.
Upstream response type, redirect policy, bounded JSON (session
creation), bounded SSE event size and allowlist (assistant.delta,
tool.started/completed names, run lifecycle) are tested against a
fake endpoint. No cookies, browser tokens, tool bodies, upstream
errors, Authentik/FGA keys, projected Kubernetes tokens, bridge HMAC,
LiteLLM/CPA metadata or internal topology leave the browser BFF.

## Physical acceptance left to the next approved increment

1. Verify in the pinned image / Hermes runtime how the API Server is
   launched alongside existing gateway/s6, expected API credential
   header, runtime session persistence, SSE event identity semantics,
   tool-run stop behavior and server reboot/resume. If unsupported,
   **do not enable**; revisit adapter.
2. Review Go generator opt-in and gate OFF, per-agent scoped Fabric
   OpenBao key/bootstrap, ClusterIP-only Service, exact ingress/egress
   BFF and tenant policies. Test network refusals physically.
3. Implement private BFF backend resolver with real EndpointSlice
   identity attestation, issuer-safe bridge and fresh OpenFGA
   authorization, durable SQL threads/leases; fail closed when any
   provider is absent. The current deployed BFF must continue only
   serving health until this is complete.
4. Separately authorize gate changes and choose a precise `dev-v*`
   candidate. Physically validate two approved AgentIdentity threads
   from a verified global user, switch A/B/A, revocation, external
   unauthorized user, sibling-tenant keys, network denied paths and
   Hermes restart/abort. Only then consider #3981 acceptance.
5. Indiba provider onboarding is separate #3979; settings/admin,
   BYOK/billing and file upload are not delivered here.

**No merge, promotion, production activation or cluster mutation is
authorized by this source/test contract.**
