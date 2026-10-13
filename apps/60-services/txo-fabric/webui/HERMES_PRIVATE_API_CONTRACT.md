# Strict durable BFF source integration — #3981 / Prompt 13

**SOURCE ONLY, NOT ACTIVATED.** The strict service now uses the #4059
PostgreSQL contract through `server/strict-chat-sessions.mjs`. Deployed
BFF remains health-only; browser and bridge gates remain false. There is
no selected SQL 003 migration, pool, credential, Hermes API listener or
8642 Service/NetworkPolicy exception.

`strictBinding:true` **requires** all durable repository methods:
`createThread/getThread/listThreads/getServerThread/bindHermesSession/claimTurn/finalizeTurn`.
The old `insert/get/list/claim` adapter is isolated in the historical
non-strict source tests. There is no shim, process mutex or fallback.

## Exact state machine

The caller supplies a verified Fabric principal, fresh `agent.chat`
authorization, current canonical TenantBundle/AgentIdentity and UID,
observed generation, independent endpoint attestation and server-only
credential. Scope is immutable and complete:

`{tenantKey, fabricTenantId, tenantBundleUID, fabricUserId, agentKey, agentUID, agentObservedGeneration}`

Creation order (strict path only):

~~~text
fresh verified user/grant/canonical agent/endpoint
  -> createThread({scope}) [durable opaque Fabric thread]
  -> repeat fresh grant/identity/endpoint before privileged upstream
  -> openHermesSession() once, using trusted revalidated endpoint
  -> bindHermesSession({scope,threadId,remoteSessionId}) once
  -> return {id,agentKey,createdAt} only
~~~

If upstream fails, the thread remains unbound. If the remote session was
really created but binding fails or conflicts, the durable thread remains
unbound and the remote may be an **orphan requiring manual reconciliation**.
Return generic unavailable. Never retry/replace/adopt an upstream session
or compensate with a remote deletion without an independently proven
physical contract. Unbound threads do not appear in strict get/list.

Strict get/list must not query by tenant/user alone or by opaque thread
ID without its complete scope. The injected *server-only*
`listCanonicalAgentKeys` enumerates candidate keys; each candidate
requires fresh grant, canonical identity/endpoint pinning and complete
scope before `getThread/listThreads`. Recheck grant and pinned UID,
observed generation and endpoint before the internal
`getServerThread` read for each returned thread. Hide changed, revoked,
unbound and foreign rows, even for a same-named agent.

Turn flow:

~~~text
fresh complete scoped lookup + fresh grant/pinning
  -> claimTurn({scope,threadId,turnId}) [durable cross-replica fence]
  -> claimed: retain leaseToken privately, reauthorize + repin,
              stream trusted Hermes only once,
              finalizeTurn(completed|failed|cancelled) with fence,
              emit terminal status only after confirmed durable finalize
  -> terminal: emit bounded prior {type:"done",status:outcome} only
               (never replay SSE/upstream POST)
  -> conflict|uncertain: safe conflict, never call Hermes
  -> denied|malformed|unavailable: generic fail closed, no Hermes
~~~

A stream ending without a trustworthy terminal event is a failure.
Controlled cancellation/abort finalizes cancelled; upstream errors
finalize failed if possible. If database/process failure prevents safe
finalization, do not fake an outcome, release/reassign a lease, or
automatically replay. Expired/uncertain ownership requires separately
reviewed privileged physical recovery. The same turn ID never authorizes
an upstream retry. The final stream resolver also pins
`agentObservedGeneration`.

The browser may provide only a bounded opaque `turnId` as a
deduplication value; headers/query/body cannot supply canonical
scope, endpoint, remote session, lease token, key, grant or outcome.
Public thread projection is restricted to Fabric
`id/agentKey/createdAt`. No remote session, private Kubernetes topology,
token, credential, error body or internal event data is exposed.

## Source evidence and remaining physical gates

Tests cover observable create/bind ordering, orphan failure, changed
identity/grants, 40 competing turn claims from two **real ephemeral
PostgreSQL pools** wired to the strict session contract, terminal status,
no upstream replay, cancellation/failure and browser metadata injection.
The real PostgreSQL CI remains a dedicated isolated trust-auth test
service, not a live CNPG migration or Hermes call.

Physical prerequisites are unchanged: separately approved SQL 003
migration and least-privilege DML role; selected platform BFF pool;
real Authentik bridge/OpenFGA; independent Pod/EndpointSlice attestation;
Fabric OpenBao scoped credential; operator API 8642 process/service
and narrowly scoped networking; pinned Hermes API/cancellation/session
compatibility and two-agent A/B/A physical acceptance.
**No deployment, cluster mutation, gate changes, merge or promotion**.

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

The repository remains future-only for **production runtime**. The
strict BFF source integration is now described above. The historic
non-strict adapter is for source tests only. No production entrypoint
constructs a chat pool or private runtime route.

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

## Go AgentIdentity typed desired-state preflight (#3981, source-only)

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
There is **no path that sets True or starts the API**. The generated
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

## Thread isolation and streaming (Prompt 13 update)

Strict chat now uses the complete scoped durable repository and fenced
turn journal as documented above. Historic insert/get/list/claim
belongs only to the non-strict source tests. The Hermes transport and
SSE allowlist remain source-only and require physical verification.

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
