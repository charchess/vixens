import { createChatHttpHandler } from "./chat-http.mjs";

const TENANT_KEY = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const FABRIC_TENANT_ID = /^TEN[0-9]{5,}$/;
const FABRIC_USER_ID = /^usr[0-9a-f]{32}$/;
const AUTHENTIK_UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const OIDC_SUB = /^[\x21-\x7e]{1,512}$/;
const MAX_TENANTS = 500;
const json = (data, status = 200) => new Response(JSON.stringify(data), {
  status, headers: {
    "Content-Type": "application/json", "Cache-Control": "no-store",
    "Vary": "Cookie", "X-Content-Type-Options": "nosniff",
  },
});
const denied = (code = 403) => json({ error: code === 503 ? "Service unavailable" : "Forbidden" }, code);
const validTenant = (entry) => entry && typeof entry.tenantKey === "string" &&
  TENANT_KEY.test(entry.tenantKey) && typeof entry.fabricTenantId === "string" &&
  FABRIC_TENANT_ID.test(entry.fabricTenantId) &&
  typeof entry.displayName === "string" &&
  entry.displayName.trim().length > 0 && entry.displayName.length <= 120 &&
  !/[\u0000-\u001f\u007f]/.test(entry.displayName);

/**
 * ONE global WebUI origin and a Fabric-owned BFF boundary, not one
 * browser application / BFF per tenant or per agent.
 *
 * This is an integration seam and deliberately has NO fabricated
 * Authentik session, role, OIDC token, tenant registry or FGA grant.
 * Required production dependencies:
 *
 * authenticate(request): server-side, session-bound Authentik GLOBAL
 *   OIDC client proof. NEVER trust browser headers, query or JSON.
 * listTenantCatalog(): immutable TenantBundle-backed server registry
 *   of {tenantKey, fabricTenantId, displayName}; may include many tenants.
 * resolveTenantAccess({identity, tenantKey, fabricTenantId}):
 *   PRIVATE trusted Fabric backend: validate active Authentik account,
 *   tenant-group membership from pinned Authentik UUIDs, per-tenant
 *   immutable Fabric identity mapping, current OpenFGA/freshness and
 *   effective Fabric policy. Return EXACTLY
 *   {tenantKey, fabricTenantId, fabricUserId, accountActive:true,
 *    fresh:true, canEnter:true, canManageTenant:boolean}. Never assign
 *   a Fabric user ID from a global OIDC sub or from a browser input.
 * getTenantChat({tenantKey, fabricTenantId}): Fabric-owned canonical
 *   route resolver returning {tenantKey, fabricTenantId,
 *   service, listTenantAgents}. The service must use the SAME verified
 *   Fabric principal and enforce its own per-agent FGA checks.
 *
 * Global OIDC client/issuer is distinct from legacy per-tenant OIDC
 * clients. Existing tenant identity ledgers are issuer-pinned; merging
 * them into this GLOBAL login requires an explicit, reviewed migration
 * (never silently overwrite/rebind an existing subject).
 *
 * A pathname tenant key is a REQUESTED scope, NEVER an entitlement.
 * The server looks up and rechecks access on every request, including
 * every chat turn, and does not expose internal store/agent addresses.
 */
export function createGlobalWebUIHandler({
  expectedOrigin, expectedIssuer, authenticate, listTenantCatalog,
  resolveTenantAccess, getTenantChat, resolvePlatformAccess,
}) {
  let origin, issuer;
  try { origin = new URL(expectedOrigin); issuer = new URL(expectedIssuer); }
  catch { throw new Error("Global Fabric WebUI security configuration missing"); }
  if (origin.protocol !== "https:" || origin.origin !== expectedOrigin ||
      origin.pathname !== "/" || origin.search || origin.hash ||
      issuer.protocol !== "https:" || issuer.href !== expectedIssuer ||
      issuer.pathname !== "/application/o/txo-fabric-webui/" || issuer.search || issuer.hash ||
      typeof authenticate !== "function" ||
      typeof listTenantCatalog !== "function" ||
      typeof resolveTenantAccess !== "function" ||
      typeof getTenantChat !== "function" ||
      (resolvePlatformAccess !== undefined && typeof resolvePlatformAccess !== "function")) {
    throw new Error("Global Fabric WebUI security configuration missing");
  }

  async function identityFor(request) {
    const identity = await authenticate(request);
    // Authentication is performed by the injected PRIVATE session verifier.
    // These checks prevent accidental use of legacy per-tenant proofs.
    if (identity?.authenticated !== true || identity?.verified !== true ||
        identity.issuer !== expectedIssuer ||
        typeof identity.oidcSubject !== "string" || !OIDC_SUB.test(identity.oidcSubject) ||
        typeof identity.authentikUserUUID !== "string" ||
        !AUTHENTIK_UUID.test(identity.authentikUserUUID)) throw Error("Denied");
    return Object.freeze({
      issuer: identity.issuer,
      oidcSubject: identity.oidcSubject,
      authentikUserUUID: identity.authentikUserUUID.toLowerCase(),
    });
  }

  async function catalog() {
    const entries = await listTenantCatalog();
    if (!Array.isArray(entries) || entries.length > MAX_TENANTS ||
        entries.some(e => !validTenant(e))) throw Error("Invalid catalog");
    const keys = new Set(), ids = new Set();
    for (const e of entries) {
      if (keys.has(e.tenantKey) || ids.has(e.fabricTenantId)) throw Error("Ambiguous catalog");
      keys.add(e.tenantKey); ids.add(e.fabricTenantId);
    }
    return entries;
  }

  async function scopedPrincipal(identity, entry) {
    const scope = await resolveTenantAccess(Object.freeze({
      identity, tenantKey: entry.tenantKey, fabricTenantId: entry.fabricTenantId,
    }));
    if (!scope || scope.accountActive !== true || scope.fresh !== true ||
        scope.canEnter !== true || scope.tenantKey !== entry.tenantKey ||
        scope.fabricTenantId !== entry.fabricTenantId ||
        typeof scope.fabricUserId !== "string" ||
        !FABRIC_USER_ID.test(scope.fabricUserId)) return null;

    return Object.freeze({
      authenticated: true, verified: true,
      // Per-tenant canonical immutable Fabric ID, not OIDC sub or email.
      // This is also the durable chat conversation subject.
      subject: scope.fabricUserId,
      fabricUserId: scope.fabricUserId,
      fabricTenantId: entry.fabricTenantId,
      tenantKey: entry.tenantKey,
      canManageTenant: scope.canManageTenant === true,
    });
  }

  return async function handle(request) {
    let route;
    try { route = new URL(request.url); }
    catch { return denied(); }
    if (route.origin !== expectedOrigin || route.username || route.password) return denied();
    if (!["GET", "POST"].includes(request.method)) return denied(405);
    let identity;
    try { identity = await identityFor(request); }
    catch { return denied(); }

    if (route.pathname === "/api/me" && request.method === "GET") {
      try {
        const entries = await catalog(), tenants = [];
        for (const entry of entries) {
          const principal = await scopedPrincipal(identity, entry);
          if (principal) tenants.push({
            tenantKey: entry.tenantKey, displayName: entry.displayName,
            capabilities: { chat: true, tenantAdmin: principal.canManageTenant },
          });
        }
        // Platform admin is an independently verified FGA/platform policy;
        // tenant admin is NEVER promoted to platform admin implicitly.
        const platform = resolvePlatformAccess
          ? await resolvePlatformAccess({ identity }) : null;
        if (platform && (platform.fresh !== true ||
            platform.accountActive !== true ||
            typeof platform.canAdminister !== "boolean")) throw Error("Stale");
        return json({ tenants, capabilities: {
          platformAdmin: platform?.canAdminister === true,
        } });
      } catch { return denied(503); }
    }

    // No client-controlled generic proxy. Only the chat route family is
    // currently exposed. Settings/admin must have separate operation-level
    // authorization, not reuse canEnter/canManageTenant blindly.
    const match = /^\/api\/tenants\/([a-z0-9-]+)(\/chat\/(?:agents|threads(?:\/[a-f0-9]{40}(?:\/turns)?)?))$/.exec(route.pathname);
    if (!match || !TENANT_KEY.test(match[1])) return denied(404);
    try {
      const entry = (await catalog()).find(e => e.tenantKey === match[1]);
      if (!entry) return denied(); // Do not disclose tenant existence.
      const principal = await scopedPrincipal(identity, entry);
      if (!principal) return denied(); // Check fresh membership on EACH route.
      const selected = await getTenantChat({
        tenantKey: entry.tenantKey, fabricTenantId: entry.fabricTenantId,
      });
      if (!selected || selected.tenantKey !== entry.tenantKey ||
          selected.fabricTenantId !== entry.fabricTenantId ||
          !selected.service) return denied(503);

      const chatPath = "/api" + match[2];
      const rewritten = new URL(request.url);
      rewritten.pathname = chatPath;
      const forwarded = new Request(rewritten.href, request);
      return createChatHttpHandler({
        authenticate: async () => principal,
        service: selected.service,
        expectedOrigin,
        listTenantAgents: selected.listTenantAgents,
      })(forwarded);
    } catch { return denied(503); }
  };
}
