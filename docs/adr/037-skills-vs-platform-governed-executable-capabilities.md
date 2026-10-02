# ADR-037: Separate agent-extensible skills from platform-governed executable capabilities

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3710, #3686, #3718, #3662  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, hermes, skills, toolsets, capabilities, authorization, retrospective

> **Retrospective ADR:** this records the skills-versus-executable-capability boundary implemented and physically accepted under #3686/#3718 before this ADR was written.

## Context

Hermes supports two extension surfaces that look superficially similar but have different risk and ownership semantics:

- **skills** are knowledge/procedure that an agent can learn, personalize and share;
- **toolsets/capabilities** expose executable authority such as terminal, browser, code execution, delegation, integrations or dynamically registered tools.

Treating both as agent-owned mutable configuration would allow an agent to turn local self-improvement into executable privilege escalation. Treating both as centrally immutable would remove an intended product property: agents should remain able to create and refine local skills.

ADR-033 already establishes the private/shared storage and trust domains for skills. This ADR adds the executable-capability boundary without changing those storage semantics.

## Decision

### Skills remain agent-extensible knowledge/procedure

Agents may continue to create, modify and personalize local skills within their private writable runtime domain.

Shared skills retain the trust, mount and authorization semantics defined by ADR-033. Publishing or consuming a skill does not by itself grant a new executable capability.

A skill may describe how to use an allowed tool, but it is not an authorization mechanism for obtaining that tool.

### Executable capability policy is platform-owned

The maximum Hermes executable capability surface is owned by platform-controlled runtime/template policy, not by mutable agent-local configuration.

Fabric expresses this through generic `AgentRuntimeProfile` capability policy rather than customer-specific controller branches.

The accepted policy model has three states:

- **On** — admitted by the platform profile and enabled;
- **Off** — forbidden by the platform profile;
- **AllowedOff** — admitted by the profile but disabled until explicitly activated for the AgentIdentity through the intended management binding.

An AgentIdentity may request only capabilities that the profile marks `AllowedOff`. It cannot use its local writable state to convert an `Off` or undeclared capability into an enabled one.

### Agent-local state may narrow, but never widen, the platform ceiling

Mutable Hermes configuration under the private runtime may disable or otherwise narrow an already-admitted capability surface.

It must not widen the surface beyond the effective platform policy.

The platform enforcement state therefore lives outside the agent-writable private configuration boundary and is applied through the final runtime capability-resolution path.

### Dynamic executable extension fails closed

Dynamic plugin/MCP discovery, lazy installation or similar runtime extension paths must not silently expand the executable surface beyond the platform admission catalog.

Platform-reviewed executable additions belong to the immutable runtime supply chain in ADR-034 and to explicit capability admission policy.

The durable architectural requirement is fail-closed admission. Exact Hermes compatibility fields, environment guards and catalog contents remain living implementation details.

### Capability policy is observable and reconciled

The effective capability policy is observable without exposing secret material.

A policy change, including an `AllowedOff` activation/deactivation, must cause observable reconciliation/rollout so operators can causally relate:

`profile/AgentIdentity intent -> effective policy -> runtime generation -> actual Hermes capability surface`.

### Toolset governance is not an operating-system sandbox

Capability registration controls which Hermes tools are exposed to the model. It is not a syscall or network sandbox.

A deliberately admitted broad capability such as terminal/code execution may overlap narrower toolsets.

Network reachability, third-party credentials and integration authorization are separate boundaries. They remain governed by their own contracts, including #3687, and must not be inferred merely from a toolset being `Off` or `On`.

## Consequences

### Positive

- agents keep self-improvement and personalization through local skills;
- shared skill libraries remain usable without becoming an executable-privilege channel;
- a writable `/opt/data` configuration cannot re-enable platform-forbidden toolsets;
- capability policy is generic across tenants/customers;
- executable additions pass through reviewed platform/runtime governance;
- `AllowedOff` supports explicit opt-in capabilities without making them agent-self-service by default;
- effective runtime capability can be audited and tied to reconciliation evidence.

### Negative

- adding or changing executable capability requires platform policy/review rather than only agent-local configuration;
- the platform must maintain compatibility between the pinned Hermes runtime catalog and Fabric capability policy;
- broad admitted tools can still overlap narrower policy intentions, so capability governance cannot substitute for network/credential isolation;
- dynamic plugin ecosystems require deliberate admission instead of automatic discovery/activation.

## Alternatives considered

### Let each agent fully manage Hermes toolsets in `/opt/data`

Rejected because the private runtime is intentionally writable by the agent and therefore cannot be the authority for a non-bypassable `Off` state.

### Make all skills centrally immutable

Rejected because agent-local skill creation and personalization are intended product behavior and were physically preserved under ADR-033/#3662.

### Use prompt instructions to forbid tools

Rejected because prompt text does not remove executable tool registration or prevent local configuration/plugin paths from exposing capabilities.

### Treat disabling a toolset as a complete security sandbox

Rejected because terminal/code execution/network authority can overlap narrower tool behavior. Capability registration, network egress and integration credentials are separate security layers.

### Auto-admit newly discovered plugins/toolsets

Rejected because an upstream/runtime extension could silently widen the agent's executable authority without a platform policy decision.

## Physical acceptance evidence

The #3686/#3718 acceptance flow physically proved that:

- a platform-`On` terminal tool executed successfully through real Hermes;
- `delegation` (`AllowedOff` baseline) and `computer_use` (`Off`) were absent in the runtime;
- a conflicting agent-local configuration attempt could not re-enable forbidden capability;
- local/shared skills remained available after capability enforcement;
- GitOps activation of `delegation` through the AgentIdentity binding changed the policy revision, rolled the Hermes pod and exposed delegation while `computer_use` remained denied;
- GitOps removal of that binding caused a second revision/rollout and returned `delegation` to denied;
- the final runtime returned exactly to the previously observed baseline policy revision.

Exact toolset inventories, current default states, compatibility shims, cron details and webhook follow-up direction remain in living capability documentation.

## References

- #3686 — Hermes toolset governance parent objective
- #3718 — implementation and complete physical `AllowedOff` round-trip acceptance
- #3662 — shared/local skill trust and lifecycle acceptance
- #3710 — accepted v0 platform-contract ADR harvesting
- ADR-033 — separate private state, shared business data and shared skills
- ADR-034 — immutable Hermes runtime supply chain
- `apps/60-services/txo-fabric/CAPABILITIES.md` — living capability inventory and compatibility contract
