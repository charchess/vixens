import test from "node:test";
import assert from "node:assert/strict";
import { createChatSessionService } from "../server/chat-sessions.mjs";
import { createChatHttpHandler } from "../server/chat-http.mjs";

test("agent discovery enforces tenant and user grants before returning labels", async () => {
  const principal = { authenticated: true, subject: "alice", tenantKey: "hairem" };
  const repo = { insert: async()=>{}, get: async()=>null, list: async()=>[], claim: async()=>async()=>{} };
  const service = createChatSessionService({ tenantKey:"hairem",repo,
    authorize: async ({principal,agentKey}) => principal.subject === "alice" && agentKey === "electra",
    resolveAgent: async()=>null,
  });
  const handler = createChatHttpHandler({
    authenticate: async()=>principal, service, expectedOrigin:"https://hairem.example.test",
    listTenantAgents: async()=>[{agentKey:"electra",label:"Electra"},{agentKey:"private",label:"Private"}],
  });
  const res=await handler(new Request("https://hairem.example.test/api/chat/agents"));
  assert.equal(res.status,200);
  assert.deepEqual(await res.json(),{agents:[{agentKey:"electra",label:"Electra"}]});
  const unavailable = createChatHttpHandler({authenticate:async()=>principal,service,expectedOrigin:"https://hairem.example.test"});
  const blocked=await unavailable(new Request("https://hairem.example.test/api/chat/agents"));
  assert.equal(blocked.status,503);
});
