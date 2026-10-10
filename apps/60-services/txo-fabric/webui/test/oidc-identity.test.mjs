import test from "node:test";
import assert from "node:assert/strict";
import { generateKeyPairSync, sign } from "node:crypto";
import { createAuthentikIdentityVerifier } from "../server/oidc-identity.mjs";

const nowSeconds = 1_797_000_000;
const now = () => nowSeconds * 1000;
const tenantKey = "hairem";
const issuer = "https://authentik.truxonline.com/application/o/txo-fabric-hairem/";
const clientId = "txo-fabric-hairem";
const nonce = "server-generated-once-1234567890";
const uuid = "12345678-1234-4123-8123-1234567890ab";
const otherUUID = "87654321-4321-4321-8321-ba0987654321";
const { publicKey, privateKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
const jwk = { ...publicKey.export({ format: "jwk" }), kid: "fabric-key-1", alg: "RS256", use: "sig" };
const defaultClaims = {
  iss: issuer, sub: "opaque-hashed-sub-untied-to-the-uuid",
  aud: clientId, nonce, iat: nowSeconds - 10, exp: nowSeconds + 1800,
  txo_fabric_user_uuid: uuid,
};
const discovery = { issuer, jwks_uri: issuer + "jwks/" };

function jwt(claims = defaultClaims, header = { alg: "RS256", kid: jwk.kid, typ: "JWT" }) {
  const body = [header, claims].map((part) => Buffer.from(JSON.stringify(part)).toString("base64url")).join(".");
  return body + "." + sign("RSA-SHA256", Buffer.from(body), privateKey).toString("base64url");
}
function source({ document = discovery, keys = [jwk], status = 200, fail = false } = {}) {
  const calls = [];
  const fetchImpl = async (url, options) => {
    calls.push({ url, options });
    if (fail) throw Error("IdP unavailable");
    if (url === issuer + ".well-known/openid-configuration") {
      return new Response(JSON.stringify(document), { status });
    }
    if (url === issuer + "jwks/") return new Response(JSON.stringify({ keys }), { status });
    throw Error("Attempt to fetch unexpected origin: " + url);
  };
  return { calls, fetchImpl };
}
function verifier(f = source(), overrides = {}) {
  return createAuthentikIdentityVerifier({
    tenantKey, issuer, clientId, now, fetchImpl: f.fetchImpl, ...overrides,
  });
}
const request = (claims = defaultClaims) => ({ idToken: jwt(claims), expectedNonce: nonce });
async function denied(promise) {
  await assert.rejects(promise, /Untrusted Fabric OIDC identity proof/);
}

test("accepts only signed OIDC subject plus separately signed immutable Authentik UUID", async () => {
  const f = source();
  const proof = await verifier(f)(request());
  assert.deepEqual(proof, {
    tenantKey: "hairem", issuer,
    oidcSubject: defaultClaims.sub, authentikUserUUID: uuid,
  });
  assert.equal(Object.isFrozen(proof), true);
  assert.equal(proof.verified, undefined);
  assert.equal(proof.fabricUserId, undefined);
  assert.deepEqual(f.calls.map((call) => call.url), [
    issuer + ".well-known/openid-configuration", issuer + "jwks/",
  ]);
  for (const call of f.calls) {
    assert.equal(call.options.redirect, "error");
    assert.equal(call.options.cache, "no-store");
  }
});

test("two tenants have distinct issuer and audience trust boundaries", async () => {
  const other = verifier(source(), {
    tenantKey: "indiba",
    issuer: "https://authentik.truxonline.com/application/o/txo-fabric-indiba/",
    clientId: "txo-fabric-indiba",
  });
  await denied(other(request()));
  const f = source();
  await denied(verifier(f)(request({ ...defaultClaims, aud: "txo-fabric-indiba" })));
  await denied(verifier(f)(request({ ...defaultClaims, iss: "https://evil.example/o/" })));
  assert.equal(f.calls.length, 0, "no network request for untrusted claim scope");
});

test("rejects forged identity, malformed signatures and token header injection", async () => {
  const f = source();
  const valid = jwt();
  const segments = valid.split(".");
  const changed = Buffer.from(JSON.stringify({
    ...defaultClaims, txo_fabric_user_uuid: otherUUID,
  })).toString("base64url");
  for (const idToken of [
    segments[0] + "." + changed + "." + segments[2],
    jwt(defaultClaims, { alg: "none", kid: jwk.kid }),
    jwt(defaultClaims, { alg: "HS256", kid: jwk.kid }),
    jwt(defaultClaims, { alg: "RS256", kid: jwk.kid, jku: "https://evil.example/keys" }),
    jwt(defaultClaims, { alg: "RS256", kid: jwk.kid, jwk }),
    "not.a.jwt.extra",
    valid + ".extra",
    "a".repeat(16385),
  ]) await denied(verifier(f)({ idToken, expectedNonce: nonce }));
});

test("rejects missing and ill-formed UUID; never assumes hashed sub is UUID", async () => {
  const f = source();
  for (const claim of [
    { ...defaultClaims, txo_fabric_user_uuid: undefined },
    { ...defaultClaims, txo_fabric_user_uuid: defaultClaims.sub },
    { ...defaultClaims, txo_fabric_user_uuid: otherUUID.toUpperCase() },
  ]) await denied(verifier(f)(request(claim)));
  assert.equal(f.calls.length, 0);
});

test("requires server-owned nonce and bounded fresh expiry at enrollment", async () => {
  const f = source();
  for (const arg of [
    { idToken: jwt(), expectedNonce: "browser-nonce-not-the-server-nonce" },
    { idToken: jwt(), expectedNonce: "" },
    request({ ...defaultClaims, nonce: undefined }),
    request({ ...defaultClaims, exp: nowSeconds }),
    request({ ...defaultClaims, iat: nowSeconds - 301 }),
    request({ ...defaultClaims, iat: nowSeconds + 31 }),
    request({ ...defaultClaims, nbf: nowSeconds + 31 }),
    request({ ...defaultClaims, exp: "not-a-number" }),
    request({ ...defaultClaims, iat: nowSeconds + 4, exp: nowSeconds + 3 }),
  ]) await denied(verifier(f)(arg));
  assert.equal(f.calls.length, 0);
});

test("discovery and keys are strictly from the configured tenant issuer; no redirects", async () => {
  for (const document of [
    { ...discovery, issuer: "https://evil.example/o/" },
    { ...discovery, jwks_uri: "http://authentik.truxonline.com/keys" },
    { ...discovery, jwks_uri: "https://evil.example/jwks/" },
    { ...discovery, jwks_uri: issuer + "../other/jwks/" },
    { ...discovery, jwks_uri: issuer + "jwks/?tenant=indiba" },
  ]) {
    const f = source({ document });
    await denied(verifier(f)(request()));
    assert.equal(f.calls.length, 1);
  }
});

test("rejects key substitution, duplicate kid, invalid key usage and empty JWKS", async () => {
  for (const keys of [
    [], [{ ...jwk, kid: "different" }], [jwk, jwk],
    [{ ...jwk, kty: "oct" }], [{ ...jwk, alg: "HS256" }],
    [{ ...jwk, use: "enc" }], [{ ...jwk, d: "secret" }],
    [{ ...jwk, key_ops: ["sign"] }],
  ]) await denied(verifier(source({ keys }))(request()));
});

test("fails closed on issuer outage, malformed response or fetch HTTP failure", async () => {
  await denied(verifier(source({ fail: true }))(request()));
  await denied(verifier(source({ status: 503 }))(request()));
  const f = source();
  const broken = verifier(f, { fetchImpl: async () => new Response("{bad", { status: 200 }) });
  await denied(broken(request()));
});

test("rejects invalid backend configuration before contacting any identity provider", () => {
  for (const value of [
    { issuer: "http://authentik.truxonline.com/application/o/txo-fabric-hairem/" },
    { issuer: issuer + "?param=1" },
    { issuer: "https://authentik.truxonline.com/application/o/txo-fabric-indiba/" },
    { issuer: "https://other.example/application/o/txo-fabric-hairem/#fragment" },
    { clientId: "txo-fabric-indiba" },
    { tenantKey: "HAIREM" },
    { fetchImpl: null },
  ]) assert.throws(() => verifier(source(), value), /Untrusted Fabric OIDC identity proof/);
});
