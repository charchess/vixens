/**
 * Portable WHATWG Request -> Response handler to be mounted ONLY behind an
 * Authentik/OIDC-backed Fabric BFF. authenticate MUST verify identity in the
 * backend; user-supplied headers or JSON must never establish a principal.
 * Origin + custom header checks protect browser-cookie deployments from CSRF.
 */
export function createChatHttpHandler({ authenticate, service, expectedOrigin }) {
  if (typeof authenticate !== "function" || !service ||
      ["create", "list", "get", "turn"].some((name) => typeof service[name] !== "function") ||
      typeof expectedOrigin !== "string" || !/^https:\/\/[^/]+$/.test(expectedOrigin)) {
    throw new Error("Chat HTTP security dependencies are required");
  }
  const json = (data, status = 200) => new Response(JSON.stringify(data), {
    status, headers: { "Content-Type": "application/json", "Cache-Control": "no-store" },
  });
  const fail = (status) => json({ error: status === 403 ? "Forbidden" : "Request failed" }, status);
  async function readBody(req) {
    if (Number(req.headers.get("content-length") || 0) > 100_512) throw new Error("bad body");
    const raw = await req.text();
    if (raw.length > 100_512) throw new Error("bad body");
    return JSON.parse(raw);
  }
  return async function handle(req) {
    let principal;
    try { principal = await authenticate(req); } catch { return fail(403); }
    if (!principal?.authenticated || !principal.subject || !principal.tenantKey) return fail(403);
    const route = new URL(req.url);
    // Exact host/Origin and fail-closed mutation checks.
    if (route.origin !== expectedOrigin) return fail(403);
    if (!["GET", "POST"].includes(req.method)) return fail(405);
    if (req.method === "POST" && (req.headers.get("origin") !== expectedOrigin ||
        req.headers.get("x-txo-request") !== "chat" ||
        !req.headers.get("content-type")?.startsWith("application/json"))) return fail(403);
    try {
      if (route.pathname === "/api/chat/threads" && req.method === "GET") {
        return json({ threads: await service.list({ principal }) });
      }
      if (route.pathname === "/api/chat/threads" && req.method === "POST") {
        const body = await readBody(req);
        if (!body || typeof body.agentKey !== "string") return fail(400);
        return json(await service.create({ principal, agentKey: body.agentKey }), 201);
      }
      const match = /^\/api\/chat\/threads\/([a-f0-9]{40})(?:\/(turns))?$/.exec(route.pathname);
      if (!match) return fail(404);
      const threadId = match[1];
      if (req.method === "GET" && !match[2]) return json(await service.get({ principal, threadId }));
      if (req.method === "POST" && match[2] === "turns") {
        const body = await readBody(req);
        if (typeof body?.input !== "string" || !body.input.trim() || body.input.length > 100000) return fail(400);
        await service.get({ principal, threadId });
        const encoder = new TextEncoder();
        const stream = new ReadableStream({
          async start(controller) {
            try {
              for await (const item of service.turn({ principal, threadId, input: body.input, signal: req.signal })) {
                controller.enqueue(encoder.encode(`data: ${JSON.stringify(item)}\n\n`));
              }
            } catch {
              controller.enqueue(encoder.encode('event: error\ndata: {"error":"Agent unavailable or forbidden"}\n\n'));
            } finally { controller.close(); }
          },
          cancel() { /* Request.signal is forwarded to Hermes; the host must abort it. */ },
        });
        return new Response(stream, { status: 200, headers: {
          "Content-Type": "text/event-stream", "Cache-Control": "no-store",
          "X-Content-Type-Options": "nosniff", "X-Accel-Buffering": "no",
        } });
      }
      return fail(404);
    } catch (err) {
      // No upstream failures, resolver details or credentials in HTTP errors.
      return fail(err?.message === "Forbidden" ? 403 : 400);
    }
  };
}
