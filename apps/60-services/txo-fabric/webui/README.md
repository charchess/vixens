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
