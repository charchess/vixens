import test from "node:test";
import assert from "node:assert/strict";
import { createPrivateHermesChatService } from "../server/private-hermes-chat.mjs";
import { createChatHttpHandler } from "../server/chat-http.mjs";
import { createPrivateBFFHandler } from "../server/private-bff-server.mjs";

const tenantKey = "indiba";
const fabricTenantId = "TEN00002";
const user = "usr" + "a".repeat(32);
const tenantUID = "0c0c0c0c-0000-4000-8000-000000000002";
const agentUID = "1d1d1d1d-0000-4000-8000-000000000002";
const principal = Object.freeze({
  authenticated: true, verified: true, tenantKey, fabricTenantId,
  fabricUserId: user, subject: user,
});
const upstream = "http://hermes-alex.tenant-indiba.svc:8642";
const events = [
  'event: assistant.commentary\ndata: {"thought":"private"}\n\n',
  'event: tool.started\ndata: {"tool":"terminal","arguments":{"authorization":"never-expose"}}\n\n',
  'event: assistant.delta\ndata: {"text":"Hello"}\n\n',
  'event: tool.completed\ndata: {"tool":"terminal","output":"private result"}\n\n',
  'event: run.completed\ndata: {}\n\n',
].join("");
const body = value => new ReadableStream({ start(c) {
  c.enqueue(new TextEncoder().encode(value)); c.close();
}});
const collect = async iterable => {
  const result = [];
  for await (const part of iterable) result.push(part);
  return result;
};
function storage() {
  const records = new Map(), claimed = new Set();
  return {
    insert: async record => {
      if (records.has(record.id)) throw Error("collision");
      records.set(record.id, record);
    },
    get: async id => records.get(id),
    list: async ({tenantKey, subject}) => [...records.values()].filter(r =>
      r.tenantKey === tenantKey && r.subject === subject),
    claim: async id => {
      if (claimed.has(id)) throw Error("lease unavailable");
      claimed.add(id);
      return async () => { claimed.delete(id); };
    },
  };
}
function harness(overrides = {}) {
  let enabled = true, bridge = true, grant = true, endpointValid = true;
  const rpc = [], checks = [], secrets = [];
  const tenant = {
    tenantKey, fabricTenantId, uid: tenantUID, deleting: false, status: "Ready",
  };
  const agent = {
    tenantKey, fabricTenantId, tenantBundleUID: tenantUID,
    agentKey: "alex", uid: agentUID, generation: 4, observedGeneration: 4,
    namespace: "tenant-indiba", serviceName: "hermes-alex",
    credentialRef: "hermes-api-alex", apiServerEnabled: true,
    status: "Ready", deleting: false,
  };
  const authorize = async arg => {
    checks.push(arg);
    return grant;
  };
  const resolveCanonicalTenant = async () => tenant;
  const resolveCanonicalAgent = async () => agent;
  const verifyEndpoint = async observed => {
    assert.equal(observed.tenantBundleUID, tenantUID);
    assert.equal(observed.agentUID, agent.uid);
    return endpointValid;
  };
  const readScopedCredential = async ref => {
    secrets.push(ref);
    assert.equal(ref.credentialRef, "hermes-api-alex");
    return "k".repeat(40);
  };
  const fetchImpl = async (url, options) => {
    rpc.push({ url, options });
    assert.equal(options.headers.Authorization, "Bearer " + "k".repeat(40));
    assert.equal(options.redirect, "error");
    assert.equal(options.cache, "no-store");
    assert.equal(options.headers.Cookie, undefined);
    assert.equal(options.headers["X-Fabric-Assertion"], undefined);
    assert.equal(options.headers["X-TXO-Request"], undefined);
    if (url === upstream + "/api/sessions") {
      assert.deepEqual(JSON.parse(options.body), { title: "Fabric conversation" });
      return new Response(JSON.stringify({ session_id: "remote-private-1" }), {
        headers: { "content-type": "application/json" },
      });
    }
    assert.equal(url, upstream + "/api/sessions/remote-private-1/chat/stream");
    return new Response(body(events), {
      headers: { "content-type": "text/event-stream" },
    });
  };
  const options = {
    tenantKey, fabricTenantId, repo: storage(), authorize,
    resolveCanonicalTenant, resolveCanonicalAgent,
    verifyEndpoint, readScopedCredential, fetchImpl,
    browserEnabled: overrides.browserEnabled ?? enabled,
    bridgeEnabled: overrides.bridgeEnabled ?? bridge,
  };
  const service = createPrivateHermesChatService(options);
  return { service, rpc, checks, secrets, tenant, agent, options,
    revoke: () => { grant = false; },
    invalidateEndpoint: () => { endpointValid = false; },
  };
}

test("private POC turn binds one canonical Fabric user, tenant and AgentIdentity UID", async () => {
  const h = harness();
  const thread = await h.service.create({ principal, agentKey: "alex" });
  assert.match(thread.id, /^[a-f0-9]{40}$/);
  assert.equal(h.rpc.length, 1);
  assert.equal(h.rpc[0].url, upstream + "/api/sessions");
  const result = await collect(h.service.turn({ principal, threadId: thread.id, input: "hello" }));
  assert.deepEqual(result, [
    { type: "tool-started", tool: "terminal" },
    { type: "text-delta", text: "Hello" },
    { type: "tool-completed", tool: "terminal", failed: false },
    { type: "done", status: "completed" },
  ]);
  assert.equal(h.rpc.length, 2);
  assert.deepEqual(await h.service.list({ principal }), [thread]);
  assert.deepEqual(await h.service.get({ principal, threadId: thread.id }), thread);
  assert.ok(h.checks.length >= 5, "fresh check on every action");
  assert.deepEqual(h.secrets.map(x => x.namespace), Array(h.secrets.length).fill("tenant-indiba"));
  assert.equal(JSON.stringify(result).includes("private"), false);
  assert.equal(JSON.stringify(result).includes("authorization"), false);
});

test("disabled browser or bridge gate refuses without IAM, key or Hermes I/O", async () => {
  for (const gates of [{ browserEnabled: false }, { bridgeEnabled: false }]) {
    const h = harness(gates);
    await assert.rejects(h.service.create({ principal, agentKey: "alex" }));
    await assert.rejects(h.service.canAccess({ principal, agentKey: "alex" }));
    assert.deepEqual(h.rpc, []);
    assert.deepEqual(h.secrets, []);
    assert.deepEqual(h.checks, []);
  }
  const bff = createPrivateBFFHandler();
  for (const path of ["/auth/start", "/api/me", "/api/tenants/indiba/chat/threads"]) {
    const res = await bff(new Request("https://webui.truxonline.com" + path));
    assert.equal(res.status, 404);
  }
});

test("forged browser identity, user, tenant and agent cannot open upstream", async () => {
  const variants = [
    { ...principal, subject: "usr" + "b".repeat(32) },
    { ...principal, verified: false },
    { ...principal, authenticated: false },
    { ...principal, tenantKey: "hairem" },
    { ...principal, fabricTenantId: "TEN00001" },
    { ...principal, fabricUserId: "oidc-sub" },
    { ...principal, subject: "oidc-sub" },
  ];
  const h = harness();
  for (const candidate of variants) {
    await assert.rejects(h.service.create({ principal: candidate, agentKey: "alex" }));
  }
  await assert.rejects(h.service.create({ principal, agentKey: "sam" }));
  await assert.rejects(h.service.create({ principal, agentKey: "../alex" }));
  assert.equal(h.rpc.length, 0);
  assert.equal(h.secrets.length, 0);
});

test("unknown/deleted/moved AgentIdentity, stale generation, forged URL and credentials fail before network", async () => {
  const invalid = [
    h => { h.agent.uid = ""; },
    h => { h.agent.tenantKey = "hairem"; },
    h => { h.agent.fabricTenantId = "TEN00001"; },
    h => { h.agent.tenantBundleUID = "2a2a2a2a-0000-4000-8000-000000000002"; },
    h => { h.agent.namespace = "tenant-hairem"; },
    h => { h.agent.serviceName = "hermes-other"; },
    h => { h.agent.serviceName = "external.example"; },
    h => { h.agent.credentialRef = "hermes-api-other"; },
    h => { h.agent.apiServerEnabled = false; },
    h => { h.agent.deleting = true; },
    h => { h.agent.status = "Deleting"; },
    h => { h.agent.generation++; },
    h => { h.tenant.uid = "3b3b3b3b-0000-4000-8000-000000000002"; },
    h => { h.tenant.status = "Degraded"; },
    h => { h.invalidateEndpoint(); },
  ];
  for (const mutate of invalid) {
    const h = harness();
    mutate(h);
    await assert.rejects(h.service.create({ principal, agentKey: "alex" }));
    assert.equal(h.rpc.length, 0);
    assert.equal(h.secrets.length, 0);
  }
});

test("revoked/stale grant denies every thread action before Hermes; same-named key cannot cross tenant", async () => {
  const h = harness();
  const thread = await h.service.create({ principal, agentKey: "alex" });
  h.revoke();
  const calls = h.rpc.length;
  await assert.rejects(collect(h.service.turn({ principal, threadId: thread.id, input: "again" })));
  await assert.rejects(h.service.get({ principal, threadId: thread.id }));
  assert.deepEqual(await h.service.list({ principal }), []);
  assert.equal(h.rpc.length, calls);
  const b = { ...principal, tenantKey: "hairem", fabricTenantId: "TEN00001" };
  await assert.rejects(collect(h.service.turn({ principal: b, threadId: thread.id, input: "again" })));
  await assert.rejects(collect(h.service.turn({ principal, threadId: "f".repeat(40), input: "again" })));
  assert.equal(h.rpc.length, calls);
});

test("AgentIdentity replacement or missing endpoint attestation revokes existing thread", async () => {
  for (const change of [
    h => { h.agent.uid = "aaaaaaa1-0000-4000-8000-000000000002"; },
    h => { h.agent.apiServerEnabled = false; },
    h => { h.invalidateEndpoint(); },
    h => { h.agent.deleting = true; },
  ]) {
    const h = harness();
    const thread = await h.service.create({ principal, agentKey: "alex" });
    change(h);
    const before = h.rpc.length;
    await assert.rejects(collect(h.service.turn({ principal, threadId: thread.id, input: "again" })));
    await assert.rejects(h.service.get({ principal, threadId: thread.id }));
    assert.equal(h.rpc.length, before, "must deny without any new Hermes I/O");
  }
});

test("missing key or network wrong-type response denies generically, without fallback", async () => {
  const h = harness();
  const noCredential = createPrivateHermesChatService({
    ...h.options, readScopedCredential: async () => { throw new Error("internal credential detail"); },
  });
  await assert.rejects(noCredential.create({ principal, agentKey: "alex" }), /unavailable/);
  assert.equal(h.rpc.length, 0);
  const wrong = createPrivateHermesChatService({
    ...h.options, fetchImpl: async () => new Response("wrong", {
      headers: { "content-type": "text/html" },
    }),
  });
  await assert.rejects(wrong.create({ principal, agentKey: "alex" }), /unavailable/);
  assert.throws(() => createPrivateHermesChatService({}), /unavailable/);
});

test("turn lease rejects concurrent replay, retry/reconnect never creates another session", async () => {
  const h = harness();
  const thread = await h.service.create({ principal, agentKey: "alex" });
  const before = h.rpc.length;
  await collect(h.service.turn({ principal, threadId: thread.id, input: "first" }));
  await collect(h.service.turn({ principal, threadId: thread.id, input: "second" }));
  assert.equal(h.rpc.filter(x => x.url.endsWith("/api/sessions")).length, 1);
  assert.equal(h.rpc.length, before + 2);
  assert.equal((await h.service.get({ principal, threadId: thread.id })).id, thread.id);
});

test("HTTP mutation origin validation strips browser headers and returns no private upstream topology", async () => {
  const h = harness();
  const expectedOrigin = "https://webui.truxonline.com";
  const handler = createChatHttpHandler({
    authenticate: async () => principal, service: h.service, expectedOrigin,
  });
  const request = (path, body, origin = expectedOrigin) => new Request(expectedOrigin + path, {
    method: "POST", headers: { Origin: origin, "X-TXO-Request": "chat",
      "Content-Type": "application/json", Cookie: "browser-cookie",
      Authorization: "Bearer browser-header",
      "X-Fabric-Assertion": "forged-browser-assertion" },
    body: JSON.stringify(body),
  });
  const bad = await handler(request("/api/chat/threads", { agentKey: "alex" }, "https://attacker.invalid"));
  assert.equal(bad.status, 403);
  assert.equal(h.rpc.length, 0);
  const create = await handler(request("/api/chat/threads", { agentKey: "alex" }));
  assert.equal(create.status, 201);
  const thread = await create.json();
  const result = await handler(request(`/api/chat/threads/${thread.id}/turns`, { input: "hello" }));
  assert.equal(result.status, 200);
  const exposed = await result.text();
  assert.match(exposed, /text-delta/);
  assert.doesNotMatch(exposed, /browser-cookie|forged-browser-assertion|authorization|private result|tenant-indiba\.svc|hermes-alex/);
  assert.equal(h.rpc.length, 2);
});
