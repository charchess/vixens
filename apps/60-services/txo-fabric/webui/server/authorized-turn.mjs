import { streamHermesChat } from "./hermes-client.mjs";

/**
 * The authorization callback MUST use an identity established server-side by
 * Fabric Authentik/OIDC, and check the requested tenant + AgentIdentity +
 * action. The resolver MUST return the trusted workload/service credential.
 * These dependencies are deliberately not mocked in a deployment: this code
 * must not be exposed as an unauthenticated public route.
 */
export async function* streamAuthorizedTurn({
  principal, tenantKey, agentKey, sessionId, input, authorize, resolveAgent,
  fetchImpl, signal,
}) {
  if (!principal || principal.authenticated !== true || !principal.subject ||
      !tenantKey || !agentKey || typeof authorize !== "function" ||
      typeof resolveAgent !== "function") {
    throw new Error("Forbidden");
  }
  const allowed = await authorize({ principal, tenantKey, agentKey, action: "agent.chat" });
  if (allowed !== true) throw new Error("Forbidden");
  const target = await resolveAgent({ tenantKey, agentKey });
  if (!target || target.tenantKey !== tenantKey || target.agentKey !== agentKey) {
    throw new Error("Agent is unavailable");
  }
  yield* streamHermesChat({
    baseURL: target.baseURL,
    apiKey: target.apiKey,
    sessionId, input, fetchImpl, signal,
  });
}
