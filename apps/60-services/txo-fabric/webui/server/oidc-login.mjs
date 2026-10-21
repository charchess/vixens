import { createHash, randomBytes, timingSafeEqual } from "node:crypto";
import { createAuthentikIdentityVerifier } from "./oidc-identity.mjs";

const BASE64URL43 = /^[A-Za-z0-9_-]{43}$/;
const CODE = /^[\x21-\x7e]{1,4096}$/;
const FABRIC_USER_ID = /^usr[0-9a-f]{32}$/;
const MAX_AGE_MS = 5 * 60 * 1000;
const MAX_TOKEN_RESPONSE = 32768;
const SCOPE = "openid profile email txo_fabric_identity";
const COOKIE_NAME = "__Host-txo-fabric-oidc";
const GLOBAL_OIDC_ISSUER = "https://authentik.truxonline.com/application/o/txo-fabric-webui/";
const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/;
const SUBJECT = /^[!-~]{1,255}$/;

function validGlobalSessionIdentity(identity) {
  return !!identity && typeof identity === "object" && !Array.isArray(identity) &&
    Object.keys(identity).length === 3 &&
    ["issuer", "oidcSubject", "authentikUserUUID"].every((key) =>
      Object.prototype.hasOwnProperty.call(identity, key)) &&
    identity.issuer === GLOBAL_OIDC_ISSUER && SUBJECT.test(identity.oidcSubject) &&
    UUID.test(identity.authentikUserUUID);
}
const deny = () => { throw new Error("Fabric OIDC login denied"); };
const random = () => randomBytes(32).toString("base64url");
const digest = (value) => createHash("sha256").update(value).digest("base64url");

function config({ tenantKey, issuer, clientId, redirectURI, domainSuffix, authentikOrigin }) {
  if (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(tenantKey || "") ||
      clientId !== `txo-fabric-${tenantKey}` || typeof domainSuffix !== "string" ||
      !/^[a-z0-9-]+(?:\.[a-z0-9-]+)+$/.test(domainSuffix)) deny();
  let provider, callback, platformIdP;
  try {
    provider = new URL(issuer); callback = new URL(redirectURI);
    platformIdP = new URL(authentikOrigin);
  } catch { deny(); }
  if (platformIdP.protocol !== "https:" || platformIdP.port ||
      platformIdP.pathname !== "/" || platformIdP.search || platformIdP.hash ||
      platformIdP.username || platformIdP.password ||
      platformIdP.origin + "/" !== authentikOrigin + "/" ||
      provider.origin !== platformIdP.origin ||
      provider.protocol !== "https:" || provider.port || provider.username || provider.password ||
      provider.search || provider.hash || provider.href !== issuer ||
      provider.pathname !== `/application/o/txo-fabric-${tenantKey}/` ||
      callback.protocol !== "https:" || callback.port || callback.username || callback.password ||
      callback.search || callback.hash || callback.href !== redirectURI ) deny();
  // TenantBundle-generated legacy callback regex is:
  // ^https://[a-z0-9-]+-<tenant>\\.<domainSuffix>/auth/callback$
  const suffix = "-" + tenantKey + "." + domainSuffix;
  const appLabel = callback.hostname.endsWith(suffix)
    ? callback.hostname.slice(0, -suffix.length) : "";
  const tenantCallback = callback.pathname === "/auth/callback" &&
    /^[a-z0-9-]+$/.test(appLabel);
  const fabricCallback = callback.origin === "https://fabric.truxonline.com" &&
    callback.pathname === "/auth/tenants/" + tenantKey + "/callback";
  if (!tenantCallback && !fabricCallback) deny();
  const authorizationEndpoint = new URL("/application/o/authorize/", provider.origin);
  const tokenEndpoint = new URL("/application/o/token/", provider.origin);
  return { authorizationEndpoint, tokenEndpoint, fabricCallback };
}

function secretEqual(a, b) {
  if (!BASE64URL43.test(a || "") || !BASE64URL43.test(b || "")) return false;
  const aBytes = Buffer.from(a, "ascii");
  const bBytes = Buffer.from(b, "ascii");
  return timingSafeEqual(aBytes, bBytes);
}

/**
 * Staged, unmounted Fabric BFF OIDC authorization-code login coordinator.
 *
 * tenantKey, issuer, clientId, domainSuffix, redirectURI, authentikOrigin
 * MUST be resolved from trusted
 * server-side TenantBundle config; they are NOT parameters on begin/complete.
 * transactions MUST be a durable, tenant-keyed store with atomic insert and
 * atomic consume (delete-before-return), shared across BFF replicas, bounded
 * retention and no logging of PKCE verifiers, nonce or binding hashes.
 *
 * The BFF sets the returned browserCookie using Secure + HttpOnly +
 * SameSite=Lax + Path=/, with NO Domain attribute. On callback it reads
 * that exact __Host- cookie from the request, NOT a request body/header.
 * State is single-use even after failed cookie checking, code exchange,
 * validation, enrollment or upstream outage.
 *
 * enrollVerifiedHuman is a privileged Fabric backend integration. It MUST
 * authenticate the BFF service to the Fabric enrollment writer, bind the
 * server-owned tenant, verify the private ledger and active account, and
 * return a canonical Fabric user ID. This module never creates a session,
 * sets principal.verified or grants an OpenFGA permission.
 */
export function createFabricOIDCLogin({
  tenantKey, issuer, clientId, domainSuffix, redirectURI, authentikOrigin, transactions,
  enrollVerifiedHuman, resolveGlobalSession, fetchImpl = fetch, now = () => Date.now(),
}) {
  const endpoints = config({ tenantKey, issuer, clientId, domainSuffix, redirectURI, authentikOrigin });
  if (!transactions || typeof transactions.insert !== "function" ||
      typeof transactions.consume !== "function" ||
      typeof enrollVerifiedHuman !== "function" ||
      (endpoints.fabricCallback && typeof resolveGlobalSession !== "function") ||
      typeof fetchImpl !== "function" || typeof now !== "function") deny();
  const verifyLoginIDToken = createAuthentikIdentityVerifier({
    tenantKey, issuer, clientId, fetchImpl, now,
  });

  return Object.freeze({
    async begin({ globalSessionBinding } = {}) {
      try {
        if (endpoints.fabricCallback && !BASE64URL43.test(globalSessionBinding || "")) deny();
        const issuedAt = now();
        if (!Number.isSafeInteger(issuedAt) || issuedAt < 0) deny();
        const state = random(), nonce = random(), pkceVerifier = random();
        const browserBinding = random();
        // insert MUST be atomic and reject collision; never overwrite a live
        // transaction or accept an incomplete record on retry.
        await transactions.insert({
          tenantKey, state,
          value: {
            nonce, pkceVerifier, bindingDigest: digest(browserBinding),
            ...(endpoints.fabricCallback ? { globalSessionDigest: digest(globalSessionBinding) } : {}),
            issuedAt, issuer, clientId, redirectURI,
          },
          expiresAt: issuedAt + MAX_AGE_MS,
        });
        const url = new URL(endpoints.authorizationEndpoint.href);
        url.search = new URLSearchParams({
          response_type: "code", client_id: clientId, redirect_uri: redirectURI,
          scope: SCOPE, state, nonce,
          code_challenge: digest(pkceVerifier), code_challenge_method: "S256",
        }).toString();
        return Object.freeze({
          authorizationURL: url.href,
          browserCookie: Object.freeze({
            name: COOKIE_NAME, value: browserBinding,
            httpOnly: true, secure: true, sameSite: "Lax", path: "/",
            maxAge: 300,
          }),
        });
      } catch { deny(); }
    },

    async complete({ state, code, browserCookieValue, globalSessionBinding } = {}) {
      if (!BASE64URL43.test(state || "") || typeof code !== "string" ||
          !CODE.test(code) || !BASE64URL43.test(browserCookieValue || "") ||
          (endpoints.fabricCallback && !BASE64URL43.test(globalSessionBinding || ""))) deny();
      let transaction;
      try {
        // A failed callback burns the state. Must be atomic across BFF pods.
        transaction = await transactions.consume({ tenantKey, state });
      } catch { deny(); }
      if (!transaction || transaction.issuer !== issuer ||
          transaction.clientId !== clientId ||
          transaction.redirectURI !== redirectURI ||
          !BASE64URL43.test(transaction.nonce || "") ||
          !BASE64URL43.test(transaction.pkceVerifier || "") ||
          !BASE64URL43.test(transaction.bindingDigest || "") ||
          (endpoints.fabricCallback && !BASE64URL43.test(transaction.globalSessionDigest || "")) ||
          !Number.isSafeInteger(transaction.issuedAt)) deny();
      let t;
      try { t = now(); } catch { deny(); }
      if (!Number.isSafeInteger(t) || t < transaction.issuedAt ||
          t - transaction.issuedAt > MAX_AGE_MS ||
          !secretEqual(transaction.bindingDigest, digest(browserCookieValue)) ||
          (endpoints.fabricCallback && !secretEqual(transaction.globalSessionDigest, digest(globalSessionBinding)))) deny();

      let globalIdentity;
      if (endpoints.fabricCallback) {
        try { globalIdentity = await resolveGlobalSession(globalSessionBinding); }
        catch { deny(); }
        if (!validGlobalSessionIdentity(globalIdentity)) deny();
      }

      try {
        const form = new URLSearchParams({
          grant_type: "authorization_code", client_id: clientId,
          code, redirect_uri: redirectURI, code_verifier: transaction.pkceVerifier,
        });
        const response = await fetchImpl(endpoints.tokenEndpoint.href, {
          method: "POST", redirect: "error", cache: "no-store",
          headers: { Accept: "application/json",
            "Content-Type": "application/x-www-form-urlencoded" },
          body: form.toString(), signal: AbortSignal.timeout(3000),
        });
        if (response.status !== 200 || response.redirected ||
            Number(response.headers?.get?.("content-length") || 0) > MAX_TOKEN_RESPONSE) deny();
        const raw = await response.text();
        if (raw.length > MAX_TOKEN_RESPONSE) deny();
        const tokens = JSON.parse(raw);
        if (typeof tokens?.id_token !== "string" || tokens.id_token.length > 16384 ||
            tokens.token_type !== "Bearer") deny();
        // Only the IdP-signed identity proof is forwarded. Do not pass the
        // raw code, bearer tokens, nonce, browser cookie or headers to Fabric.
        const proof = await verifyLoginIDToken({
          idToken: tokens.id_token, expectedNonce: transaction.nonce,
        });
        if (endpoints.fabricCallback && globalIdentity.authentikUserUUID !== proof.authentikUserUUID) deny();
        const fabricUserId = await enrollVerifiedHuman(Object.freeze({
          tenantKey: proof.tenantKey, issuer: proof.issuer,
          oidcSubject: proof.oidcSubject,
          authentikUserUUID: proof.authentikUserUUID,
          ...(endpoints.fabricCallback ? {
            globalOidcIssuer: globalIdentity.issuer,
            globalOidcSubject: globalIdentity.oidcSubject,
          } : {}),
        }));
        if (typeof fabricUserId !== "string" || !FABRIC_USER_ID.test(fabricUserId)) deny();
        return Object.freeze({
          tenantKey, issuer, oidcSubject: proof.oidcSubject, fabricUserId,
        });
      } catch { deny(); }
    },
  });
}
