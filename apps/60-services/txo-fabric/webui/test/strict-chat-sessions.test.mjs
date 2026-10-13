import test from "node:test";
import assert from "node:assert/strict";
import { randomBytes } from "node:crypto";
import { createChatSessionService } from "../server/chat-sessions.mjs";
import { createChatHttpHandler } from "../server/chat-http.mjs";

const userId = "usr" + "a".repeat(32);
const principal = Object.freeze({
  authenticated: true, verified: true, tenantKey: "indiba",
  fabricTenantId: "TEN00002", fabricUserId: userId, subject: userId,
});
const target = Object.freeze({
  tenantKey: "indiba", fabricTenantId: "TEN00002", agentKey: "alex",
  tenantBundleUID: "10101010-0000-4000-8000-000000000002",
  agentUID: "20202020-0000-4000-8000-000000000002",
  agentObservedGeneration: 4, apiServerEnabled: true, endpointVerified: true,
  baseURL: "http://hermes-alex.tenant-indiba.svc:8642", apiKey: "fake-test-credential",
});
const id = () => randomBytes(16).toString("hex");
const collect = async iterable => {
  const values = [];
  for await (const value of iterable) values.push(value);
  return values;
};
const eventBody = 'event: assistant.delta\ndata: {"text":"Hello"}\n\n' +
  'event: run.completed\ndata: {}\n\n';

function harness() {
  const trace = [], records = new Map(), journal = new Map();
  const state = {
    grant: true, agent: { ...target }, agentKeys: ["alex"],
    bindConflict: false, failFinalize: false, failOpen: false,
    forceClaim: null, onClaim: null, onGetThread: null, streamError: false,
    onUpstream: null,
  };
  const matches = (record, scope) => record &&
    Object.keys(scope).every(key => record.scope[key] === scope[key]);
  const project = rec => ({ id: rec.id, agentKey: rec.scope.agentKey, createdAt: rec.createdAt });
  const repo = {
    async createThread({ scope }) {
      trace.push("createThread");
      const rec = { id: randomBytes(20).toString("hex"), scope: { ...scope },
        remoteSessionId: null, createdAt: new Date().toISOString(), lease: null };
      records.set(rec.id, rec);
      return project(rec);
    },
    async getThread({ scope, threadId }) {
      trace.push("getThread");
      const rec = records.get(threadId);
      if (state.onGetThread) state.onGetThread();
      return matches(rec, scope) ? project(rec) : null;
    },
    async listThreads({ scope }) {
      trace.push("listThreads");
      return [...records.values()].filter(rec => matches(rec, scope)).map(project);
    },
    async getServerThread({ scope, threadId }) {
      trace.push("getServerThread");
      const rec = records.get(threadId);
      return matches(rec, scope) ? { remoteSessionId: rec.remoteSessionId } : null;
    },
    async bindHermesSession({ scope, threadId, remoteSessionId }) {
      trace.push("bindHermesSession");
      const rec = records.get(threadId);
      if (!matches(rec, scope)) return { kind: "denied" };
      if (state.bindConflict || rec.remoteSessionId) return { kind: "conflict" };
      rec.remoteSessionId = remoteSessionId;
      return { kind: "bound" };
    },
    async claimTurn({ scope, threadId, turnId }) {
      trace.push("claimTurn");
      const rec = records.get(threadId);
      if (state.onClaim) state.onClaim();
      if (state.forceClaim) return state.forceClaim;
      if (!matches(rec, scope) || !rec.remoteSessionId) return { kind: "denied" };
      const k = threadId + ":" + turnId;
      const prev = journal.get(k);
      if (prev) return prev.kind === "terminal"
        ? { kind: "terminal", outcome: prev.outcome } : { kind: "conflict" };
      if (rec.lease) return { kind: "conflict" };
      rec.lease = id();
      journal.set(k, { kind: "claimed", leaseToken: rec.lease });
      return { kind: "claimed", leaseToken: rec.lease };
    },
    async finalizeTurn({ scope, threadId, turnId, leaseToken, outcome }) {
      trace.push("finalizeTurn:" + outcome);
      if (state.failFinalize) throw new Error("private database failure");
      const rec = records.get(threadId);
      const prior = journal.get(threadId + ":" + turnId);
      if (!matches(rec, scope) || !prior || prior.leaseToken !== leaseToken) {
        return { kind: "denied" };
      }
      rec.lease = null;
      journal.set(threadId + ":" + turnId, { kind: "terminal", outcome });
      return { kind: "terminal", outcome };
    },
  };
  const service = createChatSessionService({
    tenantKey: "indiba", strictBinding: true, repo,
    authorize: async () => state.grant,
    resolveAgent: async () => state.agent,
    listAgentKeys: async () => state.agentKeys,
    openRemoteSession: async () => {
      trace.push("openHermesSession");
      if (state.failOpen) throw new Error("sensitive remote failure");
      return "fake-remote-session";
    },
    fetchImpl: async (url, options) => {
      trace.push("upstreamStart");
      assert.equal(url, target.baseURL + "/api/sessions/fake-remote-session/chat/stream");
      assert.equal(options.headers.Authorization, "Bearer fake-test-credential");
      if (state.onUpstream) state.onUpstream(options);
      if (state.streamError) throw new Error("private Hermes failure");
      return new Response(eventBody, {
        headers: { "content-type": "text/event-stream" },
      });
    },
  });
  return { service, trace, state, repo, records, journal };
}

test("strict requires the complete durable adapter: legacy mutex + claimTurn is insufficient", () => {
  const h = harness();
  const legacy = { insert() {}, get() {}, list() {}, claim() {}, claimTurn() {} };
  assert.throws(() => createChatSessionService({
    tenantKey: "indiba", strictBinding: true, repo: legacy,
    authorize: async () => true, resolveAgent: async () => target,
    listAgentKeys: async () => ["alex"], fetchImpl: async () => {},
  }), /security dependencies/);
  assert.equal(typeof h.repo.claim, "undefined");
});

test("strict create reserves durable thread before remote creation and binds exactly once", async () => {
  const h = harness();
  const thread = await h.service.create({ principal, agentKey: "alex" });
  assert.deepEqual(h.trace, ["createThread", "openHermesSession", "bindHermesSession"]);
  assert.deepEqual(Object.keys(thread).sort(), ["agentKey", "createdAt", "id"]);
  const rec = h.records.get(thread.id);
  assert.deepEqual(Object.keys(rec.scope).sort(), [
    "agentKey", "agentObservedGeneration", "agentUID", "fabricTenantId",
    "fabricUserId", "tenantBundleUID", "tenantKey",
  ]);
  assert.equal(rec.scope.agentObservedGeneration, 4);
  assert.deepEqual(await h.service.get({ principal, threadId: thread.id }), thread);
  assert.deepEqual(await h.service.list({ principal }), [thread]);
  assert.doesNotMatch(JSON.stringify(await h.service.list({ principal })), /remote|credential|lease/);
});

test("upstream failure and conflicting one-time bind preserve unbound orphan, no retry or compensation", async () => {
  for (const failure of ["failOpen", "bindConflict"]) {
    const h = harness();
    h.state[failure] = true;
    await assert.rejects(h.service.create({ principal, agentKey: "alex" }), /unavailable/);
    assert.equal(h.trace.filter(t => t === "createThread").length, 1);
    assert.equal(h.trace.filter(t => t === "openHermesSession").length, 1);
    assert.equal(h.trace.filter(t => t === "bindHermesSession").length,
      failure === "bindConflict" ? 1 : 0);
    assert.equal(h.records.size, 1);
    assert.deepEqual(await h.service.list({ principal }), [], "unbound thread is hidden");
    assert.equal([...h.records.values()][0].remoteSessionId, null);
    assert.equal(h.trace.some(t => t.includes("delete") || t.includes("retry")), false);
  }
});

test("no durable thread, credential read or upstream when scope/grant/identity/endpoint is invalid", async () => {
  const badPrincipals = [
    { ...principal, verified: false }, { ...principal, subject: "OIDC-sub" },
    { ...principal, fabricTenantId: "TEN00001" }, { ...principal, tenantKey: "hairem" },
    { ...principal, fabricUserId: "OIDC-sub" },
  ];
  const h = harness();
  for (const p of badPrincipals) {
    await assert.rejects(h.service.create({ principal: p, agentKey: "alex" }));
  }
  assert.deepEqual(h.trace, []);
  for (const change of [
    { grant: false }, { agent: { ...target, agentUID: "" } },
    { agent: { ...target, endpointVerified: false } },
    { agent: { ...target, agentObservedGeneration: undefined } },
    { agent: { ...target, tenantBundleUID: "incorrect" } },
    { agent: { ...target, fabricTenantId: "TEN00001" } },
  ]) {
    const k = harness(); Object.assign(k.state, change);
    await assert.rejects(k.service.create({ principal, agentKey: "alex" }));
    assert.deepEqual(k.trace, []);
  }
});

test("strict get/list deny cross-user and changed identity UID/generation/endpoint before upstream", async () => {
  const h = harness();
  const thread = await h.service.create({ principal, agentKey: "alex" });
  const foreign = { ...principal, subject: "usr" + "b".repeat(32), fabricUserId: "usr" + "b".repeat(32) };
  await assert.rejects(h.service.get({ principal: foreign, threadId: thread.id }), /Forbidden/);
  assert.deepEqual(await h.service.list({ principal: foreign }), []);
  for (const change of [
    { agentUID: "30303030-0000-4000-8000-000000000002" },
    { agentObservedGeneration: 5 },
    { endpointVerified: false },
    { tenantBundleUID: "40404040-0000-4000-8000-000000000002" },
  ]) {
    h.state.agent = { ...target, ...change };
    assert.deepEqual(await h.service.list({ principal }), []);
    await assert.rejects(h.service.get({ principal, threadId: thread.id }), /Forbidden/);
    await assert.rejects(collect(h.service.turn({
      principal, threadId: thread.id, turnId: id(), input: "do not send",
    })), /Forbidden/);
  }
  assert.equal(h.trace.filter(t => t === "upstreamStart").length, 0);
});

test("terminal is bounded metadata, active/conflict/uncertain/denied never replay", async () => {
  const h = harness();
  const thread = await h.service.create({ principal, agentKey: "alex" });
  const turnId = id();
  const first = await collect(h.service.turn({ principal, threadId: thread.id,
    turnId, input: "hello" }));
  assert.deepEqual(first, [
    { type: "text-delta", text: "Hello" }, { type: "done", status: "completed" },
  ]);
  const replay = await collect(h.service.turn({ principal, threadId: thread.id,
    turnId, input: "different message" }));
  assert.deepEqual(replay, [{ type: "done", status: "completed" }]);
  assert.equal(h.trace.filter(t => t === "upstreamStart").length, 1);
  for (const kind of ["conflict", "uncertain", "denied", "unknown"]) {
    h.state.forceClaim = { kind };
    await assert.rejects(collect(h.service.turn({
      principal, threadId: thread.id, turnId: id(), input: "no replay",
    })));
  }
  assert.equal(h.trace.filter(t => t === "upstreamStart").length, 1);
});

test("grant revoked between lookup and claim or after claim denies before stream", async () => {
  for (const phase of ["onGetThread", "onClaim"]) {
    const h = harness();
    const thread = await h.service.create({ principal, agentKey: "alex" });
    h.state[phase] = () => { h.state.grant = false; };
    await assert.rejects(collect(h.service.turn({
      principal, threadId: thread.id, turnId: id(), input: "no call",
    })));
    assert.equal(h.trace.filter(t => t === "upstreamStart").length, 0);
    assert.equal(h.trace.filter(t => t === "claimTurn").length,
      phase === "onClaim" ? 1 : 0);
    assert.equal(h.trace.includes("finalizeTurn:failed"), phase === "onClaim");
  }
});

test("cancelled/failed upstream and finalize outage never cause a second POST", async () => {
  for (const mode of ["cancel", "upstream", "finalize"]) {
    const h = harness();
    const thread = await h.service.create({ principal, agentKey: "alex" });
    const turnId = id();
    const controller = new AbortController();
    if (mode === "cancel") h.state.onUpstream = () => controller.abort();
    if (mode === "upstream") h.state.streamError = true;
    if (mode === "finalize") h.state.failFinalize = true;
    try {
      await collect(h.service.turn({ principal, threadId: thread.id, turnId,
        input: "first", signal: controller.signal }));
    } catch (e) { assert.match(e.message, /unavailable/); }
    assert.equal(h.trace.filter(t => t === "upstreamStart").length, 1);
    assert.ok(h.trace.includes("finalizeTurn:" + (mode === "cancel" ? "cancelled" :
      mode === "upstream" ? "failed" : "completed")));
    if (mode === "finalize") {
      h.state.forceClaim = { kind: "uncertain" };
    }
    await assert.rejects(collect(h.service.turn({ principal, threadId: thread.id,
      turnId: id(), input: "do not replay" })));
    assert.equal(h.trace.filter(t => t === "upstreamStart").length, 1);
  }
});

test("HTTP browser-supplied remote scope, lease, endpoint, token, headers are ignored", async () => {
  const h = harness(), origin = "https://webui.example.test";
  const handle = createChatHttpHandler({ service: h.service,
    authenticate: async () => principal, expectedOrigin: origin });
  const request = (path, body) => new Request(origin + path, {
    method: "POST", headers: { Origin: origin, "X-TXO-Request": "chat",
      "Content-Type": "application/json", "X-Hermes-Session": "forged",
      "X-Lease-Token": id(), "X-Fabric-Tenant-ID": "TEN99999" },
    body: JSON.stringify(body),
  });
  const created = await handle(request("/api/chat/threads", { agentKey: "alex",
    sessionId: "forged", remoteSessionId: "forged", tenantKey: "hairem",
    agentUID: "fake", endpoint: "https://attacker.test", apiKey: "fake" }));
  assert.equal(created.status, 201);
  const thread = await created.json();
  const response = await handle(request("/api/chat/threads/" + thread.id + "/turns", {
    input: "hello", turnId: id(), remoteSessionId: "forged",
    leaseToken: "forged", scope: { tenantKey: "hairem" }, resultState: "completed",
  }));
  assert.equal(response.status, 200);
  const body = await response.text();
  assert.match(body, /text-delta/);
  assert.doesNotMatch(body, /fake-test-credential|forged|TEN99999|hairem/);
  assert.equal(h.records.get(thread.id).scope.fabricTenantId, "TEN00002");
});
