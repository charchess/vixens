# Tenant-scoped AI plane and credential pools

Status: **target implementation contract for #3868, #3869 and #3885**.

The currently deployed inference path still uses the shared LiteLLM gateway and
OpenRouter. The target remains opt-in until the tenant-local AI plane passes
physical acceptance.

## Decision

The v0.1 target is a **tenant-local LiteLLM facade** plus an optional
**tenant-local CLIProxyAPI (CPA) credential broker**.

```text
Tenant workloads
      |
      v
tenant LiteLLM
  auth / logical models / routing / metering
      |
      +-- txo-coding ----> tenant CPA ----> Codex OAuth account A/B/...
      +-- txo-general ---> OpenRouter / API / OSS backend
      +-- txo-reasoning -> OpenRouter / API / OSS backend
      +-- txo-embedding -> OpenRouter / local embedding backend
```

Fabric is the control plane. LiteLLM is the tenant-facing AI gateway. CPA is an
internal backend specialized in OAuth/subscription credential lifecycle.

This supersedes the temporary #3882 interpretation where CPA itself was called
the tenant AI gateway. The CPA substrate remains useful; only its responsibility
and resource/API naming change.

## Responsibility boundaries

### TXO Fabric

Fabric owns:

- tenant ownership and topology;
- logical model/catalog policy;
- consumer allowlists;
- lifecycle of the tenant gateway and credential broker;
- non-secret status;
- generation/rotation/revocation intent.

Provider secrets never belong in TenantBundle/AgentIdentity spec or status.

### Tenant LiteLLM

LiteLLM owns the tenant-facing concerns:

- one OpenAI-compatible endpoint for tenant consumers;
- Fabric-scoped virtual keys;
- logical model aliases and allowlists;
- provider/deployment routing;
- explicitly authorized cross-provider/model fallback;
- request/token/spend metering;
- budgets and rate limits where configured.

LiteLLM requires a database for virtual keys and durable spend/budget state.
A DB-less tenant gateway must never be presented as having enforceable budgets.

Fabric provisions that state as a **dedicated CloudNativePG database and login
role per tenant gateway**, selected by `AIGatewayProfile.postgresqlProfileRef`
(default `postgresql-litellm`). It reuses the platform CNPG cluster but never
reuses the tenant application/Hindsight database. Database/role credentials live
in the database namespace, are copied only into the tenant-local
`txo-ai-gateway-runtime` Secret, and never enter CR spec/status. Schema changes
are applied by a pinned-image Prisma migration Job before the LiteLLM serving
runtime may become Ready.

### Tenant CPA

CPA owns only the subscription/OAuth provider-account pool:

- interactive OAuth connect/re-auth;
- persisted refresh/access state;
- selection between equivalent credentials;
- quota/cooldown and retry state;
- disable/revoke lifecycle;
- provider-account health observations.

CPA is not the tenant authorization source of truth and is not the primary
billing ledger.

## Fabric API contract

`TenantBundle.spec.aiGateway` references `AIGatewayProfile`.

For v0.1:

- implementation = `LiteLLM`;
- default profile = `litellm-standard`;
- this is the consumer-facing inference endpoint.

`TenantBundle.spec.aiCredentialBroker` references
`AICredentialBrokerProfile`.

For v0.1:

- implementation = `CLIProxyAPI`;
- default profile = `cliproxyapi-standard`;
- this is internal provider-credential infrastructure.

The API separation prevents CPA from accidentally becoming the consumer-facing
security/metering plane.

## Resource naming

For tenant `hairem`, the CPA broker owns:

```text
Deployment/txo-ai-credential-broker
Service/txo-ai-credential-broker
Secret/txo-ai-credential-broker-runtime
PersistentVolumeClaim/txo-ai-credential-broker-auth
NetworkPolicy/txo-ai-credential-broker
```

The future LiteLLM facade owns the `txo-ai-gateway` resource family. The two
components must coexist without resource-name collisions.

The CPA auth PVC is provider credential state, never agent state. It is not
mounted into AgentIdentity or Hindsight workloads.

## Logical models

Consumers select logical capabilities, never provider credentials:

```text
txo-auto
txo-general
txo-coding
txo-fast
txo-reasoning
txo-embedding
```

Examples:

- Hermes may receive `txo-coding` and resolve through LiteLLM -> CPA -> Codex;
- Hindsight may receive only `txo-embedding` and resolve through LiteLLM ->
  OpenRouter/BGE-M3 or a future local embedding service;
- another service may receive `txo-reasoning` without any access to CPA.

Provider account identity remains invisible to consumers.

## Retry and fallback layering

Retries must not multiply across layers.

CPA is responsible for retry/failover **between credentials serving the same
provider/model pool**.

LiteLLM is responsible for fallback **between logical deployments/providers**.

Example:

```text
txo-coding
   |
   +-> CPA
   |    +-> Codex A (cooldown)
   |    +-> Codex B (healthy)
   |
   +-> explicitly-authorized alternate provider only if the CPA pool is unavailable
```

No retry or fallback may cross the tenant boundary or widen a consumer/model
allowlist.

## Metering and accounting

LiteLLM is the primary tenant-side usage observation point.

The durable ledger should be able to attribute:

- tenant;
- consumer key / AgentIdentity / platform service;
- requested logical model;
- resolved deployment/model when safe;
- input/output tokens;
- requests;
- calculated provider cost when pricing semantics are meaningful.

For subscription-backed Codex, token usage is measurable but provider billing is
not necessarily per-token. TXO internal cost allocation must therefore remain a
separate policy from observed token usage.

CPA management/usage observations remain useful for provider quota and credential
health, but they are secondary to gateway-side metering.

## Tenant isolation

v0.1 isolation remains topological:

```text
tenant-hairem
  LiteLLM-hairem
  CPA-hairem
  OAuth PVC-hairem

tenant-indiba
  LiteLLM-indiba
  CPA-indiba
  OAuth PVC-indiba
```

No global LiteLLM router or CPA process may select provider credentials from
multiple tenants.

## OAuth lifecycle

Interactive OAuth remains an administrative operation.

1. Admin/Fabric selects one tenant broker.
2. CPA performs provider authorization.
3. Mutable auth state is persisted only on that tenant broker PVC.
4. CPA owns refresh and account eligibility.
5. Fabric exposes only log-safe lifecycle/quota state.
6. Disable/revoke removes an account from eligibility before destructive cleanup.

The exact remote Codex authorization flow still requires physical proof for the
pinned CPA version before #3868 can close.

## Migration

Current path:

```text
Hermes/Hindsight -> shared LiteLLM -> OpenRouter
```

Target path:

```text
Hermes/Hindsight -> tenant LiteLLM -> CPA and/or ordinary model backends
```

The shared gateway remains authoritative until the tenant-local gateway has an
accepted database/key/metering contract and the physical smoke tests pass.

`fabric-smoke` is a parked recovery shell under #3815 and must remain free of
steady-state runtime compute. The temporary direct-CPA activation introduced by
#3884/#3886 is retired before promotion. Physical AI-plane acceptance must use a
purpose-built disposable test tenant created for the validation window, then
removed again through GitOps.

## Required tests

Controller tests must prove:

- CPA broker state is isolated per tenant;
- broker client and management credentials are distinct and stable;
- provider tokens never enter Fabric status/generated config;
- unsupported/shared broker topology fails closed;
- CPA cannot be accepted as an `AIGatewayProfile`;
- LiteLLM is the only accepted v0.1 tenant gateway implementation;
- gateway and broker resource families do not collide;
- removing a capability cannot delete resources not owned by the TenantBundle.

Physical acceptance will later prove:

1. tenant LiteLLM starts and is the only consumer-facing endpoint;
2. LiteLLM can route a Responses API model to tenant CPA;
3. an embedding route can bypass CPA;
4. scoped keys enforce model allowlists;
5. LiteLLM metering observes CPA-backed and non-CPA routes;
6. CPA OAuth state survives restart;
7. credential failover stays inside a tenant;
8. fallback/retry behavior does not multiply unexpectedly;
9. the shared gateway remains available until explicit cutover.
