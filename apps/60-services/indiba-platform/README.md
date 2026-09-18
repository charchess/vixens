# INDIBA agent platform GitOps scaffold

This directory implements the Kubernetes/GitOps **contracts and isolation
boundaries** for the INDIBA agent platform qualified in architecture issue
#3279.

Phase 0 deliberately keeps Hermes agents `alwaysOn` and ships a minimal
user-facing Portal/BFF so the POC can be exercised without a wake-up bus or a
final frontend build pipeline.

The following custom binaries remain separate implementation deliverables:

- Control Plane API;
- Hermes Operator;
- LLM Gateway / Credential Broker;
- Hindsight INDIBA authorization extension package;
- tenant-specific Hindsight release and managed PostgreSQL binding.

## Implemented here

- `HermesRuntime` `agents.indiba.io/v1alpha1` CRD;
- `indiba-system` and reference `tenant-indiba` namespaces;
- Phase 0 `alwaysOn` lifecycle contract for the reference agent;
- `indiba-system` Pod Security `restricted`;
- `tenant-indiba` Pod Security `baseline` with `restricted` audit/warn until
  Hermes is qualified under the production restricted profile;
- tenant-scoped RBAC: the Control Plane may manage `HermesRuntime` objects but
  cannot directly create workload resources; the Hermes Operator may reconcile
  runtime resources inside the tenant cell and has no Secret read permission;
- default-deny NetworkPolicies for system and tenant cells;
- explicit Hermes -> Hindsight and Hermes/Hindsight -> LLM Gateway paths;
- service contracts for Control Plane, LLM Gateway and `hs-indiba`;
- immutable runtime configuration, memory authorization, execution-context,
  portal identity and LLM credential contracts;
- minimal Node Portal/BFF behind Authentik ForwardAuth;
- Portal -> Control Plane network boundary;
- dev ingress at `https://agents.dev.truxonline.com`;
- a representative `HermesRuntime` example under `examples/`.

## Phase 0 portal

The bootstrap Portal is deliberately small and is mounted from a ConfigMap into
a non-root `node:22-alpine` container. It provides:

- authenticated user identity from Authentik;
- a `Mes agents` view backed by `GET /v1/agents` on the future Control Plane;
- a placeholder provider-credential section for delegated OAuth.

This is a POC implementation, not the final UX. The intended replacement is a
proper SvelteKit application using the same trust boundary:

```text
browser -> Traefik -> Authentik -> Portal/BFF -> Control Plane
```

The browser must never receive direct Kubernetes, Hindsight or provider-secret
access.

## Security invariants

1. Kubernetes DNS/service discovery locates services; no runtime IP is embedded.
2. NetworkPolicy limits reachability but is never treated as authentication.
3. `bank_id`/`bankRef` is an identifier, not a credential.
4. Hindsight self-hosted must authenticate the caller through a custom
   `TenantExtension` and authorize `(AgentIdentity, MemoryBank, operation)`
   through an `OperationValidatorExtension`.
5. Hermes workloads do not receive provider API keys in the target design.
6. Delegated OAuth credentials belong to a tenant/principal/AgentIdentity
   binding; for the normal user case one principal authorizes one provider once,
   and the LLM Gateway is the unique refresh-token authority.
7. Caller-supplied tenant/user/agent headers are never authoritative. The Phase
   0 Control Plane may accept normalized Authentik claims only from the trusted
   Portal workload path.
8. Software is immutable/reconstructible; mutable state, credentials and caches
   do not share a writable scope across tenants.

## Intentionally unresolved

For Phase 0 the message/session/wake-up problem is intentionally bypassed by
`alwaysOn` Hermes runtimes. Direct wake vs durable queue/NATS, later KEDA
scale-to-zero, generic tool execution environments and dynamic package
installation remain open architecture decisions.

The tenant Hindsight -> managed PostgreSQL egress is also absent until the
managed database endpoint and credential delivery are provisioned. Default deny
therefore fails closed.

## Build

```bash
kustomize build apps/60-services/indiba-platform/overlays/dev
```

The example custom resource is not part of the Kustomize build because the
operator binary is not implemented yet.
