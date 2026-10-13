import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { createPrivateHermesResponsesService } from "../server/private-hermes-chat.mjs";
import {
  streamHermesResponses, requireHermesConversationKey, projectHermesResponsesEvent,
} from "../server/hermes-responses.mjs";

const tenantKey = "indiba";
const fabricTenantId = "TEN00002";
const user = "usr" + "a".repeat(32);
const tenantUID = "0c0c0c0c-0000-4000-8000-000000000002";
const agentUID = "1d1d1d1d-0000-4000-8000-000000000002";
const principal = Object.freeze({
  authenticated: true, verified: true, tenantKey, fabricTenantId,
  fabricUserId: user, subject: user,
});
const url = "http://hermes-alex.tenant-indiba.svc:8642/v1/responses";
const keyFor = scope => "v1_" + createHash("sha256").update(JSON.stringify(scope)).digest("base64url");
// The test-only deterministic digest above is NOT a production KDF.
// Production requires an independently scoped secret HMAC, provisioned later.
const frames = (...parts) => new ReadableStream({
  start(controller) {
    for (const part of parts) controller.enqueue(new TextEncoder().encode(part));
    controller.close();
  },
});
const sse = (event, payload) => `event: ${event}\ndata: ${JSON.stringify({ type: event, ...payload })}\n\n`;
const validStream = [
  ": keepalive\n\n",
  sse("response.created", { response: { id: "internal-1" } }),
  sse("response.output_item.added", { output_index: 0,
    item: { type: "function_call", name: "terminal", arguments: "private tool arguments" } }),
  sse("response.reasoning_summary_text.delta", { delta: "secret reasoning" }),
  sse("response.output_item.added", { output_index: 1,
    item: { type: "message", phase: "commentary", content: "private commentary" } }),
  sse("response.output_text.delta", { output_index: 1, delta: "private commentary" }),
  sse("response.output_item.done", { output_index: 0,
    item: { type: "function_call", name: "terminal", status: "completed", output: "secret tool output" } }),
  sse("response.output_text.delta", { output_index: 2, delta: "Hello" }),
  sse("response.completed", { response: { id: "internal-1", output: [], usage: { foo: "private" } } }),
].join("");
const collect = async iterable => { const out = []; for await (const e of iterable) out.push(e); return out; };

function harness(overrides = {}) {
  const calls = [], attestations = [], credentials = [], scopes = [], authorizations = [];
  const tenant = {
    tenantKey, fabricTenantId, uid: tenantUID, deleting: false, status: "Ready",
  };
  const agent = {
    tenantKey, fabricTenantId, tenantBundleUID: tenantUID,
    agentKey: "alex", uid: agentUID, generation: 4, observedGeneration: 4,
    namespace: "tenant-indiba", serviceName: "hermes-alex",
    credentialRef: "hermes-api-alex", apiServerEnabled: true, deleting: false,
    status: "Ready",
  };
  const options = {
    tenantKey, fabricTenantId,
    browserEnabled: true, bridgeEnabled: true,
    authorize: async item => { authorizations.push(item); return true; },
    resolveCanonicalTenant: async () => tenant,
    resolveCanonicalAgent: async () => agent,
    verifyEndpoint: async item => { attestations.push(item); return true; },
    deriveConversationKey: async scope => { scopes.push(scope); return keyFor(scope); },
    readScopedCredential: async item => { credentials.push(item); return "c".repeat(40); },
    fetchImpl: async (upstream, request) => {
      calls.push({ url: upstream, request });
      return new Response(frames(
        validStream.slice(0, 14), validStream.slice(14, 55), validStream.slice(55),
      ), { headers: { "content-type": "text/event-stream" } });
    },
    ...overrides,
  };
  const service = createPrivateHermesResponsesService(options);
  return { service, tenant, agent, calls, attestations, credentials, scopes, authorizations, options };
}

test("first and next authorized turns use ONLY Responses and same server-derived opaque session key", async () => {
  const h = harness();
  const first = await collect(h.service.turn({
    principal, agentKey: "alex", input: "first",
    conversation: "browser-continuity", sessionId: "browser-session",
    apiKey: "browser-key", provider: "provider-override",
    previous_response_id: "browser-response", history: ["browser context"],
    headers: { Authorization: "browser bearer", Cookie: "sensitive browser cookie" },
  }));
  const second = await collect(h.service.turn({ principal, agentKey: "alex", input: "second" }));
  const projected = [
    { type: "tool-started", tool: "terminal" },
    { type: "tool-completed", tool: "terminal", failed: false },
    { type: "text-delta", text: "Hello" },
    { type: "done", status: "completed" },
  ];
  assert.deepEqual(first, projected);
  assert.deepEqual(second, projected);
  assert.equal(h.calls.length, 2);
  const expectedKey = keyFor({
    fabricTenantId, fabricUserId: user, agentUID, agentGeneration: 4,
  });
  for (const { url: destination, request } of h.calls) {
    assert.equal(destination, url);
    assert.equal(request.method, "POST");
    assert.equal(request.redirect, "error");
    assert.equal(request.cache, "no-store");
    assert.equal(request.headers["X-Hermes-Session-Key"], expectedKey);
    assert.equal(request.headers.Authorization, "Bearer " + "c".repeat(40));
    assert.equal(request.headers.Accept, "text/event-stream");
    assert.equal(request.headers.Cookie, undefined);
    assert.deepEqual(Object.keys(request.headers).sort(),
      ["Accept", "Authorization", "Content-Type", "X-Hermes-Session-Key"].sort());
    const body = JSON.parse(request.body);
    assert.deepEqual(Object.keys(body).sort(), ["input", "conversation", "stream", "store"].sort());
    assert.equal(body.conversation, expectedKey);
    assert.equal(body.stream, true);
    assert.equal(body.store, true);
    assert.equal(body.previous_response_id, undefined);
    assert.equal(body.history, undefined);
    assert.equal(body.model, undefined);
    assert.equal(body.provider, undefined);
    assert.doesNotMatch(JSON.stringify(body), /browser-continuity|browser-session|browser-response|override/);
  }
  assert.equal(h.authorizations.length, 2, "every turn needs a fresh grant");
  assert.equal(h.attestations.length, 2, "every turn needs fresh endpoint attestation");
  assert.equal(h.credentials.length, 2, "never cache a scoped credential");
  assert.equal(h.scopes.length, 2);
  assert.deepEqual(h.scopes[0], h.scopes[1]);
  assert.deepEqual(Object.keys(h.scopes[0]).sort(),
    ["fabricTenantId", "fabricUserId", "agentUID", "agentGeneration"].sort());
  assert.doesNotMatch(JSON.stringify(projected), /private|secret|internal|tool output|commentary/);
});

test("canonical tenant, Fabric user, AgentIdentity UID and generation all separate continuity", async () => {
  const h = harness();
  const keys = [];
  await collect(h.service.turn({ principal, agentKey: "alex", input: "hello" }));
  keys.push(h.calls.at(-1).request.headers["X-Hermes-Session-Key"]);
  const anotherUser = { ...principal, fabricUserId: "usr" + "b".repeat(32),
    subject: "usr" + "b".repeat(32) };
  await collect(h.service.turn({ principal: anotherUser, agentKey: "alex", input: "hello" }));
  keys.push(h.calls.at(-1).request.headers["X-Hermes-Session-Key"]);
  h.agent.uid = "2e2e2e2e-0000-4000-8000-000000000002";
  await collect(h.service.turn({ principal, agentKey: "alex", input: "hello" }));
  keys.push(h.calls.at(-1).request.headers["X-Hermes-Session-Key"]);
  h.agent.generation += 1;
  h.agent.observedGeneration += 1;
  await collect(h.service.turn({ principal, agentKey: "alex", input: "hello" }));
  keys.push(h.calls.at(-1).request.headers["X-Hermes-Session-Key"]);
  const otherTenant = harness({
    tenantKey: "hairem", fabricTenantId: "TEN00001",
    resolveCanonicalTenant: async () => ({
      tenantKey: "hairem", fabricTenantId: "TEN00001", uid: tenantUID,
      deleting: false, status: "Ready",
    }),
    resolveCanonicalAgent: async () => ({
      ...h.agent, tenantKey: "hairem", fabricTenantId: "TEN00001",
      namespace: "tenant-hairem", serviceName: "hermes-alex",
    }),
    fetchImpl: async (destination, request) => {
      keys.push(request.headers["X-Hermes-Session-Key"]);
      assert.equal(destination, "http://hermes-alex.tenant-hairem.svc:8642/v1/responses");
      return new Response(frames(sse("response.completed", {})), {
        headers: { "content-type": "text/event-stream" },
      });
    },
  });
  await collect(otherTenant.service.turn({ principal: {
    ...principal, tenantKey: "hairem", fabricTenantId: "TEN00001",
  }, agentKey: "alex", input: "hello" }));
  assert.equal(new Set(keys).size, 5, "all canonical scope changes must partition history");
});

test("both default gates, invalid identity and stale/foreign agents fail before credential and fetch", async () => {
  for (const overrides of [{browserEnabled: false}, {bridgeEnabled: false}]) {
    const h = harness(overrides);
    await assert.rejects(collect(h.service.turn({principal, agentKey:"alex", input:"hi"})));
    assert.equal(h.authorizations.length, 0);
    assert.equal(h.credentials.length, 0);
    assert.equal(h.calls.length, 0);
  }
  const invalidPrincipals = [
    { ...principal, authenticated: false }, { ...principal, verified: false },
    { ...principal, fabricTenantId: "TEN00001" },
    { ...principal, tenantKey: "hairem" },
    { ...principal, subject: "oidc-user" },
    { ...principal, fabricUserId: "oidc-user" },
    { ...principal, subject: "usr" + "b".repeat(32) },
  ];
  for (const bad of invalidPrincipals) {
    const h = harness();
    await assert.rejects(collect(h.service.turn({ principal: bad, agentKey:"alex", input:"hi" })));
    assert.equal(h.calls.length, 0);
    assert.equal(h.credentials.length, 0);
  }
  const mutations = [
    h=>{h.agent.uid="";}, h=>{h.agent.generation++;},
    h=>{h.agent.status="NotReady";}, h=>{h.agent.deleting=true;},
    h=>{h.agent.namespace="tenant-hairem";}, h=>{h.agent.serviceName="evil.svc";},
    h=>{h.agent.credentialRef="hermes-api-foreign";},
    h=>{h.agent.tenantBundleUID="3e3e3e3e-0000-4000-8000-000000000002";},
    h=>{h.agent.apiServerEnabled=false;},
    h=>{h.tenant.deleting=true;}, h=>{h.tenant.status="NotReady";},
  ];
  for (const mutate of mutations) {
    const h = harness();
    mutate(h);
    await assert.rejects(collect(h.service.turn({principal, agentKey:"alex", input:"hi"})));
    assert.equal(h.calls.length, 0);
    assert.equal(h.credentials.length, 0);
  }
  for (const overrides of [
    { authorize: async()=>false },
    { authorize: async()=>{throw Error("private policy");} },
    { verifyEndpoint: async()=>false },
    { verifyEndpoint: async()=>{throw Error("internal pod topology");} },
  ]) {
    const h = harness(overrides);
    await assert.rejects(collect(h.service.turn({principal, agentKey:"alex", input:"hi"})),
      e=>e.message === "Agent is unavailable");
    assert.equal(h.calls.length, 0);
    assert.equal(h.credentials.length, 0);
  }
});

test("derived key is checked BEFORE credential read and cannot be raw identity or control characters", async () => {
  const bad = [null, "", "TEN00002", user, agentUID, "v1_" + "a".repeat(257),
    "v1_" + "a".repeat(40) + "\r\n", "https://outside.invalid", "hermes-alex",
    "v1_" + "a".repeat(42), "v1_" + "a".repeat(43) + "\0"];
  for (const value of bad) {
    const h = harness({ deriveConversationKey: async () => value });
    await assert.rejects(collect(h.service.turn({principal, agentKey:"alex", input:"hi"})));
    assert.equal(h.credentials.length, 0);
    assert.equal(h.calls.length, 0);
  }
  assert.throws(()=>requireHermesConversationKey("tenant-indiba"),/unavailable/);
  const h = harness({ readScopedCredential: async () => {throw Error("internal secret name");} });
  await assert.rejects(collect(h.service.turn({principal, agentKey:"alex", input:"hi"})),
    e=>e.message === "Agent is unavailable" && !e.message.includes("secret"));
  assert.equal(h.calls.length, 0);
  assert.throws(()=>createPrivateHermesResponsesService(),/unavailable/);
});

test("Responses tool projection is strict and native-session events remain independent", () => {
  const commentary = new Set();
  assert.equal(projectHermesResponsesEvent({event:"assistant.delta", data:'{"text":"native"}'},commentary),null);
  assert.equal(projectHermesResponsesEvent({event:"response.reasoning_summary_text.delta",
    data:'{"delta":"private reasoning"}'},commentary),null);
  assert.deepEqual(projectHermesResponsesEvent({event:"response.failed",
    data:sse("response.failed",{error:{message:"sensitive error"}}).split("data: ")[1].trim()}, commentary),
    {type:"done",status:"failed"});
  assert.deepEqual(projectHermesResponsesEvent({event:"response.output_item.added",
    data:JSON.stringify({item:{type:"function_call",name:"terminal",arguments:"SECRET"}})}, commentary),
    {type:"tool-started",tool:"terminal"});
  assert.equal(projectHermesResponsesEvent({event:"response.output_item.done",
    data:JSON.stringify({item:{type:"function_call",name:"\nforged",output:"PRIVATE"}})}, commentary),null);
});

test("redirect, non-SSE, missing body, malformed/oversized frame and incomplete stream deny generically", async () => {
  const key = keyFor({fabricTenantId, fabricUserId:user,agentUID,agentGeneration:4});
  const badResponses = [
    { ok:false, redirected:false, headers:new Headers({"content-type":"text/event-stream"}), body:frames("raw upstream token") },
    { ok:true, redirected:true, headers:new Headers({"content-type":"text/event-stream"}), body:frames(validStream) },
    new Response("internal header leakage", {headers:{"content-type":"text/html"}}),
    { ok:true, redirected:false, headers:new Headers({"content-type":"text/event-stream"}), body:null },
    new Response(frames("event: response.output_text.delta\ndata: not-json\n\n"), {headers:{"content-type":"text/event-stream"}}),
    new Response(frames("event: response.output_text.delta\ndata: " + "x".repeat(20000) + "\n\n"),
      {headers:{"content-type":"text/event-stream"}}),
    new Response(frames(sse("response.output_text.delta",{delta:"partial"})),
      {headers:{"content-type":"text/event-stream"}}),
  ];
  for (const returned of badResponses) {
    await assert.rejects(collect(streamHermesResponses({
      baseURL:"http://hermes-alex.tenant-indiba.svc:8642", apiKey:"c".repeat(40),
      conversationKey:key, input:"hello", fetchImpl:async()=>returned,
    })), e => e.message === "Agent is unavailable" && !/internal|token|upstream/.test(e.message));
  }
  let calls = 0;
  for (const unsafe of [
    "https://public.invalid", "http://localhost:8642",
    "http://hermes-alex.tenant-indiba.svc@evil.invalid/",
  ]) {
    await assert.rejects(collect(streamHermesResponses({
      baseURL:unsafe, apiKey:"c".repeat(40),
      conversationKey:key, input:"hello", fetchImpl:async()=>{calls++;},
    })),/unavailable/);
  }
  assert.equal(calls,0);
});

test("abort propagates to Hermes fetch, without DELETE, resume or replay", async () => {
  const controller = new AbortController();
  let calls = 0;
  const h = harness({
    fetchImpl: async (destination, request) => {
      calls++;
      assert.equal(destination, url);
      assert.equal(request.method, "POST");
      assert.equal(request.signal.aborted, false);
      controller.abort();
      assert.equal(request.signal.aborted, true);
      throw Error("upstream private abort details");
    },
  });
  await assert.rejects(collect(h.service.turn({
    principal,agentKey:"alex",input:"hi", signal:controller.signal,
  })), e=>e.message==="Agent is unavailable");
  assert.equal(calls,1);
});

test("stateless service never requires BFF SQL or changes selected production", async () => {
  const h = harness({ repo: new Proxy({}, {get(){throw Error("BFF repo must never be consulted");}}) });
  await collect(h.service.turn({principal,agentKey:"alex",input:"hello"}));
  const doc = path => readFileSync(new URL(path, import.meta.url),"utf8");
  const deployment = doc("../runtime/deployment.yaml");
  const operator = doc("../../operator/config/manager/manager.yaml");
  const bootstrap = doc("../bootstrap/migrate-job.yaml");
  const prod = doc("../../overlays/prod/kustomization.yaml");
  const bff = doc("../server/private-bff-main.mjs");
  assert.match(deployment,/name: TXO_WEBUI_BFF_BROWSER_ENABLED\s+value: "false"/);
  assert.match(operator,/name: TXO_FABRIC_PRIVATE_BRIDGE_ENABLED\s+value: "false"/);
  assert.doesNotMatch(bootstrap,/003_fabric_chat_threads/);
  assert.doesNotMatch(prod,/hermes-api-|003_fabric_chat_threads/);
  assert.doesNotMatch(bff,/createPrivateHermesResponsesService|hermes-responses/);
});
