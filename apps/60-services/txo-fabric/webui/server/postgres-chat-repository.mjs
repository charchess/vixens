import { randomBytes } from "node:crypto";

// This module is SOURCE ONLY. It is NOT composed by the production BFF.
// All authority is supplied by a separately authenticated, fresh-authorized
// BFF resolver. Validating a shape here is necessary but NOT IAM authorization.
const TENANT_KEY = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const FABRIC_TENANT_ID = /^TEN[0-9]{5,}$/;
const FABRIC_USER_ID = /^usr[a-f0-9]{32}$/;
const AGENT_KEY = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const UID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
const THREAD_ID = /^[a-f0-9]{40}$/;
const TURN_ID = /^[a-f0-9]{32}$/;
const REMOTE_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const OUTCOMES = new Set(["completed", "failed", "cancelled"]);
const MAX_MS = 8_640_000_000_000_000;
const fail = () => { throw new Error("Fabric chat repository unavailable"); };
const safeTime = value => Number.isSafeInteger(value) && value >= 0 && value <= MAX_MS;

// Always include the COMPLETE immutable server-verifiable scope in SQL,
// including UID incarnation and observed AgentIdentity generation. Never
// accept a browser-supplied user, tenant, agent, session or lease as authority.
function binding(scope) {
  if (!scope || typeof scope !== "object" || Array.isArray(scope) ||
      Object.keys(scope).sort().join(",") !== [
        "agentKey", "agentObservedGeneration", "agentUID", "fabricTenantId",
        "fabricUserId", "tenantBundleUID", "tenantKey",
      ].sort().join(",") ||
      !TENANT_KEY.test(scope.tenantKey || "") ||
      !FABRIC_TENANT_ID.test(scope.fabricTenantId || "") ||
      scope.fabricTenantId.length > 64 ||
      !FABRIC_USER_ID.test(scope.fabricUserId || "") ||
      !AGENT_KEY.test(scope.agentKey || "") ||
      !UID.test(scope.agentUID || "") || !UID.test(scope.tenantBundleUID || "") ||
      !Number.isSafeInteger(scope.agentObservedGeneration) ||
      scope.agentObservedGeneration < 1) fail();
  return [
    scope.tenantKey, scope.fabricTenantId, scope.tenantBundleUID,
    scope.fabricUserId, scope.agentKey, scope.agentUID,
    scope.agentObservedGeneration,
  ];
}
function threadKey(threadId) {
  if (typeof threadId !== "string" || !THREAD_ID.test(threadId)) fail();
  return threadId;
}
function turnKey(turnId) {
  if (typeof turnId !== "string" || !TURN_ID.test(turnId)) fail();
  return turnId;
}
const scopeWhere = `tenant_key = $2 AND fabric_tenant_id = $3 AND
 tenant_bundle_uid = $4 AND fabric_user_id = $5 AND
 agent_key = $6 AND agent_identity_uid = $7 AND
 agent_observed_generation = $8`;
function fields(scope, threadId) {
  return [threadKey(threadId), ...binding(scope)];
}
function time(value) {
  if (typeof value !== "string" || !/^(0|[1-9][0-9]{0,15})$/.test(value)) fail();
  const n = Number(value);
  if (!safeTime(n)) fail();
  return n;
}
function resultShape(result) {
  if (!result || !Number.isSafeInteger(result.rowCount) ||
      result.rowCount < 0 || !Array.isArray(result.rows) ||
      result.rows.length > result.rowCount) fail();
  return result;
}
function project(row) {
  if (!row || !THREAD_ID.test(row.thread_id || "") ||
      !AGENT_KEY.test(row.agent_key || "") || !row.created_at_ms) fail();
  const createdAt = time(row.created_at_ms);
  return Object.freeze({ id: row.thread_id, agentKey: row.agent_key,
    createdAt: new Date(createdAt).toISOString() });
}
const select = `SELECT thread_id, agent_key, created_at_ms
 FROM txo_fabric_chat_threads`;
const scoped = `thread_id = $1 AND ${scopeWhere}`;
const utcMillis = "(extract(epoch from clock_timestamp()) * 1000)::bigint";

// Future privileged BFF uses one dedicated PLATFORM PostgreSQL role with
// SELECT/INSERT/UPDATE on 003's tables. No migration, credentials, pool factory,
// Secret or runtime K8s dependency is created here.
export function createPostgresFabricChatRepository({
  pool, random = randomBytes, leaseSeconds = 60,
} = {}) {
  if (typeof pool?.query !== "function" ||
      typeof pool?.connect !== "function" ||
      typeof random !== "function" ||
      !Number.isSafeInteger(leaseSeconds) || leaseSeconds < 10 ||
      leaseSeconds > 120) fail();

  const token = length => {
    let value;
    try { value = random(length); } catch { fail(); }
    if (!Buffer.isBuffer(value) || value.length !== length) fail();
    return value.toString("hex");
  };
  async function query(sql, args) {
    try { return resultShape(await pool.query(sql, args)); }
    catch { fail(); }
  }
  async function transaction(work) {
    let client;
    try {
      client = await pool.connect();
      if (!client || typeof client.query !== "function" ||
          typeof client.release !== "function") fail();
      await client.query("BEGIN");
      const value = await work(async (sql, params = []) =>
        resultShape(await client.query(sql, params)));
      await client.query("COMMIT");
      return value;
    } catch {
      if (client) { try { await client.query("ROLLBACK"); } catch { /* fail closed */ } }
      fail();
    } finally {
      if (client) client.release();
    }
  }
  async function lockedThread(tx, scope, threadId) {
    const r = await tx(`SELECT remote_session_id, lease_token,
        lease_expires_at_ms, lease_uncertain, ${utcMillis} AS db_now_ms
      FROM txo_fabric_chat_threads WHERE ${scoped} FOR UPDATE`,
    fields(scope, threadId));
    if (r.rowCount === 0) return null;
    if (r.rowCount !== 1 || r.rows.length !== 1) fail();
    return r.rows[0];
  }

  return Object.freeze({
    async createThread({ scope } = {}) {
      const b = binding(scope);
      const threadId = token(20); // 160-bit server-created opaque Fabric ID
      const r = await query(`INSERT INTO txo_fabric_chat_threads
        (thread_id, tenant_key, fabric_tenant_id, tenant_bundle_uid,
         fabric_user_id, agent_key, agent_identity_uid, agent_observed_generation,
         created_at_ms, updated_at_ms)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,${utcMillis},${utcMillis})
        RETURNING thread_id, agent_key, created_at_ms`, [threadId, ...b]);
      if (r.rowCount !== 1 || r.rows.length !== 1) fail();
      return project(r.rows[0]);
    },

    async getThread({ scope, threadId } = {}) {
      const r = await query(`${select} WHERE ${scoped}`, fields(scope, threadId));
      if (r.rowCount === 0) return null;
      if (r.rowCount !== 1 || r.rows.length !== 1) fail();
      return project(r.rows[0]); // NO remote session, lease or credential
    },

    async listThreads({ scope, limit = 100 } = {}) {
      const b = binding(scope);
      if (!Number.isSafeInteger(limit) || limit < 1 || limit > 100) fail();
      const r = await query(`${select} WHERE ${scopeWhere.replaceAll(/\$(\d+)/g,
        (_, n) => "$" + (Number(n) - 1))}
        ORDER BY created_at_ms ASC, thread_id ASC LIMIT $8`, [...b, limit]);
      if (r.rowCount !== r.rows.length || r.rowCount > limit) fail();
      return r.rows.map(project);
    },

    // INTERNAL only. Never send the value to browser list/get or accept a
    // remote session ID from a browser endpoint. Ownership + fresh FGA remain
    // prerequisites enforced by the caller on EVERY read/action.
    async getServerThread({ scope, threadId } = {}) {
      const r = await query(`SELECT remote_session_id FROM
        txo_fabric_chat_threads WHERE ${scoped}`, fields(scope, threadId));
      if (r.rowCount === 0) return null;
      if (r.rowCount !== 1 || r.rows.length !== 1) fail();
      const sessionId = r.rows[0].remote_session_id;
      if (sessionId !== null && !REMOTE_ID.test(sessionId || "")) fail();
      return Object.freeze({ remoteSessionId: sessionId });
    },

    // Called ONLY AFTER a trusted Hermes adapter creates the remote session.
    // Atomic conditional UPDATE + unique agent-UID/session index prevent
    // replacing or adopting another thread's remote session, even on races.
    async bindHermesSession({ scope, threadId, remoteSessionId } = {}) {
      if (typeof remoteSessionId !== "string" || !REMOTE_ID.test(remoteSessionId)) fail();
      const params = [...fields(scope, threadId), remoteSessionId];
      const updated = await query(`UPDATE txo_fabric_chat_threads
        SET remote_session_id = $9, updated_at_ms = ${utcMillis}
        WHERE ${scoped} AND remote_session_id IS NULL
        RETURNING thread_id`, params);
      if (updated.rowCount === 1) return Object.freeze({ kind: "bound" });
      if (updated.rowCount !== 0) fail();
      const previous = await query(`SELECT remote_session_id
        FROM txo_fabric_chat_threads WHERE ${scoped}`, params.slice(0, 8));
      if (previous.rowCount === 0) return Object.freeze({ kind: "denied" });
      if (previous.rowCount !== 1 || previous.rows.length !== 1) fail();
      return Object.freeze({ kind: "conflict" }); // no replacement, even same value
    },

    // PostgreSQL ROW LOCK (FOR UPDATE) serializes decisions for a single
    // thread across replicas. The gap for a new turn is protected by that
    // lock, the PRIMARY KEY and one atomic BEGIN/COMMIT. No local mutex.
    async claimTurn({ scope, threadId, turnId } = {}) {
      const id = turnKey(turnId), key = fields(scope, threadId);
      return transaction(async tx => {
        const thread = await lockedThread(tx, scope, threadId);
        if (!thread) return Object.freeze({ kind: "denied" });
        const prior = await tx(`SELECT state, result_code
          FROM txo_fabric_chat_turns WHERE thread_id=$1 AND turn_id=$2`, [threadId, id]);
        if (prior.rowCount === 1) {
          const row = prior.rows[0];
          if (!row) fail();
          if (OUTCOMES.has(row.state) && row.result_code === row.state) {
            return Object.freeze({ kind: "terminal", outcome: row.state });
          }
          if (row.state !== "claimed") fail();
          return Object.freeze({ kind: "conflict" });
        }
        if (prior.rowCount !== 0) fail();
        if (thread.remote_session_id === null) return Object.freeze({ kind: "denied" });
        if (!REMOTE_ID.test(thread.remote_session_id)) fail();
        if (thread.lease_token !== null) {
          if (!TURN_ID.test(thread.lease_token || "")) fail();
          if (thread.lease_uncertain) return Object.freeze({ kind: "uncertain" });
          if (time(thread.lease_expires_at_ms) <= time(thread.db_now_ms)) {
            await tx(`UPDATE txo_fabric_chat_threads
              SET lease_uncertain=true, updated_at_ms=${utcMillis}
              WHERE ${scoped} AND lease_token IS NOT NULL`, key);
            return Object.freeze({ kind: "uncertain" });
          }
          return Object.freeze({ kind: "conflict" });
        }
        const leaseToken = token(16);
        const began = time(thread.db_now_ms);
        const expiresAt = began + leaseSeconds * 1000;
        if (!safeTime(expiresAt)) fail();
        const r = await tx(`INSERT INTO txo_fabric_chat_turns
          (thread_id, turn_id, state, lease_token, started_at_ms)
          VALUES ($1,$2,'claimed',$3,$4)`, [threadId, id, leaseToken, began]);
        if (r.rowCount !== 1) fail();
        const update = await tx(`UPDATE txo_fabric_chat_threads
          SET lease_token=$9, lease_expires_at_ms=$10,
              updated_at_ms=${utcMillis}
          WHERE ${scoped} AND lease_token IS NULL
          RETURNING thread_id`, [...key, leaseToken, expiresAt]);
        if (update.rowCount !== 1) fail();
        return Object.freeze({ kind: "claimed", leaseToken, expiresAt });
      });
    },

    async finalizeTurn({ scope, threadId, turnId, leaseToken, outcome } = {}) {
      const id = turnKey(turnId), key = fields(scope, threadId);
      if (typeof leaseToken !== "string" || !TURN_ID.test(leaseToken) ||
          !OUTCOMES.has(outcome)) fail();
      return transaction(async tx => {
        const thread = await lockedThread(tx, scope, threadId);
        if (!thread) return Object.freeze({ kind: "denied" });
        const old = await tx(`SELECT state, result_code, lease_token
          FROM txo_fabric_chat_turns WHERE thread_id=$1 AND turn_id=$2`, [threadId, id]);
        if (old.rowCount === 0) return Object.freeze({ kind: "denied" });
        if (old.rowCount !== 1 || old.rows.length !== 1) fail();
        if (old.rows[0].lease_token !== leaseToken) return Object.freeze({ kind: "denied" });
        if (OUTCOMES.has(old.rows[0].state)) {
          return Object.freeze({ kind: old.rows[0].state === outcome ? "terminal" : "conflict",
            outcome: old.rows[0].state });
        }
        if (old.rows[0].state !== "claimed" || thread.lease_token !== leaseToken) {
          return Object.freeze({ kind: "conflict" });
        }
        const updated = await tx(`UPDATE txo_fabric_chat_turns
          SET state=$3, result_code=$3, finished_at_ms=${utcMillis}
          WHERE thread_id=$1 AND turn_id=$2 AND state='claimed'
          RETURNING state`, [threadId, id, outcome]);
        if (updated.rowCount !== 1) fail();
        const released = await tx(`UPDATE txo_fabric_chat_threads
          SET lease_token=NULL, lease_expires_at_ms=NULL, lease_uncertain=false,
              updated_at_ms=${utcMillis}
          WHERE ${scoped} AND lease_token=$9 RETURNING thread_id`,
          [...key, leaseToken]);
        if (released.rowCount !== 1) fail();
        return Object.freeze({ kind: "terminal", outcome });
      });
    },

    async inspectTurn({ scope, threadId, turnId } = {}) {
      const r = await query(`SELECT t.state, t.result_code
        FROM txo_fabric_chat_turns AS t
        JOIN txo_fabric_chat_threads AS h ON h.thread_id=t.thread_id
        WHERE h.${scoped.replaceAll(/\b(tenant_key|fabric_tenant_id|tenant_bundle_uid|fabric_user_id|agent_key|agent_identity_uid|agent_observed_generation)\b/g,
          m => "h." + m)} AND t.turn_id=$9`, [...fields(scope, threadId), turnKey(turnId)]);
      if (r.rowCount === 0) return Object.freeze({ kind: "absent" });
      if (r.rowCount !== 1 || r.rows.length !== 1) fail();
      if (OUTCOMES.has(r.rows[0].state) && r.rows[0].result_code === r.rows[0].state) {
        return Object.freeze({ kind: "terminal", outcome: r.rows[0].state });
      }
      if (r.rows[0].state !== "claimed") fail();
      return Object.freeze({ kind: "conflict" });
    },

    // Expiry is NOT evidence Hermes stopped. Mark uncertain and BLOCK
    // further claims until a separate, verified physical recovery protocol.
    // A still-owning run may finalize using its fencing token; a new owner
    // is NEVER elected merely because wall-clock time passed.
    async expireStaleLease({ scope, threadId } = {}) {
      const key = fields(scope, threadId);
      return transaction(async tx => {
        const row = await lockedThread(tx, scope, threadId);
        if (!row) return Object.freeze({ kind: "denied" });
        if (row.lease_token === null) return Object.freeze({ kind: "idle" });
        if (row.lease_uncertain) return Object.freeze({ kind: "uncertain" });
        if (time(row.lease_expires_at_ms) > time(row.db_now_ms)) {
          return Object.freeze({ kind: "active" });
        }
        const updated = await tx(`UPDATE txo_fabric_chat_threads
          SET lease_uncertain=true, updated_at_ms=${utcMillis}
          WHERE ${scoped} AND lease_token IS NOT NULL`, key);
        if (updated.rowCount !== 1) fail();
        return Object.freeze({ kind: "uncertain" });
      });
    },
  });
}
