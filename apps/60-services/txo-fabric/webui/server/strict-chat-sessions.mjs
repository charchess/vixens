import { requireInternalHermesURL } from "./hermes-client.mjs";
import { streamAuthorizedTurn } from "./authorized-turn.mjs";

// Source-only strict Fabric chat adapter. The caller supplies trusted,
// independently authenticated IAM/AgentIdentity collaborators. This module
// never creates a database pool, credential, route or live runtime connection.
const AGENT = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const TENANT = AGENT;
const TENANT_ID = /^TEN[0-9]{5,}$/;
const USER = /^usr[a-f0-9]{32}$/;
const UID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
const THREAD = /^[a-f0-9]{40}$/;
const TURN = /^[a-f0-9]{32}$/;
const REMOTE = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const OUTCOME = new Set(["completed", "failed", "cancelled"]);
const REQUIRED = [
  "createThread", "getThread", "listThreads", "getServerThread",
  "bindHermesSession", "claimTurn", "finalizeTurn",
];
const deny = () => { throw new Error("Forbidden"); };
const unavailable = () => { throw new Error("Agent is unavailable"); };

export function createStrictChatSessionService({
  tenantKey, authorize, resolveAgent, listAgentKeys, repo,
  openRemoteSession, fetchImpl,
}) {
  if (!TENANT.test(tenantKey || "") || typeof authorize !== "function" ||
      typeof resolveAgent !== "function" || typeof listAgentKeys !== "function" ||
      typeof openRemoteSession !== "function" || typeof fetchImpl !== "function" ||
      !repo || REQUIRED.some(method => typeof repo[method] !== "function")) {
    throw new Error("Chat security dependencies are required");
  }

  function requirePrincipal(principal) {
    if (principal?.authenticated !== true || principal?.verified !== true ||
        principal?.tenantKey !== tenantKey ||
        !TENANT_ID.test(principal?.fabricTenantId || "") ||
        !USER.test(principal?.fabricUserId || "") ||
        principal?.subject !== principal.fabricUserId) deny();
  }

  async function freshScope(principal, agentKey, expected) {
    requirePrincipal(principal);
    if (!AGENT.test(agentKey || "")) deny();
    let target;
    try {
      if (await authorize({ principal, tenantKey, agentKey, action: "agent.chat" }) !== true) deny();
      target = await resolveAgent({ tenantKey, agentKey });
      if (!target || target.tenantKey !== tenantKey ||
          target.fabricTenantId !== principal.fabricTenantId ||
          target.agentKey !== agentKey ||
          !UID.test(target.tenantBundleUID || "") ||
          !UID.test(target.agentUID || "") ||
          !Number.isSafeInteger(target.agentObservedGeneration) ||
          target.agentObservedGeneration < 1 ||
          target.apiServerEnabled !== true || target.endpointVerified !== true ||
          typeof target.apiKey !== "string" || !target.apiKey ||
          /[\r\n]/.test(target.apiKey)) deny();
      requireInternalHermesURL(target.baseURL);
    } catch { deny(); }
    const scope = Object.freeze({
      tenantKey, fabricTenantId: principal.fabricTenantId,
      tenantBundleUID: target.tenantBundleUID,
      fabricUserId: principal.fabricUserId, agentKey,
      agentUID: target.agentUID,
      agentObservedGeneration: target.agentObservedGeneration,
    });
    if (expected && Object.keys(scope).some(key => scope[key] !== expected[key])) deny();
    return { scope, target };
  }

  // Listing and opaque-thread lookup cannot perform an unscoped DB query.
  // Only a trusted server-side agent catalog may supply candidate keys.
  async function candidates(principal) {
    requirePrincipal(principal);
    let values;
    try { values = await listAgentKeys({ principal, tenantKey }); }
    catch { unavailable(); }
    if (!Array.isArray(values) || values.length > 500 ||
        values.some(key => typeof key !== "string" || !AGENT.test(key))) unavailable();
    return [...new Set(values)];
  }

  function publicThread(record, scope) {
    if (!record || typeof record !== "object" || !THREAD.test(record.id || "") ||
        record.agentKey !== scope.agentKey ||
        typeof record.createdAt !== "string" ||
        !Number.isFinite(Date.parse(record.createdAt))) unavailable();
    return Object.freeze({
      id: record.id, agentKey: scope.agentKey, createdAt: record.createdAt,
    });
  }

  async function repository(method, args) {
    try { return await repo[method](args); } catch { unavailable(); }
  }

  async function requireBound(principal, threadId, scope) {
    // Revalidate before this *separate* privileged repository read.
    await freshScope(principal, scope.agentKey, scope);
    const internal = await repository("getServerThread", { scope, threadId });
    if (!internal || !REMOTE.test(internal.remoteSessionId || "")) deny();
    return internal.remoteSessionId;
  }

  async function boundThread(principal, threadId) {
    requirePrincipal(principal);
    if (!THREAD.test(threadId || "")) deny();
    for (const agentKey of await candidates(principal)) {
      let scope;
      try { ({ scope } = await freshScope(principal, agentKey)); }
      catch { continue; } // An ungranted, missing or replaced agent is invisible.
      const result = await repository("getThread", { scope, threadId });
      if (!result) continue;
      const thread = publicThread(result, scope);
      const sessionId = await requireBound(principal, threadId, scope);
      return { thread, sessionId, scope };
    }
    deny();
  }

  return Object.freeze({
    async canAccess({ principal, agentKey }) {
      await freshScope(principal, agentKey);
      return true;
    },

    async create({ principal, agentKey }) {
      const { scope, target } = await freshScope(principal, agentKey);
      // Persist an opaque thread *before* creating the remote session.
      const thread = publicThread(await repository("createThread", { scope }), scope);
      // Revalidate before privileged upstream work; never adopt stale targets.
      await freshScope(principal, agentKey, scope);
      let remoteSessionId;
      try {
        remoteSessionId = await openRemoteSession({
          target, tenantKey, agentKey, fetchImpl, strictBinding: true,
        });
      } catch { unavailable(); }
      if (typeof remoteSessionId !== "string" || !REMOTE.test(remoteSessionId)) unavailable();
      const bound = await repository("bindHermesSession", {
        scope, threadId: thread.id, remoteSessionId,
      });
      // A conflicted/failed bind is a recoverable orphan. NEVER retry, replace,
      // adopt, or issue an unsafe compensating remote delete.
      if (bound?.kind !== "bound") unavailable();
      return thread;
    },

    async list({ principal }) {
      const visible = [];
      for (const agentKey of await candidates(principal)) {
        let scope;
        try { ({ scope } = await freshScope(principal, agentKey)); }
        catch { continue; }
        const records = await repository("listThreads", { scope, limit: 100 });
        if (!Array.isArray(records) || records.length > 100) unavailable();
        for (const record of records) {
          try {
            const thread = publicThread(record, scope);
            await requireBound(principal, thread.id, scope);
            visible.push(thread);
          } catch { /* A vanished grant, incarnation or unbound thread is hidden. */ }
        }
      }
      return visible.sort((a, b) => a.createdAt.localeCompare(b.createdAt) || a.id.localeCompare(b.id)).slice(0, 100);
    },

    async get({ principal, threadId }) {
      return (await boundThread(principal, threadId)).thread;
    },

    async *turn({ principal, threadId, turnId, input, signal }) {
      if (!TURN.test(turnId || "") || typeof input !== "string" ||
          !input.trim() || input.length > 100000) deny();
      const { thread, sessionId, scope } = await boundThread(principal, threadId);
      await freshScope(principal, thread.agentKey, scope);
      const claim = await repository("claimTurn", { scope, threadId, turnId });
      if (claim?.kind === "terminal") {
        if (!OUTCOME.has(claim.outcome)) unavailable();
        yield { type: "done", status: claim.outcome };
        return; // Bounded prior status only; NEVER resume or replay upstream.
      }
      if (claim?.kind === "conflict" || claim?.kind === "uncertain") {
        throw new Error("Chat turn conflict");
      }
      if (claim?.kind !== "claimed" || !TURN.test(claim.leaseToken || "")) unavailable();

      let outcome = "cancelled"; // Early generator close / disconnected consumer.
      let receivedDone = false;
      let failure = false;
      try {
        // A grant may disappear after claim, even on another BFF replica.
        await freshScope(principal, thread.agentKey, scope);
        for await (const event of streamAuthorizedTurn({
          principal, tenantKey, agentKey: thread.agentKey, sessionId,
          input, authorize, resolveAgent, fetchImpl, signal,
          expectedAgentUID: scope.agentUID,
          expectedTenantBundleUID: scope.tenantBundleUID,
          expectedFabricTenantId: scope.fabricTenantId,
          expectedAgentObservedGeneration: scope.agentObservedGeneration,
          strictTransport: true,
        })) {
          if (signal?.aborted) break;
          if (receivedDone) unavailable();
          if (event.type === "done") {
            if (!OUTCOME.has(event.status)) unavailable();
            outcome = event.status;
            receivedDone = true;
          } else {
            yield event;
          }
        }
        if (signal?.aborted) outcome = "cancelled";
        else if (!receivedDone) unavailable(); // No trustworthy terminal frame.
      } catch {
        failure = true;
        outcome = signal?.aborted ? "cancelled" : "failed";
      } finally {
        // Only this private fenced token may finalize. On a DB outage do not
        // invent a terminal result, release the lease, or replay the POST.
        const final = await repository("finalizeTurn", {
          scope, threadId, turnId, leaseToken: claim.leaseToken, outcome,
        });
        if (final?.kind !== "terminal" || final.outcome !== outcome) unavailable();
      }
      if (failure) unavailable();
      if (!signal?.aborted) yield { type: "done", status: outcome };
    },
  });
}
