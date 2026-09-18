# INDIBA Agent Platform

## Status

| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|------------|--------|---------|
| Dev | [ ] | [x] Phase 0 GitOps contracts + portal | [ ] | v1alpha1 |
| Prod | [ ] | [ ] | [ ] | - |

Issue: #3279

## Purpose

This application is the production-shaped Kubernetes contract for the INDIBA
multi-tenant agent platform. It separates persistent business identity from
runtime:

```text
AgentTemplate -> AgentIdentity -> HermesRuntime -> Deployment/Pod
```

The Control Plane remains the source of truth for tenants, principals, groups,
AgentIdentity objects, grants, memory-bank registry, credentials metadata,
runtime configuration snapshots and usage events. Kubernetes is the execution
plane, not the business catalog.

## Phase 0 / POC decisions

The POC deliberately keeps the runtime model simple:

- authorized Hermes agents are `alwaysOn` (`replicas: 1`);
- no NATS/JetStream wake-up path is required yet;
- no KEDA scale-to-zero is required for Hermes runtimes yet;
- no session scheduler is required for 3-4 pilot users;
- the user portal talks only to the Control Plane contract, never directly to
  Hermes, Hindsight or Kubernetes.

The wake-up/message-bus decision remains deferred without blocking the POC.

## User portal

The Phase 0 portal is deployed in `indiba-system` behind Traefik + Authentik
ForwardAuth.

Reference URL in dev:

```text
https://agents.dev.truxonline.com
```

The initial portal is intentionally tiny and replaceable. It exposes:

- authenticated user identity from Authentik;
- `Mes agents`, populated through `GET /v1/agents` on the Control Plane;
- a placeholder entry point for delegated LLM-provider credentials.

Target product flow:

```text
browser
  -> Traefik
  -> Authentik ForwardAuth
  -> INDIBA Portal/BFF
  -> Control Plane
  -> authorized HermesRuntime
```

The browser never receives Kubernetes/Hindsight endpoints or provider secrets.
For Phase 0 the portal forwards Authentik identity claims to the Control Plane,
but the Control Plane must accept those claims only from the authenticated
portal workload/trust path; arbitrary caller-supplied identity headers are not
authoritative.

The ConfigMap-hosted Node portal is a bootstrap implementation so the POC is
usable before the final SvelteKit UX is built. Replacing it must not change the
Portal -> Control Plane trust boundary.

## Namespaces and tenant cells

- `indiba-system`: platform control-plane/operator/gateway/portal trust domain.
- `tenant-indiba`: reference tenant cell. Future tenants receive their own
  namespace, Hindsight logical service, database/credential binding and
  NetworkPolicy/RBAC bindings.

`indiba-system` enforces Pod Security `restricted`.

For Phase 0, `tenant-indiba` enforces `baseline` while auditing/warning against
`restricted`, because the current Hermes image/runtime contract is not yet
qualified under a fully restricted pod security context. This is an explicit
POC exception, not the production target.

Both namespaces start from default-deny ingress/egress.

## HermesRuntime

`HermesRuntime` is namespaced. A Control Plane may create/update/delete it only
in an authorized tenant namespace. The custom operator reconciles it to workload
resources.

The CR contains references and immutable configuration identity, not provider
credentials, raw secret values or raw NFS paths.

For Phase 0 the effective lifecycle is `alwaysOn`; the CRD retains `onDemand`
for the later production scaling design.

`configurationRevision` identifies a separately persisted immutable
`RuntimeConfigurationSnapshot`; the hash is verification, not the snapshot
itself.

## Hindsight

The service contract for INDIBA is:

```text
HermesRuntime
  -> hs-indiba.tenant-indiba.svc:8888
  -> Hindsight tenant release
  -> hindsight_indiba database
```

The current scaffold creates only the service/network contract. It does not
reuse the existing laboratory Hindsight instance as a security boundary.

Required authorization for self-hosted Hindsight:

```text
authenticated workload
  -> custom TenantExtension
  -> RequestContext(AgentIdentity)
  -> OperationValidatorExtension
  -> authorize AgentIdentity + MemoryBank + operation
```

`bank_id` is never accepted as proof of authorization. Cross-bank negative tests
are mandatory for every candidate Hindsight release.

## LLM Gateway and credentials

Target flow:

```text
Hermes/Hindsight -> LLM Gateway -> approved provider
                          |
                          -> OpenBao credential material
```

`LLMProfile` (model/routing/privacy/budget policy) and `LLMCredential`
(authentication material metadata) are separate objects.

Credential ownership supports `tenant`, `principal`, and `AgentIdentity`.
Normal delegated OAuth UX is principal-scoped: a user authorizes a provider once
and grants that credential to several AgentIdentity objects. The LLM Gateway is
the only technical owner allowed to rotate the refresh token.

Native Hermes provider OAuth stored in `~/.hermes/auth.json` is acceptable only
as a POC/transitional mode. The same writable `auth.json` must not be mounted
into multiple runtimes.

The portal is the intended UX for `connect provider`, credential status and
`reauth required`; provider refresh tokens never transit through browser code.

## Privacy/data routing

Provider routing must eventually enforce the tenant `DataPolicy`, including
approved providers, retention/training policy and geographic transfer rules.
ZDR alone is not considered sufficient evidence for lawful offshore transfers.

No prompt/response body should be emitted to technical logs by default.

## Not implemented by this GitOps slice

- Control Plane API/database schema and chat proxy;
- Hermes Operator binary;
- LLM Gateway/provider adapters/OAuth refresh code;
- final SvelteKit portal UX;
- OpenFGA deployment and exact relation model;
- Hindsight authorization extension package;
- tenant-managed PostgreSQL provisioning;
- future Agent Gateway/message/session/wake-up protocol;
- generic Python/browser execution environment strategy.

These are explicit implementation gaps, not hidden assumptions.
