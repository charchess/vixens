import test from "node:test";
import assert from "node:assert/strict";
import {
  listAccessibleTenants, listAvailableAgents, createFabricThread, streamFabricTurn,
} from "../src/api.mjs";

const id = "a".repeat(40);
const collect = async iter => { const out=[]; for await (const event of iter) out.push(event); return out; };

test("global WebUI fetches canonical authorized tenants from one same-origin session", async () => {
  const fetchImpl = async (url, options) => {
    assert.equal(url, "/api/me");
    assert.equal(options.credentials, "same-origin");
    return new Response(JSON.stringify({
      tenants: [
        { tenantKey: "hairem", displayName: "hAIrem",
          capabilities: { chat: true, tenantAdmin: true } },
        { tenantKey: "indiba", displayName: "Indiba",
          capabilities: { chat: true, tenantAdmin: false } },
      ],
      capabilities: { platformAdmin: false },
    }), { status: 200 });
  };
  assert.deepEqual(await listAccessibleTenants(fetchImpl), {
    tenants: [
      { tenantKey: "hairem", displayName: "hAIrem",
        capabilities: { chat: true, tenantAdmin: true } },
      { tenantKey: "indiba", displayName: "Indiba",
        capabilities: { chat: true, tenantAdmin: false } },
    ],
    platformAdmin: false,
  });
});

test("duplicate or malformed tenant contexts are not rendered as accessible", async () => {
  for (const tenants of [
    [{ tenantKey: "hairem", displayName: "hAIrem",
      capabilities: { chat: true, tenantAdmin: false } },
    { tenantKey: "hairem", displayName: "clone",
      capabilities: { chat: true, tenantAdmin: false } }],
    [{ tenantKey: "../hairem", displayName: "forged",
      capabilities: { chat: true, tenantAdmin: true } }],
    [{ tenantKey: "hairem", displayName: "forged", capabilities: {} }],
  ]) {
    await assert.rejects(() => listAccessibleTenants(async () =>
      new Response(JSON.stringify({ tenants }), { status: 200 })), /Espaces indisponibles/);
  }
});

test("available agent list is scoped to the selected tenant, not a global agent name", async () => {
  const fetchImpl = async (url) => {
    assert.equal(url, "/api/tenants/hairem/chat/agents");
    return new Response(JSON.stringify({ agents: [{ agentKey: "electra", label: "Electra" }] }), {status:200});
  };
  assert.deepEqual(await listAvailableAgents("hairem", fetchImpl),
    [{ agentKey: "electra", label: "Electra" }]);
  await assert.rejects(listAvailableAgents("../indiba", fetchImpl), /Tenant invalide/);
});

test("thread creation carries only tenant scope in path and agent key in JSON", async () => {
  const fetchImpl = async (url, opts) => {
    assert.equal(url, "/api/tenants/indiba/chat/threads");
    assert.equal(opts.method, "POST");
    assert.equal(opts.credentials, "same-origin");
    assert.deepEqual(JSON.parse(opts.body), { agentKey: "alex" });
    return new Response(JSON.stringify({id, agentKey:"alex"}), {status:201});
  };
  assert.equal(await createFabricThread("indiba", "alex", { fetchImpl }),id);
  await assert.rejects(createFabricThread("indiba", "../other", { fetchImpl }), /Agent invalide/);
  await assert.rejects(createFabricThread("other/../../hairem", "alex", { fetchImpl }), /Tenant invalide/);
});

test("browser consumes projected SSE in the requested tenant scope", async () => {
  const payload = [
    'data: {"type":"text-delta","text":"Hi"}',
    'data: {"type":"done","status":"completed"}'
  ].join("\n\n")+"\n\n";
  const fetchImpl = async (url) => {
    assert.equal(url, "/api/tenants/hairem/chat/threads/" + id + "/turns");
    return new Response(new ReadableStream({start(c) {
      c.enqueue(new TextEncoder().encode(payload.slice(0,23)));
      c.enqueue(new TextEncoder().encode(payload.slice(23)));
      c.close();
    }}), {status:200});
  };
  assert.deepEqual(await collect(streamFabricTurn("hairem", id,"hello",{fetchImpl})),[
    { type:"text-delta", text:"Hi" },{ type:"done",status:"completed" }
  ]);
});
