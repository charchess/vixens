import { randomBytes } from "node:crypto";
import { requireInternalHermesURL } from "./hermes-client.mjs";
import { streamAuthorizedTurn } from "./authorized-turn.mjs";

const AGENT_KEY = /^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$/;
const REMOTE_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const THREAD_ID = /^[a-f0-9]{40}$/;

const deny = () => { throw new Error("Forbidden"); };
const validPrincipal = (p) => p?.authenticated === true &&
  typeof p.subject === "string" && p.subject.trim().length > 0 &&
  typeof p.tenantKey === "string" && p.tenantKey.length > 0;

function requirePrincipal(principal, tenantKey) {
  if (!validPrincipal(principal) || principal.tenantKey !== tenantKey) deny();
}

async function requireGrant({ principal, tenantKey, agentKey, action, authorize }) {
  if (typeof agentKey !== "string" || !AGENT_KEY.test(agentKey) ||
      await authorize({ principal, tenantKey, agentKey, action }) !== true) deny();
}

function requireTarget(target, tenantKey, agentKey) {
  if (!target || target.tenantKey !== tenantKey || target.agentKey !== agentKey ||
      typeof target.apiKey !== "string" || !target.apiKey || /[\r\n]/.test(target.apiKey)) {
    throw new Error("Agent is unavailable");
  }
  return requireInternalHermesURL(target.baseURL);
}

/** No caller-controlled remote session IDs. Hermes returns one and we store it
 * server-side, indexed by a cryptographically random opaque Fabric thread ID.
 */
export async function openHermesSession({ target, tenantKey, agentKey, fetchImpl = fetch }) {
  const origin = requireTarget(target, tenantKey, agentKey);
  const response = await fetchImpl(`${origin}/api/sessions`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${target.apiKey}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({ title: "Fabric conversation" }),
  });
  if (!response.ok) throw new Error("Agent is unavailable");
  // Reject oversized/unexpected internal replies, and never expose them to UI.
  if (Number(response.headers?.get?.("content-length") || 0) > 4096) throw new Error("Agent is unavailable");
  let json;
  try {
    const body = await response.text();
    if (body.length > 4096) throw new Error("oversized");
    json = JSON.parse(body);
  } catch { throw new Error("Agent is unavailable"); }
  const sessionId = json?.session_id ?? json?.id;
  if (typeof sessionId !== "string" || !REMOTE_ID.test(sessionId)) {
    throw new Error("Agent is unavailable");
  }
  return sessionId;
}

/**
 * Storage is a REQUIRED injected dependency. In production it must be durable
 * and enforce atomic per-thread turn leases across BFF replicas. No in-memory
 * fallback is allowed here.
 *
 * repo: insert(record), get(id), list({tenantKey,subject}), claim(id)
 * claim(id) returns an async release() callback; throws if already in use.
 */
export function createChatSessionService({ tenantKey, authorize, resolveAgent, repo,
  openRemoteSession = openHermesSession, fetchImpl = fetch }) {
  if (typeof tenantKey !== "string" || !tenantKey ||
      typeof authorize !== "function" || typeof resolveAgent !== "function" ||
      !repo || ["insert", "get", "list", "claim"].some((k) => typeof repo[k] !== "function") ||
      typeof openRemoteSession !== "function" || typeof fetchImpl !== "function") {
    throw new Error("Chat security dependencies are required");
  }
  async function boundThread({ principal, threadId, action }) {
    requirePrincipal(principal, tenantKey);
    if (typeof threadId !== "string" || !THREAD_ID.test(threadId)) deny();
    const thread = await repo.get(threadId);
    if (!thread || thread.tenantKey !== tenantKey || thread.subject !== principal.subject) deny();
    await requireGrant({ principal, tenantKey, agentKey: thread.agentKey, action, authorize });
    return thread;
  }
  return {
    async canAccess({ principal, agentKey }) {
      requirePrincipal(principal, tenantKey);
      await requireGrant({ principal, tenantKey, agentKey, action: "agent.chat", authorize });
      return true;
    },
    async create({ principal, agentKey }) {
      requirePrincipal(principal, tenantKey);
      await requireGrant({ principal, tenantKey, agentKey, action: "agent.chat", authorize });
      const target = await resolveAgent({ tenantKey, agentKey });
      requireTarget(target, tenantKey, agentKey);
      const sessionId = await openRemoteSession({ target, tenantKey, agentKey, fetchImpl });
      if (typeof sessionId !== "string" || !REMOTE_ID.test(sessionId)) {
        throw new Error("Agent is unavailable");
      }
      const record = {
        id: randomBytes(20).toString("hex"), tenantKey, subject: principal.subject,
        agentKey, sessionId, createdAt: new Date().toISOString(),
      };
      await repo.insert(record);
      return { id: record.id, agentKey: record.agentKey, createdAt: record.createdAt };
    },
    async list({ principal }) {
      requirePrincipal(principal, tenantKey);
      const records = await repo.list({ tenantKey, subject: principal.subject });
      const visible = [];
      for (const r of records) {
        if (r.tenantKey !== tenantKey || r.subject !== principal.subject) continue;
        if (await authorize({ principal, tenantKey, agentKey: r.agentKey, action: "agent.chat" }) === true) {
          visible.push({ id: r.id, agentKey: r.agentKey, createdAt: r.createdAt });
        }
      }
      return visible;
    },
    async *turn({ principal, threadId, input, signal }) {
      const thread = await boundThread({ principal, threadId, action: "agent.chat" });
      const release = await repo.claim(thread.id);
      try {
        // Recheck after the turn lease is acquired; streaming checks again.
        await requireGrant({ principal, tenantKey, agentKey: thread.agentKey, action: "agent.chat", authorize });
        yield* streamAuthorizedTurn({
          principal, tenantKey, agentKey: thread.agentKey, sessionId: thread.sessionId,
          input, authorize, resolveAgent, fetchImpl, signal,
        });
      } finally {
        await release();
      }
    },
    async get({ principal, threadId }) {
      const r = await boundThread({ principal, threadId, action: "agent.chat" });
      return { id: r.id, agentKey: r.agentKey, createdAt: r.createdAt };
    },
  };
}
