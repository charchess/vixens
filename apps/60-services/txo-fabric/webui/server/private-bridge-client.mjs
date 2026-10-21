import { createHmac, randomBytes as secureRandomBytes } from "node:crypto";
import { readFile as readPrivateFile } from "node:fs/promises";

export const PRIVATE_BRIDGE_PATH = "/internal/v1/identity/resolve";
export const PRIVATE_BRIDGE_URL =
  "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090" + PRIVATE_BRIDGE_PATH;
export const PRIVATE_BRIDGE_AUDIENCE = "txo-fabric-private-identity";
export const PRIVATE_BFF_SUBJECT =
  "system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff";
const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/;
const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const USER_ID = /^usr[0-9a-f]{32}$/;
const TENANT_ID = /^TEN[0-9]{5,}$/;
export const GLOBAL_OIDC_ISSUER =
  "https://authentik.truxonline.com/application/o/txo-fabric-webui/";
const SUBJECT = /^[!-~]{1,255}$/;
const JWT = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const denied = () => new Error("Private Fabric identity unavailable");
export const PRIVATE_ENROLLMENT_PATH = "/internal/v1/identity/enroll";
export const PRIVATE_ENROLLMENT_URL =
  "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090" + PRIVATE_ENROLLMENT_PATH;


/**
 * Workload-only helper. The tenant and UUID come from a verified, durable
 * GLOBAL session + platform tenant catalogue, NEVER directly from browser
 * JSON/headers. No OIDC sub, Fabric ID or old per-tenant issuer accepted.
 *
 * The key must be the exact operator-side platform secret, mounted privately;
 * the token must be Kubernetes-projected for the dedicated #4037 audience.
 * Every attempt gets a new cryptographically secure 256-bit JTI.
 */
export function createPrivateBridgeClient({
  enabled = false,
  endpoint = PRIVATE_BRIDGE_URL,
  tokenPath = "/var/run/secrets/txo-fabric-bff/token",
  keyPath = "/var/run/secrets/txo-fabric-bridge/hmac-key",
  readFile = readPrivateFile,
  randomBytes = secureRandomBytes,
  fetchImpl = fetch,
  now = () => Date.now(),
} = {}) {
  if (typeof enabled !== "boolean" ||
      endpoint !== PRIVATE_BRIDGE_URL ||
      typeof tokenPath !== "string" || tokenPath.length < 1 ||
      typeof keyPath !== "string" || keyPath.length < 1 ||
      typeof readFile !== "function" || typeof randomBytes !== "function" ||
      typeof fetchImpl !== "function" || typeof now !== "function") throw denied();

  return Object.freeze({
    async resolve({ tenantKey, authentikUserUUID } = {}) {
      // Feature-off doesn't even read a credential or initiate network I/O.
      if (!enabled || !TENANT.test(tenantKey || "") ||
          !UUID.test(authentikUserUUID || "")) throw denied();
      try {
        const time = now();
        if (!Number.isSafeInteger(time) || time < 0) throw denied();
        const iat = Math.floor(time / 1000);
        const exp = iat + 20; // #4037 max 30 seconds, 5 seconds skew
        const nonce = randomBytes(32);
        if (!Buffer.isBuffer(nonce) || nonce.length !== 32) throw denied();
        const jti = nonce.toString("base64url");
        const [key, tokenRaw] = await Promise.all([
          readFile(keyPath), readFile(tokenPath),
        ]);
        if (!Buffer.isBuffer(key) || key.length < 32 || key.length > 1024 ||
            !Buffer.isBuffer(tokenRaw) || tokenRaw.length > 16384) throw denied();
        const bearer = tokenRaw.toString("utf8").trim();
        if (!JWT.test(bearer)) throw denied();
        const body = JSON.stringify({ tenantKey, authentikUserUUID });
        const assertion = {
          method: "POST", path: PRIVATE_BRIDGE_PATH,
          tenantKey, authentikUserUUID,
          serviceAccount: PRIVATE_BFF_SUBJECT,
          audience: PRIVATE_BRIDGE_AUDIENCE,
          iat, exp, jti,
        };
        const payload = Buffer.from(JSON.stringify(assertion)).toString("base64url");
        const signature = createHmac("sha256", key).update(payload).digest("base64url");
        const response = await fetchImpl(endpoint, {
          method: "POST", redirect: "error", cache: "no-store",
          headers: {
            "Content-Type": "application/json",
            "Authorization": "Bearer " + bearer,
            "X-Fabric-Assertion": payload + "." + signature,
          },
          body,
          signal: AbortSignal.timeout(2000),
        });
        if (response.status !== 200 || response.redirected ||
            !/^application\/json(?:\s*;|$)/i.test(response.headers?.get?.("content-type") || "") ||
            Number(response.headers?.get?.("content-length") || 0) > 2048) throw denied();
        const text = await response.text();
        if (text.length > 2048) throw denied();
        const result = JSON.parse(text);
        if (!result || typeof result !== "object" ||
            Array.isArray(result) || Object.keys(result).length !== 2 ||
            result.tenantKey !== tenantKey ||
            typeof result.fabricUserId !== "string" ||
            !USER_ID.test(result.fabricUserId)) throw denied();
        // The only output is an existing tenant-scoped Fabric identity.
        return Object.freeze({ tenantKey, fabricUserId: result.fabricUserId });
      } catch {
        // Never leak raw HTTP error bodies, tokens, UUID, HMAC, session or JTI.
        throw denied();
      }
    },

    async enroll({ tenantKey, oidcIssuer, oidcSubject, authentikUserUUID, globalOidcIssuer, globalOidcSubject } = {}) {
      if (!enabled || !TENANT.test(tenantKey || "") || !UUID.test(authentikUserUUID || "") ||
          typeof oidcIssuer !== "string" || !oidcIssuer.startsWith("https://") || oidcIssuer.length > 2048 ||
          typeof oidcSubject !== "string" || oidcSubject.length < 1 || oidcSubject.length > 2048 ||
          globalOidcIssuer !== GLOBAL_OIDC_ISSUER ||
          typeof globalOidcSubject !== "string" || !SUBJECT.test(globalOidcSubject)) throw denied();
      try {
        const time = now(); if (!Number.isSafeInteger(time) || time < 0) throw denied();
        const iat = Math.floor(time / 1000), exp = iat + 20;
        const nonce = randomBytes(32); if (!Buffer.isBuffer(nonce) || nonce.length !== 32) throw denied();
        const jti = nonce.toString("base64url");
        const [key, tokenRaw] = await Promise.all([readFile(keyPath), readFile(tokenPath)]);
        if (!Buffer.isBuffer(key) || key.length < 32 || key.length > 1024 || !Buffer.isBuffer(tokenRaw) || tokenRaw.length > 16384) throw denied();
        const bearer = tokenRaw.toString("utf8").trim(); if (!JWT.test(bearer)) throw denied();
        const body = JSON.stringify({ tenantKey, oidcIssuer, oidcSubject, authentikUserUUID, globalOidcIssuer, globalOidcSubject });
        const assertion = { method: "POST", path: PRIVATE_ENROLLMENT_PATH, tenantKey, oidcIssuer, oidcSubject, authentikUserUUID, globalOidcIssuer, globalOidcSubject, serviceAccount: PRIVATE_BFF_SUBJECT, audience: PRIVATE_BRIDGE_AUDIENCE, iat, exp, jti };
        const payload = Buffer.from(JSON.stringify(assertion)).toString("base64url");
        const signature = createHmac("sha256", key).update(payload).digest("base64url");
        const response = await fetchImpl(PRIVATE_ENROLLMENT_URL, { method: "POST", redirect: "error", cache: "no-store", headers: { "Content-Type": "application/json", Authorization: "Bearer " + bearer, "X-Fabric-Assertion": payload + "." + signature }, body, signal: AbortSignal.timeout(2000) });
        if (response.status !== 200 || response.redirected || !/^application\/json(?:\s*;|$)/i.test(response.headers?.get?.("content-type") || "") || Number(response.headers?.get?.("content-length") || 0) > 2048) throw denied();
        const text = await response.text(); if (text.length > 2048) throw denied();
        const result = JSON.parse(text);
        if (!result || typeof result !== "object" || Array.isArray(result) || Object.keys(result).length !== 2 || result.tenantKey !== tenantKey || typeof result.fabricUserId !== "string" || !USER_ID.test(result.fabricUserId)) throw denied();
        return Object.freeze({ tenantKey, fabricUserId: result.fabricUserId });
      } catch { throw denied(); }
    },
  });
}

/**
 * Source-only BFF composition seam, NOT an authentication endpoint.
 *
 * The ONLY production caller is expected to be createGlobalWebUIHandler:
 * it obtains identity from a verified, server-owned durable GLOBAL OIDC
 * session and tenantKey/fabricTenantId from the canonical server catalogue.
 * No raw browser request, group claim, issuer-substitution, Fabric user ID,
 * store/model, backend URL or AgentIdentity is accepted here.
 *
 * The live Authentik recheck prevents stale/inactive global sessions.
 * The operator's separately authenticated bridge proves fresh tenant-group
 * membership, issuer-pinned legacy ledger and FGA reconciliation lease;
 * a Fabric user ID is NEVER assigned from the global OIDC subject. An agent
 * grant (agent.chat) and effective policy are STILL a separate operation-
 * level authorization check, not implied by this tenant membership.
 *
 * The double opt-in is source-only. Runtime BFF never instantiates it,
 * even if someone flips an environment variable.
 */
export function createVerifiedTenantBridgeAccess({
  browserEnabled = false, bridgeEnabled = false,
  verifyActiveHuman, bridgeClient,
} = {}) {
  if (typeof browserEnabled !== "boolean" || typeof bridgeEnabled !== "boolean" ||
      typeof verifyActiveHuman !== "function" ||
      typeof bridgeClient?.resolve !== "function") throw denied();

  return async function resolveTenantAccess({ identity, tenantKey, fabricTenantId } = {}) {
    // No auth-provider call, secret read, token or network when either gate is
    // OFF. This function is never a direct browser-facing API.
    if (!browserEnabled || !bridgeEnabled) throw denied();
    // This is the exact server-owned durable-session projection from
    // createPostgresGlobalSessions.authenticate(). Keep an exact allowlist:
    // accepting a browser-shaped object or silently stripping extra fields
    // would turn the bridge boundary into an authority normalizer.
    if (!identity || typeof identity !== "object" || Array.isArray(identity) ||
        Object.keys(identity).length !== 5 ||
        !["authenticated", "verified", "issuer", "oidcSubject", "authentikUserUUID"].every(k =>
          Object.prototype.hasOwnProperty.call(identity, k)) ||
        identity.authenticated !== true || identity.verified !== true ||
        identity.issuer !== GLOBAL_OIDC_ISSUER ||
        typeof identity.oidcSubject !== "string" || !SUBJECT.test(identity.oidcSubject) ||
        !UUID.test(identity.authentikUserUUID || "") ||
        typeof tenantKey !== "string" || !TENANT.test(tenantKey) ||
        typeof fabricTenantId !== "string" || !TENANT_ID.test(fabricTenantId)) throw denied();
    try {
      const probe = Object.freeze({
        issuer: identity.issuer, oidcSubject: identity.oidcSubject,
        authentikUserUUID: identity.authentikUserUUID,
      });
      const active = await verifyActiveHuman(probe);
      if (!active || active.accountActive !== true || active.fresh !== true ||
          active.issuer !== probe.issuer ||
          active.oidcSubject !== probe.oidcSubject ||
          active.authentikUserUUID !== probe.authentikUserUUID) throw denied();
      const result = await bridgeClient.resolve(Object.freeze({
        tenantKey, authentikUserUUID: probe.authentikUserUUID,
      }));
      if (!result || typeof result !== "object" || Array.isArray(result) ||
          Object.keys(result).length !== 2 ||
          result.tenantKey !== tenantKey ||
          typeof result.fabricUserId !== "string" ||
          !USER_ID.test(result.fabricUserId)) throw denied();

      // canEnter is ONLY a current tenant-membership result from the real
      // operator resolver. Never invent an admin/agent.chat permission.
      return Object.freeze({
        tenantKey, fabricTenantId, fabricUserId: result.fabricUserId,
        accountActive: true, fresh: true, canEnter: true,
        canManageTenant: false,
      });
    } catch { throw denied(); }
  };
}
