import test from "node:test";
import assert from "node:assert/strict";
import {
  generateKeyPairSync, createHash, sign as rsaSign,
} from "node:crypto";
import { createGlobalOIDCLogin } from "../server/global-oidc-login.mjs";
import { createGlobalAuthHttp, createGlobalWebUIApp } from "../server/global-auth-http.mjs";

const origin = "https://webui.truxonline.com";
const idp = "https://authentik.truxonline.com";
const issuer = idp + "/application/o/txo-fabric-webui/";
const userUUID = "c38f3cc7-5b33-4266-bd85-40ce9a100061";
const sub = "opaque-global-subject-987";
const now = 1_797_000_000_000;
const nonceHash = s => createHash("sha256").update(s).digest("base64url");
const { publicKey, privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
const publicJwk = publicKey.export({ format: "jwk" });
const kid = "txo-global-test-key";
const jwt = (claims) => {
  const head = Buffer.from(JSON.stringify({ alg: "RS256", kid, typ: "JWT" })).toString("base64url");
  const body = Buffer.from(JSON.stringify(claims)).toString("base64url");
  const data = head + "." + body;
  const signature = rsaSign("RSA-SHA256", Buffer.from(data), privateKey).toString("base64url");
  return data + "." + signature;
};

function setup({ tokenOverrides = {}, active = true, tokenStatus = 200 } = {}) {
  const records = new Map(), sessions = new Map(), calls = [];
  let live = active, clock = now, counter = 0;
  const transactions = {
    async insert({ tenantKey, state, value }) {
      const key = tenantKey + ":" + state;
      if (records.has(key)) throw Error("duplicate");
      records.set(key, value);
    },
    async consume({ tenantKey, state }) {
      const key = tenantKey + ":" + state;
      const value = records.get(key);
      records.delete(key);
      return value ?? null;
    },
  };
  const fetchImpl = async (url, options) => {
    calls.push({ url, options });
    if (url === idp + "/application/o/token/") {
      assert.equal(options.method, "POST");
      assert.equal(options.redirect, "error");
      const form = new URLSearchParams(options.body);
      assert.equal(form.get("grant_type"), "authorization_code");
      assert.equal(form.get("client_id"), "txo-fabric-webui");
      assert.equal(form.get("redirect_uri"), origin + "/auth/callback");
      assert.ok(form.get("code_verifier"));
      const record = [...records.values()][0] || captured;
      assert.equal(nonceHash(form.get("code_verifier")), pendingChallenge);
      return new Response(JSON.stringify({
        token_type: "Bearer",
        id_token: jwt({
          iss: issuer, aud: "txo-fabric-webui", sub,
          txo_fabric_user_uuid: userUUID, nonce: record.nonce,
          iat: Math.floor(now / 1000), exp: Math.floor(now / 1000) + 300,
          ...tokenOverrides,
        }),
      }), { status: tokenStatus });
    }
    if (url === issuer + ".well-known/openid-configuration") {
      return Response.json({ issuer, jwks_uri: issuer + "jwks/" });
    }
    if (url === issuer + "jwks/") {
      return Response.json({ keys: [{ ...publicJwk, kid, use: "sig", alg: "RS256" }] });
    }
    throw Error("unexpected external endpoint " + url);
  };
  let captured, pendingChallenge;
  const login = createGlobalOIDCLogin({
    expectedOrigin: origin, authentikOrigin: idp,
    transactions, fetchImpl, now: () => clock,
  });
  const store = {
    async issue(proof) {
      assert.equal(proof.issuer, issuer);
      assert.equal(proof.oidcSubject, sub);
      assert.equal(proof.authentikUserUUID, userUUID);
      const token = ("A".repeat(42) + String(++counter)).slice(0,43);
      sessions.set(token, {
        authenticated: true, verified: true, ...proof,
      });
      return { token, maxAge: 28800 };
    },
    async authenticate(token) { return sessions.get(token) || null; },
    async revoke(token) { sessions.delete(token); },
  };
  const auth = createGlobalAuthHttp({
    expectedOrigin: origin, login, sessions: store,
    verifyActiveHuman: async proof => ({
      ...proof, accountActive: live, fresh: true,
    }),
  });
  return {
    transactions, calls, records, sessions, login, auth, store,
    setActive: v => { live = v; },
    setTime: v => { clock = v; },
    async begin() {
      const response = await auth.handle(new Request(origin + "/auth/start"));
      assert.equal(response.status, 302);
      const url = new URL(response.headers.get("location"));
      pendingChallenge = url.searchParams.get("code_challenge");
      const state = url.searchParams.get("state");
      captured = records.get("webui:" + state);
      assert.ok(captured);
      const binding = response.headers.get("set-cookie").match(/__Host-txo-fabric-oidc=([^;]+)/)[1];
      return { response, url, state, binding };
    },
    async complete(start, overrides = {}) {
      const request = new Request(origin + "/auth/callback?" +
        new URLSearchParams({
          state: start.state,
          code: "issued-code",
          ...overrides,
        }), { headers: { Cookie: `__Host-txo-fabric-oidc=${start.binding}` } });
      return auth.handle(request);
    },
  };
}

test("global OIDC flow validates signed JWT, nonce, PKCE, exact issuer and creates a session with no tenant grant", async () => {
  const h = setup(), start = await h.begin();
  assert.equal(start.url.searchParams.get("client_id"), "txo-fabric-webui");
  assert.equal(start.url.searchParams.get("redirect_uri"), origin + "/auth/callback");
  assert.equal(start.url.searchParams.get("code_challenge_method"), "S256");
  assert.equal(start.url.searchParams.get("scope"), "openid profile email txo_fabric_identity");
  assert.equal(start.response.headers.get("location").startsWith(idp + "/application/o/authorize/"), true);
  assert.match(start.response.headers.get("set-cookie"), /HttpOnly; Secure; SameSite=Lax/);
  assert.equal(h.records.has("webui:" + start.state), true);
  assert.equal(h.records.get("webui:" + start.state).bindingDigest, nonceHash(start.binding));
  const response = await h.complete(start);
  assert.equal(response.status, 303);
  assert.equal(response.headers.get("location"), origin + "/");
  assert.equal(response.headers.get("cache-control"), "no-store");
  const cookies = response.headers.getSetCookie();
  assert.equal(cookies.length, 2);
  assert.ok(cookies.some(s => s.startsWith("__Host-txo-fabric-oidc=;") && s.includes("Max-Age=0")));
  const session = cookies.find(s => s.startsWith("__Host-txo-fabric-session="));
  assert.match(session, /HttpOnly; Secure; SameSite=Lax/);
  assert.equal(h.sessions.size, 1);
  const [identity] = h.sessions.values();
  assert.deepEqual(identity, {
    authenticated: true, verified: true, issuer,
    oidcSubject: sub, authentikUserUUID: userUUID,
  });
  assert.equal(identity.fabricUserId, undefined);
  assert.equal(identity.tenantKey, undefined);
  assert.equal(h.records.size, 0);
  assert.equal((await h.complete(start)).status, 403);
  assert.equal(h.sessions.size, 1);
  assert.equal(h.calls.filter(x => x.options.method === "POST").length, 1);
});

test("signed token refusal, wrong nonce and inactive humans never receive sessions", async () => {
  for (const changes of [
    { iss: "https://evil.example/" },
    { aud: "txo-fabric-hairem" },
    { txo_fabric_user_uuid: "invalid" },
    { nonce: "wrong" },
    { exp: Math.floor(now / 1000) - 1 },
  ]) {
    const h = setup({ tokenOverrides: changes }), start = await h.begin();
    assert.equal((await h.complete(start)).status, 403);
    assert.equal(h.sessions.size, 0);
    assert.equal(h.records.size, 0);
  }
  const h = setup({ active: false }), start = await h.begin();
  assert.equal((await h.complete(start)).status, 403);
  assert.equal(h.sessions.size, 0);
});

test("callbacks are browser-bound, single-use across replicas and fail closed for forged origin/query/cookies", async () => {
  const h = setup(), start = await h.begin();
  assert.equal((await h.auth.handle(new Request(
    "https://other.example/auth/callback?state=" + start.state +
    "&code=code", { headers: { cookie: "__Host-txo-fabric-oidc=" + start.binding } },
  ))).status, 403);
  const bad = await h.auth.handle(new Request(origin + "/auth/callback?" +
    new URLSearchParams({ state: start.state, code: "code" }), {
    headers: { cookie: "__Host-txo-fabric-oidc=" + "A".repeat(43) },
  }));
  assert.equal(bad.status, 403);
  assert.equal((await h.complete(start)).status, 403);
  assert.equal(h.sessions.size, 0);
  const next = await h.begin();
  const duplicated = await h.auth.handle(new Request(origin + "/auth/callback?" +
    new URLSearchParams({ state: next.state, code: "code" }), {
    headers: { cookie: `__Host-txo-fabric-oidc=${next.binding}; __Host-txo-fabric-oidc=${next.binding}` },
  }));
  assert.equal(duplicated.status, 403);
  assert.equal(h.records.has("webui:" + next.state), true);
  const response = await h.complete(next);
  assert.equal(response.status, 303);
  assert.equal((await h.complete(next)).status, 403);
  for (const opts of [
    {state: next.state, code: "code", injected: "true"},
    {state: next.state, code: "code", iss: "https://attacker.example"},
  ]) {
    const r = await h.auth.handle(new Request(origin + "/auth/callback?" +
      new URLSearchParams(opts), { headers: { cookie: "__Host-txo-fabric-oidc=" + next.binding } }));
    assert.equal(r.status, 403);
  }
});

test("expired transaction denies even a correctly signed ID token", async () => {
  const h = setup(), start = await h.begin();
  h.setTime(now + 300_001);
  assert.equal((await h.complete(start)).status, 403);
  assert.equal(h.sessions.size, 0);
});

test("logout checks CSRF origin, revokes server-side and clears host-only cookie", async () => {
  const h = setup(), start = await h.begin();
  const response = await h.complete(start);
  const token = response.headers.getSetCookie().find(c => c.startsWith("__Host-txo-fabric-session="))
    .match(/__Host-txo-fabric-session=([^;]+)/)[1];
  const cookie = "__Host-txo-fabric-session=" + token;
  const req = (originHeader, customHeader) => new Request(origin + "/auth/logout", {
    method: "POST",
    headers: { cookie, origin: originHeader, "x-txo-request": customHeader },
  });
  assert.equal((await h.auth.handle(req("https://evil.example","auth"))).status, 403);
  assert.equal(h.sessions.size, 1);
  assert.equal((await h.auth.handle(req(origin,"fake"))).status, 403);
  assert.equal((await h.auth.handle(req(origin,"auth"))).status, 204);
  assert.equal(h.sessions.size, 0);
  assert.equal(await h.auth.authenticate(new Request(origin + "/api/me", { headers: { cookie } })), null);
});

test("composed global BFF authenticates once and rechecks all tenant memberships on each request", async () => {
  const h = setup();
  let allowed = true, checks = 0;
  const app = createGlobalWebUIApp({
    expectedOrigin: origin, authentikOrigin: idp,
    transactions: h.transactions, sessions: h.store,
    verifyActiveHuman: async proof => ({ ...proof, accountActive: true, fresh: true }),
    listTenantCatalog: async () => [{ tenantKey:"hairem", fabricTenantId:"TEN00001", displayName:"hAIrem" }],
    resolveTenantAccess: async ({ tenantKey, fabricTenantId }) => {
      checks++;
      return allowed ? {
        tenantKey, fabricTenantId, fabricUserId: "usr" + "b".repeat(32),
        accountActive: true, fresh: true, canEnter: true, canManageTenant: false,
      } : null;
    },
    getTenantChat: async ({ tenantKey, fabricTenantId }) => ({
      tenantKey, fabricTenantId,
      service: {
        canAccess: async () => true,
        list: async () => [], create: async () => { throw Error("no chat"); },
        get: async () => { throw Error("no chat"); },
        turn: async function* () { throw Error("no chat"); },
      },
      listTenantAgents: async () => [{ agentKey: "electra", label: "Electra" }],
    }),
    fetchImpl: async (url,opts) => {
      // Reuse the tested valid IdP network simulator.
      // This local test covers cookie lookup + mandatory per-tenant checks;
      // the full OAuth exchange is tested above against signed RSA JWTs.
      throw Error("No external calls in this session lookup test");
    },
    now: () => now,
  });
  const token = "A".repeat(42) + "1";
  h.sessions.set(token, {
    authenticated: true, verified: true, issuer,
    oidcSubject: sub, authentikUserUUID: userUUID,
  });
  const req = () => new Request(origin + "/api/tenants/hairem/chat/agents", {
    headers: { cookie: "__Host-txo-fabric-session=" + token },
  });
  assert.equal((await app(req())).status, 200);
  allowed = false;
  assert.equal((await app(req())).status, 403);
  assert.equal(checks, 2);
  assert.equal((await app(new Request(origin + "/api/me"))).status, 403);
});
