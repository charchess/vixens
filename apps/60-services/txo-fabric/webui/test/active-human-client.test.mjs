import assert from "node:assert/strict";
import test from "node:test";
import { createActiveHumanClient } from "../server/active-human-client.mjs";

const issuer = "https://authentik.truxonline.com/application/o/txo-fabric-webui/";
const uuid = "123e4567-e89b-42d3-a456-426614174000";
const response = body => new Response(JSON.stringify(body), { headers: { "content-type": "application/json" } });

test("active-human client signs the exact BFF-only request and accepts only echoed active identity", async () => {
  let request;
  const client = createActiveHumanClient({ enabled: true, now: () => 1760212800000,
    randomBytes: () => Buffer.alloc(32, 7),
    readFile: async path => Buffer.from(path.endsWith("hmac-key") ? "k".repeat(32) : "a.b.c"),
    fetchImpl: async (url, options) => {
      request = { url, options };
      return response({ oidcIssuer: issuer, oidcSubject: "global-subject", authentikUserUUID: uuid, accountActive: true, fresh: true });
    },
  });
  const verified = await client.verify({ issuer, oidcSubject: "global-subject", authentikUserUUID: uuid });
  assert.deepEqual(verified, { issuer, oidcSubject: "global-subject", authentikUserUUID: uuid, accountActive: true, fresh: true });
  assert.equal(request.url, "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090/internal/v1/identity/active");
  assert.equal(request.options.headers.Authorization, "Bearer a.b.c");
  assert.match(request.options.headers["X-Fabric-Assertion"], /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]{43}$/);
});

test("active-human client fails closed for gate-off, malformed replies and identity mismatch", async () => {
  const off = createActiveHumanClient({ fetchImpl: async () => { throw Error("must not call"); } });
  await assert.rejects(off.verify({ issuer, oidcSubject: "global-subject", authentikUserUUID: uuid }));
  const bad = createActiveHumanClient({ enabled: true, now: () => 1760212800000,
    randomBytes: () => Buffer.alloc(32, 7), readFile: async path => Buffer.from(path.endsWith("hmac-key") ? "k".repeat(32) : "a.b.c"),
    fetchImpl: async () => response({ oidcIssuer: issuer, oidcSubject: "other", authentikUserUUID: uuid, accountActive: true, fresh: true }),
  });
  await assert.rejects(bad.verify({ issuer, oidcSubject: "global-subject", authentikUserUUID: uuid }));
});
