import test from "node:test";
import assert from "node:assert/strict";
import { parseSSE, projectChatEvent } from "../server/hermes-events.mjs";
import { streamHermesChat, requireInternalHermesURL } from "../server/hermes-client.mjs";
import { streamAuthorizedTurn } from "../server/authorized-turn.mjs";

const chunks = (...parts) => new ReadableStream({
  start(controller) {
    for (const part of parts) controller.enqueue(new TextEncoder().encode(part));
    controller.close();
  },
});
const collect = async (iterable) => { const out = []; for await (const item of iterable) out.push(item); return out; };
const target = { tenantKey: "indiba", agentKey: "usr000001-agt00001", baseURL: "http://hermes.tenant-indiba.svc:8642", apiKey: "sensitive-upstream-token" };
const principal = { authenticated: true, subject: "edfoley" };

test("SSE parses fragmented UTF-8, keepalives and multiline data", async () => {
  const result = await collect(parseSSE(chunks(": ping\r\nevent: assistant.delta\r\ndata: {\"text\":\"caf", "é\"}\r\n\r\n", "event: run.completed\ndata: {}\n\n")));
  assert.deepEqual(result, [
    { event: "assistant.delta", data: '{"text":"café"}' },
    { event: "run.completed", data: "{}" },
  ]);
});

test("multiple small events in a large read are allowed", async () => {
  const data = "event: run.completed\ndata: {}\n\n".repeat(5000);
  const result = await collect(parseSSE(chunks(data), 64));
  assert.equal(result.length, 5000);
});

test("SSE rejects oversized frames rather than buffering indefinitely", async () => {
  await assert.rejects(collect(parseSSE(chunks("data: ", "x".repeat(200)), 64)), /size limit/);
});

test("non-product events and tool previews are not exposed", () => {
  assert.equal(projectChatEvent({ event: "assistant.commentary", data: '{"text":"private reasoning"}' }), null);
  assert.equal(projectChatEvent({ event: "model.internal", data: '{"token":"sensitive"}' }), null);
  assert.deepEqual(projectChatEvent({ event: "tool.started", data: '{"tool":"terminal","preview":"secret=password","args":{"token":"SECRET"}}' }), { type: "tool-started", tool: "terminal" });
  assert.deepEqual(projectChatEvent({ event: "tool.started", data: '{"tool":"\\u001b[evil"}' }), { type: "tool-started", tool: "tool" });
  assert.deepEqual(projectChatEvent({ event: "run.failed", data: '{"error":"upstream OAuth token secret"}' }), { type: "done", status: "failed" });
});

test("rejects untrusted Hermes targets and malformed session IDs", async () => {
  for (const url of ["https://evil.tld:8642", "http://localhost:8642", "http://hermes.tenant-indiba.svc@evil.com", "ftp://hermes.tenant-indiba.svc"]) {
    assert.throws(() => requireInternalHermesURL(url), /Invalid internal/);
  }
  await assert.rejects(collect(streamHermesChat({ ...target, sessionId: "../another", input: "hello", fetchImpl: () => { throw Error("unexpected fetch"); } })), /Invalid Hermes session/);
});

test("only an authorized principal in the matching tenant reaches Hermes", async () => {
  let calls = 0;
  const fetchImpl = async (url, opts) => {
    calls++;
    assert.equal(url, "http://hermes.tenant-indiba.svc:8642/api/sessions/thread-1/chat/stream");
    assert.equal(opts.headers.Authorization, "Bearer " + target.apiKey);
    assert.equal(JSON.parse(opts.body).input, "hello");
    return { ok: true, body: chunks("event: assistant.delta\ndata: {\"text\":\"Hello\"}\n\n", "event: run.completed\ndata: {}\n\n") };
  };
  const params = { principal, tenantKey: "indiba", agentKey: target.agentKey, sessionId: "thread-1", input: "hello", fetchImpl,
    authorize: async (ctx) => ctx.tenantKey === "indiba" && ctx.principal.subject === "edfoley", resolveAgent: async () => target };
  assert.deepEqual(await collect(streamAuthorizedTurn(params)), [ { type: "text-delta", text: "Hello" }, { type: "done", status: "completed" } ]);
  assert.equal(calls, 1);
  await assert.rejects(collect(streamAuthorizedTurn({ ...params, principal: { subject: "edfoley" } })), /Forbidden/);
  await assert.rejects(collect(streamAuthorizedTurn({ ...params, principal: { authenticated: true, subject: "intruder" } })), /Forbidden/);
  await assert.rejects(collect(streamAuthorizedTurn({ ...params, tenantKey: "hairem" })), /Forbidden/);
  await assert.rejects(collect(streamAuthorizedTurn({ ...params, resolveAgent: async () => ({...target, tenantKey: "hairem"}) })), /Agent is unavailable/);
  assert.equal(calls, 1, "unauthorized requests must not touch Hermes");
});

test("upstream errors cannot leak sensitive error responses", async () => {
  const fetchImpl = async () => ({ ok: false, status: 500, body: chunks("secret upstream error") });
  await assert.rejects(collect(streamHermesChat({ ...target, sessionId: "thread-1", input: "hello", fetchImpl })), (error) => error.message === "Agent is unavailable" && !error.message.includes("secret"));
});
