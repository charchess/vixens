# INDIBA Agent Platform

## Status

| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|------------|--------|---------|
| Dev | [ ] pending merge/sync | [x] Phase 0 vertical slice | [ ] E2E runtime | v1alpha1 |
| Prod | [ ] | [ ] | [ ] | - |

Issue: #3279

## Purpose

INDIBA is a multi-tenant AI-agent platform where persistent business identity is
separated from runtime execution:

```text
AgentTemplate -> AgentIdentity -> HermesRuntime -> Deployment/Pod
```

The Control Plane is the source of truth for platform identity and policy.
Kubernetes is the execution plane.

## Phase 0 architecture

Phase 0 deliberately runs authorized Hermes agents `alwaysOn` (`replicas: 1`).
It does not require NATS/JetStream, a wake-up protocol, KEDA scale-to-zero or a
session scheduler.

The POC nevertheless deploys the main architectural components so the product
can be exercised end to end:

```text
Browser
  -> Traefik
  -> Authentik
  -> Portal/BFF
  -> Control Plane MVP
       -> OpenFGA
       -> platform_core PostgreSQL
       -> HermesRuntime
            -> Hermes Operator MVP
            -> Hermes always-on
                 -> Memory Gateway POC -> Hindsight -> hindsight_indiba pgvector
                 -> LLM Gateway MVP -> OpenRouter
```

OpenBao + External Secrets deliver runtime/database/provider credentials.

## GitOps packaging

The implementation follows Vixens conventions:

- third-party packaged components use Helm through Argo CD;
- INDIBA-owned workloads and policy resources use native manifests/Kustomize;
- platform PostgreSQL uses CloudNativePG;
- the Hindsight Phase 0 database is temporarily a tenant-local
  `pgvector/pgvector:pg17` StatefulSet because a CNPG-compatible pgvector image
  has not yet been qualified.

Argo CD applications:

- `indiba-platform`: native/Kustomize resources;
- `indiba-openfga`: upstream OpenFGA Helm chart;
- `indiba-hindsight`: upstream Hindsight 0.10.0 OCI Helm chart.

## User portal

The bootstrap portal is deployed in `indiba-system` behind Traefik + Authentik
ForwardAuth.

Reference dev URL:

```text
https://agents.dev.truxonline.com
```

It exposes authenticated identity and the `Mes agents` view through the Control
Plane. The browser never receives Kubernetes, Hermes, Hindsight or provider
credentials/endpoints.

The ConfigMap-hosted Node UI is replaceable; the intended product frontend is a
SvelteKit application preserving the same BFF trust boundary.

## Control Plane MVP

The Control Plane uses `platform_core` PostgreSQL and currently implements the
minimum POC functions:

- bootstrap schema and seeded Bertrand/Nadia AgentIdentity;
- AgentIdentity -> runtime -> MemoryBank registry fields;
- OpenFGA store/model bootstrap;
- `GET /v1/agents`;
- `POST /v1/agents/{agentRef}/chat` proxy to the Hermes API server;
- internal `(AgentIdentity, bank, operation)` memory authorization;
- initial durable `usage_event` table.

Phase 0 currently has `POC_ALLOW_ANY_AUTHENTICATED=true`, allowing the controlled
pilot cohort to use the seeded agent. OpenFGA is deployed and wired, but must
become authoritative before the trust scope is widened.

## OpenFGA

OpenFGA runs in `indiba-system` with its own `openfga` database on the INDIBA
CloudNativePG cluster.

The Phase 0 relation model is intentionally minimal (`user`, `agent`,
`owner/viewer/can_use`). The full tenant/group/repository/credential/memory model
and the platform force-allow/force-deny resolver semantics remain hardening work.

## PostgreSQL

`indiba-postgresql` is a one-instance CloudNativePG cluster used for:

- `platform_core`;
- `openfga`.

Roles and credentials are delivered from OpenBao through External Secrets.

Hindsight uses a separate `hindsight_indiba` PostgreSQL/pgvector database in the
tenant namespace for Phase 0, exercising the intended separate memory database
and credential boundary. Moving this database to the target managed/CNPG pattern
requires qualifying a pgvector-capable image and migration procedure.

## Tenant cell and HermesRuntime

`tenant-indiba` is the reference Phase 0 tenant cell.

`HermesRuntime` is namespaced and protected by several controls:

- RBAC grants Control Plane/Operator access only in `tenant-indiba`;
- Kyverno Enforce denies Phase 0 `HermesRuntime` resources outside the tenant
  namespace;
- admission requires `tenantRef=TEN00001` and the matching immutable tenant UUID;
- the Operator independently verifies namespace, `tenantRef` and `tenantId`
  before creating runtime resources.

The dev overlay applies the Bertrand/Nadia example CR. The MVP Operator
reconciles it to a one-replica Hermes Deployment, Service, private PVC,
per-runtime capability Secret and runtime status.

`tenant-indiba` temporarily enforces Pod Security `baseline`, with `restricted`
audit/warn, until the Hermes startup contract is qualified under restricted PSS.

## Hindsight and memory

Hindsight 0.10.0 is a first-class Phase 0 component, not an external placeholder.
It is deployed in `tenant-indiba` and uses the dedicated `hindsight_indiba`
pgvector database.

Stable memory service abstraction:

```text
hs-indiba.tenant-indiba.svc:8888
```

The service selects the API pods of the `indiba-hindsight` Helm release.

The target self-hosted architecture authorizes memory directly inside Hindsight
with a custom `TenantExtension` + `OperationValidatorExtension`. To make bank
isolation testable immediately, Phase 0 uses a replaceable Memory Gateway:

```text
Hermes workload capability
  -> Memory Gateway
  -> Control Plane memory authorization
  -> Hindsight bank endpoint
```

The POC must demonstrate retain, recall and reflect persistence, runtime
recreation without memory loss, and negative cross-bank access even when a
foreign `bank_id` is known.

## LLM Gateway

The MVP LLM Gateway sits between Hermes/Hindsight and OpenRouter:

```text
Hermes/Hindsight -> LLM Gateway -> OpenRouter
```

It validates a platform workload capability and injects the actual OpenRouter
credential obtained from OpenBao. Provider keys therefore do not enter Hermes
or Hindsight runtime configuration.

The target credential model remains:

- `LLMProfile` = model/routing/privacy/budget policy;
- `LLMCredential` = provider authentication metadata + secret reference;
- `LLMCredentialBinding` = authorization to use a credential;
- delegated OAuth normally belongs to a principal and can be reused by several
  authorized AgentIdentity objects;
- the LLM Gateway is the unique refresh-token authority.

Delegated device OAuth is not yet implemented in the MVP; OpenRouter managed
credentials unblock the POC first.

## Network and secrets

Both INDIBA namespaces are default-deny. Explicit flows permit only the current
POC communication paths, including tenant workloads to Hindsight/LLM Gateway and
LLM Gateway HTTPS egress to OpenRouter.

OpenBao-backed External Secrets provide:

- Control Plane/PostgreSQL credentials;
- OpenFGA PostgreSQL credentials;
- Hindsight PostgreSQL credentials;
- workload gateway capability;
- OpenRouter provider credential.

## Deferred beyond Phase 0

- runtime scale-to-zero and wake-up protocol;
- NATS/JetStream/session scheduling;
- full OpenFGA model and policy resolver semantics;
- native Hindsight authorization extensions replacing the POC Memory Gateway;
- delegated OAuth/device-flow credential brokerage;
- CNPG-qualified pgvector lifecycle for tenant memory databases;
- generic Python/browser tool runtimes;
- full usage reconciliation/billing;
- production HA and multi-region operation.

These are explicit deferrals, not hidden dependencies for the first end-to-end
POC.
