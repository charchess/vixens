# TXO Fabric Operator

`txo-fabric-operator` is the authoritative reconciler behind the Fabric CRDs.
It replaces the temporary Kyverno `generate` provisioning POC; Kyverno remains a
policy/admission engine, not the Fabric lifecycle engine.

## API ownership

- `TenantBundle` describes one Fabric Cell. `metadata.name` is the canonical
  tenant slug; `spec.tenantId` is the immutable business identifier.
- `AgentIdentity` describes who an agent is and references a `TenantBundle` by
  name. Because the CR is cluster-scoped, `metadata.name` is the globally unique
  Kubernetes object identity (recommended form: `<tenant>-<agentKey>`), while
  `spec.agentKey` is the immutable tenant-local machine identity used for Hermes
  profiles, runtime resource names and the default memory-bank key. This lets
  different tenants each have an agent called `sales`, `assistant`, etc. without
  colliding. `displayName` remains purely user-facing.
- `AgentRuntimeProfile` describes how an agent runs (image, storage, resources,
  Vixens scheduling and s6 compatibility). Runtime infrastructure is not part of
  the identity object.
- `PostgreSQLProfile` describes the platform-owned persistence implementation.
  The first supported topology is `SharedCluster`: TXO references a GitOps-owned
  CloudNativePG Cluster and owns only tenant `Database`, `DatabaseRole` and
  credential Secret resources.
- `AIGatewayProfile` describes the platform-owned tenant-facing inference gateway.
  The v0.1 target is one tenant-local LiteLLM facade per `TenantBundle`, backed by
  its own CloudNativePG `Database`/`DatabaseRole` binding for virtual keys,
  metering, budgets and proxy state. The gateway database is deliberately distinct
  from tenant application/Hindsight persistence.
- `AICredentialBrokerProfile` describes an internal provider-credential broker.
  The first implementation is a pinned CLIProxyAPI instance whose mutable OAuth
  state is isolated from every AgentIdentity PVC.

The operator rejects two live `AgentIdentity` resources that claim the same
`tenantRef.name + agentKey` pair instead of letting them fight over the same
Deployment/PVC/NetworkPolicy.

The current controller reconciles:

- tenant namespace + default-deny network baseline;
- isolated Hermes PVC, Deployment and egress policy per `AgentIdentity`;
- Shared PostgreSQL persistence through CloudNativePG `Database` and
  `DatabaseRole` resources, with generated secret-backed credentials and required
  extensions declared by `PostgreSQLProfile`;
- explicit reclaim semantics for tenant PostgreSQL resources;
- opt-in tenant-scoped CLIProxyAPI credential-broker substrate, including isolated
  OAuth-state storage, generated internal/management credentials, Service and
  NetworkPolicy;
- tenant LiteLLM PostgreSQL state reconciliation through a dedicated
  `postgresql-litellm` profile, generated database credentials, tenant-local
  runtime Secret, restricted migration egress and pinned-image Prisma migration Job;
- the LiteLLM serving/routing runtime remains deliberately fail-closed until the
  next #3885 slice;
- finalizers and standard Kubernetes status conditions;
- a semantic runtime probe that verifies a real `hermes gateway run` process,
  avoiding the prior `s6 + sleep infinity` false-positive readiness state.

TXO never owns or mutates the referenced CloudNativePG `Cluster`; RBAC grants the
operator read-only access to that dependency. Tenant database credentials are
generated outside Git and never copied into Fabric status.

Hindsight and its memory-bank lifecycle are reconciled by the current controller.
Optional Fabric modules remain represented in `TenantBundle` but are reported as
pending until their reconcilers are implemented. They must not be added as more
Kyverno generate rules.

The target tenant AI plane is tracked by #3885: tenant-local LiteLLM is the
consumer-facing policy/model/metering facade, while CLIProxyAPI is an internal
OAuth/subscription credential broker behind it. Existing tenants remain on the
shared LiteLLM/OpenRouter path until the tenant-local plane passes physical
acceptance. The direct-CPA smoke topology must not be promoted as the final
architecture.

## OpenFGA tenant store lifecycle — first #3996 slice

The private Fabric-wide OpenFGA service is deployed independently from the operator. **One service does not mean one store**: the target is one authorization store per `TenantBundle.spec.tenantId` plus a separate platform-control store. A store name is derived only from the **immutable business tenant ID** (`txo-fabric-tenant-ten00001`), never the mutable display name, AgentIdentity name, username or browser input.

`internal/openfga/stores.go` currently implements the platform-private store discovery/create/adoption primitive (including conflict detection, paginated discovery, deny-on-outage and no deletion on retries). Its tests use a fake HTTP transport to verify A/B tenant isolation, non-duplication and fail-closed behavior.

**Important: this first PR does NOT wire the client into `TenantBundleReconciler`, inject the OpenBao secret into the operator, write store mappings to Kubernetes, publish an authorization model or synchronize human membership tuples. It has no live provisioning behavior and requires no production promotion.** Those tasks remain in #3996 and must be implemented with status/recovery handling in the operator, with human memberships read from Authentik (the operator owns tenant groups/applications, not their human members).

Do not run manual per-client `fga store create` as a substitute for the remaining operator reconciliation. OpenFGA store names are not a uniqueness constraint: if duplicates already exist, the client refuses to select one, rather than guessing or deleting data.

## Development

The production cluster is Kubernetes 1.34, so this module pins controller-runtime
0.22.x / k8s.io 0.34.x. Generate API/RBAC assets with controller-tools 0.19.x.

```sh
make generate manifests
make test
```

The operator image is built to GHCR after merge; a separate GitOps change enables
the manager only after an immutable image tag exists. This prevents a source PR
from deploying a non-existent image into `txo-fabric-system`.
