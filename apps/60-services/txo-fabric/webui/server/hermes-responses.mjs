import { parseSSE } from "./hermes-events.mjs";
import { requireInternalHermesURL } from "./hermes-client.mjs";

const KEY = /^v[1-9][0-9]*_[A-Za-z0-9_-]{43,128}$/;
const TOOL = /^[A-Za-z0-9_.:-]{1,80}$/;
const SAFE_TEXT_LIMIT = 8192;
const INPUT_LIMIT = 100000;
const fail = () => { throw new Error("Agent is unavailable"); };

/**
 * A scoped opaque conversation identifier. Production must inject a
 * versioned, secret-keyed HMAC/base64url derivation. This module cannot read,
 * generate, cache or select a signing secret. Raw canonical identity values
 * and URL/credential material may not be used as the key.
 */
export function requireHermesConversationKey(key, scope = {}) {
  if (typeof key !== "string" || key.length > 256 || !KEY.test(key) ||
      /[\r\n\0]/.test(key) ||
      Object.values(scope).some(value => typeof value === "string" && value === key)) fail();
  return key;
}

export function requireHermesResponsesInput(input) {
  if (typeof input !== "string" || !input.trim() || input.length > INPUT_LIMIT ||
      /[\0]/.test(input)) fail();
  return input;
}

function decode(event) {
  let data;
  try { data = JSON.parse(event.data); } catch { fail(); }
  if (!data || typeof data !== "object" || Array.isArray(data) ||
      (data.type !== undefined && data.type !== event.event)) fail();
  return data;
}

function textDelta(payload, commentary) {
  if (payload.phase === "commentary" || payload.phase === "reasoning") return null;
  // Without a canonical output index we cannot distinguish final answer
  // deltas from an in-progress commentary message. Deny rather than leak.
  if (!Number.isSafeInteger(payload.output_index) || payload.output_index < 0) fail();
  if (commentary.has(payload.output_index)) return null;
  if (typeof payload.delta !== "string" || payload.delta.length < 1 ||
      payload.delta.length > SAFE_TEXT_LIMIT || payload.delta.includes("\0")) fail();
  return { type: "text-delta", text: payload.delta };
}

/** Strictly separate Responses events from the legacy native-session SSE. */
export function projectHermesResponsesEvent(event, commentary = new Set()) {
  const known = new Set([
    "response.output_text.delta", "response.output_item.added",
    "response.output_item.done", "response.completed", "response.failed",
  ]);
  if (!known.has(event.event)) return null; // includes reasoning, IDs and usage
  const payload = decode(event);
  if (event.event === "response.output_item.added" &&
      payload.item?.type === "message" &&
      (payload.item.phase === "commentary" || payload.item.phase === "reasoning")) {
    if (!Number.isSafeInteger(payload.output_index) ||
        payload.output_index < 0 || commentary.size >= 256) fail();
    commentary.add(payload.output_index);
    return null;
  }
  if (event.event === "response.output_text.delta") return textDelta(payload, commentary);
  if (event.event === "response.completed") return { type: "done", status: "completed" };
  if (event.event === "response.failed") return { type: "done", status: "failed" };
  if (payload.item?.type !== "function_call" ||
      typeof payload.item.name !== "string" || !TOOL.test(payload.item.name)) return null;
  if (event.event === "response.output_item.added") {
    return { type: "tool-started", tool: payload.item.name };
  }
  if (event.event === "response.output_item.done") {
    return { type: "tool-completed", tool: payload.item.name,
      ...(payload.item.status === "failed" ? { failed: true } :
        payload.item.status === "completed" ? { failed: false } : {}) };
  }
  return null;
}

/**
 * One authenticated Responses POST, no retry/fallback/previous_response_id.
 * Input, target and opaque continuity key come ONLY from trusted BFF paths.
 * Hermes persists named-conversation state on its own agent PVC.
 */
export async function* streamHermesResponses({
  baseURL, apiKey, conversationKey, input, fetchImpl = fetch, signal,
} = {}) {
  // Validate everything before ANY network operation.
  let origin;
  try {
    origin = requireInternalHermesURL(baseURL);
    requireHermesConversationKey(conversationKey, { baseURL, apiKey });
    requireHermesResponsesInput(input);
    if (typeof apiKey !== "string" || apiKey.length < 16 || apiKey.length > 2048 ||
        /[\r\n\0]/.test(apiKey) || typeof fetchImpl !== "function" ||
        (signal !== undefined && !(signal instanceof AbortSignal))) fail();
  } catch { fail(); }
  const deadline = AbortSignal.timeout(180000);
  const requestSignal = signal ? AbortSignal.any([signal, deadline]) : deadline;
  try {
    const response = await fetchImpl(origin + "/v1/responses", {
      method: "POST",
      headers: {
        Authorization: "Bearer " + apiKey,
        "X-Hermes-Session-Key": conversationKey,
        "Content-Type": "application/json",
        Accept: "text/event-stream",
      },
      body: JSON.stringify({ input, conversation: conversationKey, stream: true, store: true }),
      redirect: "error", cache: "no-store", signal: requestSignal,
    });
    if (!response?.ok || response.redirected === true ||
        !response.headers?.get?.("content-type")?.toLowerCase().startsWith("text/event-stream") ||
        !response.body) fail();
    let finished = false;
    const commentary = new Set();
    for await (const event of parseSSE(response.body, 16384)) {
      if (event.data === "[DONE]") continue;
      if (finished) fail(); // no second completion or trailing privileged events
      const projected = projectHermesResponsesEvent(event, commentary);
      if (!projected) continue;
      if (projected.type === "done") finished = true;
      yield projected;
    }
    if (!finished) fail(); // no fabricated success on an incomplete SSE
  } catch { fail(); } // redact all transport/provider/body/parser details
}
