# Tenant-scoped LLM credential pools

Status: **target implementation contract for #3868**. The current production-shaped
gateway still routes `txo-default` and `txo-embedding` through OpenRouter. Nothing
in this document means that a ChatGPT/Codex credential pool is active until its
controller resources and physical acceptance tests have landed.

## Goal and trust boundary

TXO Fabric must let one tenant own several upstream LLM credentials without ever
giving those credentials to Hermes or allowing another tenant to consume them.

The public contract remains:

```text
Hermes
  |
  | Fabric virtual key
  | public logical model, e.g. txo-default
  v
TXO AI Gateway
  |
  | authenticated, tenant-scoped resolution
  v
tenant-private provider pool
  |
  +-- credential worker A
  +-- credential worker B
  +-- ...
```

The agent selects a public logical model. Fabric/LiteLLM selects the provider
deployment and credential. An agent must never select an upstream account.

## Verified LiteLLM v1.102.1 constraints

The gateway is pinned to
`ghcr.io/berriai/litellm-non_root:v1.102.1`. The design below was checked
against that exact source tag rather than against current upstream documentation.

### ChatGPT OAuth is process-scoped

In `litellm/llms/chatgpt/authenticator.py`, `Authenticator` resolves exactly one
`CHATGPT_TOKEN_DIR` and one `CHATGPT_AUTH_FILE` from process environment. Reads,
device authorization and refresh all use that file, and refresh writes the updated
OAuth state back to it.

Therefore a single LiteLLM process with several mounted ChatGPT auth files does not
provide a trustworthy account-selection boundary. For v0.1, **one ChatGPT OAuth
credential means one isolated worker process**.

### Virtual-key aliases can hide tenant-private routes

In v1.102.1, `/key/generate` accepts an `aliases` map. During request setup,
`_update_model_if_key_alias_exists` rewrites the requested model from the
authenticated virtual key's alias map before provider routing.

Fabric can therefore mint an AgentIdentity key with a contract such as:

```json
{
  "models": ["txo-default"],
  "aliases": {
    "txo-default": "txo-tenant-ten00001-default"
  }
}
```

Hermes still requests `txo-default`; the internal group is never part of its
configuration. Because key aliases are intentionally capable of redirecting model
selection, **only the Fabric control plane may author them**. Tenant/user input must
never be copied into this map unchecked.

### Dynamic frontend deployments are available but must be enabled deliberately

The exact v1.102.1 proxy implements `POST /model/new`. Persisting and reconciling
those models through the proxy database is conditional on
`store_model_in_db = true` / `STORE_MODEL_IN_DB=True`.

The current TXO gateway does not enable that behavior. Enabling it and making the
operator own model registration/deletion is a later #3868 implementation slice; it
must not be silently switched on as part of the alias-only substrate.

## Canonical identities and naming

Tenant ownership uses the immutable Fabric `TenantBundle.spec.tenantId`, not an
upstream account name.

Recommended internal naming:

```text
public logical model:
  txo-default

tenant-private model group:
  txo-tenant-<lowercase tenantId>-default
  example: txo-tenant-ten00001-default

credential identity:
  Fabric non-secret credential object
  tenantRef + credentialKey + provider

worker resources:
  txo-llm-<tenantId>-<credentialKey>
```

Provider account identifiers, OAuth access tokens, refresh tokens and auth files are
not part of AgentIdentity, TenantBundle, virtual-key metadata, logs or status.

## Credential ownership model

Each upstream credential needs an explicit Fabric-owned non-secret identity with at
least:

- `tenantRef`;
- immutable non-secret `credentialKey`;
- provider type (`ChatGPT` first);
- administrative owner/re-auth authority;
- allowed logical model families;
- desired lifecycle state such as enabled, drained or revoked.

Observed status may expose only log-safe operational data:

- `PendingAuth`, `Healthy`, `Degraded`, `RateLimited`, `Cooldown`,
  `ReauthRequired`, `Disabled`;
- last successful request time;
- last failure class/status, without response bodies that may contain secrets;
- provider-reported remaining quota when trustworthy;
- `nextEligibleAt` only when a reliable reset/retry time exists;
- an explicit unknown-recovery state otherwise.

The concrete CRD shape is implemented in a later slice. Secret material is never a
CRD spec/status field.

## OAuth state storage

ChatGPT OAuth state is mutable because LiteLLM refreshes and rewrites its auth file.
It therefore cannot be modeled as immutable Git data and should not be treated as a
normal one-way ExternalSecret.

For the v0.1 proof:

- each credential receives a dedicated persistent volume in the Fabric management
  namespace;
- only that credential's authorization job and worker may mount it;
- the worker receives `CHATGPT_TOKEN_DIR` pointing at that volume;
- the auth file is never mounted into Hermes, Hindsight or another credential
  worker;
- deleting or retaining an AgentIdentity PVC has no effect on provider access;
- disabling a credential removes its worker from routing immediately;
- revocation invalidates selection first, then removes/invalidates the persisted
  OAuth state according to the credential lifecycle policy.

The storage implementation must be encrypted/protected by the platform storage
boundary. A later hardening slice may move to a dedicated mutable credential service
if required; the isolation contract does not depend on the backing implementation.

## Device authorization

Interactive OAuth must be an administrative workflow, never a side effect of a
tenant inference request.

Target flow:

1. operator/admin requests connect or re-auth for one credential identity;
2. a short-lived authorization workload mounts only that credential's state volume;
3. it runs the v1.102.1 ChatGPT device flow and exposes the verification URL/user
   code through an authenticated admin surface;
4. LiteLLM writes the resulting auth state to the credential volume;
5. the worker becomes eligible only after the state is valid;
6. ordinary refresh happens inside the isolated worker and survives restart.

A worker with missing/invalid auth must be unready and absent from the frontend
eligible set. It must not initiate an interactive device flow because a Hermes
request happened to arrive.

## Worker topology

Each ChatGPT credential gets one isolated LiteLLM worker:

```text
txo-ai-gateway frontend
        |
        +--> tenant TEN00001 private group
        |      +--> worker credential-a
        |      +--> worker credential-b
        |
        +--> tenant TEN00002 private group
               +--> worker credential-a
```

Worker properties:

- same explicitly pinned LiteLLM release until an upgrade is reviewed;
- static ChatGPT provider configuration local to the worker;
- one credential state volume;
- ClusterIP-only service;
- ingress allowed only from the Fabric-facing gateway;
- no Fabric virtual-key database and no tenant-facing administration endpoint;
- no credential sharing between worker pods.

The frontend registers each eligible worker as a deployment of exactly one
tenant-private model group. Multiple eligible workers in that group let LiteLLM
perform normal deployment selection, retry and cooldown without exposing account
identity to the caller.

## Routing and fail-closed rules

For a tenant using a private pool:

1. Hermes authenticates with its AgentIdentity Fabric key.
2. It asks for `txo-default`.
3. The operator-owned key alias resolves that public name to the tenant-private
   group.
4. Only workers owned by that tenant are registered in that group.
5. Retry/failover remains inside that group.
6. If the group has no eligible deployment, the request fails unless an explicit
   tenant policy defines another authorized provider/model fallback.

Never derive a fallback by searching all healthy credentials globally.

The following are separate policy layers and must be tested independently:

1. another credential for the same provider/model in the same tenant;
2. another authorized provider deployment for the same tenant logical model;
3. an explicit fallback logical model.

## OpenRouter rollout safety

The existing `txo-default -> OpenRouter` mapping remains the default until a
tenant private pool is fully reconciled and physically validated. The first
implementation slices must not rewrite existing AgentIdentity keys to a tenant-private
group before that group has eligible deployments.

This preserves the current working route while #3868 is introduced incrementally.

## Required isolation tests

Fast tests must prove at least:

- current keys with no alias retain the existing OpenRouter behavior;
- a Fabric-generated alias keeps the caller's public model contract unchanged;
- hAIrem and Indiba aliases resolve to different internal group names;
- tenant/user input cannot choose an arbitrary alias target;
- missing or ambiguous tenant ownership fails closed;
- disabled/revoked credentials are not eligible;
- cooldown/fallback never selects a credential owned by another tenant;
- no token/auth-file content appears in CRD status or log-safe structures.

Physical acceptance must additionally prove two hAIrem OAuth credentials, a separate
Indiba credential, restart persistence, forced failover, revoke/re-auth, and continued
OpenRouter availability as required by #3868.
