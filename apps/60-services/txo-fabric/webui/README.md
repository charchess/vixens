## #3981 private Hermes API adapter (source-only, NOT activated)

The tested strict BFF composition for canonical tenant/user/AgentIdentity,
private future-only API port 8642, scoped credential **reference**,
mandatory independent endpoint identity attestation and durable
thread/lease contract is documented in
[HERMES_PRIVATE_API_CONTRACT.md](HERMES_PRIVATE_API_CONTRACT.md).
No Hermes API Server, `ExternalSecret`, Service, NetworkPolicy
exception or browser route is added; BFF/bridge/FGA gates stay OFF.
Tests use only fake private upstream; **no physical chat POC** is claimed.

---

# Prompt 05 / #3807 — PRIVATE BFF substrate (source + GitOps, gated OFF)

**This is NOT a live browser WebUI.** The new Node image serves health only.
No Ingress/Gateway/HTTPRoute, global callback, chat route, login session,
active Authentik credential, OpenFGA activation or BFF→operator identity
resolution is reachable after merge. Both
`TXO_WEBUI_BFF_BROWSER_ENABLED=false` and
`TXO_FABRIC_PRIVATE_BRIDGE_ENABLED=false` remain OFF. The OpenFGA rollout
gate stays OFF as well. The source modules from #4036 and #4037 are composed
only by explicit code with real injected privileged dependencies, NEVER by
a fallback test/mock principal or browser-controlled ID.

## Private runtime and deployment safety

- **One platform workload**, `Deployment/txo-fabric-webui-bff`,
  `ServiceAccount/txo-fabric-webui-bff` and
  `Service/txo-fabric-webui-bff.txo-fabric-system.svc:3000`
  (`ClusterIP` only). The JavaScript HTTP process in
  `server/private-bff-main.mjs` serves `GET/HEAD /healthz` and
  `GET/HEAD /readyz` without inspecting credentials or sessions. ALL
  other `/auth/*`, `/api/*`, chat/settings/admin and unknown paths
  return `404 Unavailable` and `Cache-Control: no-store`. Even an
  accidental feature gate `true` prevents startup until a separate
  reviewed integration is written. No request/header/cookie logging.
- Nonroot UID 10001, read-only rootfs, no Linux capabilities or ambient
  ServiceAccount token, explicit resource budgets and probes, bounded
  header/body/timeouts and SIGTERM graceful shutdown.
- Dedicated projected **short-lived** SA token
  `txo-fabric-private-identity` (3600 seconds) at
  `/var/run/secrets/txo-fabric-bff/token`, without a generic
  `/var/run/secrets/kubernetes.io/serviceaccount` mount. Operator
  TokenReview from #4037 expects exactly
  `system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff`
  plus its configured Kubernetes issuer and audience. Shared HMAC key
  is optional volume `txo-fabric-private-bridge-assertion/hmac-key`
  at `/var/run/secrets/txo-fabric-bridge/hmac-key`; absent by default.
- `server/private-bridge-client.mjs` is the ONLY client for mapping an
  **verified global immutable Authentik UUID** to a historical per-tenant
  `fabricUserId`. It pins operator
  `http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090/internal/v1/identity/resolve`,
  reads the projected SA token and shared HMAC key ONLY when explicitly
  enabled, signs exactly method/path/tenant/UUID/SA/audience/`iat`/`exp`
  (20 seconds)/fresh random 256-bit `jti`, and denies on missing
  configuration, malformed HTTP/JSON, redirect, timeout or 403. All
  errors are generic. Neither global OIDC `sub`, tenant issuer,
  client-chosen Fabric ID, user header nor OpenFGA store appears in the
  signed request. The operator's durable atomic Lease nonce CAS guards
  replay; the BFF never retries the same assertion.
- `runtime/networkpolicy.yaml` has **no BFF ingress** and only DNS 53,
  CNPG `postgresql-shared` 5432 and the exact operator workload 8090
  as BFF egress. Shared CNPG pods are isolated **by role/database**
  credentials, not by TCP port (the PostgreSQL cluster itself is shared).
  Existing operator ingress policy accepts the BFF pod's exact
  `app.kubernetes.io/name` + `app.kubernetes.io/component` labels.
  No BFF Internet, direct Authentik, OpenFGA, Hermes, CPA or broker path.
  Kubelet health probes are not exposed via ingress.
- `webui/kustomization.yaml` is included in `overlays/prod`
  and renders the private runtime plus an immutable representation
  of the **original** SQL 001/002 as a generated ConfigMap. The SQL
  ConfigMap contains schema, not passwords. No public routes are added.

## Database ownership and deferred bootstrap

`webui/bootstrap/` is a **separate, UNREFERENCED Kustomization** and
does not appear in `overlays/prod`. It stages the CNPG
`DatabaseRole/txo-fabric-webui-migrator` (DB owner / DDL role),
`DatabaseRole/txo-fabric-webui-runtime` (session runtime **DML-only**),
`Database/txo-fabric-webui` on `postgresql-shared`, four separate
platform-only ExternalSecret references, and a **suspended** job
`txo-fabric-webui-schema-v1`. This is intentional: no real technical
credentials are provisioned and the migration MUST NOT accidentally
execute after source merge. The job needs a separate migrator credential
(`txo-fabric-webui-migrator-runtime`); the BFF runtime would use only
`txo-fabric-webui-bff-runtime`, both from a separately reviewed
OpenBao platform record. No shared database superuser credential or
tenant OpenBao dependency.

Only after human-reviewed bootstrap, apply the two existing idempotent SQL
files **in order**, then `REVOKE PUBLIC` and `GRANT CONNECT`, `USAGE`,
`SELECT, INSERT, DELETE` on **only** the two BFF tables to the
runtime role. The migration is never done by the serving process.
Job `suspend: true` must be replaced by an approved activation change
and its actual success verified; using a migration credential for runtime
would violate the security contract. Persisted OAuth transactions and
session-cookie hashes remain platform-only. This is *not* tenant
credential storage; OpenBao is the **infrastructure** technical
bootstrap service, not a tenant vault.

## Image provenance / validation / next gate

`Dockerfile` stages Node 22 in a minimal nonroot runtime, no app secret
or client frontend. `.github/workflows/txo-fabric-webui-bff-image-ci.yaml`
runs `node --test`, source checks, `git diff --check` and builds +
publishes GHCR image with a source-commit `src-<sha12>` tag and an
observable OCI digest. **Reviewer must pin the verified SHA256 image
digest in GitOps**; CI does not merge, tag Git, promote or deploy a
candidate. The image is health-only even when a Kubernetes Service exists.

Future activation (separate reviewed change, never this PR):
1. Register/reconcile CNPG roles/DB only after an approved private
   technical secret exists, run/verify suspended migration and ensure
   PostgreSQL ingress NetworkPolicy allows **only BFF/migrator** from
   `txo-fabric-system` (currently not proven).
2. Verify actual Authentik account source, global OIDC issuer/callback,
   current TenantBundle catalogue, pinned per-tenant identity ledgers,
   model/membership readiness leases, operator TokenReview issuer,
   HMAC distribution and private NetworkPolicy behaviour. No browser
   route until all integrations are real, audited and fail-closed.
3. Only a separate authorization grants/chat increment #3981 may
   enable per-AgentIdentity operations and physical hAIrem/Indiba/A/B/C
   acceptance. Ask for human approval of an **exact dev-v*** candidate
   before any production promotion.

Rollback: revert the private runtime inclusion in the production
overlay or scale down via *GitOps*, keeping both gates OFF. Do not
delete CNPG databases, PVCs, Fabric retained identity ledgers, or
secret sources in order to roll back an unactivated BFF. No direct
cluster mutation or Argo forced synchronization is authorized.

---

# Fabric WebUI — global browser/BFF decision (ADR-040)

## Global browser login + durable session implementation (#3807, staged)

`server/global-oidc-login.mjs`, `server/global-postgres-sessions.mjs`
and `server/global-auth-http.mjs` now form one **real source-level**
AuthentiK browser login flow for the global app (ADR-040):

- `GET /auth/start`: redirects to the **pinned global Authentik**
  `/application/o/authorize/`, generating cryptographically random
  PKCE S256/state/nonce and a Secure+HttpOnly+SameSite=Lax+host-only
  temporary cookie. The transaction persists through the existing
  PostgreSQL adapter (#4032) with private `tenantKey=webui` namespace.
  `webui` is not a customer TenantBundle or Fabric grant.
- `GET /auth/callback`: requires that original browser binding,
  atomically consumes transaction, exchanges public-client authorization
  code with its PKCE verifier, validates the actual RS256 Authentik
  ID-token signature/audience/global issuer/nonce and the signed immutable
  Authentik UUID via the existing verifier. A REQUIRED *privileged*
  Authentik active-account read must additionally confirm identity
  and freshness. Only then can a persistent browser session be issued.
- `db/migrations/002_global_browser_sessions.sql`: stores a SHA256
  **digest of a random 256-bit cookie**, issuer, immutable Authentik UUID,
  opaque global `sub`, and fixed 8h expiry in a private Fabric BFF
  PostgreSQL DB. No raw cookie, access token, client API key, tenant
  `fabricUserId`, group or permission is stored. `authenticate()`,
  `revoke()`, bounded expiry cleanup use parameterized SQL and deny
  on outages; two BFF replicas use the same durable session state.
- After callback, redirect to `/` with **no code/state** in the URL.
  `__Host-txo-fabric-session` is Secure/HttpOnly/SameSite=Lax/Path=/.
  `POST /auth/logout` checks exact Origin and custom CSRF header
  before deleting the session digest and expiring the cookie.
- `createGlobalWebUIApp` composes this session verifier with the
  existing global `GET /api/me` tenant discovery and per-operation
  tenant-scoped chat router. An OIDC login by itself **never** grants
  any Fabric group, tenant, agent, admin or OpenFGA permission.
  Existing issuer-pinned tenant identity ledgers are NOT mutated.
- The global React shell has a login link to `/auth/start` when
  session-based workspace discovery fails.

Tests exercise real RSA-signed JWT OIDC claims with fake IdP HTTP,
invalid nonce/issuer/audience/UUID/expiry, bad cookie, callback replay,
CSRF logout, session isolation and SQL adapter expiry/outages.
**A mocked Authentik and SQL-contract simulator in CI are not proof of
physical Authentik/CNPG readiness.**

**NOT DEPLOYED / NOT OPEN TO THE INTERNET:** Production still needs
an authenticated trusted active-account resolver and issuer-safe
per-tenant canonical user lookup/enrollment (operator-led),
`TenantBundle` catalog, fresh Authentik pinned-group + OpenFGA checks,
dedicated CNPG DB/role/migrations and credential lifecycle, real
Node HTTP app image/Ingress/TLS/NetworkPolicy, durable chat sessions
and private Hermes API. The global OIDC provider staged in #4035 is
not confirmed live and no prod promotion is authorized. Do not wire
these modules with fake callbacks. Infrastructure OpenBao may supply
*only platform bootstrap technical secrets* and is never a tenant
credential vault.

---

## Platform-global Authentik OIDC provider (staged, not activated)

The **global** browser client is now declared in the existing Authentik
platform `ConfigMap/authentik-config` as `txo-fabric-webui.yaml`, and
mounted into the Authentik worker at
`/blueprints/vixens-fabric-webui.yaml` by GitOps. It is independent from
the operator-generated `txo-fabric-tenants.yaml` and exists even if there
are no tenants. Application slug/client ID are `txo-fabric-webui`.
The recommended per-application OIDC issuer is
`https://authentik.truxonline.com/application/o/txo-fabric-webui/`.
The provider is a **public authorization-code-only** client with
one `strict` redirect URI:
`https://webui.truxonline.com/auth/callback`. It includes the
signed `txo_fabric_identity` scope with immutable
`txo_fabric_user_uuid`. It creates **no human users, group membership,
tenant grants or Fabric identities**.

The **Fabric BFF** must use server-owned authorization-code + PKCE S256,
single-use state, nonce, exact issuer/audience, bounded token exchange,
strict callback host/cookie and account/tenant authorization checks.
Authentik's public OAuth provider may support PKCE without *requiring*
it for all third-party code requests; do not assume provider-side PKCE
enforcement. The Fabric server must reject callbacks not bound to a
BFF-created state/PKCE transaction regardless of provider behavior.

**Runtime gate:** a merged blueprint source and successful Kustomize
checks do NOT prove that the global provider exists in live Authentik.
The new blueprint will only be applied when the exact desired-state
revision is promoted under the existing GitOps process. Before login
activation, validate live issuer/discovery/JWKS, the strict redirect,
`txo_fabric_user_uuid` signed ID-token claim, and absence of token
or unauthorized metadata leaks. The existing tenant-specific issuer
ledgers MUST NOT be silently migrated or replaced. There is **no
global HTTP BFF session or ingress** in this PR, so an end-user cannot
log in to the product yet.

---



**Status:** source-only prototype, not deployed; `webui.truxonline.com` does not yet serve this application.

One **global** React/assistant-ui frontend has now been adapted for server-scoped
multi-tenant navigation. A **single logical Fabric BFF** routes between separately
authorized tenant chat services; runtime Hermes/LiteLLM/CPA and sensitive
credentials remain tenant-scoped.

```text
Browser -> webui.truxonline.com (global SPA + same-origin BFF)
           -> Authentik global OIDC (TO PROVISION)
           -> Fabric secure session / tenant resolution (TO IMPLEMENT)
           -> GET /api/me [only authorized tenant contexts]
           -> /api/tenants/<tenantKey>/chat/*
                -> Fabric global route middleware (source only)
                -> existing tenant-scoped chat-http and session service
                -> freshly checked agent grant / OpenFGA
                -> private Hermes -> tenant LiteLLM -> tenant CPA
```

**Existing per-tenant issuers remain in Authentik** for v0 Hermes dashboards.
A global OIDC issuer cannot simply be substituted into existing per-tenant
immutable subject/issuer ledgers. Authentik group memberships are sourced
from Authentik, not from a browser/group claim or fabricated grant.

`server/global-webui-http.mjs` includes strict global issuer, tenant
catalog, role, active account and freshness validation around the existing
chat handler. The required privileged authenticated callbacks are **not
implemented/deployed**, and MUST never accept a user-supplied tenant
principal or silently grant `verified`. The global handler itself is
not a production-ready authentication server.

**Still missing:** global Authentik app/client, real login/session and BFF
server deployment, secure service-to-service IAM enrollment, tenant FGA
grants/refresh, durable chat thread and transaction stores, secret
and NetworkPolicy boundaries, private Hermes API enablement, deployment
and physical acceptance. Admin and settings are NOT live.
`TXO_FABRIC_OPENFGA_STORES_ENABLED` remains disabled. No production
promotion is implied.

---

## Existing internal Hermes chat/OIDC components

The first implementation provides a **small, dependency-free, server-side transport contract** that forwards only whitelisted user-safe Hermes API Server events. It deliberately does not implement an HTTP server, browser authentication, a provider key store, React UI or Kubernetes manifests.

```text
Future assistant-ui React shell
  -> authenticated Fabric BFF (TO BUILD)
    -> verified principal / tenant and AgentIdentity authorization (TO BUILD)
    -> streamAuthorizedTurn (this package)
      -> pinned Hermes API Server /api/sessions/{session_id}/chat/stream
        -> existing tenant LiteLLM -> tenant CPA -> provider
```

## Checks

Run `node --test` from this directory with Node >= 20. No npm install or secrets required.

## Security invariants

- Never build the Hermes URL from a browser parameter; the backend must resolve the authorized AgentIdentity to a private Kubernetes Service, with server-side credential material.
- The `authorize` callback must be backed by an actual verified Authentik/OIDC principal and grants, not user-supplied JSON. Re-check on every agent/session operation; ownership of Hermes session IDs must be tracked separately by the BFF.
- `streamAuthorizedTurn` rejects denied principals, missing authorization dependencies and cross-tenant target mismatches before a provider request.
- Only `.svc` / `.svc.cluster.local` targets are accepted here as defense in depth; NetworkPolicy egress must additionally prevent SSRF. This is **not** a substitute for DNS/IP constraints and backend controls.
- Tool arguments/results, raw Hermes errors and non-product commentary are not forwarded to the client. Sensitive details must remain server-side.
- Error/cancellation/stream finalization semantics need integration and security testing when the BFF/API are wired.

## Blocking integration steps

1. Verify the exact Hermes v2026.9.24 API Server is present and compatible in the pinned runtime. Its upstream docs show this API, but **Fabric currently starts the gateway/dashboard and does not enable `API_SERVER_ENABLED` or supply `API_SERVER_KEY`**.
2. Add a private, Secret-backed Hermes API Server and an authenticated Fabric BFF with session-level authorization, per-user/tenant isolation and explicit NetworkPolicies. No browser → Hermes direct bearer.
3. Build tenant WebUI shell with `assistant-ui` (selected for a reversible prototype); connect using a Fabric chat API, not directly to Hermes. Node transport is not the whole chat backend.
4. Test persisted session ownership, stop/cancel, agent replace, tools, reconnect, SSE compatibility and hAIrem multi-agent acceptance before considering #3981 done. Indiba remains blocked on voluntary CPA provider enrollment through #3979.

Do **not** add a public Ingress for the Hermes API port or enable it globally before the BFF and policy enforcement exist. No prod promotion implied.

## Second slice: conversation ownership and tenant chat HTTP API

This branch adds `server/chat-sessions.mjs`, `server/chat-http.mjs` and
`test/chat-sessions.test.mjs`:

- Cryptographic opaque Fabric thread IDs mapped server-side to Hermes sessions.
- Authorize tenant, authenticated user, and agent on every read and turn.
- Require a durable conversation repository with atomic per-thread turn leases;
  no permissive/in-memory repository is bundled as a production fallback.
- Portable Web Request/Response routes for thread creation, list, lookup,
  and streaming turns, with origin checks and redacted HTTP errors.
- Unit tests for denied cross-user/tenant access, revocation, CSRF and SSE.

These are integration primitives, **not** a deployed BFF: an actual
Authentik/OIDC verifier, Fabric IAM grants, durable session repository and
Secret-backed private Hermes API Service/NetworkPolicies are still required
before mounting this handler at a public tenant host. The React assistant-ui
frontend is not yet included. The static per-agent LiteLLM key must not be
mistaken for the end-user identity used by future personal OAuth routing.

## Central OpenFGA authorization adapter (#3992)

The optional server-only module `server/openfga-authorize.mjs` implements
the pre-existing `authorize({principal,tenantKey,agentKey,action})` callback:

- `agent.chat -> can_chat`; `agent.manage -> can_manage`.
- Sends only stable Fabric user/agent identifiers and a **backend-resolved**
  tenant store + pinned authorization model ID to the **private** centralized
  OpenFGA service. No consumer-controlled URL, store ID or token.
- Requires `principal.verified` set by an **authentic** server-side Authentik
  verifier and a stable Fabric subject mapping. A forged `verified: true`
  client input is **not** authentication.
- Cross-tenant resolver mismatches, missing model/store, FGA failure, malformed
  response and all upstream errors deny access.
- Unit tests cover accepted and rejected checks without contacting cluster.

This is an **integration seam only**: the actual Authentik verifier,
`resolveTenantStore`, `resolveCanonicalAgent`, store/model provisioner,
tuple reconciliation, and API key injection from OpenBao are not wired to a
running Fabric BFF yet. Do not expose the handler publicly based on a successful
unit test. See #3988, #3990, #3992 and ADR-039.

## Signed Authentik identity proof (staged #3996)

The source-only `server/oidc-identity.mjs` verifier is the first trust-boundary
primitive for future **automatic verified human enrollment**. It is **not** a
running OIDC login flow or an enrolled human. A privileged Fabric BFF must:

1. Resolve the tenant slug, expected HTTPS issuer and audience from its
   **server-owned TenantBundle**, never from request JSON, Host or headers.
2. Own OAuth authorization-code/PKCE, state, nonce and a single-use login
   transaction; exchange the code server-side. Only pass the resulting **ID
   token** and the transaction's server-stored expected nonce to the verifier.
3. Require the Authentik provider to publish a dedicated, signed
   `txo_fabric_user_uuid` ID-token claim from Authentik's immutable
   `request.user.uuid` (via a reviewed OAuth2 scope mapping). The operator now generates the custom
   `txo_fabric_identity` scope mapping (#4025); its live publishing and
   exact signed-token behavior must still be validated before activation. Do not infer an Authentik UUID from hashed
   `sub`, username, email, groups or workspace `userRef`.
4. Use the returned immutable proof (exact issuer, opaque OIDC subject and
   Authentik UUID) only inside an authenticated Fabric-owned enrollment
   backend. It must check the live Authentik account status and immutable
   tenant binding, allocate/reuse a stable Fabric user ID, persist via CAS into
   the retained private ledger, and handle revocation/re-enrollment safely.
   No browser may choose a Fabric user ID or write the ledger.
5. Resolve the fresh canonical identity from that ledger and enforce IAM
   freshness, active membership and OpenFGA Check before marking a principal
   verified or authorizing any chat operation. This verifier returns **no**
   `principal.verified` field, Fabric ID, grant, cookie or session.

The verifier uses an exact tenant issuer, the provider's discovery document and
same-provider JWKS, RS256 signature checking, a strict audience and nonce,
short enrollment-time validity, bounded HTTP replies and no redirects.
`test/oidc-identity.test.mjs` covers forged claims, mismatched tenants,
expired tokens, changed group/account identifiers and key/discovery failures.
This code has **no runtime wiring, GitOps rollout flag change or production
promotion**. Runtime callback, scope mapping, durable identity enrollment,
account lifecycle, freshness/revocation and physical A/B/C checks remain open
under #3996 and ADR-039.

## Single-use Authentik OIDC login coordinator (staged #3996)

`server/oidc-login.mjs` now composes the existing signed-ID-token verifier
with the **real Authentik authorization-code endpoint** and a server-owned
OAuth2 PKCE S256 transaction. It always requests exactly
`openid profile email txo_fabric_identity` and uses Authentik's shared
`/application/o/authorize/` and `/application/o/token/` endpoints, not
the per-provider issuer as a token URL. The issuer, audience and redirect callback are derived from **trusted
TenantBundle** configuration and the shared Authentik origin is pinned by
Fabric platform configuration (not the browser). The callback
host must match its Authentik-generated pattern
`https://<app>-<tenant>.<domainSuffix>/auth/callback`.

The coordinator returns an HTTPS authorization URL plus the parameters for
a **Secure, HttpOnly, SameSite=Lax, Path=/, host-only `__Host-` binding
cookie**. It requires a separate **durable cross-replica** transaction
repository with atomic insert and *atomic consume before any exchange*,
not an in-process Map. A five-minute transaction holds hashed browser
binding, 32-byte state/nonce, and PKCE verifier. Browser state and code
cannot select an issuer, client, store, tenant, user, token endpoint or
redirect. Failed cookie/nonce/account/ledger/enrollment and reused
transactions deny without granting permission. The ID token is checked
with same-issuer JWKS and the original nonce; only the resulting signed,
immutable Authentik identity fields reach the privileged enrollment callback,
never the code, cookie, PKCE verifier, access token or arbitrary headers.

**Still a source-only integration seam, not a deployed authentication
service:** the mountable HTTP BFF and its strict host+callback+cookie
handling are absent. The transaction store and session store are not
provisioned. The `enrollVerifiedHuman` callback is intentionally
unimplemented at runtime: its future Fabric-only transport must authenticate
the BFF workload, bind the tenant to the service identity, and call the
operator-owned CAS writer with a live Authentik account check. A forged
`enrollVerifiedHuman` implementation would compromise the boundary;
the current module is not safe to mount with arbitrary callbacks. Successful
login returns an identity **without** setting `authenticated` or
`verified`; every chat, integration and agent operation still requires
fresh account/IAM state and an OpenFGA Check.

Tests: `node --test` includes PKCE challenge vs verifier, original nonce,
exact scoped endpoints, tenant-specific host validation, cookie binding,
single-use/replay and concurrent callback, expired/mismatched transactions,
invalid tokens, upstream outages, refusal and cross-tenant isolation.
Do not enable the OpenFGA rollout flag, create a public BFF ingress or
promote production on the strength of these synthetic source tests.

## PostgreSQL-backed replay-proof OAuth transaction store (staged #3996)

`server/oidc-postgres-transactions.mjs` now provides the actual
`transactions.insert/consume` dependency for the existing
`createFabricOIDCLogin` coordinator, using a **platform-owned PostgreSQL**
connection pool instead of a node-local Map:

- A 32-byte browser state is **SHA-256 digested** before persisting; table
  primary key is `(tenant_key, state_digest)`, with **no upsert**.
- `DELETE ... WHERE tenant_key = $1 AND state_digest = $2 RETURNING ...`
  atomically burns the transaction on **exactly one** replica, including
  if cookie validation, token exchange or enrollment subsequently fails.
  PostgreSQL's atomic row deletion, not a simulated browser test, is the
  concurrency primitive. Query parameters prevent SQL injection; failures
  are redacted and fail closed.
- The database holds only short-lived state (nonce, PKCE verifier, cookie
  binding **digest**, approved issuer/client/redirect, creation/expiry),
  never the original state, access tokens, sessions, Fabric identities or
  OpenFGA grants. A fixed five-minute TTL is enforced by a SQL CHECK and
  BFF coordinator. `pruneExpired()` removes **at most 100** expired
  transactions per call; schedule it through the future Fabric-owned BFF
  maintenance loop or another reviewed platform component. DB clocks must
  be synchronized with BFF clocks.
- SQL migration: `db/migrations/001_oidc_login_transactions.sql`;
  apply from a reviewed GitOps database migrator with exclusive schema
  rights, **not** during a browser request. Require a *dedicated Fabric BFF*
  database, encrypted PostgreSQL backups, least-privileged BFF role with
  only `SELECT/INSERT/DELETE` on this table, and private NetworkPolicy.
  Do not reuse OpenFGA CNPG credentials or provision a second independent
  Authentik/OpenFGA source of truth.
- Node tests cover a SQL-contract simulator with two independent pool
  adapters, consume-once vs replay, tenant separation, digest storage,
  malformed/forged state, TTL and bounded pruning, corrupt DB rows and
  outages. **These are not physical PostgreSQL tests** and do not prove
  real migrations, runtime NetworkPolicy or cluster readiness.

This is an implementation of the **durable storage adapter and its schema**,
not yet its Kubernetes rollout. No PostgreSQL database/role, BFF Deployment,
migration Job, secret, ingress, real authenticated BFF→operator enrollment
transport, session service or authorizer has been created. The Fabric BFF
must instantiate this adapter with **Fabric-platform-only PostgreSQL
credentials**, delivered through a reviewed Kubernetes secret lifecycle,
and verify the migration/DB/network preconditions **before** exposing
`/auth/start` or `/auth/callback`. The storage backend for BFF platform
credentials is a deployment choice: do **not** make the infrastructure
OpenBao/ESO path a mandatory tenant-facing runtime dependency. Until the
platform backend and BFF are deployed, **no end-user OIDC login is active**;
#3996 stays open.

## OpenBao trust-domain boundary (non-negotiable)

**Vixens Core OpenBao is the infrastructure/platform secret store. It is
not the TXO Fabric tenant/customer/user credential vault.** Its existing
OpenFGA/PostgreSQL and Authentik platform service credentials are
control-plane bootstrap material; a Fabric BFF technical database
password, if stored there, is likewise a platform credential, never a
customer credential. Tenant Hermes runtimes, tenant-admin users and
Fabric customer interfaces must not receive Core OpenBao API access,
root tokens, secret paths, or a Kubernetes `ClusterSecretStore/openbao`
reference allowing client data to enter that trust domain.

The accepted target in [#3728](https://github.com/charchess/vixens/issues/3728)
is a **separate IAaaS OpenBao deployment** with its own lifecycle,
storage, policies and backups, used behind the non-revealing Fabric
Credential Broker [#3729](https://github.com/charchess/vixens/issues/3729)
for tenant/user/integration credential data. This is a separate
**v0.2** workstream and must not be smuggled into #3996 (v0.1)
as a requirement for BFF user authentication.

The OAuth transaction table introduced here is **short-lived BFF
control-plane session state**, not tenant provider secret storage.
Its server-side tenant keys are isolation metadata, not an OpenBao
namespace nor an entitlement for tenants to administer the shared
database. The BFF must remain a trusted Fabric boundary and may never
copy a customer's upstream OAuth provider credentials into Vixens Core
OpenBao. Existing tenant CPA provider credentials retain their
tenant-local lifecycle; #3979 governs their future self-service UI.

Before deploying the BFF: choose a scoped **platform-only** CNPG role and
secret-generation/rotation mechanism, bind it solely to the trusted BFF
workload through service identity/RBAC/NetworkPolicy, and keep tenant
runtime access denied. A CNPG role/Secret created through a secure
platform-local generator may satisfy this without any new OpenBao
application record. **Do not create a credential in Core OpenBao merely
because the BFF stores a tenant key in a PostgreSQL row.**
