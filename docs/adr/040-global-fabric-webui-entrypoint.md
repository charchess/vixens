# ADR-040 — One global Fabric WebUI with trusted multi-tenant routing

**Date:** 2026-10-10  
**Status:** Accepted — owner decision  
**Scope:** TXO Fabric / AIaaS  
**Related:** #3807, #3981, #3996, #3979, #3728, #3729, ADR-039

## Context and product contract

The product owner chooses **one global human product entrypoint**:

```text
https://webui.truxonline.com
  -> authenticate ONE human against Authentik
  -> server returns the accessible tenants/organizations
  -> human selects one permitted workspace
  -> Fabric serves only that tenant's permitted agents/chats/settings
  -> platform admins can access independent platform-admin functions
```

One human can legitimately belong to multiple tenants or serve as a tenant admin
in one and a normal member of another. Neither changing workspace nor switching
agents should require another browser domain, another Hermes account, or a second
login. An unauthorized tenant never appears in the list, and knowing its slug
or a thread ID provides no authority.

This **supersedes** the earlier per-tenant frontend design sketched in #3807
(`hairem.truxonline.com`) and the "stateless UI instance per tenant" draft
in #3981. Those were staging options, not immutable product requirements.
Existing direct per-agent dashboard ingress remains compatibility/debug
transport; it does not become the stable Fabric product interface.

## Decision

- **One global frontend deployment/codebase** on `webui.truxonline.com`.
  `/api/me` provides an authorized tenant/workspace list and independent
  platform-admin capability. The selected tenant is a **requested scope**,
  not a grant.
- **One logical Fabric BFF service**, horizontally scalable, serving the
  same-origin API. It may use multiple **private** privileged Fabric backends
  behind it, but **no per-agent or per-tenant browser BFF is necessary**.
- **One Authentik GLOBAL OIDC application/client** for browser sign-in, using
  public PKCE authorization-code flow and exact canonical callback
  `https://webui.truxonline.com/auth/callback`. A future centrally owned
  provider uses issuer
  `https://<trusted-authentik-host>/application/o/txo-fabric-webui/`
  and requests the signed `txo_fabric_identity` UUID claim.
  Existing tenant OIDC applications and their direct dashboard access
  must stay intact during migration. No redirects accepting arbitrary
  tenant hosts or browser-chosen issuers are acceptable for the global UI.
- The backend authenticates the **global** signed session, then resolves
  the human's **immutable Authentik UUID**, active state, tenant membership
  from server-pinned Authentik group UUIDs and the existing Fabric
  **per-tenant immutable user identity ledgers**. A global `sub`,
  username, email, role or browser group claim **does not become**
  a tenant's `fabricUserId`, and must never silently rebind an existing
  issuer-pinned ledger. A signed global claim alone cannot grant access.
  Design the explicit, reviewed legacy-subject migration before rollout.
- `GET /api/me` lists only current authorized tenants, as computed by
  private server-side resolution with TTL/freshness/active-account checks.
  Tenant-administrator and platform-administrator capabilities are
  independently checked against canonical policy/FGA; a tenant admin
  is **not** automatically a platform admin.
- Chat endpoints are scoped:
  `/api/tenants/{tenantKey}/chat/agents`,
  `/api/tenants/{tenantKey}/chat/threads` and related thread/turn routes.
  Every request revalidates tenant access; the existing chat module
  independently checks per-AgentIdentity `can_chat`, thread ownership,
  tenant store/model and tenant target. No client-provided OpenFGA store,
  authorization model, Kubernetes Service, credential or Fabric user ID.
  The same agentKey can exist in many tenants without collision.
- Frontend switches tenant by remounting tenant-scoped assistant-ui
  runtimes: local drafts, cached threads and conversations cannot bleed
  from one tenant into another. Durable server-side thread ownership
  is separately keyed by immutable Fabric user ID and tenant.
- Settings/admin routes are **not** implemented by reusing `can_chat`
  or `canEnter`. Each needs its own effective Fabric policy +
  fresh OpenFGA/identity check. The frontend must mark these surfaces
  "not yet available" until the real backend exists.
- Infrastructure OpenBao remains **Vixens Core platform-only**.
  It is not the tenant/customer secret vault (#3728 and #3729). Tenant
  CPA/LiteLLM, Hermes/Hindsight, PVCs, files, workspaces, DB credentials
  and provider secrets retain their tenant-scoped lifecycle.

## Architecture

```text
Browser -> webui.truxonline.com (one origin, HTTPS)
        -> SPA static assets
        -> global Fabric BFF (private server-side session + CSRF)
           -> Authentik GLOBAL OIDC sign-in
           -> private Fabric IAM identity/tenant resolver
              -> Authentik: active user and memberships
              -> Fabric: pinned tenant/group UUIDs, immutable user IDs
              -> OpenFGA: per-tenant store, fresh agent/check grants
           -> tenant-scoped chat/session service
              -> per-tenant private Hermes API Server
                 -> tenant LiteLLM -> tenant CPA
           -> privileged platform admin API (separately authorized)
```

BFF request routing MUST fail closed on missing/ambiguous tenant catalog,
expired permissions, unknown users, changed group UUIDs, model/store
mismatch, FGA outage and unauthorized agent/session.

## Rollout phases and non-goals of the initial PR

1. **Source-level global contract:** rename/reuse the existing React
   assistant-ui frontend as global shell; server-declared tenant list;
   multi-tenant route middleware; adversarial tests (PR linked to #3807).
2. **Control-plane Authentik GLOBAL provider + HTTPS login:**
   create a platform-owned global OIDC client with **exact** callback;
   real PKCE/state/nonce/cookie + durable sessions; issuer-aware
   migration of existing per-tenant user mappings. No fabricated
   `principal.verified`.
3. **Real serving BFF:** dedicated Fabric CNPG BFF database + technical
   credentials (platform-only), migration and runtime HTTP service,
   trusted authenticated BFF→operator enrollment transport, private
   NetworkPolicies, global ingress and TLS through reviewed GitOps.
4. **Real authorization/chat:** active tenant membership and AgentIdentity
   grants, FGA model/tuple freshness + revocation, durable chat threads,
   Hermes private API enablement, settings/admin operation-level checks.
5. **Physical acceptance:** authorized user in two tenants, authorized
   tenant admin, platform admin, denied outsider, same agent name in two
   tenants, direct cross-tenant thread attempt, group removal/revocation,
   FGA outage, new tenant C created via TenantBundle. Only promote a
   precise validated `dev-v*` snapshot with explicit human approval.

The first source slice is **not a deployed global login**, operational
admin portal or a completed #3807/#3996. Do not expose a public
ingress for source-only handlers with mocked/provisional auth or
tenant resolver, and do not promote implicitly.
