import { createHash, randomBytes, timingSafeEqual } from "node:crypto";
import { createAuthentikIdentityVerifier } from "./oidc-identity.mjs";

const STATE = /^[A-Za-z0-9_-]{43}$/;
const CODE = /^[\x21-\x7e]{1,4096}$/;
const TTL = 300_000;
const COOKIE = "__Host-txo-fabric-oidc";
const SCOPE = "openid profile email txo_fabric_identity";
const fail = () => { throw new Error("Global Fabric OIDC login denied"); };
const random = () => randomBytes(32).toString("base64url");
const digest = (s) => createHash("sha256").update(s).digest("base64url");

/**
 * Global Authentik browser login using the EXISTING signed OIDC verifier.
 * It uses the stable, platform-owned `webui` namespace in the shared
 * PostgreSQL OIDC transaction repository, NOT any user-selected tenant.
 *
 * Only cryptographically verified global Authentik identity is returned.
 * NO tenant membership, Fabric user ID, role, grant, session or
 * authenticated principal is created by this component.
 */
export function createGlobalOIDCLogin({
  authentikOrigin, expectedOrigin, transactions,
  fetchImpl = fetch, now = () => Date.now(),
}) {
  let origin, idp;
  try {
    origin = new URL(expectedOrigin);
    idp = new URL(authentikOrigin);
  } catch { fail(); }
  if (origin.protocol !== "https:" || origin.origin !== expectedOrigin ||
      origin.pathname !== "/" || origin.search || origin.hash ||
      origin.href !== expectedOrigin + "/" || origin.port ||
      idp.protocol !== "https:" || idp.origin !== authentikOrigin ||
      idp.pathname !== "/" || idp.search || idp.hash ||
      idp.href !== authentikOrigin + "/" || idp.port ||
      typeof transactions?.insert !== "function" ||
      typeof transactions?.consume !== "function" ||
      typeof fetchImpl !== "function" || typeof now !== "function") fail();

  const issuer = authentikOrigin + "/application/o/txo-fabric-webui/";
  const clientId = "txo-fabric-webui";
  const callback = expectedOrigin + "/auth/callback";
  // Existing strict verifier validates RS256, same-issuer JWKS, nonce,
  // client audience and signed immutable Authentik UUID. Its tenantKey
  // "webui" is a provider label, NOT a Fabric tenant.
  const verifyToken = createAuthentikIdentityVerifier({
    tenantKey: "webui", issuer, clientId, fetchImpl, now,
  });

  return Object.freeze({
    async begin() {
      try {
        const t = now();
        if (!Number.isSafeInteger(t) || t < 0 || !Number.isSafeInteger(t + TTL)) fail();
        const state = random(), nonce = random(), pkceVerifier = random();
        const binding = random();
        await transactions.insert({
          tenantKey: "webui", state,
          value: {
            nonce, pkceVerifier, bindingDigest: digest(binding),
            issuer, clientId, redirectURI: callback, issuedAt: t,
          },
          expiresAt: t + TTL,
        });
        const authorize = new URL("/application/o/authorize/", authentikOrigin);
        authorize.search = new URLSearchParams({
          response_type: "code", client_id: clientId,
          redirect_uri: callback, scope: SCOPE, state, nonce,
          code_challenge: digest(pkceVerifier), code_challenge_method: "S256",
        }).toString();
        return Object.freeze({
          authorizationURL: authorize.href, binding,
        });
      } catch { fail(); }
    },

    async complete({ state, code, binding } = {}) {
      if (!STATE.test(state || "") || typeof code !== "string" ||
          !CODE.test(code) || !STATE.test(binding || "")) fail();
      let transaction;
      try {
        // PostgreSQL DELETE...RETURNING atomically burns the transaction
        // across pods, BEFORE signature, code exchange or account lookup.
        transaction = await transactions.consume({ tenantKey: "webui", state });
      } catch { fail(); }
      if (!transaction || transaction.issuer !== issuer ||
          transaction.clientId !== clientId ||
          transaction.redirectURI !== callback ||
          !STATE.test(transaction.nonce || "") ||
          !STATE.test(transaction.pkceVerifier || "") ||
          !STATE.test(transaction.bindingDigest || "") ||
          !Number.isSafeInteger(transaction.issuedAt)) fail();
      let t;
      try { t = now(); } catch { fail(); }
      if (!Number.isSafeInteger(t) || t < transaction.issuedAt ||
          t - transaction.issuedAt > TTL) fail();
      const expected = Buffer.from(transaction.bindingDigest, "ascii");
      const received = Buffer.from(digest(binding), "ascii");
      if (expected.length !== received.length ||
          !timingSafeEqual(expected, received)) fail();

      try {
        const response = await fetchImpl(
          new URL("/application/o/token/", authentikOrigin).href, {
            method: "POST", redirect: "error", cache: "no-store",
            headers: {
              Accept: "application/json",
              "Content-Type": "application/x-www-form-urlencoded",
            },
            body: new URLSearchParams({
              grant_type: "authorization_code", code,
              client_id: clientId, redirect_uri: callback,
              code_verifier: transaction.pkceVerifier,
            }).toString(),
            signal: AbortSignal.timeout(3000),
          });
        if (response.status !== 200 || response.redirected ||
            Number(response.headers?.get?.("content-length") || 0) > 32768) fail();
        const raw = await response.text();
        if (raw.length > 32768) fail();
        const tokens = JSON.parse(raw);
        if (tokens?.token_type !== "Bearer" ||
            typeof tokens.id_token !== "string" ||
            tokens.id_token.length > 16384) fail();

        const proof = await verifyToken({
          idToken: tokens.id_token, expectedNonce: transaction.nonce,
        });
        return Object.freeze({
          issuer: proof.issuer,
          oidcSubject: proof.oidcSubject,
          authentikUserUUID: proof.authentikUserUUID,
        });
      } catch { fail(); }
    },
  });
}
export const GLOBAL_OIDC_BINDING_COOKIE = COOKIE;
