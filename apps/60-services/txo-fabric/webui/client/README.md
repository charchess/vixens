# TXO Fabric WebUI — assistant-ui front-end spike

Prototype source for #3981 / #3832 / #3807. **Not deployed and not yet usable on hAIrem or Indiba.** Do not expose it as a public application without the identity/session/service integrations below.

## Functionality in source

- React and `@assistant-ui/react` LocalRuntime, with a French responsive shell and accessible chat primitives.
- Lists only AgentIdentity entries returned by `GET /api/chat/agents` after server-side Fabric grants are checked.
- Keeps mounted runtime state separate while switching among multiple authorized agents (A/B/A).
- Starts an opaque Fabric thread with `POST /api/chat/threads`, then streams chat via `POST /api/chat/threads/<id>/turns`.
- Text deltas render incrementally, and safe tool progress is shown without tool inputs/outputs. Cancel sends an AbortSignal.
- No provider keys, CPA tokens, Hermes credentials, cluster coordinates or tenant selector in the browser API.
- Settings and tenant administration are **not implemented**; navigation is marked as planned, not functional.

The client cannot call Hermes directly: it requires the Fabric BFF's same-origin API, with authenticated session and CSRF protections, and trusted backends. The server callback `listTenantAgents` must be supplied by real Fabric IAM and resource discovery. There is no hardcoded sample agent for live usage.

## Build and tests

From this directory: `npm install --ignore-scripts && npm run build && node --test`.
CI: `.github/workflows/txo-fabric-webui-client-ci.yaml`. Dependencies are pinned at the top level for the prototype; **before a deployable image**, commit a reviewed dependency lockfile and implement supply-chain build/pin GitOps workflow.

## Remaining product gates

1. Fabric BFF Authentik/OIDC session verification, tenant context and authorization (no forwarded unsigned headers).
2. Durable private thread repository with per-session atomic leases, history and proper replay/reconnect after reload.
3. Privately enabled Hermes API Server, per-agent credential, network policies and measured pinned protocol.
4. Resolve agent list through Fabric canonical IDs and authorized user/group grants; do not trust browser selections.
5. Integrate same application `/settings` and tenant `/admin` via #3979/#3980.
6. CI, candidate dev and **manual physical two-agent acceptance** before production promotion.

This is deliberately a **working source prototype behind mocked or future BFF APIs**, not a launched service.
