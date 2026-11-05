import { createHmac, randomBytes as secureRandomBytes } from "node:crypto";
import { readFile as readPrivateFile } from "node:fs/promises";

export const ACTIVE_HUMAN_PATH = "/internal/v1/identity/active";
export const ACTIVE_HUMAN_URL = "http://txo-fabric-operator-private-bff.txo-fabric-system.svc:8090" + ACTIVE_HUMAN_PATH;
export const ACTIVE_HUMAN_AUDIENCE = "txo-fabric-private-identity";
export const ACTIVE_HUMAN_SERVICE_ACCOUNT = "system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff";
const TOKEN_PATH = "/var/run/secrets/txo-fabric-bff/token";
const KEY_PATH = "/var/run/secrets/txo-fabric-bridge/hmac-key";
const ISSUER = "https://authentik.truxonline.com/application/o/txo-fabric-webui/";
const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/;
const SUBJECT = /^[!-~]{1,255}$/;
const JWT = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const unavailable = () => { throw new Error("Private active-human authority unavailable"); };

function exact(value, keys) {
  return value && typeof value === "object" && !Array.isArray(value) &&
    Object.keys(value).length === keys.length && keys.every(key => Object.hasOwn(value, key));
}

/** BFF-only signed active-account/freshness recheck; grants no Fabric access. */
export function createActiveHumanClient({
  enabled = false, endpoint = ACTIVE_HUMAN_URL, tokenPath = TOKEN_PATH,
  keyPath = KEY_PATH, readFile = readPrivateFile, randomBytes = secureRandomBytes,
  fetchImpl = fetch, now = () => Date.now(),
} = {}) {
  if (typeof enabled !== "boolean" || endpoint !== ACTIVE_HUMAN_URL ||
      typeof tokenPath !== "string" || !tokenPath || typeof keyPath !== "string" || !keyPath ||
      typeof readFile !== "function" || typeof randomBytes !== "function" ||
      typeof fetchImpl !== "function" || typeof now !== "function") unavailable();
  return Object.freeze({
    async verify({ issuer, oidcSubject, authentikUserUUID } = {}) {
      if (!enabled || issuer !== ISSUER || typeof oidcSubject !== "string" || !SUBJECT.test(oidcSubject) ||
          typeof authentikUserUUID !== "string" || !UUID.test(authentikUserUUID)) unavailable();
      try {
        const milliseconds = now();
        if (!Number.isSafeInteger(milliseconds) || milliseconds < 0) unavailable();
        const nonce = randomBytes(32);
        if (!Buffer.isBuffer(nonce) || nonce.length !== 32) unavailable();
        const iat = Math.floor(milliseconds / 1000);
        const [key, tokenRaw] = await Promise.all([readFile(keyPath), readFile(tokenPath)]);
        const bearer = Buffer.isBuffer(tokenRaw) ? tokenRaw.toString("utf8").trim() : "";
        if (!Buffer.isBuffer(key) || key.length < 32 || key.length > 1024 || !JWT.test(bearer)) unavailable();
        const body = Object.freeze({ oidcIssuer: issuer, oidcSubject, authentikUserUUID });
        const assertion = Object.freeze({ method: "POST", path: ACTIVE_HUMAN_PATH, ...body,
          serviceAccount: ACTIVE_HUMAN_SERVICE_ACCOUNT, audience: ACTIVE_HUMAN_AUDIENCE,
          iat, exp: iat + 20, jti: nonce.toString("base64url") });
        const payload = Buffer.from(JSON.stringify(assertion)).toString("base64url");
        const signature = createHmac("sha256", key).update(payload).digest("base64url");
        const response = await fetchImpl(endpoint, { method: "POST", redirect: "error", cache: "no-store",
          headers: { "Content-Type": "application/json", Authorization: "Bearer " + bearer,
            "X-Fabric-Assertion": payload + "." + signature }, body: JSON.stringify(body), signal: AbortSignal.timeout(2000) });
        if (response.status !== 200 || response.redirected || !/^application\/json(?:\s*;|$)/i.test(response.headers?.get?.("content-type") || "")) unavailable();
        const text = await response.text(); if (text.length > 4096) unavailable();
        const result = JSON.parse(text);
        if (!exact(result, ["oidcIssuer", "oidcSubject", "authentikUserUUID", "accountActive", "fresh"]) ||
            result.oidcIssuer !== issuer || result.oidcSubject !== oidcSubject ||
            result.authentikUserUUID !== authentikUserUUID || result.accountActive !== true || result.fresh !== true) unavailable();
        return Object.freeze({ issuer, oidcSubject, authentikUserUUID, accountActive: true, fresh: true });
      } catch { unavailable(); }
    },
  });
}
