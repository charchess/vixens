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
- `HindsightProfile` declares how the platform satisfies tenant memory.

The TXO Fabric operator owns the Kubernetes resources derived from that intent.
It reconciles continuously instead of writing generated manifests back to Git.
Kyverno remains available for admission, validation and generic security policy;
it is not a Fabric lifecycle engine.

## API contracts

### TenantBundle

`TenantBundle` is cluster-scoped. `metadata.name` is the canonical tenant slug and
`spec.tenantId` is the immutable business identifier. The operator currently
reconciles the tenant namespace and its default-deny network baseline.

PostgreSQL, Hindsight and optional modules are explicit API capabilities. A
TenantBundle selects platform-owned persistence and memory implementation profiles
through `profileRef`; it does not contain database credentials, provider secrets,
or provider-specific connection strings.

The PostgreSQL and Hindsight lifecycle controllers are not implemented yet. When
requested, these capabilities are reported through status as pending rather than
being silently treated as ready.

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
priority and compatibility settings. Tenant and agent manifests do not embed
those platform implementation details.

### PostgreSQLProfile

`PostgreSQLProfile` is cluster-scoped and describes platform persistence policy.
The initial API deliberately supports only `SharedCluster`. The concrete
`postgresql-shared` profile selects the existing Vixens CloudNativePG cluster
`databases/postgresql-shared` as the candidate Shared target and establishes the
intended contract: one logical database and one login role per tenant, required
database extensions, and explicit reclaim policy. Authoritative ownership/reuse of
that cluster is validated separately before the reconciler is implemented.

The Fabric operator must not own or mutate the referenced CloudNativePG `Cluster`.
It will eventually own only the tenant logical resources derived from the profile.
A dedicated per-tenant cluster topology is deferred until the Shared path has been
validated end-to-end; that investigation is tracked separately in #3573.

The profile contains implementation policy such as topology, naming, extension
requirements and reclaim behavior. Passwords, connection credentials and physical
connection strings are never stored in the Fabric CRD or Git.

### HindsightProfile

`HindsightProfile` is cluster-scoped and describes the tenant-scoped Hindsight API
runtime contract: immutable API image, API port, inbound API-auth mechanism,
resource envelope, optional model cache, scheduling policy and the platform
LLM-auth mode. The profile selects an auth mechanism but never embeds the API key
itself.

The initial `hindsight-standard` profile uses the upstream API-only image, requires
API-key authentication and does not provision a model-cache PVC by default. The
published full API image already contains the default local models; persistent
runtime model caching remains an explicit opt-in for a separately designed use
case. The Hindsight control plane is not part of this first Fabric contract.

The future controller will consume the tenant PostgreSQL binding, inject
secret-backed credentials and expose banks to AgentIdentity resources.
Provider/database credentials and Hindsight API keys remain secret-backed rather
than being embedded in the profile. A shared multi-tenant Hindsight service is
deferred until its authentication and database/schema isolation model is proven;
that investigation is tracked in #3574.

## Lifecycle and deletion

The operator uses explicit finalizers for destructive tenant and agent lifecycle.
Tenant Namespace and tenant default-deny NetworkPolicy are intentionally not
owned through Kubernetes garbage-collection ownerReferences: a TenantBundle
cannot delete its tenant cell while AgentIdentity resources still reference it.
The Namespace is removed only after the tenant finalizer has observed zero
remaining agents.

Agent runtime resources are reconciled from `AgentIdentity`. Existing compatible
PVCs can be adopted without rewriting their storage contract, preserving their
UID and data. The current sandbox AgentIdentity deletion path is destructive and
removes its runtime PVC; a customer-facing retention policy is a later milestone.

## Hermes runtime boundary

Hermes runs from the upstream `nousresearch/hermes-agent` image. TXO Fabric does
not copy provider credentials or legacy OAuth state into generated runtimes.
`TXO_LLM_AUTH_MODE=unconfigured` is deliberate: a healthy runtime is reported as
`AuthBlocked` until the planned scoped TXO LLM credential broker exists.

The operator starts the Hermes host gateway with `gateway run --replace` and uses
semantic process probes. Named profiles live on the agent PVC and the host gateway
serves them through Hermes' profile multiplexing model.

Hindsight bank identity is already part of the API contract, but Hindsight itself
is not provisioned by this controller version. The same is true for PostgreSQL.

## Brownfield cutover

The hAIrem sandbox is the first brownfield validation tenant. The temporary
Kyverno provisioning policies were retired first, leaving Tesla, Tina and Tiffa
runtime resources ownerless. The authoritative operator then adopts those
resources using the new CR contract.

PVC adoption is deliberately non-destructive. Deployment adoption may cause one
controlled `Recreate` rollout when the operator normalizes the old POC Deployment
spec, but the existing workspace PVC must retain its identity and data.

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
