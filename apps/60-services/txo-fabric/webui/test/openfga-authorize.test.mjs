import test from "node:test";
import assert from "node:assert/strict";
import { createOpenFGAAuthorizer, openFGAPrivateOrigin } from "../server/openfga-authorize.mjs";

const storeId = "01H0H015178Y2V4CX10C2KGHF4";
const modelId = "01H0H015178Y2V4CX10C2KGHF5";
const alice = {
  authenticated: true, verified: true, subject: "authentik-subject-123",
  tenantKey: "indiba", fabricTenantId: "ten000002", fabricUserId: "usr000001",
};
const tenant = { fabricTenantId: "ten000002", storeId, modelId };
const agent = { fabricTenantId: "ten000002", agentId: "agt00001" };
const params = {
  endpoint: "http://txo-fabric-openfga.txo-fabric-system.svc:8080",
  apiKey: "test-only-server-side-token-0123456789",
  resolveTenantStore: async () => tenant,
  resolveCanonicalAgent: async () => agent,
};
const check = { principal: alice, tenantKey:"indiba", agentKey:"sam", action:"agent.chat" };

test("central FGA Check receives only stable Fabric identities and server-side store IDs", async () => {
  let calls = 0;
  const authorize = createOpenFGAAuthorizer({
    ...params,
    fetchImpl: async (url, options) => {
      calls++;
      assert.equal(url, "http://txo-fabric-openfga.txo-fabric-system.svc:8080/stores/" + storeId + "/check");
      assert.equal(options.headers.Authorization, "Bearer " + params.apiKey);
      assert.deepEqual(JSON.parse(options.body), {
        authorization_model_id: modelId,
        tuple_key: { user:"user:usr000001", relation:"can_chat", object:"agent:agt00001" },
      });
      return new Response('{"allowed":true}', {status:200});
    }
  });
  assert.equal(await authorize(check), true);
  assert.equal(calls, 1);
});

test("management checks use another relation; deny response is false", async () => {
  const authorize = createOpenFGAAuthorizer({
    ...params,
    fetchImpl: async (_url, options) => {
      assert.equal(JSON.parse(options.body).tuple_key.relation, "can_manage");
      return new Response('{"allowed":false}', {status:200});
    },
  });
  assert.equal(await authorize({...check, action:"agent.manage"}), false);
});

test("an unsigned identity, forged tenant or unsupported action never calls FGA", async () => {
  let calls=0;
  const authorize = createOpenFGAAuthorizer({
    ...params,
    fetchImpl: async()=>{calls++; throw Error("should not fetch");}
  });
  for(const input of [
    {...check, principal:{...alice, verified:false}},
    {...check, principal:{...alice, tenantKey:"hairem"}},
    {...check, tenantKey:"hairem"},
    {...check, principal:{...alice, fabricUserId:""}},
    {...check, action:"provider.delete"},
    {...check, agentKey:"../another"}
  ]) assert.equal(await authorize(input),false);
  assert.equal(calls,0);
});

test("store or agent tenancy mismatch fails closed without network calls",async()=>{
  let calls=0;
  const fetchImpl=async()=>{calls++;throw Error("unexpected");};
  const wrongStore=createOpenFGAAuthorizer({...params,fetchImpl,
    resolveTenantStore:async()=>({...tenant,fabricTenantId:"ten000001"})});
  const wrongAgent=createOpenFGAAuthorizer({...params,fetchImpl,
    resolveCanonicalAgent:async()=>({...agent,fabricTenantId:"ten000001"})});
  assert.equal(await wrongStore(check),false);
  assert.equal(await wrongAgent(check),false);
  assert.equal(calls,0);
});

test("FGA unavailable, unauthenticated, stale model and malformed replies all deny",async()=>{
  for(const fetchImpl of [
    async()=>{throw Error("timeout");},
    async()=>new Response("internal DB error", {status:500}),
    async()=>new Response('{"allowed":"true"}', {status:200}),
    async()=>new Response("oops", {status:200}),
  ]) {
    const authorize=createOpenFGAAuthorizer({...params,fetchImpl});
    assert.equal(await authorize(check),false);
  }
  const unresolved=createOpenFGAAuthorizer({...params,resolveTenantStore:async()=>({...tenant,modelId:""}),fetchImpl:async()=>{throw Error("unexpected");}});
  assert.equal(await unresolved(check),false);
});

test("endpoint and security dependencies cannot be forged",()=>{
  for(const endpoint of [
    "https://openfga.example.com",
    "http://localhost:8080",
    "http://txo-fabric-openfga.txo-fabric-system.svc:8081",
    "http://txo-fabric-openfga.txo-fabric-system.svc:8080/stores",
  ]) assert.throws(()=>openFGAPrivateOrigin(endpoint),/Invalid/);
  assert.throws(()=>createOpenFGAAuthorizer({...params,apiKey:"short"}),/security dependencies/);
  assert.throws(()=>createOpenFGAAuthorizer({...params,resolveCanonicalAgent:null}),/security dependencies/);
});
