# Tenant-scoped LLM credential pools

Status: **target implementation contract for #3868**.

The currently deployed inference path still uses the shared LiteLLM gateway and
OpenRouter. The target described here is deliberately opt-in until the
CLIProxyAPI path has passed physical acceptance. No tenant is cut over merely by
adding the substrate.

## Decision

The v0.1 target is **one CLIProxyAPI (CPA) gateway per TenantBundle**.

TXO Fabric remains the control plane. CPA becomes the tenant-local provider
credential broker.

```text
                       TXO Fabric
                       control plane
                            |
             +--------------+--------------+
             |                             |
             v                             v
      tenant hAIrem                  tenant Indiba
             |                             |
      CLIProxyAPI hAIrem             CLIProxyAPI Indiba
        |       |                       |
        |       +-- OAuth account B     +-- OAuth account C
        +---------- OAuth account A
```

Hermes never receives an upstream provider credential. It eventually receives
only a Fabric-owned credential for the CPA belonging to its own tenant.

This supersedes the earlier #3868 design that proposed one LiteLLM worker per
OAuth credential. The alias support added in #3880 remains harmless rollout
substrate for the currently active LiteLLM path but is not the target credential
pool implementation.

## Why CPA is the credential broker

The selected baseline is CLIProxyAPI **v8.0.16**. Its upstream implementation
already owns the behaviors #3868 would otherwise have to recreate around
LiteLLM:

- multiple file/OAuth-backed credentials;
- Codex/OpenAI OAuth support;
- round-robin, weighted-round-robin and fill-first credential selection;
- retry/failover across eligible credentials;
- session affinity when wanted;
- per-credential and per-model cooldown/quota state;
- disable/unavailable lifecycle state;
- persisted mutable auth files and refresh state;
- authenticated management APIs for credential and quota observation;
- an in-memory usage-event queue and API-key usage views.

Fabric must consume those capabilities instead of reimplementing their internal
scheduler.

## Tenant isolation boundary

CPA's ordinary incoming API-key authentication is not treated as a sufficient
multi-tenant authorization boundary. In v0.1, credential pools are isolated by
topology:

- one CPA Deployment per TenantBundle;
- one tenant namespace per CPA;
- one OAuth-state PVC per CPA;
- one generated management credential per CPA;
- one generated bootstrap/client credential per CPA;
- no shared credential directory between tenants;
- no global CPA process allowed to select from several tenants' provider
  accounts.

Therefore a hAIrem failover cannot select an Indiba credential unless the
platform itself violates the Kubernetes/configuration boundary.

CPA credential/model prefixes and `routing.force-model-prefix=true` remain
enabled as defense in depth, not as the primary tenant boundary.

## Fabric API contract

`TenantBundle.spec.aiGateway` requests the tenant capability and references a
platform-owned `AIGatewayProfile`.

The profile owns implementation details such as:

- exact CPA image;
- service port;
- resources and scheduling class;
- OAuth-state storage class/size;
- bounded usage-queue retention.

The tenant CR never contains OAuth tokens, upstream account identifiers,
provider API keys or management passwords.

The first standard profile pins:

```text
eceasy/cli-proxy-api:v8.0.16
```

The image/version must be upgraded deliberately and revalidated; `latest` is
not an acceptable Fabric contract.

## Runtime resources

For a tenant named `hairem`, the reconciler owns resources in
`tenant-hairem`:

```text
Deployment/txo-ai-gateway
Service/txo-ai-gateway
Secret/txo-ai-gateway-runtime
PersistentVolumeClaim/txo-ai-gateway-auth
NetworkPolicy/txo-ai-gateway
```

The auth PVC is **gateway/provider state**, not agent state. It is never mounted
into an AgentIdentity runtime and is independent from retained Hermes workspaces.

The runtime Secret contains only Fabric-to-CPA bootstrap/control material and
the generated CPA configuration. Provider OAuth access/refresh tokens belong on
the dedicated CPA auth volume.

## CPA baseline configuration

The generated baseline deliberately disables discovery, control-panel
auto-update and plugins. It enables:

- round-robin routing;
- retry/failover;
- `force-model-prefix=true`;
- usage statistics;
- a bounded management usage queue;
- OAuth state under the dedicated `/data/auth` volume.

The Management API is protected by a generated `MANAGEMENT_PASSWORD`. Provider
credentials are never copied into TenantBundle/AgentIdentity status, labels or
annotations.

The first substrate does **not** hand the bootstrap CPA key to Hermes and does
not change `OPENAI_BASE_URL`. That cutover happens only after physical CPA
acceptance.

## OAuth lifecycle

Interactive OAuth is an administrative operation, never a side effect of an
inference request.

Target lifecycle:

1. Fabric/admin chooses the tenant gateway and requests connect/re-auth for one
   provider credential.
2. The action is performed against only that tenant CPA.
3. CPA persists the resulting mutable auth state on that tenant's gateway PVC.
4. CPA owns ordinary refresh and credential eligibility.
5. Fabric/collector observes only log-safe lifecycle/quota data.
6. Disable/revoke removes the credential from eligibility before destructive
   cleanup of its persisted token state.

### Codex device authorization caveat

CLIProxyAPI exposes Codex OAuth and its CLI includes a Codex device-login path,
but the exact **remote administrative device-flow contract required by TXO has
not yet been physically proven** for v8.0.16. The inspected management/TUI flow
for Codex is browser/callback oriented.

Therefore #3868 must remain open until the chosen administrative flow is proven
without shell access to the tenant runtime and without exposing tokens to
Hermes.

## Logical models

Agents must continue to choose a logical model, never an account.

The final mapping belongs to #3869, for example:

```text
txo-general
txo-coding
txo-fast
txo-auto
txo-embedding
```

Fabric projects those logical choices into the tenant CPA configuration. CPA
then chooses an eligible credential/provider account inside that tenant.

Provider account identity is not part of the agent-facing contract.

## Accounting, budgets and collector contract

LiteLLM is **not required merely for accounting**.

CPA v8 exposes authenticated observability/management surfaces including:

```text
GET /v8/management/observability/usage/api-keys
GET /v8/management/observability/usage/queue
```

Credential observations also expose success/failure counts, recent request
buckets, quota/cooldown state and recovery timestamps when known.

A TXO accounting collector may therefore:

1. authenticate to each tenant CPA Management API with platform-owned
   credentials;
2. drain usage events inside the configured retention window;
3. normalize tenant, agent/client key, logical/served model, tokens and provider
   observations;
4. enrich events with the platform pricing catalog;
5. persist durable accounting outside CPA;
6. derive dashboards, alerts and budget enforcement from that durable ledger.

CPA's in-memory usage queue is an **observation transport, not the billing
ledger**. Missing a polling window must be detectable, and durable budget
enforcement must not depend solely on volatile CPA state.

Because every tenant has its own CPA, tenant attribution is already fixed by the
endpoint being collected. Per-agent attribution can later use distinct
Fabric-generated CPA client keys and/or a Fabric-owned request identity
projection once that contract is proven.

## Network boundary

The tenant CPA is ClusterIP-only and is selected by the tenant default-deny
network posture.

Required intent:

- tenant workloads reach only their own CPA for model inference;
- CPA reaches DNS and explicitly permitted HTTPS provider endpoints;
- CPA Management API requires its separate generated management credential;
- no tenant workload receives that management credential;
- a future collector/control-plane path is explicitly allowed rather than
  opening the Management API publicly;
- hAIrem and Indiba CPA pods cannot mount or address each other's auth PVCs.

NetworkPolicy is defense in depth; namespace/resource ownership is the primary
credential-state isolation boundary.

## Rollout and LiteLLM retirement

During migration:

```text
existing path:
Hermes -> shared LiteLLM -> OpenRouter

target path:
Hermes -> tenant CPA -> tenant provider pool
```

The existing LiteLLM route remains authoritative until CPA acceptance is green.
There is no automatic fallback from a CPA-enabled tenant to another tenant or a
global credential pool.

LiteLLM can be removed only after all of the following are true:

- AgentIdentity inference uses the tenant CPA contract;
- Hindsight embedding routing has an accepted replacement path;
- scoped credential revocation/rotation has an equivalent Fabric/CPA contract;
- accounting/budget observation has moved to the collector path;
- no active module depends on the shared LiteLLM gateway.

Removal is a separate GitOps change, not part of the initial CPA substrate.

## Required tests

Fast controller tests must prove:

- two tenants receive separate CPA Secrets and auth PVCs;
- generated gateway credentials are different per tenant and stable across
  idempotent reconciliation;
- unsupported/shared CPA topology fails closed;
- provider tokens do not appear in generated config, CR status or logs;
- the CPA image is explicitly pinned;
- deleting/removing the capability cannot delete resources not owned by that
  TenantBundle;
- current tenants that do not request `spec.aiGateway` keep their existing
  inference path unchanged.

Physical acceptance for #3868 must prove at least:

1. one disposable tenant CPA starts from the pinned image under the generated
   security/storage contract;
2. two hAIrem Codex credentials can be authorized and survive CPA restart;
3. selection/failover stays inside hAIrem;
4. a separate Indiba CPA cannot use hAIrem credentials;
5. forced quota exhaustion/cooldown makes the next eligible hAIrem credential
   serve the request;
6. revoke/disable/re-auth works without giving a provider token to Hermes;
7. the collector can consume usage/quota observations without provider secret
   leakage;
8. the existing OpenRouter/LiteLLM path remains available until explicit
   cutover.
