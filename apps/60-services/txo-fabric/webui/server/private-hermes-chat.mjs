// Preferred first POC path: Hermes Responses owns the conversation on its PVC.
// This source-only export does not construct a BFF repository or live route.
export { createPrivateHermesResponsesService } from "./private-hermes-responses.mjs";

// Legacy native sessions are retained solely to preserve existing source
// transport tests and future governance research. Do not activate this path.
import { createChatSessionService } from "./chat-sessions.mjs";

const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const AGENT = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const TENANT_ID = /^TEN[0-9]{5,}$/;
const USER_ID = /^usr[0-9a-f]{32}$/;
const UID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
const DENY = () => new Error("Agent is unavailable");

/**
 * Future-only private composition. Every resolver and credential reader here
 * must be injected by a separately reviewed, authenticated *platform backend*.
 * NO production entrypoint currently creates it; unit test doubles must not
 * become runtime defaults. No browser-controlled URL, Secret ref, ID or token.
 */
export function createPrivateHermesChatService({
  tenantKey, fabricTenantId, repo, authorize,
  resolveCanonicalTenant, resolveCanonicalAgent, verifyEndpoint,
  readScopedCredential, fetchImpl = fetch,
  browserEnabled = false, bridgeEnabled = false,
} = {}) {
  if (!TENANT.test(tenantKey || "") || !TENANT_ID.test(fabricTenantId || "") ||
      typeof repo?.insert !== "function" || typeof repo?.get !== "function" ||
      typeof repo?.list !== "function" || typeof repo?.claim !== "function" ||
      typeof authorize !== "function" || typeof resolveCanonicalTenant !== "function" ||
      typeof resolveCanonicalAgent !== "function" || typeof verifyEndpoint !== "function" ||
      typeof readScopedCredential !== "function" || typeof fetchImpl !== "function" ||
      typeof browserEnabled !== "boolean" || typeof bridgeEnabled !== "boolean") throw DENY();

  const requireGate = (principal) => {
    if (browserEnabled !== true || bridgeEnabled !== true ||
        principal?.authenticated !== true || principal?.verified !== true ||
        principal?.tenantKey !== tenantKey ||
        principal?.fabricTenantId !== fabricTenantId ||
        !USER_ID.test(principal?.fabricUserId || "") ||
        principal?.subject !== principal.fabricUserId) throw DENY();
  };

  const safeAuthorize = async ({ principal, agentKey, action }) => {
    try {
      requireGate(principal);
      if (!AGENT.test(agentKey || "") || action !== "agent.chat") return false;
      // This is a FRESH ADR-039 OpenFGA Check + effective-policy callback.
      // A cached grant or missing/stale OpenFGAAuthorizationReady MUST deny.
      return await authorize({ principal, tenantKey, fabricTenantId, agentKey, action }) === true;
    } catch { return false; }
  };

  const resolveAgent = async ({ tenantKey: selected, agentKey }) => {
    if (selected !== tenantKey || !AGENT.test(agentKey || "")) throw DENY();
    try {
      const [tenant, agent] = await Promise.all([
        resolveCanonicalTenant({ tenantKey, fabricTenantId }),
        resolveCanonicalAgent({ tenantKey, fabricTenantId, agentKey }),
      ]);
      const namespace = "tenant-" + tenantKey;
      const serviceName = "hermes-" + agentKey;
      const credentialRef = "hermes-api-" + agentKey;
      // Verify canonical UID ownership, deletion/generation, and exact
      // operator-controlled endpoint before ANY secret read or Hermes I/O.
      if (!tenant || tenant.tenantKey !== tenantKey ||
          tenant.fabricTenantId !== fabricTenantId || !UID.test(tenant.uid || "") ||
          tenant.deleting === true || tenant.status !== "Ready" ||
          !agent || agent.tenantKey !== tenantKey ||
          agent.fabricTenantId !== fabricTenantId ||
          agent.agentKey !== agentKey || !UID.test(agent.uid || "") ||
          agent.tenantBundleUID !== tenant.uid || agent.deleting === true ||
          agent.status !== "Ready" ||
          !Number.isSafeInteger(agent.generation) || agent.generation < 1 ||
          agent.observedGeneration !== agent.generation ||
          agent.namespace !== namespace || agent.serviceName !== serviceName ||
          agent.apiServerEnabled !== true || agent.credentialRef !== credentialRef) throw DENY();

      // MUST be independent Kubernetes EndpointSlice/Pod owner+UID attestation
      // or mutually authenticated private transport; DNS alone is insufficient.
      const verified = await verifyEndpoint(Object.freeze({
        namespace, serviceName, port: 8642,
        tenantKey, fabricTenantId, tenantBundleUID: tenant.uid,
        agentKey, agentUID: agent.uid, generation: agent.generation,
      }));
      if (verified !== true) throw DENY();

      const apiKey = await readScopedCredential(Object.freeze({
        namespace, credentialRef, tenantKey, fabricTenantId,
        tenantBundleUID: tenant.uid, agentKey, agentUID: agent.uid,
      }));
      if (typeof apiKey !== "string" || apiKey.length < 16 ||
          apiKey.length > 2048 || /[\r\n]/.test(apiKey)) throw DENY();

      return Object.freeze({
        tenantKey, fabricTenantId, agentKey, agentUID: agent.uid,
        tenantBundleUID: tenant.uid, apiServerEnabled: true,
        endpointVerified: true,
        baseURL: `http://${serviceName}.${namespace}.svc:8642`,
        apiKey,
      });
    } catch { throw DENY(); }
  };

  // The existing chat service owns opaque Fabric threads, leases and server-
  // assigned Hermes sessions. strictBinding pins the exact AgentIdentity UID
  // and revalidates it, never accepting a browser Hermes session identifier.
  const service = createChatSessionService({
    tenantKey, authorize: safeAuthorize, resolveAgent,
    repo, fetchImpl, strictBinding: true,
  });
  // No privileged operation can bypass the two runtime feature gates.
  return Object.freeze(service);
}
