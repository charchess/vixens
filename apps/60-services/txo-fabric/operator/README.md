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
- `AIGatewayProfile` describes the platform-owned tenant inference gateway
  implementation. The v0.1 target is one pinned CLIProxyAPI instance per
  `TenantBundle`, with provider OAuth state isolated from every AgentIdentity PVC.

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
- opt-in tenant-scoped CLIProxyAPI gateway substrate, including isolated OAuth-state
  storage, generated client/management credentials, Service and NetworkPolicy;
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

The tenant AI gateway is an incremental #3868 substrate. Existing tenants are not
cut over from the shared LiteLLM/OpenRouter path until the pinned CLIProxyAPI runtime,
OAuth lifecycle, cross-tenant isolation and accounting collector path pass physical
acceptance.

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
