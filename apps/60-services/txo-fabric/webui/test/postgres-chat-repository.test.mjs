import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { randomBytes } from "node:crypto";
import { createPostgresFabricChatRepository } from "../server/postgres-chat-repository.mjs";

// The ordinary Node test suite has no database, credentials or network need.
// The dedicated CI job runs this file with TEST_FABRIC_CHAT_PG=1 against a
// fresh, ephemeral PostgreSQL runner using local trust auth (no password).
const realPostgres = process.env.TEST_FABRIC_CHAT_PG === "1";
const scope = Object.freeze({
  tenantKey: "indiba",
  fabricTenantId: "TEN00002",
  tenantBundleUID: "10101010-0000-4000-8000-000000000002",
  fabricUserId: "usr" + "a".repeat(32),
  agentKey: "alex",
  agentUID: "20202020-0000-4000-8000-000000000002",
  agentObservedGeneration: 4,
});
const turnId = () => randomBytes(16).toString("hex");
const foreignScopes = [
  { ...scope, tenantKey: "hairem" },
  { ...scope, fabricTenantId: "TEN00001" },
  { ...scope, tenantBundleUID: "30303030-0000-4000-8000-000000000002" },
  { ...scope, fabricUserId: "usr" + "b".repeat(32) },
  { ...scope, agentKey: "same-name" },
  { ...scope, agentUID: "40404040-0000-4000-8000-000000000002" },
  { ...scope, agentObservedGeneration: 5 },
];

test("real PostgreSQL: migrations, two independent pools and atomic cross-replica claims", {
  skip: !realPostgres && "requires ephemeral PostgreSQL service in dedicated CI",
}, async () => {
  const { default: pg } = await import("pg");
  const config = {
    host: "127.0.0.1", port: 5432, database: "postgres", user: "postgres",
    max: 8, connectionTimeoutMillis: 1500,
    idleTimeoutMillis: 2000,
  };
  const a = new pg.Pool(config);
  const b = new pg.Pool(config);
  try {
    const migration = readFileSync(new URL("../db/migrations/003_fabric_chat_threads.sql", import.meta.url), "utf8");
    await a.query(migration);
    // Required additive/idempotent: reapplying cannot destroy prior state.
    await a.query(migration);

    const r1 = createPostgresFabricChatRepository({ pool: a });
    const r2 = createPostgresFabricChatRepository({ pool: b });
    const created = await r1.createThread({ scope });
    assert.match(created.id, /^[a-f0-9]{40}$/);
    assert.deepEqual(await r2.getThread({ scope, threadId: created.id }), created);
    assert.deepEqual(await r2.listThreads({ scope }), [created]);
    assert.equal(JSON.stringify(created).includes("remote"), false);
    assert.equal(JSON.stringify(created).includes("lease"), false);

    // Missing canonical owner, user, tenant, agent UID/key or generation
    // must fail BEFORE returning private session/turn state.
    for (const foreign of foreignScopes) {
      assert.equal(await r2.getThread({ scope: foreign, threadId: created.id }), null);
      assert.deepEqual(await r2.listThreads({ scope: foreign }), []);
      assert.equal(await r2.getServerThread({ scope: foreign, threadId: created.id }), null);
      assert.deepEqual(await r2.claimTurn({ scope: foreign, threadId: created.id, turnId: turnId() }),
        { kind: "denied" });
      assert.deepEqual(await r2.bindHermesSession({
        scope: foreign, threadId: created.id, remoteSessionId: "remote-1",
      }), { kind: "denied" });
    }

    const before = await r1.claimTurn({ scope, threadId: created.id, turnId: turnId() });
    assert.deepEqual(before, { kind: "denied" }, "unbound thread cannot execute turn");

    const initial = await r1.bindHermesSession({
      scope, threadId: created.id, remoteSessionId: "remote-1",
    });
    assert.deepEqual(initial, { kind: "bound" });
    assert.deepEqual(await r2.getServerThread({ scope, threadId: created.id }),
      { remoteSessionId: "remote-1" });
    assert.deepEqual(await r2.bindHermesSession({
      scope, threadId: created.id, remoteSessionId: "remote-1",
    }), { kind: "conflict" });
    assert.deepEqual(await r2.bindHermesSession({
      scope, threadId: created.id, remoteSessionId: "hijacked-remote",
    }), { kind: "conflict" });
    assert.deepEqual(await r2.getThread({ scope, threadId: created.id }), created,
      "public projections must not disclose remote session");

    const other = await r2.createThread({ scope });
    await assert.rejects(r2.bindHermesSession({
      scope, threadId: other.id, remoteSessionId: "remote-1",
    }), /Fabric chat repository unavailable/, "same agent session must not bind to two threads");
    assert.deepEqual(await r2.getServerThread({ scope, threadId: other.id }),
      { remoteSessionId: null }, "a failed unique bind is atomic");

    const id = turnId();
    const contenders = await Promise.all(Array.from({ length: 40 }, (_, i) =>
      (i % 2 ? r2 : r1).claimTurn({ scope, threadId: created.id, turnId: id })));
    assert.equal(contenders.filter(r => r.kind === "claimed").length, 1,
      "PostgreSQL lock must fence all but one claim across pools");
    assert.equal(contenders.filter(r => r.kind === "conflict").length, 39);
    const owned = contenders.find(r => r.kind === "claimed");
    assert.match(owned.leaseToken, /^[a-f0-9]{32}$/);
    assert.ok(owned.expiresAt > 0);
    assert.deepEqual(await r2.claimTurn({ scope, threadId: created.id, turnId: turnId() }),
      { kind: "conflict" }, "other turn cannot run while lease is active");

    await assert.rejects(r2.finalizeTurn({
      scope, threadId: created.id, turnId: id, leaseToken: owned.leaseToken,
      outcome: "arbitrary-tool-output",
    }), /unavailable/);
    assert.deepEqual(await r2.finalizeTurn({
      scope, threadId: created.id, turnId: id, leaseToken: turnId(),
      outcome: "completed",
    }), { kind: "denied" });
    assert.deepEqual(await r2.finalizeTurn({
      scope, threadId: created.id, turnId: id, leaseToken: owned.leaseToken,
      outcome: "completed",
    }), { kind: "terminal", outcome: "completed" });
    const after = await r1.claimTurn({ scope, threadId: created.id, turnId: id });
    assert.deepEqual(after, { kind: "terminal", outcome: "completed" },
      "exact same scoped retry returns bounded prior terminal state, never another run");
    assert.deepEqual(await r2.inspectTurn({ scope, threadId: created.id, turnId: id }),
      { kind: "terminal", outcome: "completed" });
    assert.deepEqual(await r1.finalizeTurn({
      scope, threadId: created.id, turnId: id, leaseToken: owned.leaseToken,
      outcome: "completed",
    }), { kind: "terminal", outcome: "completed" }, "terminal finalize idempotent");
    assert.deepEqual(await r1.finalizeTurn({
      scope, threadId: created.id, turnId: id, leaseToken: owned.leaseToken,
      outcome: "failed",
    }), { kind: "conflict", outcome: "completed" }, "terminal state immutable");

    const cancelledId = turnId();
    const current = await r2.claimTurn({ scope, threadId: created.id, turnId: cancelledId });
    assert.equal(current.kind, "claimed");
    await r2.finalizeTurn({ scope, threadId: created.id, turnId: cancelledId,
      leaseToken: current.leaseToken, outcome: "cancelled" });
    assert.deepEqual(await r1.inspectTurn({ scope, threadId: created.id, turnId: cancelledId }),
      { kind: "terminal", outcome: "cancelled" });

    const unknown = await r1.createThread({ scope });
    await assert.rejects(r1.bindHermesSession({
      scope, threadId: unknown.id, remoteSessionId: "https://foreign.invalid/",
    }), /unavailable/);
    await assert.rejects(r1.claimTurn({ scope: { ...scope, agentObservedGeneration: 0 },
      threadId: created.id, turnId: turnId() }), /unavailable/);
    for (const bad of ["", "A".repeat(32), "f".repeat(33), "../forged", "0".repeat(10000)]) {
      await assert.rejects(r1.claimTurn({
        scope, threadId: created.id, turnId: bad,
      }), /unavailable/, "invalid opaque idempotency value");
    }

    // Losing an owner at lease expiry MUST NOT elect a successor, because
    // expiry does not prove that the upstream Hermes tool/run has stopped.
    const staleId = turnId();
    const stale = await r1.claimTurn({ scope, threadId: created.id, turnId: staleId });
    assert.equal(stale.kind, "claimed");
    await a.query(`UPDATE txo_fabric_chat_threads
      SET lease_expires_at_ms=1 WHERE thread_id=$1`, [created.id]);
    assert.deepEqual(await r2.expireStaleLease({ scope, threadId: created.id }),
      { kind: "uncertain" });
    assert.deepEqual(await r1.claimTurn({ scope, threadId: created.id, turnId: turnId() }),
      { kind: "uncertain" }, "expired lease must not allow automatic replay");
    assert.deepEqual(await r2.claimTurn({ scope, threadId: created.id, turnId: staleId }),
      { kind: "conflict" });
    for (const foreign of foreignScopes) {
      assert.deepEqual(await r2.expireStaleLease({ scope: foreign, threadId: created.id }),
        { kind: "denied" });
    }
    // An original owner that still has its exact private fencing token can
    // finalize. Only then is a new claim safe; no automatic lease theft.
    assert.deepEqual(await r2.finalizeTurn({ scope, threadId: created.id, turnId: staleId,
      leaseToken: stale.leaseToken, outcome: "failed" }),
    { kind: "terminal", outcome: "failed" });
    assert.equal((await r1.claimTurn({
      scope, threadId: created.id, turnId: turnId(),
    })).kind, "claimed");
  } finally {
    await Promise.all([a.end(), b.end()]);
  }
});

test("source migration is additive, credential-free and not selected by current bootstrap", () => {
  const sql = readFileSync(new URL("../db/migrations/003_fabric_chat_threads.sql", import.meta.url), "utf8");
  const bootstrap = readFileSync(new URL("../bootstrap/migrate-job.yaml", import.meta.url), "utf8");
  const prod = readFileSync(new URL("../../overlays/prod/kustomization.yaml", import.meta.url), "utf8");
  assert.equal((sql.match(/CREATE TABLE IF NOT EXISTS/g) || []).length, 2);
  assert.match(sql, /FOR PUBLIC|FROM PUBLIC/);
  assert.match(sql, /PRIMARY KEY \(thread_id, turn_id\)/);
  assert.match(sql, /UNIQUE INDEX IF NOT EXISTS/);
  assert.doesNotMatch(sql, /\b(DROP|TRUNCATE|CREATE ROLE|PASSWORD)\s+(TABLE|ROLE|\S+)/i);
  assert.match(bootstrap, /suspend: true/);
  assert.doesNotMatch(bootstrap, /003_fabric_chat_threads/);
  assert.doesNotMatch(prod, /003_fabric_chat_threads|openbao-fabric-store/);
});
