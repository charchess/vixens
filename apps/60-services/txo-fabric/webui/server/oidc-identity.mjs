import { createPublicKey, verify as verifySignature } from "node:crypto";

const SLUG = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const SAFE_SUBJECT = /^[!-~]{1,255}$/;
const BASE64URL = /^[A-Za-z0-9_-]+$/;
const KID = /^[A-Za-z0-9._:-]{1,128}$/;
const MAX_TOKEN_BYTES = 16384;
const MAX_RESPONSE_BYTES = 65536;
const CLOCK_SKEW_SECONDS = 30;
const MAX_ENROLLMENT_TOKEN_AGE_SECONDS = 300;

const reject = () => { throw new Error("Untrusted Fabric OIDC identity proof"); };

function trustedIssuer(tenantKey, issuer, clientId) {
  if (!SLUG.test(tenantKey || "") || typeof issuer !== "string" ||
      typeof clientId !== "string" || clientId !== `txo-fabric-${tenantKey}`) reject();
  let url;
  try { url = new URL(issuer); } catch { reject(); }
  if (url.protocol !== "https:" || !url.hostname || url.username || url.password ||
      url.port || url.search || url.hash || url.pathname !==
      `/application/o/txo-fabric-${tenantKey}/` ||
      url.href !== issuer || url.hostname !== url.host.toLowerCase()) reject();
  return url;
}

function decodeSegment(segment) {
  if (typeof segment !== "string" || !BASE64URL.test(segment)) reject();
  const bytes = Buffer.from(segment, "base64url");
  if (bytes.toString("base64url") !== segment) reject();
  return bytes;
}

function decodeObject(segment) {
  try {
    const value = JSON.parse(decodeSegment(segment).toString("utf8"));
    if (!value || typeof value !== "object" || Array.isArray(value)) reject();
    return value;
  } catch { reject(); }
}

async function readJSON(url, fetchImpl) {
  let response;
  try {
    response = await fetchImpl(url, {
      method: "GET", headers: { Accept: "application/json" },
      redirect: "error", cache: "no-store", signal: AbortSignal.timeout(3000),
    });
    if (response.status !== 200 || response.redirected ||
        Number(response.headers?.get?.("content-length") || 0) > MAX_RESPONSE_BYTES) reject();
    const body = await response.text();
    if (body.length > MAX_RESPONSE_BYTES) reject();
    return JSON.parse(body);
  } catch { reject(); }
}

function trustedJWKSURL(value, expectedIssuer) {
  if (typeof value !== "string") reject();
  let url;
  try { url = new URL(value); } catch { reject(); }
  // The issuer is configured by Fabric, not an untrusted discovery response.
  // Never let an IdP document turn this into a generic HTTP/SSRF client.
  if (url.origin !== expectedIssuer.origin || url.pathname !==
      expectedIssuer.pathname + "jwks/" || url.username || url.password ||
      url.search || url.hash || url.href !== value) reject();
  return url.href;
}

/**
 * Source-only verified OIDC login proof, intended for a privileged Fabric BFF.
 *
 * issuer/clientId/tenantKey MUST be selected server-side from TenantBundle.
 * expectedNonce MUST come from a single-use, server-side OAuth login transaction,
 * never the browser token or headers. The BFF must separately verify OAuth
 * state/PKCE/code exchange and refuse a reused transaction.
 *
 * The UUID claim is intended to be minted by an Authentik scope mapping from
 * request.user.uuid. It is not supplied by the browser or derived from sub,
 * email, username, groups or userRef. A missing claim fails closed.
 *
 * This does NOT allocate Fabric IDs, write the identity ledger, set
 * principal.verified, create a session, grant access, or register a route.
 */
export function createAuthentikIdentityVerifier({
  tenantKey, issuer, clientId, fetchImpl = fetch,
  now = () => Date.now(),
}) {
  const expectedIssuer = trustedIssuer(tenantKey, issuer, clientId);
  if (typeof fetchImpl !== "function" || typeof now !== "function") reject();
  const discoveryURL = expectedIssuer.href + ".well-known/openid-configuration";

  return async function verifyLoginIDToken({ idToken, expectedNonce } = {}) {
    if (typeof idToken !== "string" || idToken.length > MAX_TOKEN_BYTES ||
        typeof expectedNonce !== "string" || expectedNonce.length < 16 ||
        expectedNonce.length > 256 || !SAFE_SUBJECT.test(expectedNonce)) reject();
    const parts = idToken.split(".");
    if (parts.length !== 3 || !parts.every((p) => p.length > 0)) reject();
    const [head, payload, signature] = parts;
    const header = decodeObject(head);
    const claims = decodeObject(payload);
    if (header.alg !== "RS256" || !KID.test(header.kid || "") ||
        (header.typ !== undefined && header.typ !== "JWT") ||
        header.jku !== undefined || header.jwk !== undefined ||
        header.x5u !== undefined || header.crit !== undefined ||
        header.enc !== undefined || header.zip !== undefined) reject();

    const t = Math.floor(now() / 1000);
    if (!Number.isSafeInteger(t) || claims.iss !== expectedIssuer.href ||
        claims.aud !== clientId || !SAFE_SUBJECT.test(claims.sub || "") ||
        !UUID.test(claims.txo_fabric_user_uuid || "") ||
        claims.nonce !== expectedNonce ||
        !Number.isSafeInteger(claims.iat) || !Number.isSafeInteger(claims.exp) ||
        claims.iat > t + CLOCK_SKEW_SECONDS ||
        claims.iat < t - MAX_ENROLLMENT_TOKEN_AGE_SECONDS ||
        claims.exp <= t || claims.exp <= claims.iat ||
        (claims.nbf !== undefined &&
          (!Number.isSafeInteger(claims.nbf) || claims.nbf > t + CLOCK_SKEW_SECONDS)) ||
        claims.azp !== undefined && claims.azp !== clientId) reject();

    const discovery = await readJSON(discoveryURL, fetchImpl);
    if (discovery?.issuer !== expectedIssuer.href) reject();
    const jwksURL = trustedJWKSURL(discovery.jwks_uri, expectedIssuer);
    const jwks = await readJSON(jwksURL, fetchImpl);
    if (!Array.isArray(jwks?.keys) || jwks.keys.length === 0 ||
        jwks.keys.length > 32) reject();

    const candidates = jwks.keys.filter((key) => key?.kid === header.kid);
    if (candidates.length !== 1) reject();
    const jwk = candidates[0];
    if (jwk.kty !== "RSA" || jwk.alg !== undefined && jwk.alg !== "RS256" ||
        jwk.use !== undefined && jwk.use !== "sig" ||
        jwk.key_ops !== undefined &&
          (!Array.isArray(jwk.key_ops) || !jwk.key_ops.includes("verify")) ||
        typeof jwk.n !== "string" || typeof jwk.e !== "string" ||
        jwk.d !== undefined || jwk.p !== undefined || jwk.q !== undefined) reject();
    let valid;
    try {
      const key = createPublicKey({ key: { kty: "RSA", n: jwk.n, e: jwk.e }, format: "jwk" });
      valid = verifySignature("RSA-SHA256", Buffer.from(head + "." + payload),
        key, decodeSegment(signature));
    } catch { reject(); }
    if (!valid) reject();

    return Object.freeze({
      tenantKey, issuer: expectedIssuer.href, oidcSubject: claims.sub,
      authentikUserUUID: claims.txo_fabric_user_uuid,
    });
  };
}
