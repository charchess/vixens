# ADR-031: AgentIdentity is the Fabric runtime agent identity

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3709, #3662, #3673  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, agentidentity, hermes, runtime, retrospective

> **Retrospective ADR:** this records a contract already implemented in TXO Fabric before the ADR was written.

## Context

Early Hermes and hAIrem documentation sometimes used `persona` to describe an agent profile or behavioral identity. TXO Fabric subsequently introduced `AgentIdentity` as the explicit API object that represents a logical agent and reconciles its Hermes runtime.

Maintaining both `Persona` and `AgentIdentity` as implied runtime abstractions would create ambiguous ownership for identity, state, lifecycle and routing. The implemented Fabric controller does not need a separate Persona object for the current one-agent-per-runtime model.

## Decision

`AgentIdentity` is the current Fabric runtime agent identity.

The model is:

```text
TenantBundle / tenant
  └── AgentIdentity
        └── one logical Hermes runtime
              ├── behavioral configuration
              ├── private runtime state
              ├── local skills
              ├── external memory binding
              └── authorized capabilities/integrations
```

For the current Fabric contract:

- one `AgentIdentity` represents one logical agent;
- one reconciled Hermes runtime/pod serves that agent;
- Kubernetes pod identity is disposable implementation detail; replacing a pod does not replace the logical `AgentIdentity`;
- Fabric does not maintain an independent runtime `Persona` object between `AgentIdentity` and Hermes;
- configuration/profile objects may describe runtime classes or behavior, but they do not become a second logical agent identity;
- user-facing or historical prose may still use *persona* descriptively, but active Fabric technical documentation must not imply a separate Persona runtime resource.

If a future design needs several independently addressable personas inside one runtime, that is a new architecture decision. It must define ownership of state, memory, routing, authorization and lifecycle rather than being treated as an implicit extension of this model.

## Consequences

### Positive

- identity, lifecycle and reconciliation have one explicit owner;
- documentation maps directly to the Fabric API and controller behavior;
- stable agent identity is independent from ephemeral pod names/IPs;
- future human routing and integration authorization can target `AgentIdentity` consistently.

### Negative

- older Hermes/hAIrem material may continue to contain the word `persona` in historical or descriptive contexts;
- multi-persona-per-runtime designs are intentionally not represented by the current API.

## Alternatives considered

### Keep a separate Persona resource

Rejected for the current model because it would duplicate `AgentIdentity` without an independent lifecycle or authorization boundary.

### Treat each pod as the agent identity

Rejected because pod replacement is an operational event and must not change the logical agent identity.

## References

- #3709 — retrospective Fabric ADR backfill
- #3662 — shared agent workspaces and shared skill libraries
- #3673 — real Hermes agent migration / functional parity
- `apps/60-services/txo-fabric/` — current Fabric API/controller implementation
