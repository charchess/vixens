import test from "node:test";
import assert from "node:assert/strict";
import { listAvailableAgents, createFabricThread, streamFabricTurn } from "../src/api.mjs";

const id = "a".repeat(40);
const collect = async iter => { const out=[]; for await (const event of iter) out.push(event); return out; };

test("available agents come from Fabric API", async () => {
  const result = await listAvailableAgents(async () =>
    new Response(JSON.stringify({ agents: [{ agentKey: "electra", label: "Electra" }] }), {status:200}));
  assert.deepEqual(result, [{ agentKey: "electra", label: "Electra" }]);
});

test("thread creation carries only the agent identity", async () => {
  const fetchImpl = async (url, opts) => {
    assert.equal(url, "/api/chat/threads");
    assert.equal(opts.method,"POST");
    assert.deepEqual(JSON.parse(opts.body), { agentKey: "electra" });
    return new Response(JSON.stringify({id, agentKey:"electra"}), {status:201});
  };
  assert.equal(await createFabricThread("electra", { fetchImpl }),id);
  await assert.rejects(createFabricThread("../other", { fetchImpl }), /Agent invalide/);
});

test("browser consumes the projected stream, including completion", async () => {
  const payload = [
    'data: {"type":"text-delta","text":"Hi"}',
    'data: {"type":"done","status":"completed"}'
  ].join("\n\n")+"\n\n";
  const fetchImpl = async () => new Response(new ReadableStream({start(c) {
    c.enqueue(new TextEncoder().encode(payload.slice(0,23)));
    c.enqueue(new TextEncoder().encode(payload.slice(23)));
    c.close();
  }}), {status:200});
  assert.deepEqual(await collect(streamFabricTurn(id,"hello",{fetchImpl})),[
    { type:"text-delta", text:"Hi" },{ type:"done",status:"completed" }
  ]);
});
