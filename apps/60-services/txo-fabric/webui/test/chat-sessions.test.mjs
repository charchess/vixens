import test from "node:test";
import assert from "node:assert/strict";
import { createChatSessionService, openHermesSession } from "../server/chat-sessions.mjs";
import { createChatHttpHandler } from "../server/chat-http.mjs";

const alice = { authenticated: true, subject: "alice", tenantKey: "hairem" };
const bob = { authenticated: true, subject: "bob", tenantKey: "hairem" };
const indiba = { authenticated: true, subject: "alice", tenantKey: "indiba" };
const targets = Object.fromEntries(["electra", "sabrina"].map(agentKey =>
  [agentKey, { agentKey, tenantKey: "hairem", baseURL: `http://${agentKey}.tenant-hairem.svc:8642`, apiKey: `secret-${agentKey}` }]));
const repo = () => {
  const records = new Map(), locked = new Set();
  return {
    insert: async r => { if (records.has(r.id)) throw Error("Duplicate ID"); records.set(r.id, r); },
    get: async id => records.get(id),
    list: async ({tenantKey,subject}) => [...records.values()].filter(r=>r.tenantKey===tenantKey&&r.subject===subject),
    claim: async id => { if (locked.has(id)) throw Error("Thread busy"); locked.add(id); return async()=>locked.delete(id); },
  };
};
const frames = body => new ReadableStream({start(c) {c.enqueue(new TextEncoder().encode(body)); c.close();}});
function harness() {
  const rpc = [], grants = new Set(["alice:electra", "alice:sabrina", "bob:electra"]);
  const service = createChatSessionService({tenantKey:"hairem",repo:repo(),
    authorize:async ({principal,tenantKey,agentKey})=>tenantKey==="hairem"&&grants.has(`${principal.subject}:${agentKey}`),
    resolveAgent:async ({agentKey})=>targets[agentKey],
    openRemoteSession:async ({agentKey})=>`remote-${agentKey}-${rpc.length+1}`,
    fetchImpl:async (url,opts)=>{rpc.push({url,opts}); return {ok:true,body:frames('event: assistant.delta\ndata: {"text":"Hello"}\n\nevent: run.completed\ndata: {}\n\n')};},
  });
  return {service,rpc,grants};
}
const collect=async stream=>{const out=[];for await(const e of stream)out.push(e);return out;};

test("per-user & per-agent opaque IDs, switching agents does not reuse threads",async()=>{
  const {service}=harness();
  const one=await service.create({principal:alice,agentKey:"electra"});
  const two=await service.create({principal:alice,agentKey:"sabrina"});
  assert.match(one.id,/^[a-f0-9]{40}$/); assert.notEqual(one.id,two.id);
  assert.deepEqual((await service.list({principal:alice})).map(x=>x.agentKey),["electra","sabrina"]);
  assert.deepEqual(await service.list({principal:bob}),[]);
  assert.deepEqual(await collect(service.turn({principal:alice,threadId:one.id,input:"hi"})),[{type:"text-delta",text:"Hello"},{type:"done",status:"completed"}]);
  assert.deepEqual(await collect(service.turn({principal:alice,threadId:two.id,input:"hi"})),[{type:"text-delta",text:"Hello"},{type:"done",status:"completed"}]);
  assert.equal((await service.get({principal:alice,threadId:one.id})).agentKey,"electra");
});

test("cross-user, cross-tenant, revoked grant and guessed thread refuse before Hermes",async()=>{
  const {service,rpc,grants}=harness();
  const thread=await service.create({principal:alice,agentKey:"electra"});
  for(const principal of [bob,indiba,{authenticated:false,subject:"alice",tenantKey:"hairem"}]) {
    await assert.rejects(collect(service.turn({principal,threadId:thread.id,input:"hi"})),/Forbidden/);
    await assert.rejects(service.get({principal,threadId:thread.id}),/Forbidden/);
  }
  await assert.rejects(collect(service.turn({principal:alice,threadId:"a".repeat(40),input:"hi"})),/Forbidden/);
  grants.delete("alice:electra");
  await assert.rejects(collect(service.turn({principal:alice,threadId:thread.id,input:"hi"})),/Forbidden/);
  assert.deepEqual(await service.list({principal:alice}),[]);
  assert.equal(rpc.length,0);
});

test("Hermes target cannot cross tenant; server-issued session IDs are validated",async()=>{
  const fetchImpl=async (url,opts)=>{
    assert.equal(url,"http://electra.tenant-hairem.svc:8642/api/sessions");
    assert.equal(opts.headers.Authorization,"Bearer secret-electra");
    return {ok:true,headers:new Headers(),text:async()=>'{"id":"real-session-42"}'};
  };
  assert.equal(await openHermesSession({target:targets.electra,tenantKey:"hairem",agentKey:"electra",fetchImpl}),"real-session-42");
  await assert.rejects(openHermesSession({target:targets.electra,tenantKey:"indiba",agentKey:"electra",fetchImpl}),/Agent is unavailable/);
  await assert.rejects(openHermesSession({target:targets.electra,tenantKey:"hairem",agentKey:"electra",fetchImpl:async()=>({ok:true,headers:new Headers(),text:async()=>'{"id":"..%2f"}'})}),/Agent is unavailable/);
});

test("HTTP rejects CSRF and cross-user access, streaming forwards sanitized events",async()=>{
  const {service}=harness();
  const origin="https://hairem.example.test";
  // Test identity adapter only. Production authenticate MUST verify Authentik/OIDC.
  const authenticate=async req=>req.headers.get("authorization")==="Bearer alice-session"?alice:bob;
  const handle=createChatHttpHandler({authenticate,service,expectedOrigin:origin});
  const req=(url,method="GET",body,subject="alice-session",extra={})=>new Request(origin+url,{method,...(body?{body:JSON.stringify(body)}:{}),headers:{Authorization:`Bearer ${subject}`,Origin:origin,"X-TXO-Request":"chat","Content-Type":"application/json",...extra}});
  assert.equal((await handle(req("/api/chat/threads","POST",{agentKey:"electra"},"alice-session",{Origin:"https://attacker.example"}))).status,403);
  const created=await handle(req("/api/chat/threads","POST",{agentKey:"electra"}));
  assert.equal(created.status,201);const thread=await created.json();
  assert.equal((await handle(req(`/api/chat/threads/${thread.id}`,"GET",null,"bob-session"))).status,403);
  assert.equal((await handle(req("/api/chat/threads"))).status,200);
  const response=await handle(req(`/api/chat/threads/${thread.id}/turns`,"POST",{input:"hi"}));
  assert.equal(response.status,200);
  const result=await response.text();
  assert.match(result,/"text-delta"/);assert.match(result,/"done"/);assert.doesNotMatch(result,/secret-electra/);
});

test("security dependencies are required, never optional",()=>{
  assert.throws(()=>createChatSessionService({tenantKey:"hairem"}),/security dependencies/);
  assert.throws(()=>createChatHttpHandler({authenticate:async()=>alice,expectedOrigin:"https://hairem.example.test"}),/security dependencies/);
});
