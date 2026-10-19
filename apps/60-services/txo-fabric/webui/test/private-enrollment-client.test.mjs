import test from "node:test";
import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import {
  createPrivateEnrollmentClient,
  PRIVATE_ENROLLMENT_PATH,
  PRIVATE_ENROLLMENT_URL,
  PRIVATE_ENROLLMENT_AUDIENCE,
  PRIVATE_BFF_SUBJECT,
  GLOBAL_OIDC_ISSUER,
} from "../server/private-enrollment-client.mjs";

const uuid = "550e8400-e29b-41d4-a716-446655440001";
const now = 1_760_214_000_000;
const key = Buffer.alloc(32, 91);
const token = Buffer.from("eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJ0ZXN0In0.c2ln");
const identity = Object.freeze({
  tenantKey: "hairem",
  issuer: GLOBAL_OIDC_ISSUER,
  oidcSubject: "opaque-global-sub",
  authentikUserUUID: uuid,
});
const expected = Object.freeze({
  tenantKey: "hairem",
  fabricUserId: "usraaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
});

function fixture({ enabled = true, response } = {}) {
  const calls = [];
  const reads = [];
  const client = createPrivateEnrollmentClient({
    enabled,
    now: () => now,
    readFile: async path => {
      reads.push(path);
      return path.endsWith("hmac-key") ? key : token;
    },
    randomBytes: () => Buffer.alloc(32, 8),
    fetchImpl: async (url, options) => {
      calls.push({ url, options });
      return response || new Response(JSON.stringify(expected), {
        status: 200, headers: { "Content-Type": "application/json" },
      });
    },
  });
  return { client, calls, reads };
}

test("feature-off enrollment performs no credential read, entropy, or network I/O", async () => {
  const f = fixture({ enabled: false });
  await assert.rejects(f.client.enroll(identity), /unavailable/);
  assert.deepEqual(f.calls, []);
  assert.deepEqual(f.reads, []);
});

test("enrollment signs the exact operator one-shot assertion contract", async () => {
  const f = fixture();
  assert.deepEqual(await f.client.enroll(identity), expected);
  assert.equal(f.calls.length, 1);
  assert.equal(f.reads.length, 2);
  const { url, options } = f.calls[0];
  assert.equal(url, PRIVATE_ENROLLMENT_URL);
  assert.equal(options.method, "POST");
  assert.equal(options.redirect, "error");
  assert.equal(options.cache, "no-store");
  assert.equal(options.headers["Content-Type"], "application/json");
  const request = JSON.parse(options.body);
  assert.deepEqual(request, {
    tenantKey: identity.tenantKey,
    oidcIssuer: identity.issuer,
    oidcSubject: identity.oidcSubject,
    authentikUserUUID: identity.authentikUserUUID,
  });
  const [payload, mac] = options.headers["X-Fabric-Assertion"].split(".");
  assert.equal(createHmac("sha256", key).update(payload).digest("base64url"), mac);
  assert.deepEqual(JSON.parse(Buffer.from(payload, "base64url").toString("utf8")), {
    method: "POST", path: PRIVATE_ENROLLMENT_PATH,
    tenantKey: identity.tenantKey,
    oidcIssuer: identity.issuer,
    oidcSubject: identity.oidcSubject,
    authentikUserUUID: identity.authentikUserUUID,
    serviceAccount: PRIVATE_BFF_SUBJECT,
    audience: PRIVATE_ENROLLMENT_AUDIENCE,
    iat: Math.floor(now / 1000), exp: Math.floor(now / 1000) + 20,
    jti: Buffer.alloc(32, 8).toString("base64url"),
  });
});

test("enrollment rejects malformed identity, endpoint override, and malformed operator replies before granting", async () => {
  const f = fixture();
  for (const invalid of [
    { ...identity, tenantKey: "../hairem" },
    { ...identity, issuer: "https://evil.example/application/o/txo-fabric-webui/" },
    { ...identity, oidcSubject: "" },
    { ...identity, authentikUserUUID: "not-a-uuid" },
    { ...identity, fabricUserId: expected.fabricUserId },
  ]) await assert.rejects(f.client.enroll(invalid), /unavailable/);
  assert.deepEqual(f.calls, []);
  assert.throws(() => createPrivateEnrollmentClient({ enabled: true, endpoint: PRIVATE_ENROLLMENT_URL + "?forged=1" }), /unavailable/);
  const bad = fixture({ response: new Response(JSON.stringify({ ...expected, storeID: "browser-visible" }), {
    status: 200, headers: { "Content-Type": "application/json" },
  }) });
  await assert.rejects(bad.client.enroll(identity), /unavailable/);
});
