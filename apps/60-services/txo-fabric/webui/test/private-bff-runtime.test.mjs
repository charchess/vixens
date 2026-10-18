import test from "node:test";
import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import { createPrivateBFFHandler, createPrivateBFFServer, composeGlobalBFF, createProductionBFFApp, resolveRuntimeGate } from "../server/private-bff-server.mjs";
import { createDeclaredBrowserArtifacts } from "../server/browser-artifacts.mjs";
import {
  createPrivateBridgeClient, createVerifiedTenantBridgeAccess,
  PRIVATE_BRIDGE_URL, PRIVATE_BRIDGE_PATH, PRIVATE_BRIDGE_AUDIENCE,
  PRIVATE_BFF_SUBJECT, GLOBAL_OIDC_ISSUER,
} from "../server/private-bridge-client.mjs";

const uuid = "550e8400-e29b-41d4-a716-446655440001";
const now = 1_760_214_000_000;
const key = Buffer.alloc(32, 107);
const token = Buffer.from("eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.sig");
const expected = { tenantKey: "indiba", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" };

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


const canonicalHuman = Object.freeze({
  authenticated: true,
  verified: true,
  issuer: GLOBAL_OIDC_ISSUER,
  oidcSubject: "opaque-global-sub",
  authentikUserUUID: uuid,
});
const canonicalTenantId = "TEN00001";

function privilegedBridgeFixture({ browserEnabled = true, bridgeEnabled = true,
  humanResult, bridgeResult, activeThrows = false, bridgeThrows = false } = {}) {
  const counts = { account: 0, bridge: 0 };
  const verifyActiveHuman = async (identity) => {
    counts.account++;
    if (activeThrows) throw Error("provider unavailable");
    return humanResult === undefined
      ? { ...identity, accountActive: true, fresh: true } : humanResult;
  };
  const bridgeClient = {
    async resolve() {
      counts.bridge++;
      if (bridgeThrows) throw Error("operator bridge unavailable");
      return bridgeResult === undefined
        ? { tenantKey: "indiba", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } : bridgeResult;
    },
  };
  const access = createVerifiedTenantBridgeAccess({
    browserEnabled, bridgeEnabled, verifyActiveHuman, bridgeClient,
  });
  return { access, counts, verifyActiveHuman, bridgeClient };
}

test("exact durable global session + server-owned tenant catalogue compose bridge membership only", async () => {
  const f = privilegedBridgeFixture();
  const first = await f.access({
    identity: canonicalHuman, tenantKey: "indiba", fabricTenantId: canonicalTenantId,
  });
  const second = await f.access({
    identity: canonicalHuman, tenantKey: "indiba", fabricTenantId: canonicalTenantId,
  });
  assert.deepEqual(first, {
    tenantKey: "indiba", fabricTenantId: canonicalTenantId,
    fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    accountActive: true, fresh: true, canEnter: true, canManageTenant: false,
  });
  assert.deepEqual(second, first);
  assert.deepEqual(f.counts, { account: 2, bridge: 2 },
    "new live human verification + unique bridge assertion required per operation");
  assert.ok(Object.isFrozen(first));
  // Both gates must be explicitly true even in a source-only composition.
  assert.throws(() => composeGlobalBFF({
    pool: { query() {} }, verifyActiveHuman: f.verifyActiveHuman,
    listTenantCatalog: () => [], bridgeClient: f.bridgeClient, getTenantChat: () => {},
  }), /dependencies/);
});

test("browser/bridge gate OFF performs zero privileged reads or calls", async () => {
  for (const setting of [{browserEnabled: false}, {bridgeEnabled: false},
    {browserEnabled: false, bridgeEnabled: false}]) {
    const f = privilegedBridgeFixture(setting);
    await assert.rejects(f.access({
      identity: canonicalHuman, tenantKey: "indiba", fabricTenantId: canonicalTenantId,
    }), /unavailable/);
    assert.deepEqual(f.counts, { account: 0, bridge: 0 });
  }
});

test("global issuer UUID and requested tenant cannot smuggle Fabric authority", async () => {
  const forged = [
    { identity: { ...canonicalHuman, issuer: "https://other.example.invalid/application/o/txo-fabric-webui/" } },
    { identity: { ...canonicalHuman, authentikUserUUID: "global-sub" } },
    { identity: { ...canonicalHuman, verified: false } },
    { identity: { ...canonicalHuman, authenticated: false } },
    { identity: { ...canonicalHuman, fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } },
    { identity: { ...canonicalHuman, oidcSubject: "" } },
    { tenantKey: "../indiba" },
    { fabricTenantId: "TEN00002_FROM_BROWSER" },
    { fabricTenantId: "indiba" },
    { fabricTenantId: undefined },
  ];
  for (const override of forged) {
    const f = privilegedBridgeFixture();
    await assert.rejects(f.access({
      identity: canonicalHuman, tenantKey: "indiba",
      fabricTenantId: canonicalTenantId, ...override,
    }), /unavailable/);
    assert.deepEqual(f.counts, {account: 0, bridge: 0});
  }
});

test("account revoke, freshness loss, bridge outage and foreign Fabric ID always deny", async () => {
  for (const settings of [
    { humanResult: { ...canonicalHuman, accountActive: false, fresh: true } },
    { humanResult: { ...canonicalHuman, accountActive: true, fresh: false } },
    { humanResult: { ...canonicalHuman, authentikUserUUID: "550e8400-e29b-41d4-a716-446655440002",
      accountActive: true, fresh: true } },
    { activeThrows: true },
    { bridgeThrows: true },
    { bridgeResult: { tenantKey: "sibling", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } },
    { bridgeResult: { tenantKey: "indiba", fabricUserId: "global-sub" } },
    { bridgeResult: { tenantKey: "indiba", fabricUserId: "usr000002" } },
    { bridgeResult: { tenantKey: "indiba", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      storeId: "client-supplied" } },
  ]) {
    const f = privilegedBridgeFixture(settings);
    await assert.rejects(f.access({
      identity: canonicalHuman, tenantKey: "indiba", fabricTenantId: canonicalTenantId,
    }), /^Error: Private Fabric identity unavailable$/);
    assert.equal(f.counts.account, 1);
    assert.equal(f.counts.bridge, settings.humanResult || settings.activeThrows ? 0 : 1);
  }
});

test("feature OFF serves a static Fabric entry page without privileged dependencies", async () => {
  let privilegedCalls = 0;
  const handler = createPrivateBFFHandler();
  const home = await handler(new Request("https://fabric.truxonline.com/"));
  assert.equal(home.status, 200);
  assert.equal(home.headers.get("content-type"), "text/html; charset=utf-8");
  assert.equal(home.headers.get("cache-control"), "no-store");
  assert.match(await home.text(), /Fabric/i);
  assert.doesNotMatch(await (await handler(new Request("https://fabric.truxonline.com/"))).text(), /secret|cookie|tenant|agent|chat/i);
  for (const path of ["/healthz", "/readyz"]) {
    const r = await handler(new Request("https://webui.truxonline.com" + path));
    assert.equal(r.status, 200);
    assert.equal(r.headers.get("cache-control"), "no-store");
  }
  for (const path of [
    "/auth/start", "/auth/callback?code=secret&state=secret",
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
  assert.throws(() => resolveRuntimeGate({}), /jointly enabled/);
  assert.throws(() => resolveRuntimeGate({ TXO_WEBUI_BFF_BROWSER_ENABLED: "true" }), /jointly enabled/);
  assert.equal(resolveRuntimeGate({
    TXO_WEBUI_BFF_BROWSER_ENABLED: "false",
    TXO_FABRIC_PRIVATE_BRIDGE_ENABLED: "false",
  }), false);
  for (const env of [
    { TXO_WEBUI_BFF_BROWSER_ENABLED: "false", TXO_FABRIC_PRIVATE_BRIDGE_ENABLED: "true" },
    { TXO_WEBUI_BFF_BROWSER_ENABLED: "true", TXO_FABRIC_PRIVATE_BRIDGE_ENABLED: "false" },
  ]) assert.throws(() => resolveRuntimeGate(env), /jointly enabled/);
  assert.equal(resolveRuntimeGate({
    TXO_WEBUI_BFF_BROWSER_ENABLED: "true",
    TXO_FABRIC_PRIVATE_BRIDGE_ENABLED: "true",
  }), true);
  assert.throws(() => createPrivateBFFHandler({ enabled: true }), /activation/);
  assert.throws(() => composeGlobalBFF({}), /dependencies/);
  assert.throws(() => createPrivateBFFHandler({ enabled: "false" }), /gate/);
});

test("actual private Node HTTP process denies browser auth/API while health answers", async () => {
  const server = createPrivateBFFServer({ handle: createPrivateBFFHandler() });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  try {
    const { port } = server.address();
    const home = await fetch("http://127.0.0.1:" + port + "/");
    assert.equal(home.status, 200);
    assert.equal(home.headers.get("content-type"), "text/html; charset=utf-8");
    assert.match(await home.text(), /Fabric/i);
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

test("private HTTP adapter fixes requests to the canonical Fabric origin", async () => {
  let seen;
  const server = createPrivateBFFServer({
    handle: async request => {
      seen = request.url;
      return new Response("ok");
    },
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  try {
    const { port } = server.address();
    assert.equal((await fetch("http://127.0.0.1:" + port + "/healthz")).status, 200);
    assert.equal(seen, "https://webui.truxonline.com/healthz");
  } finally {
    await new Promise(resolve => server.close(resolve));
  }
  assert.throws(() => createPrivateBFFServer({
    handle: async () => new Response("ok"), expectedOrigin: "https://evil.example",
  }), /configuration invalid/);
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
    new Response(JSON.stringify({ tenantKey: "hairem", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" })),
    new Response(JSON.stringify({ tenantKey: "indiba", fabricUserId: "global-sub" })),
    new Response(JSON.stringify({ ...expected, issuer: "forged" })),
    new Response(JSON.stringify({ tenantKey: "indiba", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", extra: true })),
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

test("enabled gates require complete production providers before an app is composed", async () => {
  const env = {
    TXO_WEBUI_BFF_BROWSER_ENABLED: "true",
    TXO_FABRIC_PRIVATE_BRIDGE_ENABLED: "true",
  };
  assert.equal(resolveRuntimeGate(env), true);
  await assert.rejects(createProductionBFFApp({ env }), /providers unavailable/);
  await assert.rejects(createProductionBFFApp({ env, providers: {
    ready: async () => true, browserArtifacts: { get: async () => null },
  } }), /dependencies unavailable/);
  const providers = {
    ready: async () => true,
    pool: { query: async () => ({ rows: [] }) },
    verifyActiveHuman: async identity => ({ ...identity, accountActive: true, fresh: true }),
    listTenantCatalog: async () => [],
    bridgeClient: { resolve: async () => ({ tenantKey: "indiba", fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }) },
    getTenantChat: async () => null,
    browserArtifacts: { get: async () => null },
  };
  const composition = await createProductionBFFApp({ env, providers });
  assert.equal(composition.enabled, true);
  assert.equal(typeof composition.app, "function");
  assert.equal(await composition.ready(), true);
  const handler = createPrivateBFFHandler(composition);
  assert.equal((await handler(new Request("https://fabric.truxonline.com/"))).status, 403,
    "the server fixes requests to the canonical origin before the app sees them");
});

test("browser artifacts are manifest-allowlisted and reject traversal, source maps and source files", async () => {
  const files = new Map([
    ["/dist/asset-manifest.json", JSON.stringify({
      "index.html": { file: "assets/app-123.js", css: ["assets/app-123.css"] },
    })],
    ["/dist/index.html", "<!doctype html><script src=\"/assets/app-123.js\"></script>"],
    ["/dist/assets/app-123.js", "console.log('browser')"],
    ["/dist/assets/app-123.css", "body{}"],
    ["/dist/server/private-bff-server.mjs", "secret source"],
  ]);
  const artifacts = await createDeclaredBrowserArtifacts({
    assetDir: "/dist", manifestPath: "/dist/asset-manifest.json",
    readFile: async (path, encoding) => {
      const value = files.get(path);
      if (value === undefined) throw new Error("missing");
      return encoding ? value : Buffer.from(value);
    },
  });
  assert.equal((await artifacts.get("/")).contentType, "text/html; charset=utf-8");
  assert.equal((await artifacts.get("/assets/app-123.js")).contentType, "text/javascript; charset=utf-8");
  for (const path of ["/asset-manifest.json", "/assets/app-123.js.map", "/../server/private-bff-server.mjs",
    "/%2e%2e/server/private-bff-server.mjs", "/client/src/App.tsx", "/assets/undeclared.js"]) {
    assert.equal(await artifacts.get(path), null, path);
  }
});
