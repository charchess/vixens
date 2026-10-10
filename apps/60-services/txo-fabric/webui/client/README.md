# TXO Fabric WebUI — global multi-tenant assistant-ui prototype

**Canonical product origin:** `https://webui.truxonline.com` (ADR-040 / #3807). One global React frontend and one logical Fabric BFF, independently scalable. **Source only: NOT DEPLOYED and NOT yet accessible by end users.**

## Source functionality

- A single session at `GET /api/me` returns **only** backend-authorized tenants and their server-checked `chat` / `tenantAdmin` capabilities and independent `platformAdmin`. The user can switch workspace without changing domains.
- One tenant workspace mounts one set of assistant-ui runtimes; a tenant switch **remounts** everything, discarding previous tenant-local drafts, Hermes thread references and chat state.
- The selected tenant **only requests a scope**: `/api/tenants/{tenantKey}/chat/agents`, `/api/tenants/{tenantKey}/chat/threads`, `/api/tenants/{tenantKey}/chat/threads/{id}/turns`. Server side MUST resolve active Authentik memberships, Fabric stable IDs and fresh FGA grants on every request. Browser tenant/path is never an entitlement.
- Multiple agents, concurrent tenant-local chat runtime state, SSE text/tool status and cancellation reuse the existing assistant-ui prototype.
- No browser→Hermes, credentials, store/model IDs or user supplied FabricUserId. Settings and admin are **visibly marked as not implemented** even for users with appropriate roles; no privilege is implied by a navigation element.
- The backend contract is `server/global-webui-http.mjs`, wrapping existing tenant `server/chat-http.mjs`. This is NOT an implemented cookie session, OIDC provider, tenant registry or credentialed runtime. Do not attach dummy callbacks to a production Ingress.

## Build/tests

```sh
npm install --ignore-scripts
npm run build
node --test
```

CI: `.github/workflows/txo-fabric-webui-client-ci.yaml`. Dependency lockfile, reviewed image build/pin workflow and GitOps deployment are still needed before runtime rollout.

## Remaining security and deployment gates

1. Deploy **one Authentik global OIDC client** with exact `https://webui.truxonline.com/auth/callback` and a signed immutable Authentik user UUID claim, plus safe migration/lookup of existing **per-tenant issuer-pinned Fabric identity ledgers**. Legacy per-tenant OIDC/dashboard apps must not break.
2. Real BFF same-origin HTTP server, HttpOnly Secure sessions/CSRF/state/nonce, durable transactions, dedicated Fabric-private PostgreSQL credentials and migration, private service-authenticated enrollment writer.
3. TenantBundle-backed canonical registry; active Authentik user/group UUID membership verification; fail-closed per-tenant FGA freshness, effective policy, independent tenant-admin and platform-admin checks.
4. Private Hermes API Server enablement, durable BFF chat thread repository and per-operation authorized agent resolver, tenant isolation NetworkPolicy, global HTTPS/TLS ingress.
5. Physical A/B/C acceptance including one user in multiple tenants, unauthorized tenant denial, admins, revoke, outages. **No automatic prod promotion.**

See ADR-040, ADR-039 and #3996 for the full trust boundary.
