# TXO Fabric — tenant-neutral management plane

This directory defines the tenant-neutral TXO Fabric management-plane bootstrap,
its authoritative Fabric API, and the operator that reconciles tenant cells.

A fresh deployment remains healthy with **zero tenants**: no customer namespace,
Hermes runtime, Hindsight bank, Paperclip or Valkey workload is created until a
`TenantBundle` or `AgentIdentity` is declared.

## Ownership model

Git and Argo CD own the high-level desired state:

- `TenantBundle` declares one Fabric Cell and its requested capabilities;
- `AgentIdentity` declares one tenant-local agent identity;
- `AgentRuntimeProfile` declares the platform-owned runtime implementation;
- `PostgreSQLProfile` declares how the platform satisfies tenant persistence;
- `HindsightProfile` declares how the platform satisfies tenant memory;
- `IntegrationConnection` and `IntegrationBinding` declare logical external
  integration metadata and per-agent authorization without embedding credential
  values.

The TXO Fabric operator owns the Kubernetes resources derived from that intent.
It reconciles continuously instead of writing generated manifests back to Git.
Kyverno remains available for admission, validation and generic security policy;
it is not a Fabric lifecycle engine.

## API contracts

### TenantBundle

`TenantBundle` is cluster-scoped. `metadata.name` is the canonical tenant slug and
`spec.tenantId` is the immutable business identifier. The operator reconciles the
tenant namespace, its default-deny network baseline and the mandatory tenant AI
plane for every Active tenant.

`spec.lifecycle.mode` defaults to `Active`. Active tenants automatically receive
tenant LiteLLM + tenant CPA; `spec.aiGateway` and `spec.aiCredentialBroker`
remain only as v1alpha1 profile overrides and are not enable/disable switches.
`Parked` is the recovery-shell state: AI-plane compute is stopped while durable
OAuth/database state is preserved for reactivation.

PostgreSQL, Hindsight and optional modules remain explicit API capabilities. A
TenantBundle selects platform-owned persistence and memory implementation profiles
through `profileRef`; it does not contain database credentials, provider secrets,
or provider-specific connection strings.

Shared PostgreSQL and tenant-scoped Hindsight are reconciled by the operator.
Optional modules remain pending until their lifecycle reconcilers are implemented
rather than being silently treated as ready.

### AgentIdentity

`AgentIdentity` is cluster-scoped and separates global Fabric identity from the
stable tenant-local runtime key. Canonical populated agents use stable keys such as
`usr000001-agt00012`; derived runtime resources therefore use names such as
`hermes-usr000001-agt00012`, `hermes-usr000001-agt00012-data`, and the
corresponding agent-scoped network resources.

Different tenants may use the same tenant-local `agentKey` while globally scoped
Fabric CR names and tenant ownership keep those identities distinct. Do not derive
identity from mutable display names, Pod names or group membership.

### AgentRuntimeProfile

`AgentRuntimeProfile` separates identity from infrastructure policy. The profile
selects the Hermes image, storage class and size, resource envelope, scheduling
priority, compatibility settings and platform-owned executable toolset policy.
Tenant and agent manifests do not embed those implementation details. The
three-state Hermes capability contract and pinned-runtime inventory are documented
in [CAPABILITIES.md](CAPABILITIES.md).

### IntegrationConnection / IntegrationBinding

The v0 integration contract keeps logical connection metadata and per-agent
authorization in Fabric rather than in writable Hermes `/opt/data`. Credential
values remain outside the CRDs and Git. The current v0 adapter, temporary broad
POC egress, authenticated `fabric-smoke` canary, revocation semantics and v0.2
broker migration boundary are documented in [INTEGRATIONS.md](INTEGRATIONS.md).

### PostgreSQLProfile

`PostgreSQLProfile` is cluster-scoped and describes platform persistence policy.
The initial API deliberately supports only `SharedCluster`. The concrete
`postgresql-shared` profile selects the GitOps-owned Vixens CloudNativePG cluster
`databases/postgresql-shared` and establishes the contract: one logical database
and one login role per tenant, required database extensions, and explicit reclaim
policy.

The Fabric operator does not own or mutate the referenced CloudNativePG `Cluster`.
Its RBAC access to that dependency is read-only. It owns only the tenant logical
`Database`, `DatabaseRole` and generated credential Secret derived from the
profile. A dedicated per-tenant cluster topology is deferred until the Shared path
has been validated end-to-end; that investigation is tracked separately in #3573.

The controller derives stable PostgreSQL identifiers from the immutable tenant ID,
creates a `kubernetes.io/basic-auth` credential Secret outside Git, then waits for
CloudNativePG to apply the login role before creating the database. Required
extensions such as `vector` are declared through `Database.spec.extensions`, so
TXO does not connect directly to PostgreSQL to execute lifecycle SQL.

Reclaim is explicit rather than relying on ownerReferences. With `Retain`, deleting
the Fabric CR removes the CloudNativePG management CR while CloudNativePG retains
the logical database/role; the generated credential Secret is retained with them.
This prevents Kubernetes garbage collection from bypassing the profile's reclaim
contract.

Passwords, connection credentials and physical connection strings are never stored
in the Fabric CRD or Git.

### HindsightProfile

`HindsightProfile` is cluster-scoped and describes the tenant-scoped Hindsight API
runtime contract: immutable API image, API port, inbound API-auth mechanism,
resource envelope, optional model cache, scheduling policy and the platform
LLM-auth mode. The profile selects an auth mechanism but never embeds the API key
itself.

The initial `hindsight-standard` profile uses the upstream API image and requires
API-key authentication. Hindsight generative/reflection LLM processing remains
disabled with `HINDSIGHT_API_LLM_PROVIDER=none`. The canonical embedding path is
the tenant-local LiteLLM facade using the logical `txo-embedding` model and a
Hindsight-scoped virtual key. The OpenRouter provider credential exists only on
the platform-owned tenant gateway runtime; it never enters the Hindsight Pod.

Existing tenants may temporarily remain on the historical shared gateway while
their tenant OpenRouter credential and `txo-embedding` route are prepared. The
operator performs a failure-safe shared -> tenant cutover: it prepares the
destination key first, rolls Hindsight onto the tenant gateway, and revokes the
shared key only after the new runtime is Available. A tenant that has completed
the cutover never silently falls back to the shared gateway.

The operator consumes the tenant PostgreSQL binding, creates tenant-local runtime
Secrets, and reconciles one Hindsight API Deployment and Service per tenant.
Database credentials, the Hindsight tenant API key and the scoped embedding-gateway
credential remain Secret-backed rather than being embedded in the profile or
TenantBundle. AgentIdentity resources resolve deterministic bank IDs against their
tenant's Hindsight service.

When a Hindsight-enabled TenantBundle explicitly sets
`memory.hindsight.humanAccess: true` and declares `humanAccess.web`, Fabric
additionally reconciles the paired upstream Hindsight Control Plane image, a private
Service, default-deny-compatible NetworkPolicies and a stable authenticated route:

`https://hindsight-<tenant>.<domainSuffix>`

The Control Plane receives the tenant API key only through a Secret-backed
server-side environment variable and talks to the private Hindsight API inside the
tenant namespace. Browsers never receive the API key, the memory API is not routed
publicly, and PostgreSQL remains hidden behind Hindsight.

For tenants declaring `humanAccess.web`, the Fabric operator also publishes the
structural Authentik desired state through the aggregate
`auth/txo-fabric-authentik-blueprints` ConfigMap. That generated blueprint owns
the tenant structural groups, public OIDC provider/application and policy bindings,
plus the Hindsight Proxy Provider/application when Hindsight human access is
requested. It also publishes the deterministic provider union consumed by the
embedded Authentik outpost, avoiding per-tenant outpost-provider clobbering.

Fabric owns this **structural IAM intent only**. Authentik remains the live source
for human users, passwords, MFA and group membership. The separate
`/outpost.goauthentik.io` route is sent directly to the embedded Authentik outpost
so the sign-in flow itself is not recursively protected.

`TenantBundle.status.memory.hindsight.endpoint` remains the private API endpoint,
while `humanEndpoint` reports the stable authenticated WebUI URL.

A shared multi-tenant Hindsight service remains deferred until its authentication
and database/schema isolation model is proven; that investigation is tracked
separately in #3574.

## Lifecycle and deletion

The operator uses explicit finalizers for destructive tenant and agent lifecycle.
Tenant Namespace and tenant default-deny NetworkPolicy are intentionally not
owned through Kubernetes garbage-collection ownerReferences: a TenantBundle
cannot delete its tenant cell while AgentIdentity resources still reference it.
The Namespace is removed only after the tenant finalizer has observed zero
remaining agents and the Fabric-owned PostgreSQL management resources have been
released according to their reclaim policy.

Agent runtime resources are reconciled from `AgentIdentity`. Existing compatible
PVCs can be adopted without rewriting their storage contract, preserving their
UID and data. Runtime workspace deletion is explicit through
`spec.runtime.storage.retentionPolicy`, with values `Retain` or `Delete`. The safe
default is `Retain`; `Delete` is an immutable opt-in for disposable identities.
When an AgentIdentity using `Retain` is deleted, the operator removes the PVC's
AgentIdentity ownerReference and leaves tenant/agent retention labels in place so
a replacement identity with the same tenant and `agentKey` can re-adopt it.

For a bound `Retain` runtime PVC, the operator also promotes the concrete
PersistentVolume reclaim policy to `Retain`. This is the backend-safety contract:
it protects durable state even when a brownfield PVC was originally provisioned
from a StorageClass whose default reclaim policy was `Delete`. The controller only
promotes to the safer policy; an AgentIdentity using `Delete` does not
automatically downgrade an already-retained PV. Disposable profiles should
therefore continue to use a Delete StorageClass when backend deletion is desired.

A TenantBundle cannot delete its tenant Namespace while retained Fabric agent PVCs
remain inside it. Intentional destructive tenant removal must therefore release
those workspaces explicitly first, for example by using an AgentIdentity declared
with `retentionPolicy: Delete`. The `fabric-smoke` fixture opts into `Delete` so it
remains suitable for destructive lifecycle tests; customer identities inherit
`Retain` unless they explicitly choose otherwise.

The scoped LiteLLM virtual key is revoked by deterministic alias on a best-effort
basis and its tenant-local model-access Secret is deleted with the agent lifecycle.

## Hermes runtime boundary

Hermes runs from the TXO-owned immutable runtime image derived from a reviewed
upstream `nousresearch/hermes-agent` release. TXO Fabric does not copy upstream
provider credentials or legacy OAuth state into generated runtimes.

For every Active tenant, model access is mediated by that tenant's LiteLLM
facade. The operator provisions one LiteLLM virtual key per `AgentIdentity`,
restricted to the logical `txo-agent` model and tagged with tenant/agent
metadata. Only that scoped key is written to the tenant namespace.

The platform-owned Hermes managed scope pins the behavioral model route in
`/etc/hermes/config.yaml`, so the pinned Hermes gateway runtime sees the same
route on every surface and the user-owned `/opt/data/config.yaml` cannot replace
it:

- `model.default=txo-agent`;
- `model.provider=custom`;
- `model.base_url=http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`.

Hermes additionally receives the secret/runtime bridge:

- `TXO_LLM_AUTH_MODE=gateway`;
- `OPENAI_BASE_URL=http://txo-ai-gateway.tenant-<tenant>.svc:4000/v1`;
- `OPENAI_API_KEY` from `Secret/hermes-<agentKey>-model-access`;
- `HERMES_MODEL=txo-agent`.

Upstream provider/OAuth credentials remain behind LiteLLM/CPA. Hermes network
egress permits DNS, its tenant-local LiteLLM, and its tenant Hindsight service
when configured; it has no direct CPA path and no shared-LiteLLM steady-state
route. The shared gateway is retained only as a bounded legacy credential source
for failure-safe cutover cleanup.

`ModelAccessReady=True` records successful scoped credential reconciliation.
A Parked tenant cannot resolve a model backend for an AgentIdentity and therefore
fails closed instead of falling back to shared or direct provider access.

The operator starts the Hermes host gateway with `gateway run --replace` and uses
semantic process probes. Each AgentIdentity uses its pod-level
`HERMES_HOME=/opt/data` as the single private Hermes home; TXO Fabric does not add
a second named-profile layer inside that runtime.

Hindsight bank identity is part of the AgentIdentity contract. When tenant memory
is configured, the operator exposes the resolved bank through the tenant-scoped
Hindsight service while PostgreSQL remains hidden behind Hindsight from the agent.

## Historical brownfield cutover

The now-retired hAIrem sandbox was the first brownfield validation path. The
temporary Kyverno provisioning policies were retired first, leaving the historical
Tesla, Tina and Tiffa runtime resources ownerless so the authoritative operator
could prove adoption through the new CR contract.

This section records that migration mechanism; it is not the current Client0
population or an onboarding recipe for new tenants.

PVC adoption is deliberately non-destructive. Deployment adoption may cause one
controlled `Recreate` rollout when the operator normalizes the old POC Deployment
spec, but the existing workspace PVC must retain its identity and data.

A retained brownfield PVC that still uses Hermes' former named-profile layout can
opt into `runtime.storage.adoptLegacyProfile: true`. In that mode the operator
validates `profiles/<agentKey>` from the full retained volume during bootstrap,
then mounts that existing directory with Kubernetes `subPath` as the runtime's
single `/opt/data`. The profile is neither copied nor flattened. Hermes still sees
`HERMES_HOME=/opt/data`, so this is a storage-layout adoption mechanism rather
than a return to named-profile runtime multiplexing. The opt-in requires retained
storage and must be checkpointed before production cutover. New identities use the normal flat runtime home and do not need this brownfield-only flag.

## Zero-tenant acceptance

Without tenant declarations:

```console
kubectl get tenantbundles
No resources found

kubectl get agentidentities
No resources found
```

The following must also be true:

- no `tenant-*` namespace exists because of TXO Fabric;
- no Hermes runtime exists because of TXO Fabric;
- no tenant database, memory bank, repository, Paperclip or Valkey instance is
  created implicitly.

`hAIrem` receives no special platform treatment. Test and production tenant cells
use the same API contracts as future external customers.
