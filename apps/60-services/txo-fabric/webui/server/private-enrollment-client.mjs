import { createHmac, randomBytes as secureRandomBytes } from "node:crypto";
import { readFile as readPrivateFile } from "node:fs/promises";

export const PRIVATE_ENROLLMENT_PATH = "/internal/v1/identity/enroll";
export const PRIVATE_ENROLLMENT_URL =
  "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090" + PRIVATE_ENROLLMENT_PATH;
export const PRIVATE_ENROLLMENT_AUDIENCE = "txo-fabric-private-identity";
export const PRIVATE_BFF_SUBJECT =
  "system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff";
export const GLOBAL_OIDC_ISSUER =
  "https://authentik.truxonline.com/application/o/txo-fabric-webui/";

const TOKEN_PATH = "/var/run/secrets/txo-fabric-bff/token";
const KEY_PATH = "/var/run/secrets/txo-fabric-bridge/hmac-key";
const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/;
const TENANT = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const SUBJECT = /^[!-~]{1,255}$/;
const USER = /^usr[0-9a-f]{32}$/;
const JWT = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const fail = () => new Error("Private Fabric enrollment unavailable");
const exact = (value, keys) => value && typeof value === "object" &&
  !Array.isArray(value) && Object.keys(value).length === keys.length &&
  keys.every(key => Object.hasOwn(value, key));

// Workload-only client for the operator's enrollment endpoint. Its input comes
// only from a server-verified GLOBAL OIDC callback; browser values, Fabric IDs,
// model/store names and endpoints are not accepted.
export function createPrivateEnrollmentClient({
  enabled = false,
  endpoint = PRIVATE_ENROLLMENT_URL,
  tokenPath = TOKEN_PATH,
  keyPath = KEY_PATH,
  readFile = readPrivateFile,
  randomBytes = secureRandomBytes,
  fetchImpl = fetch,
  now = () => Date.now(),
} = {}) {
  if (typeof enabled !== "boolean" || endpoint !== PRIVATE_ENROLLMENT_URL ||
      typeof tokenPath !== "string" || !tokenPath || typeof keyPath !== "string" || !keyPath ||
      typeof readFile !== "function" || typeof randomBytes !== "function" ||
      typeof fetchImpl !== "function" || typeof now !== "function") throw fail();

  return Object.freeze({
    async enroll(input) {
      // Gate-off is an I/O boundary: no key/token read, entropy, or request.
      if (!enabled || !exact(input, ["tenantKey", "issuer", "oidcSubject", "authentikUserUUID"]) ||
          !TENANT.test(input.tenantKey) || input.issuer !== GLOBAL_OIDC_ISSUER ||
          !SUBJECT.test(input.oidcSubject) || !UUID.test(input.authentikUserUUID)) throw fail();
      try {
        const millis = now();
        if (!Number.isSafeInteger(millis) || millis < 0) throw fail();
        const nonce = randomBytes(32);
        if (!Buffer.isBuffer(nonce) || nonce.length !== 32) throw fail();
        const [key, rawToken] = await Promise.all([readFile(keyPath), readFile(tokenPath)]);
        const bearer = Buffer.isBuffer(rawToken) ? rawToken.toString("utf8").trim() : "";
        if (!Buffer.isBuffer(key) || key.length < 32 || key.length > 1024 || !JWT.test(bearer)) throw fail();
        const iat = Math.floor(millis / 1000);
        const request = {
          tenantKey: input.tenantKey,
          oidcIssuer: input.issuer,
          oidcSubject: input.oidcSubject,
          authentikUserUUID: input.authentikUserUUID,
        };
        const assertion = {
          method: "POST", path: PRIVATE_ENROLLMENT_PATH, ...request,
          serviceAccount: PRIVATE_BFF_SUBJECT, audience: PRIVATE_ENROLLMENT_AUDIENCE,
          iat, exp: iat + 20, jti: nonce.toString("base64url"),
        };
        const payload = Buffer.from(JSON.stringify(assertion)).toString("base64url");
        const signature = createHmac("sha256", key).update(payload).digest("base64url");
        const response = await fetchImpl(endpoint, {
          method: "POST", redirect: "error", cache: "no-store",
          headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer " + bearer,
            "X-Fabric-Assertion": payload + "." + signature,
          },
          body: JSON.stringify(request), signal: AbortSignal.timeout(2000),
        });
        if (response.status !== 200 || response.redirected ||
            !/^application\/json(?:\s*;|$)/i.test(response.headers?.get?.("content-type") || "") ||
            Number(response.headers?.get?.("content-length") || 0) > 2048) throw fail();
        const text = await response.text();
        if (text.length > 2048) throw fail();
        const result = JSON.parse(text);
        if (!exact(result, ["tenantKey", "fabricUserId"]) || result.tenantKey !== input.tenantKey ||
            !USER.test(result.fabricUserId)) throw fail();
        return Object.freeze({ tenantKey: result.tenantKey, fabricUserId: result.fabricUserId });
      } catch { throw fail(); }
    },
  });
}
