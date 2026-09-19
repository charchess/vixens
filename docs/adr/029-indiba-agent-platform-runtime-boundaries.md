# ADR-029: INDIBA agent platform runtime and trust boundaries

- **Status:** Accepted for Phase 0 bootstrap; several production execution details remain open
- **Date:** 2026-09-18
- **Deciders:** Vixens maintainers
- **Tags:** indiba, agents, hermes, hindsight, security, oauth, gitops, portal

## Context

INDIBA needs persistent, user-scoped AI collaborators without making Kubernetes
objects the business source of truth. Hermes runtimes may eventually scale to
zero or be recreated while identity, memory, repositories and credentials
survive.

The platform also needs tenant isolation, bank-level Hindsight authorization,
central provider credential handling including delegated OAuth device flows, and
a user-facing entry point that does not expose Kubernetes/Hindsight directly.

## Decision

### Identity and runtime

`AgentTemplate`, `AgentIdentity`, `MemoryBank`, `AgentHome`,
`LLMCredential` and `RuntimeConfigurationSnapshot` are Control Plane domain
objects.

`HermesRuntime` is the Kubernetes execution contract and may be created or
deleted dynamically outside GitOps. GitOps manages the CRD, controllers,
policies and stable platform services.

### Phase 0 runtime mode

For the first INDIBA POC, authorized Hermes runtimes are `alwaysOn` with one
replica. Phase 0 therefore does not require a wake-up bus, KEDA scale-to-zero or
a session scheduler.

The CRD retains the `onDemand` lifecycle value for later phases, but Phase 0
must not silently introduce a runtime wake-up architecture before it is decided.

### User portal

A user portal/BFF is part of Phase 0.

```text
browser
  -> Traefik
  -> Authentik ForwardAuth
  -> INDIBA Portal
  -> Control Plane
  -> authorized HermesRuntime
```

The browser must not call Hermes, Hindsight or Kubernetes APIs directly.

The bootstrap portal may forward normalized Authentik identity claims to the
Control Plane, but those claims are trusted only when the request originates
from the Portal's authenticated workload/network path. Arbitrary caller headers
are never authoritative. A later verified OIDC/JWT path can replace this Phase 0
mechanism without changing the Portal -> Control Plane boundary.

### Tenant boundary

A tenant receives a logical cell:

- Kubernetes namespace;
- Hindsight logical service/release;
- tenant Hindsight database and SQL credential;
- namespace RBAC and default-deny network policy.

Dedicated infrastructure can replace shared infrastructure later without
changing the Control Plane references.

For Phase 0, the tenant namespace enforces Pod Security `baseline` and audits/
warns against `restricted` until the current Hermes runtime image has been
qualified under a fully restricted pod security context. This is a documented
POC exception; `restricted` remains the production target.

### Hindsight bank authorization

Hindsight service reachability is not enough. Self-hosted Hindsight must map the
authenticated caller to an `AgentIdentity` and authorize the requested bank and
operation.

Target enforcement is a custom Hindsight `TenantExtension` plus
`OperationValidatorExtension`. Bank identifiers are not secrets.

### LLM credentials

The LLM Gateway/Credential Broker is the central provider-authentication
authority.

- `LLMProfile` describes model/routing/data/budget policy.
- `LLMCredential` describes provider authentication metadata and an OpenBao
  secret reference.
- `LLMCredentialBinding` grants use of a credential to allowed principals or
  AgentIdentity objects.
- delegated OAuth is normally principal-scoped;
- one user/provider authorization can serve several authorized agents;
- only the Gateway rotates the refresh token.

Hermes-native OAuth remains an allowed POC exception, never the production
multi-agent credential-sharing mechanism. The portal is the intended user
surface for provider connection/status/re-authentication.

### GitOps boundary

Git manages stable platform resources, including the Portal deployment and its
trust boundary. Runtime objects created in response to user activity are
controller-owned and do not need a corresponding Git commit.

## Consequences

Positive:

- the POC can operate without solving wake-up/queue semantics first;
- users have a product entry point instead of a Kubernetes/operator workflow;
- pods can be replaced without losing agent identity;
- provider credentials are not copied into every Hermes profile;
- bank access is enforceable below the tenant boundary;
- tenant isolation can be strengthened progressively;
- provider routing can enforce privacy/geography policy centrally.

Costs:

- Phase 0 always-on agents consume resources continuously;
- tenant Pod Security temporarily enforces `baseline` instead of `restricted`;
- custom Control Plane, operator, gateway and Hindsight extension software must
  be implemented and maintained;
- gateway/provider protocol compatibility must be tested, especially Codex
  OAuth/streaming/tool-call behavior;
- bank authorization needs cross-bank regression tests;
- managed PostgreSQL and secret lifecycle require explicit runbooks.

## Open decisions

Deferred beyond Phase 0:

- Agent Gateway message persistence and wake-up: direct HTTP vs durable queue;
- KEDA vs direct Control Plane scaling for HermesRuntime;
- concurrency model for simultaneous sessions on one AgentIdentity;
- generic Python/browser tool runtimes and dynamic package policy;
- final OpenFGA relation model;
- final SvelteKit portal implementation and direct OIDC/JWT verification model.

These decisions must not be silently encoded into implementation.
