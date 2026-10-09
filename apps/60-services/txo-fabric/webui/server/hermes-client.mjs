import { parseSSE, projectChatEvent } from "./hermes-events.mjs";

const SESSION_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;

/**
 * Only the trusted Fabric backend may construct this transport. The hostname
 * restriction is a defense in depth against accidental public/SSRF targets;
 * the caller must still resolve the tenant's AgentIdentity via authoritative
 * Kubernetes/Fabric state, never from a client-supplied URL.
 */
export function requireInternalHermesURL(baseURL) {
  let url;
  try { url = new URL(baseURL); } catch { throw new Error("Invalid internal Hermes target"); }
  if (!(["http:", "https:"].includes(url.protocol)) ||
      !(url.hostname.endsWith(".svc") || url.hostname.endsWith(".svc.cluster.local")) ||
      url.username || url.password || url.search || url.hash ||
      (url.pathname !== "/" && url.pathname !== "")) {
    throw new Error("Invalid internal Hermes target");
  }
  return url.origin;
}

export async function* streamHermesChat({ baseURL, apiKey, sessionId, input, fetchImpl = fetch, signal }) {
  const origin = requireInternalHermesURL(baseURL);
  if (typeof apiKey !== "string" || !apiKey || /[\r\n]/.test(apiKey)) {
    throw new Error("Hermes service credential unavailable");
  }
  if (typeof sessionId !== "string" || !SESSION_ID.test(sessionId)) {
    throw new Error("Invalid Hermes session ID");
  }
  if (typeof input !== "string" || !input.trim() || input.length > 100000) {
    throw new Error("Invalid chat input");
  }
  const url = origin + "/api/sessions/" + encodeURIComponent(sessionId) + "/chat/stream";
  const response = await fetchImpl(url, {
    method: "POST",
    headers: {
      Authorization: "Bearer " + apiKey,
      "Content-Type": "application/json",
      Accept: "text/event-stream",
    },
    body: JSON.stringify({ input }),
    signal,
  });
  // Never return the CPA/Hermes/provider raw error body to a browser.
  if (!response.ok || !response.body) {
    throw new Error("Agent is unavailable");
  }
  for await (const event of parseSSE(response.body)) {
    const projected = projectChatEvent(event);
    if (projected) yield projected;
  }
}
