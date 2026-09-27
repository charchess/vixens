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
- `AgentRuntimeProfile` declares the platform-owned runtime implementation.

The TXO Fabric operator owns the Kubernetes resources derived from that intent.
It reconciles continuously instead of writing generated manifests back to Git.
Kyverno remains available for admission, validation and generic security policy;
it is not a Fabric lifecycle engine.

## API contracts

### TenantBundle

`TenantBundle` is cluster-scoped. `metadata.name` is the canonical tenant slug and
`spec.tenantId` is the immutable business identifier. The operator currently
reconciles the tenant namespace and its default-deny network baseline.

PostgreSQL, Hindsight and optional modules are already explicit API capabilities,
but their lifecycle controllers are not implemented yet. When requested they are
reported through status as pending rather than being silently treated as ready.

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
