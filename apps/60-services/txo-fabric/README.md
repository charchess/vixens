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
tenant namespace, its default-deny network baseline and the requested Shared
PostgreSQL persistence capability.

PostgreSQL, Hindsight and optional modules are explicit API capabilities. A
TenantBundle selects platform-owned persistence and memory implementation profiles
through `profileRef`; it does not contain database credentials, provider secrets,
or provider-specific connection strings.

Shared PostgreSQL and tenant-scoped Hindsight are reconciled by the operator.
Optional modules remain pending until their lifecycle reconcilers are implemented
rather than being silently treated as ready.

### AgentIdentity

`AgentIdentity` is cluster-scoped and separates global Fabric identity from the
stable tenant-local runtime key. For example, `hairem-sandbox-tina` may use
`agentKey: tina`; runtime names remain `hermes-tina`, `hermes-tina-data`, and
`hermes-tina-egress`.

This lets different tenants use the same local keys while keeping Kubernetes CR
names globally unique.

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

The initial `hindsight-standard` profile uses the upstream API-only image, requires
API-key authentication, and keeps embeddings/reranking local. The published full
API image contains the default local models; persistent runtime model caching
remains an explicit opt-in for a separately designed use case. The Hindsight
control plane is not part of this Fabric contract.

The operator consumes the tenant PostgreSQL binding, creates a tenant-local runtime
Secret, and reconciles one Hindsight Deployment and Service per tenant. Provider
and database credentials plus the Hindsight API key remain secret-backed rather
than being embedded in the profile. AgentIdentity resources resolve deterministic
bank IDs against their tenant's Hindsight service. A shared multi-tenant Hindsight
service remains deferred until its authentication and database/schema isolation
model is proven; that investigation is tracked separately in #3574.

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

Model access is mediated by the shared TXO AI gateway. The operator provisions one
LiteLLM virtual key per `AgentIdentity`, restricted to the local `txo-default`
model alias and tagged with tenant/agent metadata. Only that scoped key is written
to the tenant namespace. Hermes receives:

- `TXO_LLM_AUTH_MODE=gateway`;
- `OPENAI_BASE_URL=http://txo-ai-gateway.txo-fabric-system.svc:4000/v1`;
- `OPENAI_API_KEY` from `Secret/hermes-<agentKey>-model-access`;
- `HERMES_MODEL=txo-default`.

Upstream provider credentials remain in the gateway boundary. Hermes network
egress permits DNS, the local AI gateway, and its tenant Hindsight service when
configured; direct provider Internet egress is not granted for model access.
`ModelAccessReady=True` records successful scoped credential reconciliation, and
an available Hermes Deployment can then make the AgentIdentity `Ready` instead of
the previous deliberate `AuthBlocked` state.

The operator starts the Hermes host gateway with `gateway run --replace` and uses
semantic process probes. Each AgentIdentity uses its pod-level
`HERMES_HOME=/opt/data` as the single private Hermes home; TXO Fabric does not add
a second named-profile layer inside that runtime.

Hindsight bank identity is part of the AgentIdentity contract. When tenant memory
is configured, the operator exposes the resolved bank through the tenant-scoped
Hindsight service while PostgreSQL remains hidden behind Hindsight from the agent.

## Brownfield cutover

The hAIrem sandbox is the first brownfield validation tenant. The temporary
Kyverno provisioning policies were retired first, leaving Tesla, Tina and Tiffa
runtime resources ownerless. The authoritative operator then adopts those
resources using the new CR contract.

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
