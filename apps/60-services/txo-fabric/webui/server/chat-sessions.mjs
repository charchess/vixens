import { randomBytes } from "node:crypto";
import { requireInternalHermesURL } from "./hermes-client.mjs";
import { streamAuthorizedTurn } from "./authorized-turn.mjs";

const AGENT_KEY = /^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$/;
const REMOTE_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const THREAD_ID = /^[a-f0-9]{40}$/;
const TURN_ID = /^[a-f0-9]{32}$/;

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
export async function openHermesSession({ target, tenantKey, agentKey, fetchImpl = fetch, strictBinding = false, signal }) {
  const origin = requireTarget(target, tenantKey, agentKey);
  const response = await fetchImpl(`${origin}/api/sessions`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${target.apiKey}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({ title: "Fabric conversation" }),
    redirect: "error", cache: "no-store", signal,
  });
  if (!response.ok || (strictBinding && (response.redirected ||
      !response.headers?.get?.("content-type")?.toLowerCase().startsWith("application/json")))) {
    throw new Error("Agent is unavailable");
  }
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
  openRemoteSession = openHermesSession, fetchImpl = fetch, strictBinding = false }) {
  if (typeof tenantKey !== "string" || !tenantKey ||
      typeof authorize !== "function" || typeof resolveAgent !== "function" ||
      !repo || ["insert", "get", "list", "claim"].some((k) => typeof repo[k] !== "function") ||
      typeof openRemoteSession !== "function" || typeof fetchImpl !== "function" ||
      typeof strictBinding !== "boolean" ||
      (strictBinding && typeof repo.claimTurn !== "function")) {
    throw new Error("Chat security dependencies are required");
  }
  function strictPrincipal(principal) {
    if (strictBinding && (principal?.verified !== true ||
        principal?.subject !== principal?.fabricUserId ||
        typeof principal?.fabricTenantId !== "string" ||
        !/^TEN[0-9]{5,}$/.test(principal.fabricTenantId))) deny();
  }
  async function boundThread({ principal, threadId, action }) {
    requirePrincipal(principal, tenantKey);
    strictPrincipal(principal);
    if (typeof threadId !== "string" || !THREAD_ID.test(threadId)) deny();
    const thread = await repo.get(threadId);
    if (!thread || thread.tenantKey !== tenantKey || thread.subject !== principal.subject ||
        (strictBinding && (thread.fabricTenantId !== principal.fabricTenantId ||
         typeof thread.agentUID !== "string" || !thread.agentUID))) deny();
    await requireGrant({ principal, tenantKey, agentKey: thread.agentKey, action, authorize });
    if (strictBinding) await requirePinnedAgent(thread);
    return thread;
  }
  async function requirePinnedAgent(thread) {
    const target = await resolveAgent({ tenantKey, agentKey: thread.agentKey });
    if (!target || target.agentKey !== thread.agentKey ||
        target.tenantKey !== tenantKey ||
        target.fabricTenantId !== thread.fabricTenantId ||
        target.agentUID !== thread.agentUID ||
        target.tenantBundleUID !== thread.tenantBundleUID ||
        target.apiServerEnabled !== true || target.endpointVerified !== true) deny();
    requireTarget(target, tenantKey, thread.agentKey);
    return target;
  }
  return {
    async canAccess({ principal, agentKey }) {
      requirePrincipal(principal, tenantKey);
      strictPrincipal(principal);
      await requireGrant({ principal, tenantKey, agentKey, action: "agent.chat", authorize });
      if (strictBinding) {
        const target = await resolveAgent({ tenantKey, agentKey });
        if (target?.fabricTenantId !== principal.fabricTenantId ||
            target.apiServerEnabled !== true || target.endpointVerified !== true) deny();
        requireTarget(target, tenantKey, agentKey);
      }
      return true;
    },
    async create({ principal, agentKey }) {
      requirePrincipal(principal, tenantKey);
      strictPrincipal(principal);
      await requireGrant({ principal, tenantKey, agentKey, action: "agent.chat", authorize });
      const target = await resolveAgent({ tenantKey, agentKey });
      requireTarget(target, tenantKey, agentKey);
      if (strictBinding && (target.fabricTenantId !== principal.fabricTenantId ||
          typeof target.agentUID !== "string" || !target.agentUID ||
          typeof target.tenantBundleUID !== "string" || !target.tenantBundleUID ||
          target.apiServerEnabled !== true || target.endpointVerified !== true)) deny();
      const sessionId = await openRemoteSession({ target, tenantKey, agentKey, fetchImpl, strictBinding });
      if (typeof sessionId !== "string" || !REMOTE_ID.test(sessionId)) {
        throw new Error("Agent is unavailable");
      }
      const record = {
        id: randomBytes(20).toString("hex"), tenantKey, subject: principal.subject,
        agentKey, sessionId, createdAt: new Date().toISOString(),
        ...(strictBinding ? { fabricTenantId: principal.fabricTenantId,
          agentUID: target.agentUID, tenantBundleUID: target.tenantBundleUID } : {}),
      };
      await repo.insert(record);
      return { id: record.id, agentKey: record.agentKey, createdAt: record.createdAt };
    },
    async list({ principal }) {
      requirePrincipal(principal, tenantKey);
      strictPrincipal(principal);
      const records = await repo.list({ tenantKey, subject: principal.subject });
      const visible = [];
      for (const r of records) {
        if (r.tenantKey !== tenantKey || r.subject !== principal.subject ||
            (strictBinding && r.fabricTenantId !== principal.fabricTenantId)) continue;
        try {
          if (await authorize({ principal, tenantKey, agentKey: r.agentKey, action: "agent.chat" }) === true) {
            if (strictBinding) await requirePinnedAgent(r);
            visible.push({ id: r.id, agentKey: r.agentKey, createdAt: r.createdAt });
          }
        } catch { /* Removed/foreign AgentIdentity or unavailable source: hide. */ }
      }
      return visible;
    },
    async *turn({ principal, threadId, turnId, input, signal }) {
      if (strictBinding && (typeof turnId !== "string" || !TURN_ID.test(turnId))) deny();
      const thread = await boundThread({ principal, threadId, action: "agent.chat" });
      const release = await repo.claim(thread.id);
      try {
        // Recheck after acquiring the atomic lease: ownership and grants may
        // have changed while waiting for another replica.
        await requireGrant({ principal, tenantKey, agentKey: thread.agentKey, action: "agent.chat", authorize });
        if (strictBinding) {
          await requirePinnedAgent(thread);
          // Atomic persistent uniqueness across replicas: a cancelled / lost
          // HTTP response cannot result in a second upstream Hermes turn.
          // There is deliberately no replay/resume fallback on duplicates.
          if (await repo.claimTurn({ threadId: thread.id, tenantKey,
            subject: principal.subject, turnId }) !== true) deny();
        }
        yield* streamAuthorizedTurn({
          principal, tenantKey, agentKey: thread.agentKey, sessionId: thread.sessionId,
          input, authorize, resolveAgent, fetchImpl, signal,
          ...(strictBinding ? { expectedAgentUID: thread.agentUID,
            expectedTenantBundleUID: thread.tenantBundleUID,
            expectedFabricTenantId: thread.fabricTenantId, strictTransport: true } : {}),
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
