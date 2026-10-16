# ADR-042: Declare tenant agent populations in TenantBundle

**Date:** 2026-10-16  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3849, #4087  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, tenantbundle, agentidentity, declarative, lifecycle

## Context

`AgentIdentity` is the stable runtime identity for one logical agent (ADR-031). A normal tenant fleet repeats tenant reference, runtime profile, storage retention and access defaults across multiple standalone manifests. This makes a simple requested population verbose and prevents the tenant declaration from being a complete statement of its intended agent fleet.

hAIrem already contains 13 independently declared `AgentIdentity` resources with durable private state. They must not be silently adopted, recreated, renamed or deleted by introducing population support.

## Decision

The high-level population contract belongs in `TenantBundle.spec.agentPopulation`; `AgentIdentity` remains the only runtime identity and reconciliation target.

The initial contract is intentionally narrow:

```yaml
spec:
  agentPopulation:
    defaults:
      runtime:
        profileRef: hermes-default
      storage:
        retentionPolicy: Retain
    instances:
      - key: a1
        displayName: A1
      - key: a2
        displayName: A2
```

The controller creates only missing generated `AgentIdentity` objects with deterministic names `<tenant-name>-<agent-key>`. Each generated child has a managed label and an owner reference to its `TenantBundle` only after the child has been created from explicit population intent.

The controller does **not** adopt an existing standalone `AgentIdentity`. If the deterministic name already exists without the generated ownership marker, reconciliation is fail-closed and reports an adoption conflict. Migration of a brownfield identity requires an explicit, separately reviewed migration marker and proof that the immutable tenant reference, agent key, runtime storage retention and private-state lifecycle are compatible.

Changing or removing an instance from a population is not implicit deletion. The first implementation reports a retained/orphaned generated child as requiring an explicit lifecycle transition. A later implementation may add a dedicated per-instance lifecycle field after its retention semantics and finalizer behavior are accepted.

Human references and Authentik group memberships remain part of the tenant declarative contract, but enrollment in the Fabric identity ledger continues to require a verified signed OIDC identity. Population YAML never creates a human ledger binding or OpenFGA authorization tuple by itself.

## Consequences

### Positive

- one tenant manifest can request a named standard fleet without duplicating runtime boilerplate;
- generated agents retain their own immutable keys, private state, Hindsight banks and model-access resources;
- standalone `AgentIdentity` manifests remain supported;
- initial hAIrem adoption is explicitly blocked rather than risking durable state;
- human identity authorization remains cryptographically grounded.

### Negative

- the first API intentionally does not offer automatic brownfield adoption;
- removal needs a later explicit lifecycle contract instead of behaving like a Deployment replica scale-down;
- per-agent deviations use bounded overrides only after defaults/merge rules are tested.

## Alternatives considered

### Separate AgentPopulation custom resource

Rejected for POC 1. The product request is a complete tenant definition containing tenant groups, human references and agents. Keeping the population on `TenantBundle` gives one declared tenant boundary while retaining `AgentIdentity` as the runtime primitive.

### Implicitly adopt existing AgentIdentity objects

Rejected. It can silently change owner references and deletion/lifecycle behavior for durable workspaces. An explicit migration is required.

### Treat instances as anonymous replicas

Rejected. Each agent has a distinct identity, agent key, memory bank and private runtime state.

## References

- ADR-031 — AgentIdentity is the Fabric runtime agent identity
- ADR-039 — Fabric OpenFGA authorization boundary
- #3849 — declarative tenant agent populations
- #4087 — hAIrem POC 1 delivery
