import {
  requireHermesConversationKey, requireHermesResponsesInput, streamHermesResponses,
} from "./hermes-responses.mjs";

const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const AGENT = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const TENANT_ID = /^TEN[0-9]{5,}$/;
const USER_ID = /^usr[0-9a-f]{32}$/;
const UID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
const deny = () => { throw new Error("Agent is unavailable"); };

/**
 * Future-only stateless Fabric BFF -> Hermes Responses composition.
 * No SQL, thread repository, remote session ID, lease or retry. The
 * infrastructure callbacks MUST be separately approved trusted backends
 * (issuer-safe identity, fresh FGA, endpoint attestation, Fabric-plane API
 * credential and keyed HMAC derivation). No production entrypoint uses it.
 */
export function createPrivateHermesResponsesService({
  tenantKey, fabricTenantId, authorize, resolveCanonicalTenant,
  resolveCanonicalAgent, verifyEndpoint, readScopedCredential,
  deriveConversationKey, fetchImpl = fetch,
  browserEnabled = false, bridgeEnabled = false,
} = {}) {
  if (!TENANT.test(tenantKey || "") || !TENANT_ID.test(fabricTenantId || "") ||
      typeof authorize !== "function" || typeof resolveCanonicalTenant !== "function" ||
      typeof resolveCanonicalAgent !== "function" || typeof verifyEndpoint !== "function" ||
      typeof readScopedCredential !== "function" ||
      typeof deriveConversationKey !== "function" ||
      typeof fetchImpl !== "function" ||
      typeof browserEnabled !== "boolean" || typeof bridgeEnabled !== "boolean") deny();

  const requirePrincipal = principal => {
    if (!browserEnabled || !bridgeEnabled ||
        principal?.authenticated !== true || principal?.verified !== true ||
        principal?.tenantKey !== tenantKey ||
        principal?.fabricTenantId !== fabricTenantId ||
        !USER_ID.test(principal?.fabricUserId || "") ||
        principal.subject !== principal.fabricUserId) deny();
  };

  async function* turn({ principal, agentKey, input, signal } = {}) {
    // This is an explicit server-side API. The browser never supplies the
    // tenant, UID, generation, session key, endpoint, provider or credential.
    try {
      requirePrincipal(principal);
      if (!AGENT.test(agentKey || "")) deny();
      requireHermesResponsesInput(input);
      if (await authorize({ principal, tenantKey, fabricTenantId,
        agentKey, action: "agent.chat" }) !== true) deny();

      const [tenant, agent] = await Promise.all([
        resolveCanonicalTenant({ tenantKey, fabricTenantId }),
        resolveCanonicalAgent({ tenantKey, fabricTenantId, agentKey }),
      ]);
      const namespace = "tenant-" + tenantKey;
      const serviceName = "hermes-" + agentKey;
      const credentialRef = "hermes-api-" + agentKey;
      if (!tenant || tenant.tenantKey !== tenantKey ||
          tenant.fabricTenantId !== fabricTenantId || !UID.test(tenant.uid || "") ||
          tenant.deleting === true || tenant.status !== "Ready" ||
          !agent || agent.tenantKey !== tenantKey ||
          agent.fabricTenantId !== fabricTenantId || agent.agentKey !== agentKey ||
          !UID.test(agent.uid || "") || agent.tenantBundleUID !== tenant.uid ||
          agent.deleting === true || agent.status !== "Ready" ||
          !Number.isSafeInteger(agent.generation) || agent.generation < 1 ||
          agent.observedGeneration !== agent.generation ||
          agent.namespace !== namespace || agent.serviceName !== serviceName ||
          agent.apiServerEnabled !== true || agent.credentialRef !== credentialRef) deny();

      // Trust only the independently observed Service/EndpointSlice/Pod UID,
      // NEVER a DNS lookup, browser-supplied URL or mutable workload label.
      if (await verifyEndpoint(Object.freeze({
        namespace, serviceName, port: 8642,
        tenantKey, fabricTenantId, tenantBundleUID: tenant.uid,
        agentKey, agentUID: agent.uid, generation: agent.generation,
      })) !== true) deny();

      // The derivation receives ONLY immutable canonical IDs and no secret
      // from this module. Future production implementation: versioned HMAC
      // with a separately approved, scoped Fabric-plane signing key.
      const scope = Object.freeze({
        fabricTenantId, fabricUserId: principal.fabricUserId,
        agentUID: agent.uid, agentGeneration: agent.generation,
      });
      const conversationKey = requireHermesConversationKey(
        await deriveConversationKey(scope),
        { ...scope, tenantKey, agentKey, namespace, serviceName, credentialRef },
      );

      // An invalid HMAC/key MUST deny before the scoped credential read.
      const apiKey = await readScopedCredential(Object.freeze({
        namespace, credentialRef, tenantKey, fabricTenantId,
        tenantBundleUID: tenant.uid, agentKey, agentUID: agent.uid,
      }));
      if (typeof apiKey !== "string" || apiKey.length < 16 || apiKey.length > 2048 ||
          /[\r\n\0]/.test(apiKey) || conversationKey === apiKey) deny();

      yield* streamHermesResponses({
        baseURL: `http://${serviceName}.${namespace}.svc:8642`,
        apiKey, conversationKey, input, fetchImpl, signal,
      });
    } catch { deny(); } // do not disclose denied identity/grant/endpoint or upstream data
  }

  return Object.freeze({ turn });
}
