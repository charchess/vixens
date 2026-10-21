import { createServer } from "node:http";
import { Readable } from "node:stream";
import { pipeline } from "node:stream/promises";
import { createGlobalWebUIApp } from "./global-auth-http.mjs";
import { createPostgresOIDCTransactions } from "./oidc-postgres-transactions.mjs";
import { createPostgresGlobalSessions } from "./global-postgres-sessions.mjs";
import { createVerifiedTenantBridgeAccess } from "./private-bridge-client.mjs";
import { createDeclaredBrowserArtifacts } from "./browser-artifacts.mjs";

export const WEBUI_PUBLIC_ORIGIN = "https://fabric.truxonline.com";
export const AUTHENTIK_PUBLIC_ORIGIN = "https://authentik.truxonline.com";

const NO_STORE = Object.freeze({
  "Cache-Control": "no-store",
  "X-Content-Type-Options": "nosniff",
  "Referrer-Policy": "no-referrer",
});
const deny = (status = 404) => new Response("Unavailable", { status, headers: NO_STORE });
const OFF_ROUTES = new Set(["/healthz", "/readyz"]);
const ENTRY_HEADERS = Object.freeze({
  ...NO_STORE,
  "Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
  "Content-Type": "text/html; charset=utf-8",
  "X-Frame-Options": "DENY",
});
const ENTRY_PAGE = "<!doctype html><html lang=\"fr\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Fabric</title></head><body><main><h1>Fabric</h1><p>Le service est en préparation.</p></main></body></html>";

function entryPage(method) {
  return new Response(method === "HEAD" ? null : ENTRY_PAGE, { status: 200, headers: ENTRY_HEADERS });
}

/**
 * The imported Authentik OIDC, durable sessions and multi-tenant routing
 * contracts are composed only with ACTUAL privileged providers. No fallback
 * test/mocked user verification, tenant catalog, chat adapter or platform
 * authorization callback is permitted in the runnable feature-OFF process.
 */
export function composeGlobalBFF({
  expectedOrigin, authentikOrigin, pool, verifyActiveHuman,
  listTenantCatalog, bridgeClient, getTenantChat, browserArtifacts,
  browserEnabled = false, bridgeEnabled = false,
  resolvePlatformAccess, fetchImpl,
} = {}) {
  // Never compose a browser-visible BFF using a caller-supplied permissive
  // resolveTenantAccess callback. The source-only bridge path is mandatory
  // and both gates must be explicitly approved. Runtime remains health-only.
  if (browserEnabled !== true || bridgeEnabled !== true ||
      typeof pool?.query !== "function" ||
      typeof verifyActiveHuman !== "function" ||
      typeof listTenantCatalog !== "function" ||
      typeof bridgeClient?.resolve !== "function" || typeof bridgeClient?.enroll !== "function" ||
      typeof getTenantChat !== "function" || typeof browserArtifacts?.get !== "function") {
    throw new Error("Global Fabric BFF privileged dependencies unavailable");
  }
  const resolveTenantAccess = createVerifiedTenantBridgeAccess({
    browserEnabled, bridgeEnabled, verifyActiveHuman, bridgeClient,
  });
  const transactions = createPostgresOIDCTransactions({ pool });
  const sessions = createPostgresGlobalSessions({ pool });
  return createGlobalWebUIApp({
    expectedOrigin, authentikOrigin, transactions, sessions,
    verifyActiveHuman, listTenantCatalog, resolveTenantAccess,
    getTenantChat, browserArtifacts, resolvePlatformAccess,
    enrollTenant: async ({ tenantKey, issuer, oidcSubject, authentikUserUUID } = {}) =>
      bridgeClient.enroll(Object.freeze({
        tenantKey, oidcIssuer: issuer, oidcSubject, authentikUserUUID,
      })),
    fetchImpl,
  });
}

/**
 * Production composition boundary. Gate-on accepts no inferred/default
 * identity, catalog, bridge, chat or browser provider. Deployments must wire
 * every privileged provider explicitly; otherwise startup fails before listen.
 */
export async function createProductionBFFApp({ env = process.env } = {}) {
  const enabled = resolveRuntimeGate(env);
  // Gate-off is an isolation boundary: do not inspect caller-provided factories
  // or provider objects, which could read credentials or create privileged clients.
  if (!enabled) return Object.freeze({ enabled: false, app: null });

  // The gate precedes the production-only import. A disabled process cannot
  // load, select or construct a privileged Responses provider.
  const { createProductionResponsesComposition } = await import("./production-responses-composition.mjs");
  return createProductionResponsesComposition();
}

/**
 * Feature-off has NO route that reads cookies, sends OIDC redirects, issues
 * sessions or dispatches to a privileged adapter. Health requires no DB,
 * IAM, FGA or bridge access, so a missing secret never creates a session.
 * Feature-on composition is deliberately impossible without a real
 * separately reviewed app handler and all its server-side dependencies.
 */
export function createPrivateBFFHandler({ enabled = false, app, ready } = {}) {
  if (typeof enabled !== "boolean") throw new Error("Invalid WebUI BFF feature gate");
  if (enabled && (typeof app !== "function" || typeof ready !== "function")) {
    throw new Error("Unverified WebUI BFF activation");
  }
  return async function handle(request) {
    let url;
    try { url = new URL(request.url); }
    catch { return deny(); }
    if (url.username || url.password || url.hash || url.search ||
        !["GET", "HEAD", "POST"].includes(request.method)) return deny();
    if (OFF_ROUTES.has(url.pathname)) {
      if (request.method !== "GET" && request.method !== "HEAD") return deny(405);
      const ok = url.pathname === "/healthz" || !enabled || await ready().catch(() => false);
      return new Response(request.method === "HEAD" ? null : (ok ? "ok" : "unavailable"), {
        status: ok ? 200 : 503, headers: NO_STORE,
      });
    }
    if (!enabled && url.pathname === "/" && (request.method === "GET" || request.method === "HEAD")) {
      return entryPage(request.method);
    }
    if (!enabled) return deny();
    try {
      if (await ready() !== true) return deny(503);
      const response = await app(request);
      return response instanceof Response ? response : deny(503);
    } catch { return deny(503); }
  };
}

// HTTP adapter is a real Node server, not an in-memory WHATWG simulation.
// Strictly bounded request bodies; never dump headers, cookies or tokens.
export function createPrivateBFFServer({
  handle, expectedOrigin = WEBUI_PUBLIC_ORIGIN, maxBodyBytes = 1024 * 1024,
} = {}) {
  if (typeof handle !== "function" || !Number.isSafeInteger(maxBodyBytes) ||
      maxBodyBytes < 1 || maxBodyBytes > 1024 * 1024 ||
      expectedOrigin !== WEBUI_PUBLIC_ORIGIN) {
    throw new Error("Private BFF server configuration invalid");
  }
  const server = createServer(async (req, res) => {
    const disconnect = new AbortController();
    const onDisconnect = () => { if (!res.writableFinished) disconnect.abort(); };
    res.once("close", onDisconnect);
    try {
      const chunks = [];
      let size = 0;
      // Even while gated OFF, do not buffer an arbitrarily large body.
      if (req.method !== "GET" && req.method !== "HEAD") {
        for await (const chunk of req) {
          size += chunk.length;
          if (size > maxBodyBytes) {
            res.writeHead(413, NO_STORE);
            res.end("Unavailable");
            return;
          }
          chunks.push(chunk);
        }
      }
      const path = req.url || "/";
      if (path.length > 4096 || !path.startsWith("/") ||
          path.startsWith("//") || path.includes("\\") ||
          /[\u0000-\u001f]/.test(path)) {
        res.writeHead(404, NO_STORE);
        res.end("Unavailable");
        return;
      }
      const request = new Request(expectedOrigin + path, {
        method: req.method, headers: req.headers, signal: disconnect.signal,
        ...((req.method === "GET" || req.method === "HEAD") ? {} : { body: Buffer.concat(chunks) }),
      });
      const response = await handle(request);
      const headers = Object.fromEntries(response.headers);
      headers["cache-control"] = "no-store";
      res.writeHead(response.status, headers);
      if (req.method === "HEAD") {
        res.end();
        return;
      }
      if (response.headers.get("content-type")?.toLowerCase().startsWith("text/event-stream")) {
        // Browser SSE must be incremental; buffering until completion breaks
        // tool progress and cancellation. pipeline propagates backpressure and
        // disconnects through the Request.signal to the Hermes adapter.
        if (!response.body) throw new Error("Missing SSE body");
        await pipeline(Readable.fromWeb(response.body), res, { signal: disconnect.signal });
        return;
      }
      const data = Buffer.from(await response.arrayBuffer());
      if (data.length > maxBodyBytes) throw new Error("Response exceeds bounded limit");
      res.end(data);
    } catch {
      if (!res.destroyed) {
        if (!res.headersSent) res.writeHead(503, NO_STORE);
        res.end("Unavailable");
      }
    } finally {
      res.off("close", onDisconnect);
    }
  });
  server.requestTimeout = 8000;
  server.headersTimeout = 5000;
  server.keepAliveTimeout = 5000;
  server.maxHeadersCount = 32;
  return server;
}

export function resolveRuntimeGate(env = process.env) {
  const browserEnabled = env.TXO_WEBUI_BFF_BROWSER_ENABLED;
  const bridgeEnabled = env.TXO_FABRIC_PRIVATE_BRIDGE_ENABLED;
  if (browserEnabled === "false" && bridgeEnabled === "false") return false;
  if (browserEnabled === "true" && bridgeEnabled === "true") return true;
  throw new Error("WebUI browser and private bridge gates must be jointly enabled");
}
