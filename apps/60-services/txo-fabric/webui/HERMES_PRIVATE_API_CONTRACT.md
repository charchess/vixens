
## #4037 source-only BFF → private bridge composition (2026-10-14)

The operator already implements the **separate private 8090 listener** gated by
`TXO_FABRIC_PRIVATE_BRIDGE_ENABLED=false`, exact BFF SA subject
`system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff`, Kubernetes
TokenReview with audience `txo-fabric-private-identity`, pinned Kubernetes
issuer/short-lived projected token validation, short HMAC assertion and a
cross-replica create-only JTI Lease. Its read-only resolver consults active
Authentik UUID/group pins, immutable per-tenant *legacy issuer-pinned* Fabric
identity ledger, current generation/store/model and bounded
`OpenFGAAuthorizationReady` reconciliation condition. Denial is generic;
a replay or TokenReview outage fails closed. **No source rewrite of the
ledger or creation of a FabricUserID is allowed.**

The missing BFF **composition seam** is now an explicit
`createVerifiedTenantBridgeAccess` used by `composeGlobalBFF`: the global
WebUI request handler first validates the durable OIDC browser session and
server-owned tenant catalogue, then the adapter demands the exact global
Authentik issuer and user UUID, independently re-checks the Authentik account
for `accountActive=true` and `fresh=true`, and invokes the HMAC/JTI-signed
private bridge client for that canonical tenant scope. Only the operator's
tenant-scoped, already-persisted `usr<32hex>` identity is accepted. A result
is projected as `accountActive/fresh/canEnter=true` solely after both fresh
checks; `canManageTenant=false` by default. **This is a tenant membership,
not an `agent.chat` grant**: the separately reviewed chat action-level
OpenFGA/effective-policy and AgentIdentity ownership check is still required.

`composeGlobalBFF` no longer accepts a generic ad-hoc
`resolveTenantAccess` provider; the bridge-backed seam and explicit
`browserEnabled=true` plus `bridgeEnabled=true` would both be required
before a future server could compose it. **The deployed BFF does not
instantiate this factory.** Actual `private-bff-main.mjs` stays
health-only with `TXO_WEBUI_BFF_BROWSER_ENABLED=false`; operator private
bridge and OpenFGA rollout gates remain false. No Identity HTTP browser
route, URL, Secret, store/model, raw token, credential, key, nonce or
upstream service is exposed by this source change. Physical BFF
session/canary, bridge-to-operator workload identity and Hermes grants
remain separate authorization gates; no real conversation is enabled.

---

# #3981 — Current POC choice: stateless Fabric BFF → Hermes Responses

**This section supersedes the historical native-session and durable-BFF-thread
integration proposals further down.** Electra's read-only pinned-canary
inspection (#3981 comments #6151494831 / #6151593897 / #6152609781)
found Hermes' existing PVC-backed `/opt/data/state.db`, a persisted
`X-Hermes-Session-Key` gateway-session mapping, and a server-owned
Responses named `conversation` chain. The native
`/api/sessions/{session_id}/chat/stream` route **still requires a
session_id**; a header alone does not select that native session. The
**first POC path** is therefore **`POST /v1/responses`**, authenticated
by the per-agent API bearer and using both `X-Hermes-Session-Key` and
`conversation` with the same server-derived opaque key.

```text
browser message, canonical agent selection only
  → signed global OIDC session / issuer-safe identity bridge (OFF)
  → server-verified immutable Fabric user+tenant / fresh FGA agent.chat (OFF)
  → canonical TenantBundle + AgentIdentity UID/generation/status
  → independent private EndpointSlice/Pod ownership attestation
  → pure scoped conversation-key derivation callback
  → read ONLY approved per-AgentIdentity credential
  → POST http://hermes-<agentKey>.tenant-<tenantKey>.svc:8642/v1/responses
      Authorization: Bearer [from trusted BFF reader]
      X-Hermes-Session-Key: [opaque derived HMAC value]
      body: {input, conversation:[same opaque value], stream:true, store:true}
  → Hermes PVC state.db owns conversation/session/history/chaining
  → allowlisted Fabric SSE event projection, no internal IDs or tool payload
```

`server/hermes-responses.mjs` is an independent Responses-only
transport/projection. `createPrivateHermesResponsesService` is the
**future-only stateless** exported factory from
`server/private-hermes-chat.mjs`. It has **no repo, SQL, thread,
lease, remote session ID, remote history cache, `previous_response_id`,
DB connection, automatic retry, or fallback route**. The existing
`createPrivateHermesChatService`, `chat-sessions.mjs`,
`hermes-client.mjs` native-session paths, and #4059's
`postgres-chat-repository.mjs`/SQL 003 remain **dormant historical
source seams**, NOT the first POC transport and NOT selected in prod.
Retained native tests continue to cover their own event protocol;
Responses events use a separate strict projector.

### Conversation identity / authorization boundary

Every call receives **only** the server-verified canonical
`{fabricTenantId, fabricUserId, agentUID, agentGeneration}` to
`deriveConversationKey(scope)`, invoked after fresh issuer-safe
principal validation, grant `agent.chat`, tenant/AgentIdentity
ownership/readiness/observedGeneration and EndpointSlice/Pod attestation.
The callback is **required** and **injected**: no HMAC signing secret,
credential resolver, environment variable, OpenBao auth, Secret or
Kubernetes object is added in this source lot. In production, only a
separately reviewed Fabric-plane **versioned HMAC-SHA256/base64url**
construction with a scoped secret and tested version rotation can
provide a cryptographically unguessable key. Test-only deterministic
digest fixtures **are not a credential derivation scheme**.

Accepted key is an opaque versioned, 43+ base64url-character token,
printable without NUL/control characters, bounded at 256 characters,
different from any canonical raw tenant/user/agent/Kubernetes ID,
API credential or URL. The key is NEVER returned to the browser or
logged. Changing user, tenant, AgentIdentity UID or generation yields
different Hermes continuation history. The first POC has one named
conversation per canonical user+tenant+exact agent incarnation, not a
browser-selectable thread/history index; product-level multiple threads,
history listing and settings are deferred.

New `turn({principal,agentKey,input,signal})` ignores extraneous
untrusted browser `conversation`, `sessionId`, `responseId`, `url`,
`provider`, `model`, history and request headers. Only the canonical
server-side agent key is looked up and independently checked against
TenantBundle, AgentIdentity and exact `hermes-<key>` endpoint. No
fresh grant, missing issuer/bridge, stale AgentIdentity, deleted
identity, failed endpoint attestation, invalid HMAC or absent scoped
credential → **deny before any Hermes I/O**. Invalid HMAC also denies
**before credential read**.

Only `response.output_text.delta`, valid `function_call` item
started/done, and `response.completed|response.failed` are projected
to safe Fabric text/tool-name/status events. Commentary, reasoning,
tool arguments, raw tool output, model/provider details, usage and
upstream response/session IDs are ignored. The client rejects redirect,
non-SSE responses, absent stream body, malformed/big frames, incomplete
streams and missing trusted dependencies generically without logging
tokens or leaking topology.

**Transport abort is not Hermes run Stop**. Browser disconnect
propagates `AbortSignal`; it does NOT prove a Hermes tool/run was
cancelled. `DELETE /v1/responses/{id}` deletes stored Responses
content and MUST NOT be called as a substitute for Stop. No automatic
retry or response replay after an ambiguous streamed POST, no cursor
or `GET/DELETE` Responses behavior in this first slice. Explicit
Stop/reconnect/idempotency semantics require a separately tested
Hermes Runs or equivalent contract and owner approval.

### Deployment and evidence boundary

This PR changes only Node source/tests/docs. The real BFF entrypoint
does **not** construct the Responses factory and still serves
health-only with `TXO_WEBUI_BFF_BROWSER_ENABLED=false`; operator
`TXO_FABRIC_PRIVATE_BRIDGE_ENABLED=false`, FGA store rollout OFF,
Hermes API Server OFF. No private Hermes Service/NetworkPolicy
exception/8642 listener, BFF egress, Secret/ExternalSecret, Fabric
OpenBao credential, browser route, SQL 003 migration or DB pool is
selected. Fake fetch tests prove *source* boundaries, not real
Hermes SSE/state persistence or physical A/B/A conversations.

Activation depends on separately reviewed and physically verified
Fabric-plane HMAC+Hermes API credentials, private cluster Service and
BFF-only NetworkPolicies, endpoint attestation, signed identity bridge,
fresh OpenFGA grants and two-agent A/B/A/revocation/stream continuity
tests on a precisely approved candidate; no public route or production
promotion is authorized by a source merge.

---


## Durable Fabric chat state seam — Prompt 12 (#3981)

`server/postgres-chat-repository.mjs` supplies a **source-only**
PostgreSQL repository, using the **existing platform-only BFF CNPG
database** as its future home (not a tenant database and not a new
stateful plane). The additive
`db/migrations/003_fabric_chat_threads.sql` creates two dedicated
tables and indexes, with `IF NOT EXISTS` and `REVOKE ... FROM PUBLIC`.
**003 is not referenced by the currently selected SQL ConfigMap or
suspended migration Job**, intentionally: this source PR cannot migrate
a database or introduce a live credential dependency. Never edit
applied 001/002. A future separately reviewed GitOps migration must
run 003 and grant only `SELECT, INSERT, UPDATE` on these two tables
to the dedicated BFF role, not DDL, DELETE, TRUNCATE, tenant access or
shared superuser rights. Run migrations only with the platform
migrator, never on BFF startup. Verify backups and data retention.

### Authority and repository API

A real verified BFF resolver, not browser JSON, must supply **all** of:

```text
{tenantKey, fabricTenantId (immutable TEN...), tenantBundleUID,
 fabricUserId (immutable usr...), agentKey, agentUID,
 agentObservedGeneration}
```

This exact binding is validated and included in *every SQL lookup and
state transition*. It is **not** independently an IAM grant: the BFF
must still verify OIDC session, issuer-safe bridge, fresh OpenFGA
`agent.chat` and endpoint/AgentIdentity ownership **before every
repository action**. The repo intentionally rejects absent/unknown
scope and never trusts a browser-provided group, tenant slug, session
ID, secret reference or endpoint. Input must come from the privileged,
independently attested canonical resolver.

The exported `createPostgresFabricChatRepository({pool})` requires a
real parameterized PostgreSQL `pool.query` + `pool.connect` interface.
No credential/pool construction, process Map/mutex or network fallback
exists in the module. Its methods and only their server-side callers:

| Method | Non-secret result / invariant |
| --- | --- |
| `createThread({scope})` | Generate 160-bit opaque Fabric thread ID, persist immutable scope and times before attempting any remote session creation |
| `getThread({scope,threadId})`, `listThreads({scope,limit})` | Project **only** Fabric thread ID, canonical agent key and created time; remote Hermes session/lease are never public |
| `getServerThread({scope,threadId})` | Internal adapter-only remote session metadata; return no record on foreign scope |
| `bindHermesSession({scope,threadId,remoteSessionId})` | One-time conditional `UPDATE` and unique `agentUID+remoteSessionId`; remote session provided **only by a trusted upstream adapter after successful creation**, never from an HTTP parameter |
| `claimTurn({scope,threadId,turnId})` | Row-locked atomic journal + per-thread fencing token, `claimed`, `conflict`, `terminal`, `uncertain` or `denied` |
| `finalizeTurn({scope,threadId,turnId,leaseToken,outcome})` | Only original holder may complete/failed/cancelled; terminal result immutable/idempotent; releases lease transactionally |
| `inspectTurn({scope,threadId,turnId})` | Persisted scoped terminal metadata only, not message content |
| `expireStaleLease({scope,threadId})` | Mark expired work **uncertain**, never silently elect a new owner |

The browser-provided `turnId` is a **bounded 128-bit hex deduplication
value only**, scoped by the server's verified binding and opaque Fabric
thread ID. It cannot grant access or select a tenant/agent. One
PostgreSQL row lock `FOR UPDATE` serializes claims from distinct BFF
replicas; PK `(thread_id,turn_id)` and transactional BEGIN/COMMIT
prevent a SELECT/INSERT gap. A later identical retry returns the
same bounded terminal outcome metadata or `conflict`, and **must never
re-POST upstream or fake/resume prior SSE content**. A claimed lease
not yet finished returns conflict for all other turns on that thread.

**Expiry is not permission to replay.** If a BFF crashes, the
PostgreSQL lease expires but Hermes may still be running. The
repository latches `lease_uncertain=true`, denies every new
claim and keeps the journal/audit state. The still-owning holder can
finalize with its original opaque fencing token. Any forced recovery
and explicit verification that Hermes actually stopped is a **future
separately reviewed privileged recovery API and physical runbook**.
There is no TTL-based auto-release or background cleanup in this PR.
This preserves safety, at the price of manual intervention after a
lost-owner crash. The thread and all turn records stay retained.

### Safe state and production integration boundary

Persisted: immutable tenant/user/agent IDs and UIDs, observed generation,
server-created opaque Fabric thread and remote session IDs, opaque
turn/lease tokens (inside the platform DB), bounded terminal enum
`completed|failed|cancelled`, and timestamps. **Not persisted**:
browser cookies, OIDC/FGA/Hermes/CPA bearer/API credentials, raw user
messages, tool inputs/outputs, SSE contents, HTTP headers, arbitrary
error text, Pod IPs/private topology, issuer subject or audit secrets.
Database query parameters and results are never logged by this module.

This repository is a **new future-only seam**. The historical
`chat-sessions.mjs` `insert/get/list/claim/claimTurn` test abstraction
still opens Hermes sessions before inserting a thread and therefore
**must not be directly substituted with this durable repo**: that
would permit orphan remote sessions on a BFF crash and weaken
transactional ownership. A future **separate integration/activation
lot** must change strict BFF thread creation to
`createThread -> trusted Hermes API session -> bindHermesSession`,
recheck scope/grant at each get/list/claim/finalize, and reconcile
`claimed|terminal|uncertain` results without any automatic upstream
replay. The current BFF stays health-only; this PR intentionally does
not compose either interface at runtime or silently adapt an insecure
legacy interface. Source tests use real ephemeral PostgreSQL via two
independent pools; they prove SQL atomicity, not the live Hermes API.

Remaining physical gates: scoped Fabric OpenBao credential; operator
private API Server Service/NetworkPolicy; real EndpointSlice/Pod UID
attestation; issuer-safe workload bridge; fresh OpenFGA and durable
BFF principal; Agent A/B/A conversation, SSE abort/restart, revocation
and sibling denial after exact candidate approval. No public route,
Kubernetes Secret, ExternalSecret, feature gate or runtime activation
is included.

---
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

## Go AgentIdentity typed desired-state preflight (#4087, source-only)

The operator now accepts an **optional** typed
`spec.runtime.privateAPI` as an intent declaration:

```yaml
# Contract example ONLY — not included in any tenant manifest or prod overlay.
# This does not start the Hermes API process.
runtime:
  privateAPI:
    enabled: false
```

For an *unselected future intent* with `enabled: true`, the API requires
a canonical `tenantId` matching the actual immutable
`TenantBundle.spec.tenantId` and a non-secret `credentialRef` equal to
`hermes-api-<AgentIdentity.spec.agentKey>`. The controller validates
those against the owning TenantBundle, AgentIdentity immutable UID,
generation, deletion state and normal Kubernetes source relationship.
**Neither displayName nor browser-provided tenant slug, Host header,
group, OIDC `sub`, URL, namespace, arbitrary port, Pod/Service name,
credential bytes or wildcard becomes authority.** No optional arbitrary
endpoint or policy is modelled; the future namespace and port remain
platform-owned and externally attested.

The condition `PrivateHermesAPIReady` has `status=False` in **all**
states in this version: `Disabled` when absent/false, `InvalidConfiguration`
for conflicting/incomplete/mismatched declarations and `NotReady` even
for *valid* opt-in intent. Its `observedGeneration` records the source
generation and its message carries no secret or user-provided value.
There is **no path that sets True or starts the API**. Although upstream Hermes
documentation describes an API server alongside `gateway run`, this operator has
not verified that contract against its pinned image, generated entrypoint, or a
private endpoint identity. Environment intent and Deployment availability are
not listener or endpoint attestation. The generated
Hermes Pod/Deployment, dashboard, model credential, PVC, services and
NetworkPolicies remain byte-for-byte unchanged whether the field is
omitted, false, malformed in a fake object or enabled in a *nonselected*
source contract. No API credential lookup or projected volume runs.

Unknown fields are not authorised by this typed API: a strict JSON
input validator rejects them, and the Kubernetes CRD structural schema
defines only `enabled`, `tenantId` and `credentialRef` (with CEL
conditional requirements and immutable field validation). **Kubernetes
may prune unknown fields in non-strict field-validation mode rather than
reject them at admission**; clients/provisioners must use
`fieldValidation=Strict`, and future activation must prove their
strict server-side validation. Never interpret a pruned field as
an approved authority request.

The BFF [private resolver contract](server/private-hermes-chat.mjs)
still requires `apiServerEnabled=true` **plus** independent Kubernetes
EndpointSlice/Pod UID attestation, a ready scoped credential, real fresh
OpenFGA authorisation and both global feature gates. The current Go
`PrivateHermesAPIReady=False` means that assertion cannot be made by
this operator. Before changing that, a distinct reviewed Go/physical
increment must verify pinned Hermes entrypoint/API key parser/SSE
semantics, Fabric-plane secret lifecycle, explicit BFF 8642 policies,
service ownership/selector, revoke/fail-closed behavior and two-agent
A/B/A acceptance.

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

The repo dependency `insert/get/list/claim/claimTurn` MUST be implemented with
**durable atomic, cross-replica** thread storage, turn-idempotency journal and leases before
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
