import test from "node:test";
import assert from "node:assert/strict";
import { createGlobalWebUIHandler } from "../server/global-webui-http.mjs";

const origin = "https://webui.truxonline.com";
const issuer = "https://authentik.truxonline.com/application/o/txo-fabric-webui/";
const uuid = "c38f3cc7-5b33-4266-bd85-40ce9a100061";
const user = "usr" + "e".repeat(32);
const tenants = [
  { tenantKey: "hairem", fabricTenantId: "TEN00001", displayName: "hAIrem" },
  { tenantKey: "indiba", fabricTenantId: "TEN00002", displayName: "Indiba" },
  { tenantKey: "other", fabricTenantId: "TEN00003", displayName: "Other" },
];
const identity = Object.freeze({
  authenticated: true, verified: true, issuer,
  oidcSubject: "opaque-global-subject", authentikUserUUID: uuid,
});
function harness({ account = identity, catalog = tenants, platformAdmin = false } = {}) {
  const audit = [], allowed = new Set(["hairem", "indiba"]);
  let fresh = true, isActive = true;
  const serviceFor = (tenantKey) => ({
    canAccess: async ({ principal, agentKey }) =>
      principal.tenantKey === tenantKey && principal.fabricUserId === user &&
      agentKey === "alex" && allowed.has(tenantKey),
    create: async ({ principal, agentKey }) => {
      audit.push({ action: "create", tenantKey, principal });
      if (principal.tenantKey !== tenantKey || agentKey !== "alex") throw Error("Forbidden");
      return { id: "a".repeat(40), agentKey };
    },
    get: async () => { throw Error("Forbidden"); },
    list: async () => [],
    turn: async function* () { throw Error("Forbidden"); },
  });
  const handler = createGlobalWebUIHandler({
    expectedOrigin: origin, expectedIssuer: issuer,
    authenticate: async () => account,
    listTenantCatalog: async () => catalog,
    resolveTenantAccess: async ({ identity: proof, tenantKey, fabricTenantId }) => {
      audit.push({ action: "membership", tenantKey, fabricTenantId, identity: proof });
      if (!allowed.has(tenantKey)) return null;
      return {
        tenantKey, fabricTenantId, fabricUserId: user,
        accountActive: isActive, fresh, canEnter: true,
        canManageTenant: tenantKey === "hairem",
      };
    },
    resolvePlatformAccess: async () => ({
      fresh, accountActive: isActive, canAdminister: platformAdmin,
    }),
    getTenantChat: async ({ tenantKey, fabricTenantId }) => ({
      tenantKey, fabricTenantId, service: serviceFor(tenantKey),
      listTenantAgents: async ({ principal }) => {
        audit.push({ action: "discovery", tenantKey, principal });
        return [{ agentKey: "alex", label: tenantKey + " Alex" }];
      },
    }),
  });
  const request = (path, init) => new Request(origin + path, init);
  return { handler, request, audit, allowed, setFresh: v => { fresh = v; },
    setActive: v => { isActive = v; } };
}

test("one global origin lists authorized tenants, not every TenantBundle, without leaking roles", async () => {
  const { handler, request } = harness();
  const r = await handler(request("/api/me"));
  assert.equal(r.status, 200);
  assert.equal(r.headers.get("Cache-Control"), "no-store");
  assert.equal(r.headers.get("Vary"), "Cookie");
  assert.deepEqual(await r.json(), {
    tenants: [
      { tenantKey: "hairem", displayName: "hAIrem",
        capabilities: { chat: true, tenantAdmin: true } },
      { tenantKey: "indiba", displayName: "Indiba",
        capabilities: { chat: true, tenantAdmin: false } },
    ],
    capabilities: { platformAdmin: false },
  });
});

test("global platform admin requires its own independent backend grant", async () => {
  const { handler, request } = harness({ platformAdmin: true });
  const r = await handler(request("/api/me"));
  assert.equal((await r.json()).capabilities.platformAdmin, true);
});

test("same agent key routes to the selected authorized tenant, never guessed internal endpoints", async () => {
  const { handler, request, audit } = harness();
  for (const tenantKey of ["hairem", "indiba"]) {
    const agents = await handler(request(`/api/tenants/${tenantKey}/chat/agents`));
    assert.equal(agents.status, 200);
    assert.deepEqual(await agents.json(), {
      agents: [{ agentKey: "alex", label: tenantKey + " Alex" }],
    });
    const created = await handler(request(`/api/tenants/${tenantKey}/chat/threads`, {
      method: "POST",
      headers: { origin, "Content-Type": "application/json", "X-TXO-Request": "chat" },
      body: JSON.stringify({ agentKey: "alex" }),
    }));
    assert.equal(created.status, 201);
    assert.equal((await created.json()).agentKey, "alex");
  }
  const routed = audit.filter(x => x.action === "create");
  assert.deepEqual(routed.map(x => x.tenantKey), ["hairem", "indiba"]);
  assert.deepEqual(routed.map(x => x.principal.fabricTenantId), ["TEN00001", "TEN00002"]);
  assert.ok(routed.every(x => x.principal.subject === user && x.principal.verified === true));
  assert.ok(routed.every(x => !("issuer" in x.principal)));
});

test("every request rechecks tenant membership, account deactivation and stale IAM state", async () => {
  const { handler, request, audit, allowed, setFresh, setActive } = harness();
  assert.equal((await handler(request("/api/tenants/hairem/chat/agents"))).status, 200);
  allowed.delete("hairem");
  assert.equal((await handler(request("/api/tenants/hairem/chat/agents"))).status, 403);
  allowed.add("hairem");
  setFresh(false);
  assert.equal((await handler(request("/api/tenants/hairem/chat/agents"))).status, 403);
  setFresh(true);
  setActive(false);
  assert.equal((await handler(request("/api/tenants/hairem/chat/agents"))).status, 403);
  assert.equal(audit.filter(x => x.action === "discovery").length, 1);
});

test("cross-tenant, absent and forged catalog keys never route to a different tenant", async () => {
  const { handler, request, audit } = harness();
  for (const path of [
    "/api/tenants/other/chat/agents",
    "/api/tenants/unknown/chat/agents",
    "/api/tenants/HAIREM/chat/agents",
    "/api/tenants/../chat/agents",
    "/api/tenants/hairem/../../admin",
    "/api/tenants/hairem/secrets",
  ]) {
    const response = await handler(request(path));
    assert.ok([403, 404].includes(response.status));
  }
  assert.equal(audit.filter(x => x.action === "discovery").length, 0);
});

test("global Authentik issuer, UUID and signed subject must match the approved GLOBAL client", async () => {
  for (const account of [
    null, { ...identity, verified: false },
    { ...identity, authenticated: false },
    { ...identity, issuer: "https://authentik.truxonline.com/application/o/txo-fabric-hairem/" },
    { ...identity, authentikUserUUID: "not-a-uuid" },
    { ...identity, oidcSubject: "\nforged" },
  ]) {
    const { handler, request } = harness({ account });
    assert.equal((await handler(request("/api/me"))).status, 403);
  }
  assert.throws(() => createGlobalWebUIHandler({}), /security configuration/);
});

test("wrong public origin and CSRF-rejected POST never create threads", async () => {
  const { handler, request, audit } = harness();
  assert.equal((await handler(new Request("https://hairem.truxonline.com/api/me"))).status, 403);
  for (const extra of [
    { origin: "https://evil.example", "Content-Type": "application/json", "X-TXO-Request": "chat" },
    { origin, "Content-Type": "text/plain", "X-TXO-Request": "chat" },
    { origin, "Content-Type": "application/json" },
  ]) {
    const response = await handler(request("/api/tenants/hairem/chat/threads", {
      method: "POST", headers: extra, body: JSON.stringify({ agentKey: "alex" }),
    }));
    assert.equal(response.status, 403);
  }
  assert.equal(audit.filter(x => x.action === "create").length, 0);
});

test("malformed duplicate catalog and access resolver outages fail closed", async () => {
  for (const catalog of [
    [{ ...tenants[0] }, { ...tenants[0] }],
    [{ ...tenants[0] }, { ...tenants[1], fabricTenantId: "TEN00001" }],
    [{ ...tenants[0], displayName: "Admin\nInjected" }],
    new Array(501).fill(tenants[0]),
  ]) {
    const { handler, request } = harness({ catalog });
    assert.equal((await handler(request("/api/me"))).status, 503);
    assert.equal((await handler(request("/api/tenants/hairem/chat/agents"))).status, 503);
  }
  const handler = createGlobalWebUIHandler({
    expectedOrigin: origin, expectedIssuer: issuer,
    authenticate: async () => identity,
    listTenantCatalog: async () => tenants,
    resolveTenantAccess: async () => { throw Error("upstream secret"); },
    getTenantChat: async () => { throw Error("must not reach router"); },
  });
  assert.equal((await handler(new Request(origin + "/api/me"))).status, 503);
  assert.equal((await handler(new Request(origin + "/api/tenants/hairem/chat/agents"))).status, 503);
});

test("settings, admin and identity writes are NOT routed through tenant chat access", async () => {
  const { handler, request, audit } = harness({ platformAdmin: true });
  for (const path of [
    "/api/admin", "/api/tenants/hairem/admin", "/api/tenants/hairem/settings",
    "/api/tenants/hairem/chat/../../credentials",
    "/api/tenants/hairem/chat/threads/" + "f".repeat(40) + "/export",
  ]) assert.equal((await handler(request(path))).status, 404);
  assert.equal(audit.filter(x => x.action === "create").length, 0);
});
