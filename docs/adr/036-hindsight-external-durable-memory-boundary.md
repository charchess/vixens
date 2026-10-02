# ADR-036: Keep durable agent memory behind the tenant Hindsight service boundary

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3710, #3685, #3697, #3674, #3703, #3706  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, hermes, hindsight, memory, lifecycle, credentials, retrospective

> **Retrospective ADR:** this records the Hermes/Hindsight memory boundary implemented and physically accepted under #3685/#3697 before this ADR was written.

## Context

A Fabric `AgentIdentity` has private Hermes runtime state under `/opt/data`, but durable semantic memory has different sharing, lifecycle and security requirements.

Making the private runtime PVC the only durable-memory domain would couple long-term memory to runtime replacement and retention policy. Giving Hermes direct PostgreSQL access would also expose database credentials and persistence details to the agent runtime even though Hindsight already owns that storage concern.

The accepted v0 implementation established Hindsight as a tenant-scoped memory service and physically proved that an agent can lose and recreate its private runtime state while reconnecting to the same durable memory bank.

This decision is distinct from runtime composition and inference governance:

- ADR-034 governs how required Hermes providers/extensions are supplied immutably;
- ADR-035 governs external LLM/embedding inference and credentials;
- this ADR governs the logical memory-service, identity and lifecycle boundary.

## Decision

### Hindsight is the external durable memory service

When a `TenantBundle` enables Hindsight memory, Fabric-managed Hermes runtimes use that tenant-local Hindsight service as their external durable memory provider.

Hindsight is additive to Hermes local/private state. Selecting the external provider does not turn Hindsight into the owner of all Hermes runtime state and does not replace `/opt/data`.

### Memory bank identity follows the logical AgentIdentity

Each `AgentIdentity` resolves a deterministic Hindsight bank identity.

The bank identity is stable for the same logical tenant/agent identity so a recreated runtime can reconnect to the same durable memory. The current derivation and override fields are living API details documented in the Fabric implementation; changing the formula without changing the logical identity contract does not require a new ADR.

Different AgentIdentities do not implicitly share a bank merely because they belong to the same tenant.

### PostgreSQL stays behind Hindsight

Hermes talks to the Hindsight service API, not directly to Hindsight's PostgreSQL datastore.

Database topology and PostgreSQL credentials remain owned by the Hindsight service boundary. They are not projected into the Hermes runtime and are not part of the `AgentIdentity` contract.

Hermes receives only the service-level Hindsight authorization required to access the tenant memory service. Secret values remain outside Git and non-secret status/diagnostics.

### Private runtime lifecycle and durable memory lifecycle are independent

The private Hermes PVC and the Hindsight bank are separate lifecycle domains.

- `Retain` / `Delete` for the private runtime controls `/opt/data`;
- replacing a pod does not replace the logical memory bank;
- recreating an AgentIdentity with the same logical identity may reconnect to the same bank even when its disposable private PVC was replaced;
- deletion of Hindsight memory is governed by the tenant memory/database lifecycle, not as an accidental side effect of deleting one Hermes PVC.

This separation is deliberate: private local state can be reset without silently erasing tenant-owned durable memory.

### Memory-service credentials do not grant inference or database authority

A Hindsight API credential authorizes use of the memory service. It is not a PostgreSQL credential and does not grant upstream model-provider authority.

Hindsight embeddings and any future generative/reflection inference remain subject to ADR-035. Enabling durable memory must not create a direct provider-credential or direct-provider-egress bypass.

### Provider composition and provider selection remain separate

The Hindsight provider required by Hermes is supplied through the immutable runtime mechanism in ADR-034.

Baking the provider into the image does not automatically enable it for every agent. Fabric configuration decides whether a tenant/AgentIdentity uses Hindsight and supplies the corresponding service endpoint, bank identity and scoped service credential.

## Consequences

### Positive

- durable memory survives disposable Hermes runtime/PVC replacement when the logical bank identity is preserved;
- PostgreSQL credentials and topology stay outside agent runtimes;
- memory isolation can be reasoned about per logical AgentIdentity bank;
- private/local Hermes state remains independently retainable or disposable;
- memory-provider composition is reproducible without making runtime startup depend on plugin downloads;
- external inference remains governed by the centralized gateway boundary instead of being smuggled through the memory feature.

### Negative

- operating an agent with durable memory now depends on the tenant Hindsight service and its persistence layer;
- lifecycle operations must distinguish private runtime deletion from durable memory deletion;
- diagnostics must consider both local Hermes state and the external memory service;
- stable logical bank identity becomes part of compatibility expectations during AgentIdentity migrations/refactors.

## Alternatives considered

### Keep all durable memory only in the private Hermes PVC

Rejected because it couples long-term memory to runtime/PVC lifecycle and makes disposable runtime replacement destructive to durable memory.

### Give Hermes direct PostgreSQL access

Rejected because it leaks persistence credentials/topology across the service boundary and makes Hermes responsible for Hindsight's storage implementation.

### Use one tenant-wide memory bank for every agent

Rejected because it removes the accepted per-AgentIdentity isolation boundary and makes memory provenance/authorization ambiguous.

### Copy Hindsight data when an AgentIdentity is recreated

Rejected because the stable logical bank is already the durable identity. Copying memory through the private-runtime lifecycle would conflate two intentionally separate domains.

## Physical acceptance evidence

The accepted #3685 flow physically proved that:

- Hermes used the Hindsight provider against the tenant-local service;
- a retained marker was recalled by a fresh provider process/session;
- a second AgentIdentity using a different deterministic bank did not recall the first agent's marker;
- deleting/recreating the disposable AgentIdentity and its private PVC removed private-runtime state while the recreated runtime reconnected to the same Hindsight bank and recalled the durable marker;
- Hermes did not receive PostgreSQL/database credentials;
- Hindsight generative LLM remained disabled while embeddings stayed routed through the TXO AI Gateway.

Exact provider versions, endpoints, bank-format strings and diagnostic commands remain living implementation documentation.

## References

- #3685 — Hermes/Hindsight functional objective and final physical acceptance
- #3697 — per-AgentIdentity Hindsight provider wiring
- #3696 — immutable Hermes runtime/provider supply chain
- #3674 — centralized inference boundary
- #3703 — accepted single-Hermes-home topology
- #3706 — recall correctness adjustment and physical acceptance
- #3710 — accepted v0 platform-contract ADR harvesting
- ADR-033 — private agent state and shared-data/skills lifecycle boundaries
- ADR-034 — immutable Hermes runtime supply chain
- ADR-035 — centralized Fabric inference boundary
- `apps/60-services/txo-fabric/` — living Fabric memory/runtime configuration
