import { createPrivateHermesResponsesService } from "./private-hermes-responses.mjs";

const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const TENANT_ID = /^TEN[0-9]{5,}$/;
const unavailable = () => { throw new Error("Reviewed Fabric Responses providers unavailable"); };

/**
 * Compose the only approved first-conversation transport: stateless BFF to
 * Hermes-owned Responses. This intentionally has no thread repository, legacy
 * session adapter, browser-supplied target, or callback fallback.
 *
 * All inputs are server-owned provider contracts. The deployed runtime has no
 * authoritative implementations for these contracts yet, so private-bff-main
 * deliberately invokes the unavailable factory rather than fabricating one.
 */
export function createReviewedResponsesComposition({
  authorize, resolveCanonicalTenant, resolveCanonicalAgent, verifyEndpoint,
  readScopedCredential, deriveConversationKey, listTenantAgents,
  fetchImpl = fetch,
} = {}) {
  if (typeof authorize !== "function" ||
      typeof resolveCanonicalTenant !== "function" ||
      typeof resolveCanonicalAgent !== "function" ||
      typeof verifyEndpoint !== "function" ||
      typeof readScopedCredential !== "function" ||
      typeof deriveConversationKey !== "function" ||
      typeof listTenantAgents !== "function" ||
      typeof fetchImpl !== "function") unavailable();

  return Object.freeze({
    async getTenantChat({ tenantKey, fabricTenantId } = {}) {
      if (!TENANT.test(tenantKey || "") || !TENANT_ID.test(fabricTenantId || "")) unavailable();
      const service = createPrivateHermesResponsesService({
        tenantKey,
        fabricTenantId,
        authorize,
        resolveCanonicalTenant,
        resolveCanonicalAgent,
        verifyEndpoint,
        readScopedCredential,
        deriveConversationKey,
        fetchImpl,
        browserEnabled: true,
        bridgeEnabled: true,
      });
      return Object.freeze({
        tenantKey,
        fabricTenantId,
        service,
        // Candidate labels come from a canonical server provider; the chat
        // handler still performs a fresh agent.chat check for every candidate.
        listTenantAgents: async ({ principal } = {}) =>
          listTenantAgents({ principal, tenantKey, fabricTenantId }),
      });
    },
  });
}

/**
 * Explicit runtime boundary. No currently reviewed deployment configuration
 * provides the authoritative identity/FGA/AgentIdentity/endpoint/credential
 * and HMAC providers, therefore gates-on startup must stop before listening.
 * This factory must be replaced only by a separately reviewed real provider
 * assembly; it may not be replaced by environment-selected modules or mocks.
 */
export function createUnavailableProductionResponsesComposition() {
  unavailable();
}
