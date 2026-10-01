# ADR-033: Separate private agent state, shared business data and shared skills

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3709, #3662  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, storage, skills, workspace, authorization, retrospective

> **Retrospective ADR:** this records the storage/trust contract implemented and physically accepted under #3662 before this ADR was written.

## Context

A useful multi-agent tenant needs three kinds of durable data with different trust, mutability and lifecycle semantics:

1. private runtime state owned by one agent;
2. business files shared between authorized people/agents;
3. procedural/behavioral skills shared between authorized agents.

Treating these as one large shared filesystem would make authorization implicit and could accidentally turn ordinary business content into behavioral input. Treating all skills as centrally immutable would also remove a useful Hermes property: each agent can create and improve its own local skills.

## Decision

TXO Fabric keeps the three domains physically and semantically separate.

### Private agent runtime: `/opt/data`

- private to one `AgentIdentity`;
- backed by the agent's RWO runtime storage;
- writable by the agent;
- contains Hermes profile/config/state and locally created or personalized skills;
- follows the explicit private-runtime retention lifecycle (`Retain` or `Delete`).

### Shared business workspace: `/workspace/shared`

- contains organization/group/user business files and collaboration data;
- uses shared RWX storage independent from an individual agent PVC;
- `reference` content is mounted read-only to consumers;
- `collaborative` content is writable only by authorized populations;
- business data is not automatically registered as Hermes skill input.

### Shared skill libraries: `/workspace/skills`

- contains shared procedural/behavioral knowledge intended for Hermes skill discovery;
- uses storage separate from shared business files even when both use the same storage backend;
- `skillsReference` is read-only to consuming agents;
- `skillsCollaborative` is writable by the authorized population;
- Hermes discovers only the libraries mounted for the agent;
- agents remain free to keep private/local skills under `/opt/data`.

Organization/group/user membership is the common authorization population model for both shared business data and shared skills. Fabric does not introduce a second IAM system solely for skills.

Authorization is enforced by the effective storage/mount topology, not by prompt instructions. An agent must not receive a broad tenant-wide writable mount when only a narrower scope is authorized.

Shared workspace and shared skill lifecycle belongs to the tenant/shared domain, not to one `AgentIdentity`. Replacing or deleting an agent does not implicitly delete shared data.

### Skill-name collisions

For the pinned Hermes runtime used when this decision was accepted, a local and external skill with the same bare name produce an explicit collision requiring disambiguation/rename. Fabric does not define a fictional local-over-shared precedence rule. If upstream Hermes changes this behavior, living compatibility documentation and tests must be updated; a change to the Fabric trust model itself would require a new decision.

## Consequences

### Positive

- private state, collaboration data and behavioral input have explicit trust boundaries;
- agents retain self-improvement/personalization through local skills;
- teams can collaborate on skills without granting every agent tenant-wide write access;
- reference material can be physically immutable from the consumer runtime;
- shared data survives individual runtime replacement.

### Negative

- more PVCs/mounts are required than a single shared tenant filesystem;
- authorization and lifecycle tests must cover each enabled scope/mode;
- users must consciously publish/copy material between private, collaborative and reference domains.

## Alternatives considered

### One tenant-wide shared writable filesystem

Rejected because it collapses authorization and lifecycle domains and creates excessive lateral write access.

### Put shared skills inside the business workspace

Rejected because ordinary shared files must not automatically become behavioral input.

### Make all skills centrally managed and read-only

Rejected because it would remove per-agent local skill creation and adaptation, which is part of the intended Hermes operating model.

## References

- #3662 — implementation and physical acceptance of shared workspaces/skill libraries
- #3709 — retrospective Fabric ADR backfill
- `apps/60-services/txo-fabric/SHARED-SKILLS.md` — living skill-sharing contract
