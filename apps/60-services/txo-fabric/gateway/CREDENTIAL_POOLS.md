# Tenant-scoped AI plane and credential pools

Status: **canonical v0.1 contract for #3868, #3885 and #3918**.

The tenant-local AI plane has passed physical hAIrem acceptance for
Hermes -> tenant LiteLLM -> tenant CPA -> Codex OAuth. It is the canonical
topology for every Active tenant; it is no longer an opt-in architecture.

## Decision

Every Active tenant receives a **tenant-local LiteLLM facade** plus a
**tenant-local CLIProxyAPI (CPA) credential broker**.

```text
Tenant workloads
      |
      v
tenant LiteLLM
  auth / logical models / routing / metering
      |
      +-- txo-agent ----> tenant CPA ----> Codex OAuth account A/B/...
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

`TenantBundle.spec.lifecycle.mode` is the topology/lifecycle switch:

- absent or `Active`: reconcile the mandatory tenant LiteLLM + CPA AI plane;
- `Parked`: stop LiteLLM/CPA compute while preserving durable recovery state.

`TenantBundle.spec.aiGateway` and `spec.aiCredentialBroker` are retained only
as v1alpha1 compatibility/profile-override fields. Their absence does **not**
disable either component for an Active tenant.

For v0.1:

- default gateway profile = `litellm-standard`;
- implementation = `LiteLLM`;
- default credential-broker profile = `cliproxyapi-standard`;
- implementation = `CLIProxyAPI`.

LiteLLM remains the only consumer-facing inference endpoint. CPA remains internal
provider-credential infrastructure. The API separation prevents CPA from
accidentally becoming the consumer-facing security/metering plane.

## Resource naming

For tenant `hairem`, the CPA broker owns:

```text
Deployment/txo-ai-credential-broker
Service/txo-ai-credential-broker
Secret/txo-ai-credential-broker-runtime
PersistentVolumeClaim/txo-ai-credential-broker-auth
NetworkPolicy/txo-ai-credential-broker
```

The tenant LiteLLM facade owns the `txo-ai-gateway` resource family. The two
components coexist without resource-name collisions.

The CPA auth PVC is provider credential state, never agent state. It is not
mounted into AgentIdentity or Hindsight workloads.

## Logical models

Consumers select logical capabilities, never provider credentials:

```text
txo-auto
txo-general
txo-agent
txo-fast
txo-reasoning
txo-embedding
```

Examples:

- Hermes may receive `txo-agent` and resolve through LiteLLM -> CPA -> Codex;
- Hindsight may receive only `txo-embedding` and resolve through LiteLLM ->
  OpenRouter/BGE-M3 or a future local embedding service;
- another service may receive `txo-reasoning` without any access to CPA.

Provider account identity remains invisible to consumers.

### OpenRouter credential contract

OpenRouter-backed logical routes are enabled per tenant, never by copying the
historical shared provider key into every tenant.

The platform secret contract is:

```text
Secret/txo-fabric-system/txo-ai-provider-<tenant>
  OPENROUTER_API_KEY
      |
      v
Secret/tenant-<tenant>/txo-ai-gateway-runtime
      |
      v
tenant LiteLLM
```

The source Secret is expected to be projected from OpenBao through External
Secrets Operator. No provider value belongs in Git, TenantBundle, AgentIdentity,
Hindsight runtime configuration, or CR status.

When the credential exists, Fabric adds `txo-embedding` to that tenant LiteLLM,
currently backed by `openrouter/baai/bge-m3`, and permits HTTPS egress from the
gateway. Hindsight itself receives only a LiteLLM virtual key scoped to
`txo-embedding`; it has no direct provider egress.

Credential appearance/rotation is watched by the TenantBundle controller through
the deterministic `txo-ai-provider-<tenant>` Secret name. A missing provider
credential does not break `txo-agent`/CPA. For an existing tenant that has not
yet migrated Hindsight, the historical shared embedding route remains a bounded
migration source **only when a pre-existing Hindsight runtime carries an embedding key or an explicit shared-backend migration marker**. An API-only Secret is not sufficient evidence of legacy use.
A newly provisioned tenant without its own OpenRouter credential must wait,
not bootstrap Hindsight against the shared gateway. Likewise, if OpenRouter
is enrolled but the tenant LiteLLM embedding route is not ready yet, a new
tenant waits for its destination instead of creating a shared key. Once
Hindsight has adopted the tenant route, loss of the provider credential fails
closed instead of falling back to shared inference.

## Retry and fallback layering

Retries must not multiply across layers.

CPA is responsible for retry/failover **between credentials serving the same
provider/model pool**.

LiteLLM is responsible for fallback **between logical deployments/providers**.

Example:

```text
txo-agent
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
2. The Fabric admin client reads that tenant's CPA management credential from
   Kubernetes and starts CPA's remote Codex OAuth flow.
3. The admin opens the returned OpenAI authorization URL.
4. For the browser/PKCE flow used by pinned CPA v8.0.16, the admin copies the
   final localhost callback URL from the browser and pastes it into the admin
   client. The client forwards that callback to CPA's authenticated management
   API.
5. CPA exchanges the code and persists mutable access/refresh/account state only
   on that tenant broker PVC under `/data/auth`.
6. CPA owns refresh and account eligibility.
7. Fabric exposes only log-safe lifecycle/quota state.
8. Disable/revoke removes an account from eligibility before destructive cleanup.

The first v0.1 administrative surface is deliberately internal and ships inside
the Fabric operator image:

```bash
kubectl -n txo-fabric-system exec -it deploy/txo-fabric-operator -- \
  /txo-fabric-admin oauth-connect --tenant hairem --provider codex
```

The command never prints the CPA management password or provider tokens. Network
policy permits this management path only from the Fabric operator pod to
tenant-owned CPA pods. Hermes and other tenant workloads do not gain CPA
management access.

A future Fabric admin UI may wrap the same contract; it must not move OAuth
tokens into Git, TenantBundle/AgentIdentity spec or status, or agent runtimes.

The remote Codex authorization flow is physically proven on hAIrem against the
production-pinned CPA version: CPA persisted one root auth JSON, populated its
model registry, and served a successful `txo-agent` request through LiteLLM.

## Migration

The historical shared LiteLLM is no longer a normal AgentIdentity backend.
#3911 provides the bounded legacy credential-cutover path: prepare the tenant
destination, roll Hermes, then revoke the shared source only after runtime
adoption.

For Active tenants, the steady-state AgentIdentity path is:

```text
Hermes -> tenant LiteLLM -> tenant CPA -> Codex OAuth pool
```

Additional logical routes such as `txo-embedding` and ordinary API/OpenRouter
backends are completed under #3885; they belong behind the same tenant LiteLLM
facade and do not reintroduce tenant-selectable gateway topology.

`fabric-smoke` is explicitly `lifecycle.mode: Parked` under #3815 and remains
free of steady-state AI-plane compute while durable recovery state is preserved.
hAIrem and Indiba are real Active beta tenants and use the same generic topology.

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
9. legacy shared-gateway cleanup is failure-safe and no AgentIdentity steady-state path depends on it.
