import { createHmac, randomBytes as secureRandomBytes } from "node:crypto";
import { readFile as readPrivateFile } from "node:fs/promises";

export const RESPONSES_AUTHORITY_PATH = "/internal/v1/responses-authority/resolve";
export const RESPONSES_AUTHORITY_URL =
  "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090" + RESPONSES_AUTHORITY_PATH;
export const RESPONSES_AUTHORITY_AUDIENCE = "txo-fabric-private-identity";
export const RESPONSES_AUTHORITY_SERVICE_ACCOUNT =
  "system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff";

const TOKEN_PATH = "/var/run/secrets/txo-fabric-bff/token";
const KEY_PATH = "/var/run/secrets/txo-fabric-bridge/hmac-key";
const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const TENANT_ID = /^TEN[0-9]{5,}$/;
const AGENT = /^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$/;
const USER = /^usr[0-9a-f]{32}$/;
const UID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
const JWT = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const unavailable = () => { throw new Error("Private Responses authority unavailable"); };

function exact(value, keys) {
  return value && typeof value === "object" && !Array.isArray(value) &&
    Object.keys(value).length === keys.length && keys.every(key => Object.hasOwn(value, key));
}

function validScope(scope, operation) {
  const keys = operation === "resolve-tenant" ? ["tenantKey", "fabricTenantId"] :
    operation === "resolve-agent" ? ["tenantKey", "fabricTenantId", "agentKey"] :
    operation === "authorize-chat" ? ["tenantKey", "fabricTenantId", "agentKey", "fabricUserId"] :
    operation === "verify-runtime" ? ["tenantKey", "fabricTenantId", "agentKey", "tenantBundleUID", "tenantGeneration", "agentIdentityUID", "agentGeneration"] : null;
  if (!keys || !exact(scope, keys)) return false;
  const { tenantKey, fabricTenantId, agentKey, fabricUserId, tenantBundleUID, tenantGeneration, agentIdentityUID, agentGeneration } = scope;
  if (!TENANT.test(tenantKey) || !TENANT_ID.test(fabricTenantId)) return false;
  if (operation === "resolve-tenant") return true;
  if (operation === "resolve-agent") return AGENT.test(agentKey);
  if (operation === "authorize-chat") return AGENT.test(agentKey) && USER.test(fabricUserId);
  return AGENT.test(agentKey) && UID.test(tenantBundleUID) && Number.isSafeInteger(tenantGeneration) && tenantGeneration > 0 && UID.test(agentIdentityUID) && Number.isSafeInteger(agentGeneration) && agentGeneration > 0;
}

function validResponse(operation, scope, value) {
  if (operation === "resolve-tenant") {
    return exact(value, ["tenant"]) && exact(value.tenant, ["tenantKey", "fabricTenantId", "tenantBundleUID", "status"]) &&
      value.tenant.tenantKey === scope.tenantKey && value.tenant.fabricTenantId === scope.fabricTenantId && UID.test(value.tenant.tenantBundleUID) && value.tenant.status === "Ready";
  }
  if (operation === "resolve-agent") {
    return exact(value, ["agent"]) && exact(value.agent, ["tenantKey", "fabricTenantId", "agentKey", "agentIdentityUID", "agentGeneration", "status"]) &&
      value.agent.tenantKey === scope.tenantKey && value.agent.fabricTenantId === scope.fabricTenantId && value.agent.agentKey === scope.agentKey && UID.test(value.agent.agentIdentityUID) && Number.isSafeInteger(value.agent.agentGeneration) && value.agent.agentGeneration > 0 && value.agent.status === "Ready";
  }
  if (operation === "authorize-chat") return exact(value, ["authorization"]) && exact(value.authorization, ["allowed"]) && value.authorization.allowed === true;
  return exact(value, ["runtime"]) && exact(value.runtime, ["attested"]) && value.runtime.attested === true;
}

/**
 * BFF-only client for the signed operator Responses authority. Gate-off does
 * not read a token/key, allocate entropy, or send a request. Browser input is
 * never accepted here: callers supply only server-derived canonical scope.
 */
export function createResponsesAuthorityClient({
  enabled = false, endpoint = RESPONSES_AUTHORITY_URL, tokenPath = TOKEN_PATH,
  keyPath = KEY_PATH, readFile = readPrivateFile, randomBytes = secureRandomBytes,
  fetchImpl = fetch, now = () => Date.now(),
} = {}) {
  if (typeof enabled !== "boolean" || endpoint !== RESPONSES_AUTHORITY_URL ||
      typeof tokenPath !== "string" || !tokenPath || typeof keyPath !== "string" || !keyPath ||
      typeof readFile !== "function" || typeof randomBytes !== "function" ||
      typeof fetchImpl !== "function" || typeof now !== "function") unavailable();

  const call = async (operation, scope) => {
    if (!enabled || !validScope(scope, operation)) unavailable();
    try {
      const milliseconds = now();
      if (!Number.isSafeInteger(milliseconds) || milliseconds < 0) unavailable();
      const nonce = randomBytes(32);
      if (!Buffer.isBuffer(nonce) || nonce.length !== 32) unavailable();
      const iat = Math.floor(milliseconds / 1000);
      const [key, tokenRaw] = await Promise.all([readFile(keyPath), readFile(tokenPath)]);
      const bearer = Buffer.isBuffer(tokenRaw) ? tokenRaw.toString("utf8").trim() : "";
      if (!Buffer.isBuffer(key) || key.length < 32 || key.length > 1024 || !JWT.test(bearer)) unavailable();
      const body = { operation, ...scope };
      const assertion = { method: "POST", path: RESPONSES_AUTHORITY_PATH, ...body,
        serviceAccount: RESPONSES_AUTHORITY_SERVICE_ACCOUNT, audience: RESPONSES_AUTHORITY_AUDIENCE,
        iat, exp: iat + 20, jti: nonce.toString("base64url") };
      const payload = Buffer.from(JSON.stringify(assertion)).toString("base64url");
      const signature = createHmac("sha256", key).update(payload).digest("base64url");
      const response = await fetchImpl(endpoint, {
        method: "POST", redirect: "error", cache: "no-store",
        headers: { "Content-Type": "application/json", Authorization: "Bearer " + bearer,
          "X-Fabric-Assertion": payload + "." + signature },
        body: JSON.stringify(body), signal: AbortSignal.timeout(2000),
      });
      if (response.status !== 200 || response.redirected ||
          !/^application\/json(?:\s*;|$)/i.test(response.headers?.get?.("content-type") || "") ||
          Number(response.headers?.get?.("content-length") || 0) > 4096) unavailable();
      const text = await response.text();
      if (text.length > 4096) unavailable();
      const parsed = JSON.parse(text);
      if (!validResponse(operation, scope, parsed)) unavailable();
      return Object.freeze(parsed);
    } catch { unavailable(); }
  };

  return Object.freeze({
    resolveTenant: scope => call("resolve-tenant", scope),
    resolveAgent: scope => call("resolve-agent", scope),
    authorizeChat: scope => call("authorize-chat", scope),
    verifyRuntime: scope => call("verify-runtime", scope),
  });
}
