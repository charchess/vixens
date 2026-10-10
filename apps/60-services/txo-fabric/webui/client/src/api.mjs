const THREAD = /^[a-f0-9]{40}$/;
const AGENT = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/;
const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

async function requireJSON(response) {
  if (!response.ok) throw new Error("Accès refusé ou service indisponible");
  return response.json();
}
const headers = { "Content-Type": "application/json", "X-TXO-Request": "chat" };
const tenantURL = (tenantKey) => {
  if (typeof tenantKey !== "string" || !TENANT.test(tenantKey)) throw new Error("Tenant invalide");
  return `/api/tenants/${tenantKey}/chat`;
};

/**
 * Global Fabric WebUI shell. The only public origin is webui.truxonline.com.
 * The backend resolves permitted tenants from a SIGNED session, live
 * Authentik memberships, canonical Fabric user mapping and fresh IAM/FGA.
 * A tenant name in a URL is only a requested scope, NEVER a grant.
 */
export async function listAccessibleTenants(fetchImpl = fetch) {
  const response = await fetchImpl("/api/me", {
    credentials: "same-origin", cache: "no-store",
  });
  const body = await requireJSON(response);
  if (!Array.isArray(body?.tenants) || body.tenants.length > 500)
    throw new Error("Espaces indisponibles");
  const seen = new Set();
  const tenants = [];
  for (const t of body.tenants) {
    if (!t || typeof t.tenantKey !== "string" || !TENANT.test(t.tenantKey) ||
        seen.has(t.tenantKey) || typeof t.displayName !== "string" ||
        !t.displayName.trim() || t.displayName.length > 120 ||
        typeof t.capabilities?.chat !== "boolean" ||
        typeof t.capabilities?.tenantAdmin !== "boolean") throw new Error("Espaces indisponibles");
    seen.add(t.tenantKey);
    tenants.push({
      tenantKey: t.tenantKey, displayName: t.displayName,
      capabilities: {
        chat: t.capabilities.chat, tenantAdmin: t.capabilities.tenantAdmin,
      },
    });
  }
  return { tenants, platformAdmin: body.capabilities?.platformAdmin === true };
}

export async function listAvailableAgents(tenantKey, fetchImpl = fetch) {
  const response = await fetchImpl(tenantURL(tenantKey) + "/agents", {
    credentials: "same-origin", cache: "no-store",
  });
  const body = await requireJSON(response);
  if (!Array.isArray(body?.agents)) throw new Error("Liste des agents invalide");
  return body.agents.filter(a => typeof a?.agentKey === "string" && AGENT.test(a.agentKey) &&
    typeof a.label === "string" && a.label.length <= 100);
}

/** @param {string} tenantKey
 * @param {string} agentKey
 * @param {{fetchImpl?: typeof fetch, signal?: AbortSignal}} [options]
 */
export async function createFabricThread(tenantKey, agentKey, { fetchImpl = fetch, signal } = {}) {
  const base = tenantURL(tenantKey);
  if (typeof agentKey !== "string" || !AGENT.test(agentKey)) throw new Error("Agent invalide");
  const response = await fetchImpl(base + "/threads", {
    method: "POST", credentials: "same-origin", cache: "no-store", headers,
    body: JSON.stringify({ agentKey }), signal,
  });
  const body = await requireJSON(response);
  if (typeof body?.id !== "string" || !THREAD.test(body.id) || body.agentKey !== agentKey) {
    throw new Error("Identifiant de conversation invalide");
  }
  return body.id;
}

/** Read server-projected SSE only. Never accept provider credentials in replies.
 * @param {string} tenantKey
 * @param {string} threadId
 * @param {string} input
 * @param {{fetchImpl?: typeof fetch, signal?: AbortSignal}} [options]
 */
export async function* streamFabricTurn(tenantKey, threadId, input, { fetchImpl = fetch, signal } = {}) {
  const base = tenantURL(tenantKey);
  if (typeof threadId !== "string" || !THREAD.test(threadId) ||
      typeof input !== "string" || !input.trim()) throw new Error("Requête invalide");
  const response = await fetchImpl(`${base}/threads/${threadId}/turns`, {
    method: "POST", credentials: "same-origin", cache: "no-store", headers,
    body: JSON.stringify({ input }), signal,
  });
  if (!response.ok || !response.body) throw new Error("Conversation indisponible");
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "", completed = false;
  try {
    while (true) {
      const { done, value } = await reader.read();
      buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
      let boundary;
      while ((boundary = buffer.indexOf("\n\n")) !== -1) {
        const frame = buffer.slice(0, boundary).replace(/\r/g, "");
        buffer = buffer.slice(boundary + 2);
        const event = frame.match(/^event: ([^\n]+)/m)?.[1] || "message";
        const data = frame.split("\n").filter(line => line.startsWith("data: ")).map(line => line.slice(6)).join("\n");
        if (event === "error") throw new Error("L'agent n'a pas pu répondre");
        if (!data) continue;
        let obj;
        try { obj = JSON.parse(data); } catch { throw new Error("Flux invalide"); }
        if (obj?.type === "text-delta" && typeof obj.text === "string") yield obj;
        else if (obj?.type === "tool-started" || obj?.type === "tool-completed") {
          yield { type: obj.type, tool: typeof obj.tool === "string" ? obj.tool : "tool" };
        } else if (obj?.type === "done") {
          if (obj.status !== "completed") throw new Error("Génération interrompue");
          completed = true;
          yield { type: "done", status: "completed" };
          return;
        }
      }
      if (buffer.length > 65_536) throw new Error("Flux trop volumineux");
      if (done) break;
    }
    if (!completed) throw new Error("Flux interrompu");
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}
