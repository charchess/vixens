const THREAD = /^[a-f0-9]{40}$/;
const AGENT = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/;

async function requireJSON(response) {
  if (!response.ok) throw new Error("Accès refusé ou service indisponible");
  return response.json();
}
const headers = { "Content-Type": "application/json", "X-TXO-Request": "chat" };

export async function listAvailableAgents(fetchImpl = fetch) {
  const response = await fetchImpl("/api/chat/agents", { credentials: "same-origin", cache: "no-store" });
  const body = await requireJSON(response);
  if (!Array.isArray(body?.agents)) throw new Error("Liste des agents invalide");
  return body.agents.filter(a => typeof a?.agentKey === "string" && AGENT.test(a.agentKey) &&
    typeof a.label === "string" && a.label.length <= 100);
}

export async function createFabricThread(agentKey, { fetchImpl = fetch, signal } = {}) {
  if (typeof agentKey !== "string" || !AGENT.test(agentKey)) throw new Error("Agent invalide");
  const response = await fetchImpl("/api/chat/threads", {
    method: "POST", credentials: "same-origin", cache: "no-store", headers,
    body: JSON.stringify({ agentKey }), signal,
  });
  const body = await requireJSON(response);
  if (typeof body?.id !== "string" || !THREAD.test(body.id) || body.agentKey !== agentKey) {
    throw new Error("Identifiant de conversation invalide");
  }
  return body.id;
}

/** Read server-projected SSE only. Never accept provider credentials in replies. */
export async function* streamFabricTurn(threadId, input, { fetchImpl = fetch, signal } = {}) {
  if (typeof threadId !== "string" || !THREAD.test(threadId) ||
      typeof input !== "string" || !input.trim()) throw new Error("Requête invalide");
  const response = await fetchImpl(`/api/chat/threads/${threadId}/turns`, {
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
