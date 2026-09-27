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

The operator rejects two live `AgentIdentity` resources that claim the same
`tenantRef.name + agentKey` pair instead of letting them fight over the same
Deployment/PVC/NetworkPolicy.

The first controller milestone intentionally reconciles only the behavior already
proven by the sandbox POC:

- tenant namespace + default-deny network baseline;
- isolated Hermes PVC, Deployment and egress policy per `AgentIdentity`;
- finalizers and standard Kubernetes status conditions;
- a semantic runtime probe that verifies a real `hermes gateway run` process,
  avoiding the prior `s6 + sleep infinity` false-positive readiness state.

PostgreSQL, Hindsight, memory-bank lifecycle and optional Fabric modules are
represented in `TenantBundle` but deliberately reported as pending until their
reconcilers are implemented. They must not be added as more Kyverno generate
rules.

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
