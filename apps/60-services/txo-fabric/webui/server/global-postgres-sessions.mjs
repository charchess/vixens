import { createHash, randomBytes } from "node:crypto";

const TOKEN = /^[A-Za-z0-9_-]{43}$/;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const SUBJECT = /^[!-~]{1,255}$/;
const ISSUER = /^https:\/\/[a-z0-9.-]+\/application\/o\/txo-fabric-webui\/$/;
const TTL = 8 * 60 * 60 * 1000;
const MAX_MS = 8_640_000_000_000_000;
const fail = () => { throw new Error("Global Fabric session store unavailable"); };
const digest = (s) => createHash("sha256").update(s).digest("base64url");
const safeTime = (x) => Number.isSafeInteger(x) && x >= 0 && x <= MAX_MS;
const validToken = (s) => typeof s === "string" && TOKEN.test(s) &&
  Buffer.from(s, "base64url").toString("base64url") === s;
function stringTime(s) {
  if (typeof s !== "string" || !/^(0|[1-9][0-9]{0,15})$/.test(s)) fail();
  const n = Number(s);
  if (!safeTime(n)) fail();
  return n;
}

/**
 * Durable cross-replica human browser session store for the ONE global BFF.
 * Stores only a SHA256 digest of a CSPRNG cookie, global Authentik signed
 * identity and a bounded absolute expiry. NO Fabric user, tenant, grant,
 * provider key or mutable username is stored. The actual tenant membership,
 * active state and OpenFGA access must be revalidated on every operation.
 *
 * pool MUST belong to a platform-only CNPG role with narrow read/write
 * table grants, private NP and encrypted backups. Never use tenant or
 * OpenFGA credentials. SQL query params/results are never logged.
 */
export function createPostgresGlobalSessions({ pool, now = () => Date.now() } = {}) {
  if (typeof pool?.query !== "function" || typeof now !== "function") fail();
  async function query(sql, params) {
    try {
      const out = await pool.query(sql, params);
      if (!out || !Number.isSafeInteger(out.rowCount) || !Array.isArray(out.rows)) fail();
      return out;
    } catch { fail(); }
  }
  return Object.freeze({
    async issue({ issuer, oidcSubject, authentikUserUUID } = {}) {
      if (typeof issuer !== "string" || !ISSUER.test(issuer) ||
          typeof oidcSubject !== "string" || !SUBJECT.test(oidcSubject) ||
          typeof authentikUserUUID !== "string" || !UUID.test(authentikUserUUID)) fail();
      let t;
      try { t = now(); } catch { fail(); }
      if (!safeTime(t) || !safeTime(t + TTL)) fail();
      const token = randomBytes(32).toString("base64url");
      const result = await query(`INSERT INTO txo_fabric_webui_sessions
        (session_digest, issuer, oidc_subject, authentik_uuid, issued_at_ms, expires_at_ms)
        VALUES ($1, $2, $3, $4, $5, $6)`, [
        digest(token), issuer, oidcSubject, authentikUserUUID, t, t + TTL,
      ]);
      if (result.rowCount !== 1) fail();
      return Object.freeze({ token, maxAge: TTL / 1000 });
    },

    async authenticate(token) {
      if (!validToken(token)) return null;
      let t;
      try { t = now(); } catch { fail(); }
      if (!safeTime(t)) fail();
      const result = await query(`SELECT issuer, oidc_subject, authentik_uuid,
          issued_at_ms, expires_at_ms
        FROM txo_fabric_webui_sessions
        WHERE session_digest = $1 AND expires_at_ms > $2`, [digest(token), t]);
      if (result.rowCount === 0 && result.rows.length === 0) return null;
      if (result.rowCount !== 1 || result.rows.length !== 1) fail();
      const row = result.rows[0];
      if (!row || typeof row.issuer !== "string" || !ISSUER.test(row.issuer) ||
          typeof row.oidc_subject !== "string" || !SUBJECT.test(row.oidc_subject) ||
          typeof row.authentik_uuid !== "string" || !UUID.test(row.authentik_uuid)) fail();
      const issuedAt = stringTime(row.issued_at_ms);
      const expiresAt = stringTime(row.expires_at_ms);
      if (expiresAt !== issuedAt + TTL || t < issuedAt || t >= expiresAt) fail();
      // This indicates a signed login+server issued browser session, NOT
      // an active tenant membership, Fabric ID or FGA authorization grant.
      return Object.freeze({
        authenticated: true, verified: true,
        issuer: row.issuer,
        oidcSubject: row.oidc_subject,
        authentikUserUUID: row.authentik_uuid,
      });
    },

    async revoke(token) {
      if (!validToken(token)) return;
      const result = await query(`DELETE FROM txo_fabric_webui_sessions
        WHERE session_digest = $1`, [digest(token)]);
      if (result.rowCount < 0 || result.rowCount > 1) fail();
    },

    async pruneExpired() {
      let t;
      try { t = now(); } catch { fail(); }
      if (!safeTime(t)) fail();
      const result = await query(`DELETE FROM txo_fabric_webui_sessions
        WHERE session_digest IN (
          SELECT session_digest FROM txo_fabric_webui_sessions
          WHERE expires_at_ms <= $1
          ORDER BY expires_at_ms ASC LIMIT 100
        )`, [t]);
      if (result.rowCount < 0 || result.rowCount > 100) fail();
      return result.rowCount;
    },
  });
}
