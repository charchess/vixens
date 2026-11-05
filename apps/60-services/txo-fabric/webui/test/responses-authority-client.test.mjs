import assert from "node:assert/strict";
import test from "node:test";
import { createHmac } from "node:crypto";
import {
  createResponsesAuthorityClient, RESPONSES_AUTHORITY_PATH,
  RESPONSES_AUTHORITY_URL,
} from "../server/responses-authority-client.mjs";

const uid = "6d0a9f4e-8024-4d39-97a5-cc1560f98dc7";
const agentUID = "9d0a9f4e-8024-4d39-97a5-cc1560f98dc7";
const tenant = { tenantKey: "hairem", fabricTenantId: "TEN00001" };
const agent = { ...tenant, agentKey: "electra" };
const user = "usr0123456789abcdef0123456789abcdef";
const json = value => new Response(JSON.stringify(value), { headers: { "content-type": "application/json" } });

function enabledClient(fetchImpl, extras = {}) {
  return createResponsesAuthorityClient({
    enabled: true, fetchImpl, now: () => 1760212800000,
    randomBytes: () => Buffer.alloc(32, 7),
    readFile: async path => Buffer.from(path.endsWith("hmac-key") ? "k".repeat(32) : "a.b.c"),
    ...extras,
  });
}

test("Responses authority gate-off performs no I/O", async () => {
  let reads = 0, random = 0, requests = 0;
  const client = createResponsesAuthorityClient({
    readFile: async () => { reads++; }, randomBytes: () => { random++; }, fetchImpl: async () => { requests++; },
  });
  await assert.rejects(client.resolveTenant(tenant));
  assert.deepEqual({ reads, random, requests }, { reads: 0, random: 0, requests: 0 });
});

test("Responses authority signs an exact tenant resolution scope", async () => {
  let captured;
  const client = enabledClient(async (url, options) => {
    captured = { url, options };
    return json({ tenant: { ...tenant, tenantBundleUID: uid, status: "Ready" } });
  });
  const result = await client.resolveTenant(tenant);
  assert.equal(captured.url, RESPONSES_AUTHORITY_URL);
  assert.equal(captured.options.redirect, "error");
  assert.equal(captured.options.cache, "no-store");
  assert.deepEqual(JSON.parse(captured.options.body), { operation: "resolve-tenant", ...tenant });
  const [payload, signature] = captured.options.headers["X-Fabric-Assertion"].split(".");
  assert.equal(signature, createHmac("sha256", "k".repeat(32)).update(payload).digest("base64url"));
  assert.deepEqual(JSON.parse(Buffer.from(payload, "base64url")), {
    method: "POST", path: RESPONSES_AUTHORITY_PATH, operation: "resolve-tenant", ...tenant,
    serviceAccount: "system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff",
    audience: "txo-fabric-private-identity", iat: 1760212800, exp: 1760212820,
    jti: Buffer.alloc(32, 7).toString("base64url"),
  });
  assert.deepEqual(result, { tenant: { ...tenant, tenantBundleUID: uid, status: "Ready" } });
});

test("Responses authority validates each operation's exact returned shape", async () => {
  const client = enabledClient(async (_url, options) => {
    const request = JSON.parse(options.body);
    if (request.operation === "resolve-agent") return json({ agent: { ...agent, agentIdentityUID: agentUID, agentGeneration: 4, status: "Ready" } });
    if (request.operation === "authorize-chat") return json({ authorization: { allowed: true } });
    return json({ runtime: { attested: true } });
  });
  assert.equal((await client.resolveAgent(agent)).agent.agentIdentityUID, agentUID);
  assert.equal((await client.authorizeChat({ ...agent, fabricUserId: user })).authorization.allowed, true);
  assert.equal((await client.verifyRuntime({ ...agent, tenantBundleUID: uid, tenantGeneration: 3, agentIdentityUID: agentUID, agentGeneration: 4 })).runtime.attested, true);
});

test("Responses authority rejects input injection and malformed or substituted replies", async () => {
  let requests = 0;
  const client = enabledClient(async () => { requests++; return json({ tenant: { ...tenant, tenantBundleUID: uid, status: "Ready" } }); });
  await assert.rejects(client.resolveTenant({ ...tenant, endpoint: "http://evil.invalid" }));
  assert.equal(requests, 0);
  for (const response of [
    { tenant: { ...tenant, tenantBundleUID: uid, status: "Ready" }, extra: true },
    { tenant: { ...tenant, fabricTenantId: "TEN99999", tenantBundleUID: uid, status: "Ready" } },
    { tenant: { ...tenant, tenantBundleUID: uid, status: "Stale" } },
    { agent: { ...agent, agentIdentityUID: agentUID, agentGeneration: 4, status: "Ready" } },
  ]) {
    const invalid = enabledClient(async () => json(response));
    await assert.rejects(invalid.resolveTenant(tenant));
  }
});

test("Responses authority rejects an endpoint override and unsafe transport result", async () => {
  assert.throws(() => createResponsesAuthorityClient({ endpoint: "http://evil.invalid" }));
  const redirected = enabledClient(async () => ({ status: 200, redirected: true, headers: new Headers({ "content-type": "application/json" }), text: async () => "{}" }));
  await assert.rejects(redirected.resolveTenant(tenant));
});
