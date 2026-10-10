import { createHash } from "node:crypto";

const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const BASE64URL43 = /^[A-Za-z0-9_-]{43}$/;
const MAX_TRANSACTION_AGE_MS = 300_000;
const MAX_STRING_BYTES = 1024;
const TABLE = "txo_fabric_oidc_login_transactions";

const INSERT = `INSERT INTO ${TABLE}
  (tenant_key, state_digest, issuer, client_id, redirect_uri,
   nonce, pkce_verifier, binding_digest, issued_at_ms, expires_at_ms)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`;
const CONSUME = `DELETE FROM ${TABLE}
  WHERE tenant_key = $1 AND state_digest = $2
  RETURNING issuer, client_id, redirect_uri, nonce, pkce_verifier,
            binding_digest, issued_at_ms, expires_at_ms`;
// PostgreSQL does not have DELETE LIMIT. Bounded pruning avoids scanning
// arbitrarily many expired rows during a login, with the expiry index.
const PRUNE = `DELETE FROM ${TABLE}
  WHERE (tenant_key, state_digest) IN (
    SELECT tenant_key, state_digest FROM ${TABLE}
    WHERE expires_at_ms < $1 ORDER BY expires_at_ms ASC LIMIT 100
  )`;

const fail = () => { throw new Error("Fabric OAuth transaction store unavailable"); };
const safeTimestamp = (value) =>
  Number.isSafeInteger(value) && value >= 0 && value <= 8_640_000_000_000_000;
const validKey = (value) => typeof value === "string" && BASE64URL43.test(value) &&
  Buffer.from(value, "base64url").toString("base64url") === value;
const sha256 = (state) => createHash("sha256").update(state).digest("base64url");

function requireInput(tenantKey, state) {
  if (typeof tenantKey !== "string" || !TENANT.test(tenantKey) || !validKey(state)) fail();
  return sha256(state);
}

function requiredString(value) {
  if (typeof value !== "string" || value.length < 1 ||
      Buffer.byteLength(value) > MAX_STRING_BYTES || /[\u0000-\u001f\u007f]/.test(value)) fail();
  return value;
}

function timestamp(value) {
  // node-postgres returns PostgreSQL BIGINT (int8) as a decimal string.
  if (typeof value !== "string" || !/^(0|[1-9][0-9]{0,15})$/.test(value)) fail();
  const n = Number(value);
  if (!safeTimestamp(n)) fail();
  return n;
}

/**
 * Durable, cross-replica adapter for createFabricOIDCLogin transactions.
 *
 * Pass a PRIVILEGED PLATFORM-OWNED PostgreSQL query interface (e.g. pg Pool)
 * with a dedicated, least-privileged database role. Call the migration once
 * through GitOps before mounting the real BFF. Never share this role or table
 * with tenant pods, browsers or Authentik clients.
 *
 * An atomic DELETE ... RETURNING, scoped by server-owned tenant + SHA-256
 * state digest, guarantees ONE successful consume across BFF replicas and
 * fail-closed replay. No SELECT-then-DELETE window or process-local Map.
 * DB query errors are deliberately redacted.
 *
 * PKCE verifiers/nonce are short-lived sensitive data: PostgreSQL requires
 * encrypted disks/backups and strict read grants. Do not log SQL parameters,
 * decoded rows or pool errors. This module does NOT create sessions, API
 * routes, grants, or a second IAM source of truth.
 */
export function createPostgresOIDCTransactions({ pool, now = () => Date.now() } = {}) {
  if (!pool || typeof pool.query !== "function" || typeof now !== "function") fail();

  async function query(sql, args) {
    try {
      const result = await pool.query(sql, args);
      if (!result || !Number.isSafeInteger(result.rowCount) ||
          !Array.isArray(result.rows)) fail();
      return result;
    } catch { fail(); }
  }

  return Object.freeze({
    async insert({ tenantKey, state, value, expiresAt } = {}) {
      const digest = requireInput(tenantKey, state);
      if (!value || typeof value !== "object" || Array.isArray(value) ||
          !validKey(value.nonce) || !validKey(value.pkceVerifier) ||
          !validKey(value.bindingDigest) || !safeTimestamp(value.issuedAt) ||
          !safeTimestamp(expiresAt) ||
          expiresAt !== value.issuedAt + MAX_TRANSACTION_AGE_MS) fail();
      const issuer = requiredString(value.issuer);
      const clientId = requiredString(value.clientId);
      const redirectURI = requiredString(value.redirectURI);
      // No upsert: a collision must fail, never overwrite an existing login.
      const result = await query(INSERT, [
        tenantKey, digest, issuer, clientId, redirectURI,
        value.nonce, value.pkceVerifier, value.bindingDigest,
        value.issuedAt, expiresAt,
      ]);
      if (result.rowCount !== 1) fail();
    },

    async consume({ tenantKey, state } = {}) {
      const digest = requireInput(tenantKey, state);
      const result = await query(CONSUME, [tenantKey, digest]);
      if (result.rowCount === 0 && result.rows.length === 0) return null;
      if (result.rowCount !== 1 || result.rows.length !== 1) fail();
      const row = result.rows[0];
      if (!row || !validKey(row.nonce) || !validKey(row.pkce_verifier) ||
          !validKey(row.binding_digest)) fail();
      const issuedAt = timestamp(row.issued_at_ms);
      const expiresAt = timestamp(row.expires_at_ms);
      if (expiresAt !== issuedAt + MAX_TRANSACTION_AGE_MS) fail();
      return Object.freeze({
        issuer: requiredString(row.issuer),
        clientId: requiredString(row.client_id),
        redirectURI: requiredString(row.redirect_uri),
        nonce: row.nonce, pkceVerifier: row.pkce_verifier,
        bindingDigest: row.binding_digest, issuedAt,
      });
    },

    // Opportunistic *bounded* cleanup. Invoke during BFF login or a private
    // maintenance reconciler. Use db clock in production for multi-pod safety;
    // the passed JS clock enables deterministic unit tests.
    async pruneExpired() {
      let current;
      try { current = now(); } catch { fail(); }
      if (!safeTimestamp(current)) fail();
      const result = await query(PRUNE, [current]);
      if (result.rowCount < 0 || result.rowCount > 100) fail();
      return result.rowCount;
    },
  });
}
