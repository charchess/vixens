# Fabric WebUI — Hermes chat transport (initial implementation slice)

Status: **internal prototype; not deployed; not a usable WebUI yet.** Related #3981, #3832, #3807 and UC-005 in `/usecase/`.

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
