import test from "node:test";
import assert from "node:assert/strict";
import { createHash, randomBytes } from "node:crypto";
import { readFileSync } from "node:fs";
import { createPostgresOIDCTransactions } from "../server/oidc-postgres-transactions.mjs";

const scope = "hairem";
const uuid = randomBytes(32).toString("base64url");
const nonce = randomBytes(32).toString("base64url");
const verifier = randomBytes(32).toString("base64url");
const binding = randomBytes(32).toString("base64url");
const issuedAt = 1_797_000_000_000;
const age = 300_000;
const sha256 = (s) => createHash("sha256").update(s).digest("base64url");
const record = (issued = issuedAt) => ({
  tenantKey: scope, state: uuid,
  value: {
    nonce, pkceVerifier: verifier, bindingDigest: sha256(binding),
    issuer: "https://authentik.truxonline.com/application/o/txo-fabric-hairem/",
    clientId: "txo-fabric-hairem",
    redirectURI: "https://chat-hairem.truxonline.com/auth/callback",
    issuedAt: issued,
  },
  expiresAt: issued + age,
});

// In-process SQL contract simulator; production uses PostgreSQL's atomic
// DELETE ... RETURNING statement (verified separately by rendered SQL).
// One shared fake DB is intentionally accessed from independent adapters.
function postgresSimulation() {
  const rows = new Map(), calls = [];
  let fail = false;
  return {
    rows, calls,
    setFailure(v) { fail = v; },
    async query(sql, args) {
      if (fail) throw Error("database password may not be leaked");
      calls.push({ sql, args });
      if (sql.startsWith("INSERT INTO txo_fabric_oidc_login_transactions")) {
        const [tenant_key, state_digest, issuer, client_id, redirect_uri,
          nonce_, pkce_verifier, binding_digest, issued_at_ms, expires_at_ms] = args;
        const key = tenant_key + ":" + state_digest;
        if (rows.has(key)) throw Error("duplicate key exposes internals");
        rows.set(key, {
          tenant_key, state_digest, issuer, client_id, redirect_uri,
          nonce: nonce_, pkce_verifier, binding_digest,
          issued_at_ms: String(issued_at_ms), expires_at_ms: String(expires_at_ms),
        });
        return { rowCount: 1, rows: [] };
      }
      if (sql.startsWith("DELETE FROM txo_fabric_oidc_login_transactions\n  WHERE tenant_key")) {
        const key = args[0] + ":" + args[1];
        const row = rows.get(key);
        rows.delete(key);
        return { rowCount: row ? 1 : 0, rows: row ? [row] : [] };
      }
      if (sql.startsWith("DELETE FROM txo_fabric_oidc_login_transactions\n  WHERE (tenant_key")) {
        const expired = [...rows].filter(([, v]) => Number(v.expires_at_ms) < args[0])
          .sort((a, b) => Number(a[1].expires_at_ms) - Number(b[1].expires_at_ms))
          .slice(0, 100);
        for (const [key] of expired) rows.delete(key);
        return { rowCount: expired.length, rows: [] };
      }
      throw Error("unexpected SQL");
    },
  };
}

test("migrated schema uses immutable tenant-scoped digest key and bounded expiry", () => {
  const migration = readFileSync(new URL("../db/migrations/001_oidc_login_transactions.sql", import.meta.url), "utf8");
  assert.match(migration, /PRIMARY KEY \(tenant_key, state_digest\)/);
  assert.match(migration, /expires_at_ms = issued_at_ms \+ 300000/);
  assert.match(migration, /CREATE INDEX IF NOT EXISTS txo_oidc_expiry/);
  assert.match(migration, /REVOKE ALL .* FROM PUBLIC/);
  assert.doesNotMatch(migration, /(?:PASSWORD|Bearer)\s+['"][^'"]{10}/i);
});

test("two replicas atomically consume one hashed state once, without storing original bearer state", async () => {
  const db = postgresSimulation();
  const a = createPostgresOIDCTransactions({ pool: db });
  const b = createPostgresOIDCTransactions({ pool: db });
  await a.insert(record());
  assert.equal(db.rows.size, 1);
  const [stored] = db.rows.values();
  assert.equal(stored.tenant_key, scope);
  assert.equal(stored.state_digest, sha256(uuid));
  assert.equal(JSON.stringify(stored).includes(uuid), false);
  assert.equal(db.calls[0].sql.includes("ON CONFLICT"), false);
  assert.match(db.calls[0].sql, /INSERT INTO/);
  const [one, two] = await Promise.all([a.consume({ tenantKey: scope, state: uuid }),
    b.consume({ tenantKey: scope, state: uuid })]);
  assert.equal([one, two].filter(Boolean).length, 1);
  const consumed = one || two;
  assert.deepEqual(consumed, { ...record().value });
  assert.equal(Object.isFrozen(consumed), true);
  assert.equal(db.rows.size, 0);
  assert.equal(await a.consume({ tenantKey: scope, state: uuid }), null);
  assert.equal(db.calls[1].sql.includes("DELETE FROM"), true);
  assert.equal(db.calls[1].sql.includes("RETURNING"), true);
  assert.match(db.calls[1].sql, /WHERE tenant_key = \$1 AND state_digest = \$2/);
  assert.equal(db.calls[1].args[0], scope);
  assert.equal(db.calls[1].args[1], sha256(uuid));
});

test("cross-tenant state cannot consume or replace an existing transaction", async () => {
  const db = postgresSimulation();
  const a = createPostgresOIDCTransactions({ pool: db });
  await a.insert(record());
  const differentTenant = await a.consume({ tenantKey: "indiba", state: uuid });
  assert.equal(differentTenant, null);
  assert.equal(db.rows.size, 1);
  await assert.rejects(() => a.insert(record()), /Fabric OAuth transaction store unavailable/);
  assert.equal(db.rows.size, 1);
  const valid = await a.consume({ tenantKey: scope, state: uuid });
  assert.equal(valid.nonce, nonce);
});

test("wrong, missing and noncanonical state/nonce, tenant or TTL never reach SQL", async () => {
  const db = postgresSimulation();
  const store = createPostgresOIDCTransactions({ pool: db });
  const valid = record();
  const mutated = [
    { ...valid, tenantKey: "../indiba" },
    { ...valid, tenantKey: "HAIREM" },
    { ...valid, state: "a" },
    { ...valid, state: uuid + "!" },
    { ...valid, value: { ...valid.value, nonce: "oops" } },
    { ...valid, value: { ...valid.value, pkceVerifier: "oops" } },
    { ...valid, value: { ...valid.value, bindingDigest: "oops" } },
    { ...valid, value: { ...valid.value, issuedAt: 1.5 } },
    { ...valid, expiresAt: valid.expiresAt + 1 },
    { ...valid, value: { ...valid.value, issuer: "https://evil.example\nAuthorization: Bearer evil" } },
    { ...valid, value: { ...valid.value, redirectURI: "x".repeat(1025) } },
    { ...valid, value: null },
  ];
  for (const item of mutated)
    await assert.rejects(() => store.insert(item), /Fabric OAuth transaction store unavailable/);
  for (const bad of [
    {}, { tenantKey: "HAIREM", state: uuid },
    { tenantKey: scope, state: "../" }, { tenantKey: scope, state: "" },
  ]) await assert.rejects(() => store.consume(bad), /Fabric OAuth transaction store unavailable/);
  assert.equal(db.calls.length, 0);
});

test("invalid or inconsistent returned database rows are denied, not accepted as OAuth claims", async () => {
  for (const changed of [
    { nonce: "invalid" },
    { pkce_verifier: "invalid" },
    { binding_digest: "invalid" },
    { issued_at_ms: "1.2" },
    { issued_at_ms: "18446744073709551615" },
    { expires_at_ms: String(issuedAt + age + 1) },
    { issuer: "" },
    { redirect_uri: "https://a/\nInjected: true" },
  ]) {
    const db = postgresSimulation(), store = createPostgresOIDCTransactions({ pool: db });
    await store.insert(record());
    const existing = [...db.rows.values()][0];
    Object.assign(existing, changed);
    await assert.rejects(() => store.consume({ tenantKey: scope, state: uuid }),
      /Fabric OAuth transaction store unavailable/);
    assert.equal(db.rows.size, 0, "invalid row must also be destroyed, not reused");
  }
});

test("opportunistic expiry pruning is bounded and does not remove active records", async () => {
  const db = postgresSimulation();
  let tick = issuedAt + age + 1;
  const store = createPostgresOIDCTransactions({ pool: db, now: () => tick });
  for (let i = 0; i < 125; i++) {
    await store.insert({ ...record(), state: randomBytes(32).toString("base64url") });
  }
  await store.insert({ ...record(tick), state: randomBytes(32).toString("base64url") });
  assert.equal(await store.pruneExpired(), 100);
  assert.equal(db.rows.size, 26);
  assert.equal(await store.pruneExpired(), 25);
  assert.equal(db.rows.size, 1);
  assert.equal(await store.pruneExpired(), 0);
  tick = -1;
  await assert.rejects(() => store.pruneExpired(), /Fabric OAuth transaction store unavailable/);
});

test("pool outage, bad result shape and missing pool fail closed with redacted error", async () => {
  assert.throws(() => createPostgresOIDCTransactions({}), /store unavailable/);
  assert.throws(() => createPostgresOIDCTransactions({ pool: {} }), /store unavailable/);
  const db = postgresSimulation();
  const store = createPostgresOIDCTransactions({ pool: db });
  db.setFailure(true);
  await assert.rejects(() => store.insert(record()), (err) =>
    err.message === "Fabric OAuth transaction store unavailable");
  await assert.rejects(() => store.consume({ tenantKey: scope, state: uuid }), (err) =>
    err.message === "Fabric OAuth transaction store unavailable");
  const broken = createPostgresOIDCTransactions({
    pool: { query: async () => ({ rows: [], rowCount: "bad" }) },
  });
  await assert.rejects(() => broken.consume({ tenantKey: scope, state: uuid }),
    /Fabric OAuth transaction store unavailable/);
});
