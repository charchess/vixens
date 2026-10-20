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

// These are the complete, server-owned capabilities required by the fixed
// production assembly. They are intentionally neither environment-selected nor
// accepted as arguments from the BFF entrypoint.
const AUTHORITATIVE_RESPONSE_CAPABILITIES = Object.freeze([
  "authorize",
  "resolveCanonicalTenant",
  "resolveCanonicalAgent",
  "verifyEndpoint",
  "readScopedCredential",
  "deriveConversationKey",
  "listTenantAgents",
]);

function unavailableAuthoritativeProviders() {
  // Do not name a backend, endpoint, secret location or topology in this error.
  throw new Error("Authoritative Fabric Responses providers unavailable");
}

function loadFixedAuthoritativeResponsesProviders() {
  // The real physical providers do not exist in this source slice. This fixed
  // seam is deliberately parameterless: it must only be completed by reviewed
  // server-owned identity/FGA, AgentIdentity, endpoint, credential and HMAC
  // implementations satisfying every capability above.
  void AUTHORITATIVE_RESPONSE_CAPABILITIES;
  unavailableAuthoritativeProviders();
}

function createAuthoritativeProductionResponsesComposition() {
  return createReviewedResponsesComposition(loadFixedAuthoritativeResponsesProviders());
}

/**
 * Production-only entrypoint used after both runtime gates are true. It is not
 * a dependency-injection hook: callers cannot supply providers or select a
 * backend. Missing physical dependencies fail before the HTTP listener starts.
 */
export function createProductionResponsesComposition() {
  return createAuthoritativeProductionResponsesComposition();
}
