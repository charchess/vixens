import { createGlobalOIDCLogin, GLOBAL_OIDC_BINDING_COOKIE } from "./global-oidc-login.mjs";
import { createGlobalWebUIHandler } from "./global-webui-http.mjs";

const SESSION_COOKIE = "__Host-txo-fabric-session";
const AUTH_REQUEST = "auth";
const BASE64URL43 = /^[A-Za-z0-9_-]{43}$/;
const OIDC_SUB = /^[!-~]{1,255}$/;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const fail = () => { throw new Error("Global Fabric authentication unavailable"); };
const headers = () => new Headers({
  "Cache-Control": "no-store",
  "Pragma": "no-cache",
  "Referrer-Policy": "no-referrer",
  "X-Content-Type-Options": "nosniff",
  "Vary": "Cookie",
});
const cookieValue = (request, name) => {
  const raw = request.headers.get("cookie");
  if (raw === null) return null;
  if (raw.length > 8192) fail();
  let matched = null;
  for (const pair of raw.split(";")) {
    const part = pair.trim();
    const i = part.indexOf("=");
    if (i < 1) fail();
    if (part.slice(0, i) !== name) continue;
    if (matched !== null || !BASE64URL43.test(part.slice(i + 1))) fail();
    matched = part.slice(i + 1);
  }
  return matched;
};
const setCookie = (name, value, maxAge) =>
  `${name}=${value}; Path=/; Max-Age=${maxAge}; HttpOnly; Secure; SameSite=Lax`;
const clearCookie = (name) =>
  `${name}=; Path=/; Max-Age=0; HttpOnly; Secure; SameSite=Lax`;
const denied = (status = 403) => new Response("Access denied", { status, headers: headers() });

/**
 * Browser routes for the SINGLE Fabric global origin. Uses a real signed
 * Authentik global OIDC code/PKCE coordinator and a cross-replica SQL-backed
 * session store. The injected active-human verifier is a privileged
 * Fabric-owned Authentik lookup and MUST authenticate its service-to-service
 * transport, reject stale/inactive users and confirm the exact immutable UUID
 * before issuing a session. It never allocates a Fabric user or grant.
 *
 * This adapter is WHATWG Request/Response source, not a deployed HTTP
 * process. No safe public exposure until its real privileged dependencies,
 * TLS ingress, CNPG migrations and NetworkPolicy are installed.
 */
export function createGlobalAuthHttp({
  expectedOrigin, login, sessions, verifyActiveHuman,
}) {
  if (typeof expectedOrigin !== "string" || !/^https:\/\/[^/?#]+$/.test(expectedOrigin) ||
      typeof login?.begin !== "function" || typeof login?.complete !== "function" ||
      typeof sessions?.issue !== "function" ||
      typeof sessions?.authenticate !== "function" ||
      typeof sessions?.revoke !== "function" ||
      typeof verifyActiveHuman !== "function") fail();

  async function authenticate(request) {
    const token = cookieValue(request, SESSION_COOKIE);
    if (!token) return null;
    return sessions.authenticate(token);
  }

  async function handle(request) {
    let url;
    try { url = new URL(request.url); } catch { return denied(); }
    if (url.origin !== expectedOrigin || url.username || url.password ||
        url.hash) return denied();
    if (url.pathname === "/auth/start") {
      if (request.method !== "GET" || url.search) return denied(405);
      try {
        const result = await login.begin();
        const h = headers();
        // Redirects only to the trusted Authentik global authorize endpoint
        // from the pinned coordinator, never from query or user input.
        h.set("Location", result.authorizationURL);
        h.append("Set-Cookie",
          setCookie(GLOBAL_OIDC_BINDING_COOKIE, result.binding, 300));
        return new Response(null, { status: 302, headers: h });
      } catch { return denied(503); }
    }
    if (url.pathname === "/auth/callback") {
      const h = headers();
      // Always delete the temporary OAuth binding cookie on callback.
      h.append("Set-Cookie", clearCookie(GLOBAL_OIDC_BINDING_COOKIE));
      if (request.method !== "GET") return new Response("Access denied", { status: 405, headers: h });
      try {
        const params = [...url.searchParams.entries()];
        if (params.length < 2 || params.length > 3 ||
            !params.some(([k]) => k === "state") ||
            !params.some(([k]) => k === "code") ||
            new Set(params.map(([k]) => k)).size !== params.length ||
            params.some(([k]) => !["state", "code", "iss"].includes(k))) fail();
        // Authorization Server Issuer Identification (RFC 9207) is
        // optional, but if supplied it MUST match our pinned global IdP.
        const state = url.searchParams.get("state");
        const code = url.searchParams.get("code");
        const iss = url.searchParams.get("iss");
        const binding = cookieValue(request, GLOBAL_OIDC_BINDING_COOKIE);
        if (!binding || !BASE64URL43.test(state || "") || !code) fail();
        if (iss && iss !== undefined) {
          // A trusted actual issuer is checked inside complete() using
          // the signed ID token; refuse unsolicited issuer parameters.
          // We intentionally do not accept arbitrary iss values here.
          // See the exact value from the signed proof below.
          if (typeof iss !== "string" || !iss.startsWith("https://") ||
              iss.includes("\r") || iss.includes("\n")) fail();
        }
        const proof = await login.complete({ state, code, binding });
        if (iss !== null && iss !== proof.issuer) fail();
        if (!OIDC_SUB.test(proof.oidcSubject || "") ||
            !UUID.test(proof.authentikUserUUID || "") ||
            typeof proof.issuer !== "string") fail();
        // BFF must not mint even a global session for an inactive user.
        const live = await verifyActiveHuman(Object.freeze({
          issuer: proof.issuer,
          oidcSubject: proof.oidcSubject,
          authentikUserUUID: proof.authentikUserUUID,
        }));
        if (!live || live.accountActive !== true || live.fresh !== true ||
            live.issuer !== proof.issuer ||
            live.oidcSubject !== proof.oidcSubject ||
            live.authentikUserUUID !== proof.authentikUserUUID) fail();

        const session = await sessions.issue(proof);
        if (!BASE64URL43.test(session?.token || "") ||
            !Number.isSafeInteger(session.maxAge) ||
            session.maxAge <= 0 || session.maxAge > 8 * 3600) fail();
        // The Authentik authorization code and state never reach the
        // landing URL or WebUI JavaScript.
        h.set("Location", expectedOrigin + "/");
        h.append("Set-Cookie",
          setCookie(SESSION_COOKIE, session.token, session.maxAge));
        return new Response(null, { status: 303, headers: h });
      } catch { return new Response("Access denied", { status: 403, headers: h }); }
    }
    if (url.pathname === "/auth/logout") {
      if (request.method !== "POST" || url.search ||
          request.headers.get("origin") !== expectedOrigin ||
          request.headers.get("x-txo-request") !== AUTH_REQUEST) return denied();
      try {
        const token = cookieValue(request, SESSION_COOKIE);
        if (token) await sessions.revoke(token);
        const h = headers();
        h.append("Set-Cookie", clearCookie(SESSION_COOKIE));
        return new Response(null, { status: 204, headers: h });
      } catch { return denied(503); }
    }
    return denied(404);
  }

  return Object.freeze({ handle, authenticate });
}

/**
 * Secure source-level assembly of global login and workspace routing.
 * DOES NOT instantiate any fake tenant mapping, memory DB or admin grants.
 * Platform owner must provide the real private membership/agent resolvers.
 */
export function createGlobalWebUIApp({
  expectedOrigin, authentikOrigin, transactions, sessions,
  verifyActiveHuman, listTenantCatalog, resolveTenantAccess,
  getTenantChat, resolvePlatformAccess,
  fetchImpl = fetch, now = () => Date.now(),
}) {
  const login = createGlobalOIDCLogin({
    expectedOrigin, authentikOrigin, transactions, fetchImpl, now,
  });
  const auth = createGlobalAuthHttp({
    expectedOrigin, login, sessions, verifyActiveHuman,
  });
  const app = createGlobalWebUIHandler({
    expectedOrigin,
    expectedIssuer: authentikOrigin + "/application/o/txo-fabric-webui/",
    authenticate: auth.authenticate,
    listTenantCatalog, resolveTenantAccess, getTenantChat,
    resolvePlatformAccess,
  });
  return async function handle(request) {
    let url;
    try { url = new URL(request.url); } catch { return denied(); }
    if (url.pathname.startsWith("/auth/")) return auth.handle(request);
    return app(request);
  };
}
