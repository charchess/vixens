import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { createPostgresGlobalSessions } from "../server/global-postgres-sessions.mjs";

const issuer = "https://authentik.truxonline.com/application/o/txo-fabric-webui/";
const identity = Object.freeze({
  issuer, oidcSubject: "signed-and-verified-global-opaque-subject",
  authentikUserUUID: "c38f3cc7-5b33-4266-bd85-40ce9a100061",
});
const now = 1_797_000_000_000;
const digest = token => createHash("sha256").update(token).digest("base64url");

function pool() {
  const rows = new Map(), calls = [];
  let broken = false;
  return {
    rows, calls,
    outage: value => { broken = value; },
    async query(sql, params) {
      if (broken) throw Error("DB username+password must never leak to browser");
      calls.push({ sql, params });
      if (sql.startsWith("INSERT INTO txo_fabric_webui_sessions")) {
        const [session_digest, issuer, oidc_subject, authentik_uuid,
          issued_at_ms, expires_at_ms] = params;
        if (rows.has(session_digest)) throw Error("collision");
        rows.set(session_digest, {
          issuer, oidc_subject, authentik_uuid,
          issued_at_ms: String(issued_at_ms),
          expires_at_ms: String(expires_at_ms),
        });
        return { rowCount: 1, rows: [] };
      }
      if (sql.startsWith("SELECT issuer")) {
        const row = rows.get(params[0]);
        if (!row || Number(row.expires_at_ms) <= params[1])
          return { rowCount: 0, rows: [] };
        return { rowCount: 1, rows: [row] };
      }
      if (sql.startsWith("DELETE FROM txo_fabric_webui_sessions\n        WHERE session_digest =")) {
        const found = rows.delete(params[0]);
        return { rowCount: found ? 1 : 0, rows: [] };
      }
      if (sql.startsWith("DELETE FROM txo_fabric_webui_sessions\n        WHERE session_digest IN")) {
        const expired = [...rows.entries()]
          .filter(([, v]) => Number(v.expires_at_ms) <= params[0])
          .slice(0, 100);
        for (const [key] of expired) rows.delete(key);
        return { rowCount: expired.length, rows: [] };
      }
      throw Error("unexpected SQL");
    },
  };
}
test("PostgreSQL schema stores hashes, exact global OIDC identity and bounded TTL, not tenant grants", () => {
  const sql = readFileSync(
    new URL("../db/migrations/002_global_browser_sessions.sql", import.meta.url), "utf8");
  assert.match(sql, /session_digest varchar\(43\) PRIMARY KEY/);
  assert.match(sql, /expires_at_ms = issued_at_ms \+ 28800000/);
  assert.match(sql, /REVOKE ALL .* FROM PUBLIC/);
  assert.doesNotMatch(sql, /fabric_user_id|tenant_key|token_value|access_token|api_key/);
});

test("session tokens are random, hashed in DB and scoped to the global OIDC issuer", async () => {
  const db = pool(), a = createPostgresGlobalSessions({ pool: db, now: () => now });
  const b = createPostgresGlobalSessions({ pool: db, now: () => now });
  const first = await a.issue(identity), second = await a.issue(identity);
  assert.match(first.token, /^[A-Za-z0-9_-]{43}$/);
  assert.notEqual(first.token, second.token);
  assert.equal(first.maxAge, 28800);
  assert.equal(db.rows.size, 2);
  assert.ok(db.rows.has(digest(first.token)));
  assert.equal(JSON.stringify([...db.rows]).includes(first.token), false);
  assert.deepEqual(await b.authenticate(first.token), {
    ...identity, authenticated: true, verified: true,
  });
  assert.equal((await b.authenticate(first.token)).fabricUserId, undefined);
  await b.revoke(first.token);
  assert.equal(await a.authenticate(first.token), null);
  assert.ok(await a.authenticate(second.token));
  await b.revoke(first.token);
});

test("expiry, invalid token, foreign issuer, corrupted rows and outage deny", async () => {
  const db = pool();
  let clock = now;
  const store = createPostgresGlobalSessions({ pool: db, now: () => clock });
  const {token} = await store.issue(identity);
  assert.equal(await store.authenticate("bad"), null);
  clock += 28_800_000;
  assert.equal(await store.authenticate(token), null);
  clock = now;
  for (const change of [
    { issuer: "https://authentik.truxonline.com/application/o/txo-fabric-hairem/" },
    { authentik_uuid: "malformed" },
    { issued_at_ms: "NaN" },
    { expires_at_ms: String(now + 99999) },
  ]) {
    const row = db.rows.get(digest(token)), saved = { ...row };
    Object.assign(row, change);
    await assert.rejects(() => store.authenticate(token), /session store unavailable/);
    Object.assign(row, saved);
  }
  db.outage(true);
  await assert.rejects(() => store.authenticate(token), (e) =>
    e.message === "Global Fabric session store unavailable");
  await assert.rejects(() => store.issue(identity), /session store unavailable/);
  db.outage(false);
  await assert.rejects(() => store.issue({
    ...identity, issuer: "https://authentik.truxonline.com/application/o/txo-fabric-indiba/",
  }), /session store unavailable/);
});

test("expired sessions are pruned in bounded 100-row batches", async () => {
  const db = pool();
  let clock = now;
  const store = createPostgresGlobalSessions({ pool: db, now: () => clock });
  for (let i = 0; i < 125; i++) await store.issue(identity);
  clock += 28_800_000;
  assert.equal(await store.pruneExpired(), 100);
  assert.equal(db.rows.size, 25);
  assert.equal(await store.pruneExpired(), 25);
  assert.equal(db.rows.size, 0);
});
