/**
 * Parse a WHATWG response body as an SSE stream. It accepts events spanning
 * multiple UTF-8 chunks and never buffers an unbounded upstream event.
 */
export async function* parseSSE(body, maxEventChars = 65536) {
  if (!body || typeof body.getReader !== "function") {
    throw new Error("Hermes stream is missing");
  }
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let eventName = "message";
  let data = [];
  let eventSize = 0;

  function consume(line) {
    if (line === "") {
      const result = data.length ? { event: eventName, data: data.join("\n") } : null;
      eventName = "message";
      data = [];
      eventSize = 0;
      return result;
    }
    if (line.startsWith(":")) return null;
    const index = line.indexOf(":");
    const field = index === -1 ? line : line.slice(0, index);
    let value = index === -1 ? "" : line.slice(index + 1);
    if (value.startsWith(" ")) value = value.slice(1);
    if (field === "event") eventName = value;
    if (field === "data") {
      eventSize += value.length;
      if (eventSize > maxEventChars) throw new Error("Hermes event exceeds size limit");
      data.push(value);
    }
    return null;
  }

  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let newline;
      while ((newline = buffer.indexOf("\n")) !== -1) {
        let line = buffer.slice(0, newline);
        buffer = buffer.slice(newline + 1);
        if (line.endsWith("\r")) line = line.slice(0, -1);
        const item = consume(line);
        if (item) yield item;
      }
      if (buffer.length > maxEventChars) throw new Error("Hermes event exceeds size limit");
    }
    buffer += decoder.decode();
    if (buffer) consume(buffer.endsWith("\r") ? buffer.slice(0, -1) : buffer);
    const last = consume("");
    if (last) yield last;
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}

const PUBLIC_EVENTS = new Set([
  "assistant.delta",
  "tool.started",
  "tool.completed",
  "run.completed",
  "run.failed",
  "run.cancelled",
]);

/** Only project user-facing, non-secret events. Do not forward tool arguments,
 * outputs, model private commentary, internal reasoning or upstream errors.
 */
export function projectChatEvent(message) {
  if (!PUBLIC_EVENTS.has(message.event)) return null;
  let payload;
  try {
    payload = JSON.parse(message.data);
  } catch {
    throw new Error("Malformed Hermes event");
  }
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
    throw new Error("Malformed Hermes event");
  }
  switch (message.event) {
    case "assistant.delta": {
      const text = typeof payload.text === "string" ? payload.text : payload.delta;
      return typeof text === "string" ? { type: "text-delta", text } : null;
    }
    case "tool.started":
    case "tool.completed": {
      // Tool name is informational. Never expose arguments, previews or results.
      const tool = typeof payload.tool === "string" && /^[A-Za-z0-9_.:-]{1,80}$/.test(payload.tool) ? payload.tool : "tool";
      return { type: message.event === "tool.started" ? "tool-started" : "tool-completed", tool,
        ...(message.event === "tool.completed" ? { failed: payload.error === true } : {}) };
    }
    case "run.completed":
      return { type: "done", status: "completed" };
    case "run.failed":
      return { type: "done", status: "failed" };
    case "run.cancelled":
      return { type: "done", status: "cancelled" };
    default:
      return null;
  }
}
