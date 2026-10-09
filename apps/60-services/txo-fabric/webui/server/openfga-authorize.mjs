const ULID = /^[0-9A-HJKMNP-TV-Z]{26}$/;
const KEY = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const SERVICE = new Set([
  "txo-fabric-openfga.txo-fabric-system.svc",
  "txo-fabric-openfga.txo-fabric-system.svc.cluster.local",
]);
const ACTION_RELATION = Object.freeze({
  "agent.chat": "can_chat",
  "agent.manage": "can_manage",
});

/** Defense in depth, not a substitute for a Kubernetes NetworkPolicy. */
export function openFGAPrivateOrigin(endpoint) {
  let url;
  try { url = new URL(endpoint); } catch { throw new Error("Invalid Fabric OpenFGA service"); }
  if (url.protocol !== "http:" || !SERVICE.has(url.hostname) ||
      url.port !== "8080" || (url.pathname !== "/" && url.pathname !== "") ||
      url.search || url.hash || url.username || url.password) {
    throw new Error("Invalid Fabric OpenFGA service");
  }
  return url.origin;
}

/**
 * Only a trusted Fabric BFF may instantiate this authorizer.
 * The principal.verified flag is set AFTER real server-side OIDC verification
 * and Fabric stable-ID mapping; it MUST NOT originate from HTTP headers or
 * client JSON. No browser, Hermes pod or tenant service calls FGA directly.
 *
 * Resolver callbacks are platform-owned and must verify actual tenant
 * ownership in durable Fabric state.
 */
export function createOpenFGAAuthorizer({
  endpoint, apiKey, resolveTenantStore, resolveCanonicalAgent,
  fetchImpl = fetch, timeoutMs = 3000,
}) {
  const origin = openFGAPrivateOrigin(endpoint);
  if (typeof apiKey !== "string" || apiKey.length < 16 ||
      /[\r\n]/.test(apiKey) ||
      typeof resolveTenantStore !== "function" ||
      typeof resolveCanonicalAgent !== "function" ||
      typeof fetchImpl !== "function" ||
      !Number.isSafeInteger(timeoutMs) || timeoutMs < 100 || timeoutMs > 10000) {
    throw new Error("Fabric OpenFGA security dependencies missing");
  }

  return async function authorize({ principal, tenantKey, agentKey, action }) {
    const relation = ACTION_RELATION[action];
    if (!relation || !principal || principal.authenticated !== true ||
        principal.verified !== true || typeof principal.subject !== "string" ||
        !principal.subject.trim() ||
        typeof principal.tenantKey !== "string" ||
        principal.tenantKey !== tenantKey || !KEY.test(tenantKey || "") ||
        typeof principal.fabricUserId !== "string" ||
        !KEY.test(principal.fabricUserId) ||
        typeof principal.fabricTenantId !== "string" ||
        !KEY.test(principal.fabricTenantId) ||
        typeof agentKey !== "string" || !KEY.test(agentKey)) {
      return false;
    }

    try {
      const tenant = await resolveTenantStore({
        tenantKey, fabricTenantId: principal.fabricTenantId,
      });
      // The OpenFGA API token is service-wide. Only Fabric enforces store
      // scope. Never accept client-provided store and model IDs.
      if (!tenant || tenant.fabricTenantId !== principal.fabricTenantId ||
          !ULID.test(tenant.storeId || "") ||
          !ULID.test(tenant.modelId || "")) return false;
      const agent = await resolveCanonicalAgent({
        tenantKey, fabricTenantId: principal.fabricTenantId, agentKey,
      });
      if (!agent || agent.fabricTenantId !== principal.fabricTenantId ||
          typeof agent.agentId !== "string" || !KEY.test(agent.agentId)) return false;

      const url = origin + "/stores/" + tenant.storeId + "/check";
      const response = await fetchImpl(url, {
        method: "POST",
        headers: {
          Authorization: "Bearer " + apiKey,
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify({
          authorization_model_id: tenant.modelId,
          tuple_key: {
            user: "user:" + principal.fabricUserId,
            relation,
            object: "agent:" + agent.agentId,
          },
        }),
        signal: AbortSignal.timeout(timeoutMs),
      });
      if (!response.ok || Number(response.headers?.get?.("content-length") || 0) > 4096) return false;
      const raw = await response.text();
      if (raw.length > 4096) return false;
      const result = JSON.parse(raw);
      return result?.allowed === true;
    } catch {
      // Missing store, bad model, outage, auth error and timeout all deny.
      // The caller may log a non-secret diagnostic separately.
      return false;
    }
  };
}
