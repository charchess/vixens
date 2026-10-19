import { createHmac, randomBytes as secureRandomBytes } from "node:crypto";
import { readFile as readPrivateFile } from "node:fs/promises";

export const BROWSER_AUTHORITY_PATH = "/internal/v1/browser-authority/resolve";
export const BROWSER_AUTHORITY_URL = "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090" + BROWSER_AUTHORITY_PATH;
export const BROWSER_AUTHORITY_AUDIENCE = "txo-fabric-private-identity";
export const BROWSER_AUTHORITY_SERVICE_ACCOUNT = "system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff";
const TOKEN_PATH = "/var/run/secrets/txo-fabric-bff/token";
const KEY_PATH = "/var/run/secrets/txo-fabric-bridge/hmac-key";
const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/;
const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const TENANT_ID = /^TEN[0-9]{5,}$/;
const USER = /^usr[0-9a-f]{32}$/;
const AGENT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const UID = /^[A-Za-z0-9._-]{1,253}$/;
const JWT = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const fail = () => new Error("Private browser authority unavailable");
const exact = (value, keys) => value && typeof value === "object" && !Array.isArray(value) && Object.keys(value).length === keys.length && keys.every((key) => Object.hasOwn(value, key));
const tenant = (value) => value && typeof value === "object" && !Array.isArray(value) && TENANT.test(value.tenantKey) && TENANT_ID.test(value.fabricTenantId) && typeof value.displayName === "string" && value.displayName.length > 0 && value.displayName.length <= 128 && !/[\u0000-\u001f\u007f]/.test(value.displayName);

export function validateBrowserAuthorityResponse(kind, value) {
  if (kind === "catalog") {
    if (!exact(value, ["tenants"]) || !Array.isArray(value.tenants) || value.tenants.length > 500) throw fail();
    const keys = new Set(), ids = new Set();
    for (const entry of value.tenants) {
      if (!exact(entry, ["tenantKey", "fabricTenantId", "displayName"]) || !tenant(entry) || keys.has(entry.tenantKey) || ids.has(entry.fabricTenantId)) throw fail();
      keys.add(entry.tenantKey); ids.add(entry.fabricTenantId);
    }
    return Object.freeze({ tenants: Object.freeze(value.tenants.map((entry) => Object.freeze({ ...entry }))) });
  }
  if (kind === "tenant") {
    if (!exact(value, ["tenant"]) || !exact(value.tenant, ["tenantKey", "fabricTenantId", "displayName", "fabricUserId", "accountActive", "fresh", "canEnter"]) || !tenant(value.tenant) || !USER.test(value.tenant.fabricUserId) || value.tenant.accountActive !== true || value.tenant.fresh !== true || value.tenant.canEnter !== true) throw fail();
    return Object.freeze({ tenant: Object.freeze({ ...value.tenant }) });
  }
  if (kind === "agent") {
    if (!exact(value, ["agent"]) || !exact(value.agent, ["agentKey", "displayName", "agentUID", "generation"]) || !AGENT.test(value.agent.agentKey) || typeof value.agent.displayName !== "string" || value.agent.displayName.length < 1 || value.agent.displayName.length > 128 || /[\u0000-\u001f\u007f]/.test(value.agent.displayName) || !UID.test(value.agent.agentUID) || !Number.isSafeInteger(value.agent.generation) || value.agent.generation < 1) throw fail();
    return Object.freeze({ agent: Object.freeze({ ...value.agent }) });
  }
  throw fail();
}

// Source-only client. Gate-off performs zero filesystem, entropy, or network
// I/O. Production BFF composition intentionally does not instantiate it.
export function createBrowserAuthorityClient({ enabled = false, readFile = readPrivateFile, randomBytes = secureRandomBytes, fetchImpl = fetch, now = () => Date.now() } = {}) {
  if (typeof enabled !== "boolean" || typeof readFile !== "function" || typeof randomBytes !== "function" || typeof fetchImpl !== "function" || typeof now !== "function") throw fail();
  const call = async (kind, input) => {
    const expected = kind === "catalog" ? ["authentikUserUUID"] : kind === "tenant" ? ["authentikUserUUID", "tenantKey"] : kind === "agent" ? ["authentikUserUUID", "tenantKey", "agentKey"] : null;
    if (!enabled || !expected || !exact(input, expected) || !UUID.test(input.authentikUserUUID) || (kind !== "catalog" && !TENANT.test(input.tenantKey)) || (kind === "agent" && !AGENT.test(input.agentKey))) throw fail();
    try {
      const millis = now(); if (!Number.isSafeInteger(millis) || millis < 0) throw fail();
      const nonce = randomBytes(32); if (!Buffer.isBuffer(nonce) || nonce.length !== 32) throw fail();
      const [key, rawToken] = await Promise.all([readFile(KEY_PATH), readFile(TOKEN_PATH)]);
      const bearer = Buffer.isBuffer(rawToken) ? rawToken.toString("utf8").trim() : "";
      if (!Buffer.isBuffer(key) || key.length < 32 || key.length > 1024 || !JWT.test(bearer)) throw fail();
      const iat = Math.floor(millis / 1000);
      const assertion = { method: "POST", path: BROWSER_AUTHORITY_PATH, kind, ...input, serviceAccount: BROWSER_AUTHORITY_SERVICE_ACCOUNT, audience: BROWSER_AUTHORITY_AUDIENCE, iat, exp: iat + 20, jti: nonce.toString("base64url") };
      const payload = Buffer.from(JSON.stringify(assertion)).toString("base64url");
      const signature = createHmac("sha256", key).update(payload).digest("base64url");
      const response = await fetchImpl(BROWSER_AUTHORITY_URL, { method: "POST", redirect: "error", cache: "no-store", headers: { "Content-Type": "application/json", Authorization: "Bearer " + bearer, "X-Fabric-Assertion": payload + "." + signature }, body: JSON.stringify({ kind, ...input }), signal: AbortSignal.timeout(2000) });
      if (response.status !== 200 || response.redirected || !/^application\/json(?:\s*;|$)/i.test(response.headers?.get?.("content-type") || "")) throw fail();
      const text = await response.text(); if (text.length > 65536) throw fail();
      return validateBrowserAuthorityResponse(kind, JSON.parse(text));
    } catch { throw fail(); }
  };
  return Object.freeze({ catalog: (input) => call("catalog", input), tenant: (input) => call("tenant", input), agent: (input) => call("agent", input) });
}
