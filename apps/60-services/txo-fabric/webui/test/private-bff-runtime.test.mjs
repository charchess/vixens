import test from "node:test";
import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import { createPrivateBFFHandler, createPrivateBFFServer, composeGlobalBFF, resolveRuntimeGate } from "../server/private-bff-server.mjs";
import {
  createPrivateBridgeClient, PRIVATE_BRIDGE_URL, PRIVATE_BRIDGE_PATH,
  PRIVATE_BRIDGE_AUDIENCE, PRIVATE_BFF_SUBJECT,
} from "../server/private-bridge-client.mjs";

const uuid = "550e8400-e29b-41d4-a716-446655440001";
const now = 1_760_214_000_000;
const key = Buffer.alloc(32, 107);
const token = Buffer.from("eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.sig");
const expected = { tenantKey: "indiba", fabricUserId: "usr000002" };

function fixture({ enabled = true, response, failRead = false, failFetch = false } = {}) {
  const calls = [];
  const readPaths = [];
  const client = createPrivateBridgeClient({
    enabled,
    now: () => now,
    readFile: async (path) => {
      readPaths.push(path);
      if (failRead) throw Error("sensitive credential error");
      return path.endsWith("hmac-key") ? key : token;
    },
    fetchImpl: async (url, options) => {
      calls.push({ url, options });
      if (failFetch) throw Error("sensitive response body");
      return response || new Response(JSON.stringify(expected), {
        status: 200, headers: { "Content-Type": "application/json" },
      });
    },
  });
  return { client, calls, readPaths };
}

test("feature OFF serves private health only and never invokes any app dependency", async () => {
  let privilegedCalls = 0;
  const handler = createPrivateBFFHandler();
  for (const path of ["/healthz", "/readyz"]) {
    const r = await handler(new Request("https://webui.truxonline.com" + path));
    assert.equal(r.status, 200);
    assert.equal(r.headers.get("cache-control"), "no-store");
  }
  for (const path of [
    "/", "/auth/start", "/auth/callback?code=secret&state=secret",
    "/auth/logout", "/api/me", "/api/tenants/indiba/chat/threads",
    "/api/tenants/indiba/chat/agents", "/settings", "/admin",
  ]) {
    const r = await handler(new Request("https://webui.truxonline.com" + path, {
      headers: { cookie: "__Host-txo-fabric-session=secret", authorization: "Bearer forged" },
    }));
    assert.equal(r.status, 404);
    assert.equal(r.headers.get("cache-control"), "no-store");
    assert.doesNotMatch(await r.text(), /secret|indiba|thread/i);
  }
  assert.equal(privilegedCalls, 0);
  assert.throws(() => resolveRuntimeGate({}), /not wired/);
  assert.throws(() => resolveRuntimeGate({ TXO_WEBUI_BFF_BROWSER_ENABLED: "true" }), /not wired/);
  assert.equal(resolveRuntimeGate({ TXO_WEBUI_BFF_BROWSER_ENABLED: "false" }), false);
  assert.throws(() => createPrivateBFFHandler({ enabled: true }), /activation/);
  assert.throws(() => composeGlobalBFF({}), /dependencies/);
  assert.throws(() => createPrivateBFFHandler({ enabled: "false" }), /gate/);
});

test("actual private Node HTTP process denies browser auth/API while health answers", async () => {
  const server = createPrivateBFFServer({ handle: createPrivateBFFHandler() });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  try {
    const { port } = server.address();
    for (const path of ["/healthz", "/readyz"]) {
      const r = await fetch("http://127.0.0.1:" + port + path);
      assert.equal(r.status, 200);
      assert.equal(await r.text(), "ok");
    }
    for (const path of ["/auth/start", "/auth/callback", "/api/me", "/api/tenants/hairem/chat/threads"]) {
      const r = await fetch("http://127.0.0.1:" + port + path, {
        headers: { cookie: "sensitive", authorization: "Bearer browser" },
      });
      assert.equal(r.status, 404);
      assert.equal(r.headers.get("cache-control"), "no-store");
    }
  } finally {
    await new Promise(resolve => server.close(resolve));
  }
});

test("bridge HMAC matches exact Go method/path/tenant/UUID/SA/audience/iAt/exp/JTI contract", async () => {
  const f = fixture();
  const a = await f.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid });
  const b = await f.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid });
  assert.deepEqual(a, expected);
  assert.deepEqual(b, expected);
  assert.equal(f.calls.length, 2);
  assert.equal(f.readPaths.length, 4);
  const seen = new Set();
  for (const { url, options } of f.calls) {
    assert.equal(url, PRIVATE_BRIDGE_URL);
    assert.equal(options.method, "POST");
    assert.equal(options.redirect, "error");
    assert.equal(options.cache, "no-store");
    assert.equal(options.headers.Authorization, "Bearer " + token.toString("utf8"));
    assert.equal(options.headers["Content-Type"], "application/json");
    assert.deepEqual(JSON.parse(options.body), { tenantKey: "indiba", authentikUserUUID: uuid });
    const [encoded, mac] = options.headers["X-Fabric-Assertion"].split(".");
    assert.equal(mac, createHmac("sha256", key).update(encoded).digest("base64url"));
    const data = JSON.parse(Buffer.from(encoded, "base64url").toString());
    assert.deepEqual(Object.keys(data), [
      "method", "path", "tenantKey", "authentikUserUUID", "serviceAccount",
      "audience", "iat", "exp", "jti",
    ]);
    assert.equal(data.method, "POST");
    assert.equal(data.path, PRIVATE_BRIDGE_PATH);
    assert.equal(data.tenantKey, "indiba");
    assert.equal(data.authentikUserUUID, uuid);
    assert.equal(data.serviceAccount, PRIVATE_BFF_SUBJECT);
    assert.equal(data.audience, PRIVATE_BRIDGE_AUDIENCE);
    assert.equal(data.iat, Math.floor(now / 1000));
    assert.equal(data.exp - data.iat, 20);
    assert.match(data.jti, /^[A-Za-z0-9_-]{43}$/);
    assert.ok(!seen.has(data.jti), "JTI reused");
    seen.add(data.jti);
    assert.ok(options.signal);
  }
});

test("client cannot bypass feature off, wrong endpoint or missing credentials", async () => {
  const off = fixture({ enabled: false });
  await assert.rejects(off.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid }), /unavailable/);
  assert.equal(off.calls.length, 0);
  assert.equal(off.readPaths.length, 0);
  const broken = fixture({ failRead: true });
  await assert.rejects(broken.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid }), /unavailable/);
  assert.equal(broken.calls.length, 0);
  assert.throws(() => createPrivateBridgeClient({ enabled: true, endpoint: "http://evil:8090/internal/v1/identity/resolve" }), /unavailable/);
  assert.throws(() => createPrivateBridgeClient({ enabled: true, endpoint: PRIVATE_BRIDGE_URL + "?sub=forged" }), /unavailable/);
  const forged = fixture();
  await assert.rejects(forged.client.resolve({ tenantKey: "../other", authentikUserUUID: uuid }), /unavailable/);
  await assert.rejects(forged.client.resolve({ tenantKey: "indiba", authentikUserUUID: "global-sub" }), /unavailable/);
  assert.equal(forged.calls.length, 0);
});

test("bridge rejects timeout, forbidden, malformed, foreign tenant/user and redirect without fallback", async () => {
  for (const response of [
    new Response("sensitive", { status: 403 }),
    new Response("not found", { status: 404 }),
    new Response("bad request", { status: 503 }),
    new Response("corrupt", { status: 200 }),
    new Response(JSON.stringify({ tenantKey: "hairem", fabricUserId: "usr000002" })),
    new Response(JSON.stringify({ tenantKey: "indiba", fabricUserId: "global-sub" })),
    new Response(JSON.stringify({ ...expected, issuer: "forged" })),
    new Response(JSON.stringify({ tenantKey: "indiba", fabricUserId: "usr000002", extra: true })),
    new Response(JSON.stringify(expected), { status: 302, headers: { Location: "https://evil.test" } }),
  ]) {
    const f = fixture({ response });
    await assert.rejects(f.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid }), /^Error: Private Fabric identity unavailable$/);
    assert.equal(f.calls.length, 1, "client retried or used fallback");
  }
  const f = fixture({ failFetch: true });
  await assert.rejects(f.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid }), /unavailable/);
  assert.equal(f.calls.length, 1);
});

test("bridge makes no request when signer cannot obtain real CSPRNG nonce", async () => {
  const c = createPrivateBridgeClient({
    enabled: true, now: () => now, randomBytes: () => Buffer.alloc(1),
    readFile: async () => { throw Error("must not read"); },
    fetchImpl: async () => { throw Error("must not call"); },
  });
  await assert.rejects(c.resolve({ tenantKey: "indiba", authentikUserUUID: uuid }), /unavailable/);
});

test("BFF denial does not log signing key, UUID, bearer or opaque browser cookie", async () => {
  const old = console.error;
  const writes = [];
  console.error = (...values) => writes.push(values.join(" "));
  try {
    const f = fixture({ failFetch: true });
    await assert.rejects(f.client.resolve({ tenantKey: "indiba", authentikUserUUID: uuid }), /unavailable/);
    assert.deepEqual(writes, []);
  } finally {
    console.error = old;
  }
});
