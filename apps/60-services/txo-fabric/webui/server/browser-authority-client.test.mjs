import test from "node:test";
import assert from "node:assert/strict";
import { createBrowserAuthorityClient, validateBrowserAuthorityResponse } from "./browser-authority-client.mjs";

const uuid = "550e8400-e29b-41d4-a716-446655440000";
const user = "usr0123456789abcdef0123456789abcdef";
const tenant = { tenantKey: "hairem", fabricTenantId: "TEN00001", displayName: "hAIrem" };

function response(body) {
  return { status: 200, redirected: false, headers: { get: () => "application/json" }, text: async () => JSON.stringify(body) };
}

test("browser authority response shapes expose only approved records", () => {
  assert.deepEqual(validateBrowserAuthorityResponse("catalog", { tenants: [tenant] }), { tenants: [tenant] });
  assert.deepEqual(validateBrowserAuthorityResponse("tenant", { tenant: { ...tenant, fabricUserId: user, accountActive: true, fresh: true, canEnter: true } }).tenant.fabricUserId, user);
  assert.deepEqual(validateBrowserAuthorityResponse("agent", { agent: { agentKey: "sam", displayName: "Sam", agentUID: "uid-123", generation: 7 } }).agent.agentKey, "sam");
  for (const bad of [
    ["catalog", { tenants: [{ ...tenant, endpoint: "http://x" }] }],
    ["tenant", { tenant: { ...tenant, fabricUserId: user, accountActive: true, fresh: true, canEnter: true, groups: [] } }],
    ["agent", { agent: { agentKey: "sam", displayName: "Sam", agentUID: "uid-123", generation: 7, namespace: "tenant-hairem" } }],
  ]) assert.throws(() => validateBrowserAuthorityResponse(...bad));
});

test("gate-off has zero I/O and rejects injection", async () => {
  let reads = 0, random = 0, requests = 0;
  const client = createBrowserAuthorityClient({ readFile: async () => { reads++; }, randomBytes: () => { random++; }, fetchImpl: async () => { requests++; } });
  await assert.rejects(client.catalog({ authentikUserUUID: uuid }));
  await assert.rejects(client.tenant({ authentikUserUUID: uuid, tenantKey: "hairem", endpoint: "http://evil" }));
  assert.equal(reads, 0); assert.equal(random, 0); assert.equal(requests, 0);
});

test("enabled client signs exact server route and strict response", async () => {
  let request;
  const client = createBrowserAuthorityClient({ enabled: true, now: () => 1760212800000, randomBytes: () => Buffer.alloc(32, 3), readFile: async (path) => Buffer.from(path.endsWith("hmac-key") ? "k".repeat(32) : "a.b.c"), fetchImpl: async (url, options) => { request = { url, options }; return response({ agent: { agentKey: "sam", displayName: "Sam", agentUID: "uid-123", generation: 7 } }); } });
  assert.deepEqual(await client.agent({ authentikUserUUID: uuid, tenantKey: "hairem", agentKey: "sam" }), { agent: { agentKey: "sam", displayName: "Sam", agentUID: "uid-123", generation: 7 } });
  assert.match(request.url, /\/internal\/v1\/browser-authority\/resolve$/);
  assert.deepEqual(JSON.parse(request.options.body), { kind: "agent", authentikUserUUID: uuid, tenantKey: "hairem", agentKey: "sam" });
  assert.equal(request.options.headers.Authorization, "Bearer a.b.c");
});
