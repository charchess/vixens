import test from "node:test";
import assert from "node:assert/strict";
import { createHash, generateKeyPairSync, sign } from "node:crypto";
import { createFabricOIDCLogin } from "../server/oidc-login.mjs";

const tenantKey = "hairem";
const issuer = "https://authentik.truxonline.com/application/o/txo-fabric-hairem/";
const clientId = "txo-fabric-hairem";
const authentikOrigin = "https://authentik.truxonline.com";
const redirectURI = "https://chat-hairem.truxonline.com/auth/callback";
const domainSuffix = "truxonline.com";
const accountUUID = "12345678-1234-4123-8123-1234567890ab";
const sub = "opaque-hashed-sub-not-a-uuid";
const fabricUserId = "usr" + "a".repeat(32);
const seconds = 1797000000;
const millis = seconds * 1000;
const { publicKey, privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
const jwk = { ...publicKey.export({ format: "jwk" }), kid: "scope-key-1", use: "sig", alg: "RS256" };
const digest = (data) => createHash("sha256").update(data).digest("base64url");
const token = (claims) => {
  const payload = [
    { alg: "RS256", kid: jwk.kid, typ: "JWT" },
    { iss: issuer, aud: clientId, sub, iat: seconds - 10, exp: seconds + 100,
      txo_fabric_user_uuid: accountUUID, ...claims },
  ].map((value) => Buffer.from(JSON.stringify(value)).toString("base64url")).join(".");
  return payload + "." + sign("RSA-SHA256", Buffer.from(payload), privateKey).toString("base64url");
};

function memoryStore() {
  const pending = new Map();
  let inserted = 0, consumed = 0;
  return {
    pending,
    get inserted() { return inserted; },
    get consumed() { return consumed; },
    async insert({ tenantKey, state, value, expiresAt }) {
      const key = tenantKey + ":" + state;
      if (pending.has(key)) throw new Error("duplicate transaction");
      if (expiresAt - value.issuedAt !== 300_000) throw Error("invalid TTL");
      inserted++;
      pending.set(key, structuredClone(value));
    },
    async consume({ tenantKey, state }) {
      consumed++;
      const key = tenantKey + ":" + state;
      const value = pending.get(key);
      pending.delete(key); // Atomic get-and-delete in production durable implementation.
      return value;
    },
  };
}

function setup(options = {}) {
  const store = options.store || memoryStore();
  const exchanges = [], enrollments = [], network = [];
  let currentTime = millis;
  const fetchImpl = options.fetchImpl || (async (url, args) => {
    network.push({ url, args });
    if (url === "https://authentik.truxonline.com/application/o/token/") {
      const form = new URLSearchParams(args.body);
      exchanges.push(form);
      const value = [...store.pending.values()][0] ||
        options.consumedValue;
      const nonce = value?.nonce;
      return new Response(JSON.stringify({
        token_type: "Bearer",
        id_token: token({
          nonce: options.tokenNonce || nonce,
          ...(options.tokenClaims || {}),
        }),
      }), { status: options.tokenStatus || 200 });
    }
    if (url === issuer + ".well-known/openid-configuration") {
      return new Response(JSON.stringify({ issuer, jwks_uri: issuer + "jwks/" }));
    }
    if (url === issuer + "jwks/") {
      return new Response(JSON.stringify({ keys: [jwk] }));
    }
    throw new Error("unexpected endpoint " + url);
  });
  const enrollVerifiedHuman = options.enrollVerifiedHuman || (async (proof) => {
    enrollments.push(proof);
    return fabricUserId;
  });
  const login = createFabricOIDCLogin({
    tenantKey, issuer, clientId, domainSuffix, redirectURI, authentikOrigin, transactions: store,
    now: () => currentTime, fetchImpl, enrollVerifiedHuman,
  });
  async function started() {
    const start = await login.begin();
    const url = new URL(start.authorizationURL);
    const state = url.searchParams.get("state");
    // Capture the atomic, server-owned pending record before callback consumption.
    const consumedValue = structuredClone(store.pending.get(tenantKey + ":" + state));
    return {
      start, url, state, consumedValue,
      input: { state, code: "server-issued-authorization-code",
        browserCookieValue: start.browserCookie.value },
    };
  }
  return { login, started, store, exchanges, enrollments, network,
    setTime(value) { currentTime = value; } };
}

async function completeFixture(fixture, start) {
  // Network mock needs the transaction captured BEFORE atomic consumption.
  // Real Authentik supplies the ID token after validating code + PKCE.
  fixture.store.pending.set("__test_snapshot", start.consumedValue);
  try { return await fixture.login.complete(start.input); }
  finally { fixture.store.pending.delete("__test_snapshot"); }
}

test("single-use browser-bound OIDC code+S256 PKCE produces only a verified private enrollment", async () => {
  const f = setup();
  const begin = await f.started();
  const { url, start, input, consumedValue } = begin;
  assert.equal(url.origin, "https://authentik.truxonline.com");
  assert.equal(url.pathname, "/application/o/authorize/");
  assert.equal(url.searchParams.get("scope"), "openid profile email txo_fabric_identity");
  assert.equal(url.searchParams.get("client_id"), clientId);
  assert.equal(url.searchParams.get("redirect_uri"), redirectURI);
  assert.equal(url.searchParams.get("response_type"), "code");
  assert.equal(url.searchParams.get("code_challenge_method"), "S256");
  assert.equal(url.searchParams.get("code_challenge"), digest(consumedValue.pkceVerifier));
  assert.equal(url.searchParams.get("nonce"), consumedValue.nonce);
  assert.equal(start.browserCookie.name, "__Host-txo-fabric-oidc");
  assert.deepEqual({
    httpOnly: start.browserCookie.httpOnly, secure: start.browserCookie.secure,
    sameSite: start.browserCookie.sameSite, path: start.browserCookie.path,
  }, { httpOnly: true, secure: true, sameSite: "Lax", path: "/" });
  assert.notEqual(start.browserCookie.value, url.searchParams.get("state"));
  assert.equal(consumedValue.bindingDigest, digest(start.browserCookie.value));
  assert.equal(f.store.pending.has(tenantKey + ":" + input.state), true);

  const result = await completeFixture(f, begin);
  assert.deepEqual(result, { tenantKey, issuer, oidcSubject: sub, fabricUserId });
  assert.equal(result.verified, undefined);
  assert.equal(result.authenticated, undefined);
  assert.equal(result.authentikUserUUID, undefined);
  assert.equal(f.store.pending.has(tenantKey + ":" + input.state), false);
  assert.equal(f.enrollments.length, 1);
  assert.deepEqual(f.enrollments[0], {
    tenantKey, issuer, oidcSubject: sub, authentikUserUUID: accountUUID,
  });
  assert.equal(f.enrollments[0].fabricUserId, undefined);
  assert.equal(f.exchanges.length, 1);
  assert.equal(f.exchanges[0].get("code_verifier"), consumedValue.pkceVerifier);
  assert.equal(f.exchanges[0].get("redirect_uri"), redirectURI);
  assert.equal(f.exchanges[0].get("client_secret"), null);
  const post = f.network.find(({ args }) => args?.method === "POST");
  assert.equal(post.url, "https://authentik.truxonline.com/application/o/token/");
  assert.equal(post.args.redirect, "error");
  assert.equal(post.args.cache, "no-store");
  assert.equal(post.args.headers["Content-Type"], "application/x-www-form-urlencoded");
  await assert.rejects(() => f.login.complete(input), /Fabric OIDC login denied/);
  assert.equal(f.enrollments.length, 1);
  assert.equal(f.exchanges.length, 1);
});

test("concurrent callbacks can consume a given login transaction only once", async () => {
  const f = setup();
  const start = await f.started();
  f.store.pending.set("__test_snapshot", start.consumedValue);
  const results = await Promise.allSettled([
    f.login.complete(start.input), f.login.complete(start.input),
  ]);
  f.store.pending.delete("__test_snapshot");
  assert.equal(results.filter((x) => x.status === "fulfilled").length, 1);
  assert.equal(results.filter((x) => x.status === "rejected").length, 1);
  assert.equal(f.enrollments.length, 1);
  assert.equal(f.exchanges.length, 1);
});

test("wrong browser cookie burns the state before code exchange or enrollment", async () => {
  const f = setup();
  const start = await f.started();
  await assert.rejects(() => f.login.complete({
    ...start.input, browserCookieValue: "A".repeat(43),
  }), /Fabric OIDC login denied/);
  await assert.rejects(() => completeFixture(f, start), /Fabric OIDC login denied/);
  assert.equal(f.exchanges.length, 0);
  assert.equal(f.enrollments.length, 0);
});

test("expired, future and mismatched transactions are denied after atomic consumption", async () => {
  for (const mutation of [
    (f) => f.setTime(millis + 300_001),
    (f) => f.setTime(millis - 1),
    (_, start) => { start.consumedValue.issuer = "https://evil.example/"; },
    (_, start) => { start.consumedValue.clientId = "txo-fabric-indiba"; },
    (_, start) => { start.consumedValue.redirectURI = "https://evil.example/auth/callback"; },
    (_, start) => { start.consumedValue.pkceVerifier = "forged"; },
    (_, start) => { start.consumedValue.bindingDigest = "wrong"; },
  ]) {
    const f = setup();
    const start = await f.started();
    mutation(f, start);
    // Replace the actual transaction, NOT merely the test-only capture.
    f.store.pending.set(tenantKey + ":" + start.state, start.consumedValue);
    await assert.rejects(() => f.login.complete(start.input), /Fabric OIDC login denied/);
    assert.equal(f.store.pending.size, 0);
    assert.equal(f.exchanges.length, 0);
    assert.equal(f.enrollments.length, 0);
  }
});

test("cross-tenant state cannot be spent in another tenant coordinator", async () => {
  const store = memoryStore();
  const a = setup({ store });
  const start = await a.started();
  const b = createFabricOIDCLogin({
    tenantKey: "indiba", issuer: "https://authentik.truxonline.com/application/o/txo-fabric-indiba/",
    clientId: "txo-fabric-indiba", domainSuffix, authentikOrigin, redirectURI: "https://chat-indiba.truxonline.com/auth/callback",
    transactions: store, fetchImpl: async () => { throw Error("must not call"); },
    enrollVerifiedHuman: async () => { throw Error("must not call"); },
    now: () => millis,
  });
  await assert.rejects(() => b.complete(start.input), /Fabric OIDC login denied/);
  assert.equal(store.pending.has(tenantKey + ":" + start.state), true);
  assert.equal(a.enrollments.length, 0);
});

test("bad nonce or forged ID token never calls the privileged enrollment service", async () => {
  for (const tokenClaims of [
    { nonce: "wrong-server-nonce" },
    { txo_fabric_user_uuid: "not-a-uuid" },
    { aud: "txo-fabric-indiba" },
    { iss: "https://evil.example/" },
    { exp: seconds - 1 },
  ]) {
    const f = setup({ tokenClaims });
    const start = await f.started();
    await assert.rejects(() => completeFixture(f, start), /Fabric OIDC login denied/);
    assert.equal(f.enrollments.length, 0);
    assert.equal(f.store.pending.has(tenantKey + ":" + start.state), false);
  }
});

test("token endpoint HTTP failures, invalid JSON, redirects, and enrollment refusal fail closed", async () => {
  const source = setup({ tokenStatus: 503 });
  const start = await source.started();
  await assert.rejects(() => completeFixture(source, start), /Fabric OIDC login denied/);
  assert.equal(source.enrollments.length, 0);
  const broken = setup({ fetchImpl: async () => new Response("{invalid", { status: 200 }) });
  const b = await broken.started();
  await assert.rejects(() => broken.login.complete(b.input), /Fabric OIDC login denied/);
  const redirect = setup({ fetchImpl: async () => new Response("", { status: 302 }) });
  const r = await redirect.started();
  await assert.rejects(() => redirect.login.complete(r.input), /Fabric OIDC login denied/);
  const refused = setup({ enrollVerifiedHuman: async () => { throw new Error("cannot trust Fabric ledger"); } });
  const c = await refused.started();
  await assert.rejects(() => completeFixture(refused, c), /Fabric OIDC login denied/);
});

test("malformed callbacks, missing security collaborators and untrusted redirect hosts are denied", async () => {
  const f = setup();
  for (const input of [
    {}, { state: "bad", code: "a", browserCookieValue: "A".repeat(43) },
    { state: "A".repeat(43), code: "", browserCookieValue: "A".repeat(43) },
    { state: "A".repeat(43), code: "evil\nInjected: yes", browserCookieValue: "A".repeat(43) },
  ]) await assert.rejects(() => f.login.complete(input), /Fabric OIDC login denied/);
  assert.equal(f.network.length, 0);
  const shared = {
    tenantKey, issuer, clientId, domainSuffix, redirectURI, authentikOrigin,
    transactions: f.store, enrollVerifiedHuman: () => fabricUserId,
  };
  for (const override of [
    { tenantKey: "indiba" },
    { issuer: "https://evil.example/application/o/txo-fabric-hairem/" },
    { authentikOrigin: "https://evil.example" },
    { authentikOrigin: "http://authentik.truxonline.com" },
    { authentikOrigin: "https://authentik.truxonline.com/evil" },
    { issuer: "http://authentik.truxonline.com/application/o/txo-fabric-hairem/" },
    { clientId: "txo-fabric-indiba" },
    { redirectURI: "https://chat-indiba.truxonline.com/auth/callback" },
    { redirectURI: "http://chat-hairem.truxonline.com/auth/callback" },
    { redirectURI: "https://chat-hairem.evil.example/auth/callback" },
    { redirectURI: "https://hairem-chat.truxonline.com/auth/callback" },
    { redirectURI: "https://chat-hairem.truxonline.com/auth/callback?code=evil" },
    { redirectURI: "https://chat-hairem.truxonline.com:444/auth/callback" },
    { transactions: null }, { enrollVerifiedHuman: null },
    { domainSuffix: "evil.example" },
  ]) assert.throws(() => createFabricOIDCLogin({ ...shared, ...override }),
    /Fabric OIDC login denied/, JSON.stringify(override));
});
