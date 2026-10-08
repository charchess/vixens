# TXO Fabric — agent functional configuration (native v0.1 slice and proposed extensions)

> **Implementation note (2026-10-08):** the minimal `AgentFunctionalProfile`
> CRD and optional per-agent reference exist. The initial native Hermes
> implementation renders `spec.instructions` as a tenant-owned read-only
> **SKILL.md** reference. It does **not** inject or enforce a global system
> prompt, and never edits `SOUL.md`. The larger `role/requirements`
> structure illustrated below remains **Proposed / NON-DEPLOYABLE**.
> Do not apply the example manifests to real tenants or promote without
> explicit approval and physical acceptance.

## Why the split matters

Today an `AgentIdentity` selects `hermes-default` and retains its private
state; `AgentRuntimeProfile` fixes the runtime and the maximum platform
toolset policy. Skills, workspace scopes and integration access are already
separate and authorized. None of those resources is a reusable tenant-owned
*business-function baseline* that two already-deployed agents can reference.

Target composition:

```text
TenantBundle/indiba
  ├── AgentFunctionalProfile/indiba-sales        # proposed, tenant-owned
  │      └── non-secret role instructions + requirements
  ├── AgentIdentity/ten00002-usr000001-agt00001 (Sam)
  │      ├── runtime.profileRef: hermes-default
  │      ├── functional.profileRef: indiba-sales   # proposed
  │      ├── /opt/data + private/local skills (Sam)
  │      └── Hindsight bank ten00002-usr000001-agt00001
  └── AgentIdentity/ten00002-usr000001-agt00002 (Alex)
         ├── runtime.profileRef: hermes-default
         ├── functional.profileRef: indiba-sales   # proposed
         ├── /opt/data + private/local skills (Alex)
         └── Hindsight bank ten00002-usr000001-agt00002
```

Sales, HR and prospecting are **roles**, not runtime engines. The same role
need not imply the same agent identity, bank, user, credentials or access.

## Proposed v0.1 API sketch (NON-DEPLOYABLE)

The names and fields below are **candidates**, not a new API contract yet.
The source of approved baseline instructions and its read-only rendering
surface must be validated against the pinned Hermes release before implementing.

```yaml
# Illustrative shape only; API/CRD does NOT exist.
apiVersion: fabric.truxonline.io/v1alpha1
kind: AgentFunctionalProfile
metadata:
  name: indiba-sales
spec:
  tenantRef:
    name: indiba
  role:
    displayName: Sales assistant
    # Only approved, non-secret tenant-scoped sources; exact contract TBD.
    instructionsRef:
      configMapName: indiba-sales-instructions
      key: role.md
  requirements:
    skills:
      - scope: group
        ownerKey: sales
        skillName: soncas
    integrations: [] # no authorization/grants implied
    logicalModels: [txo-agent] # capability requirement, not credential
    toolsets: [] # check platform ceiling; no automatic AllowedOff activation
---
# Conceptual AgentIdentity additions only; not valid against today's CRD.
kind: AgentIdentity
metadata:
  name: ten00002-usr000001-agt00001
spec:
  tenantRef:
    name: indiba
  agentKey: usr000001-agt00001
  displayName: Sam
  runtime:
    profileRef: hermes-default
  functional:
    profileRef: indiba-sales
---
kind: AgentIdentity
metadata:
  name: ten00002-usr000001-agt00002
spec:
  tenantRef:
    name: indiba
  agentKey: usr000001-agt00002
  displayName: Alex
  runtime:
    profileRef: hermes-default
  functional:
    profileRef: indiba-sales
```

The active Indiba manifests already bind their existing agents to user scope
`edfoley`, group `sales`, and to separate Hindsight banks. The technical
workspace user-to-owner-ID/IAM mapping remains open under #3802/#3852.
**Do not** replace `edfoley` with `usr000001` in live workspace access
or rename banks/PVCs to make this example true.

## Responsibilities and fail-closed resolution

| Source | What it owns | What it must never grant |
| --- | --- | --- |
| TenantBundle + Authentik | Tenant structure, live human membership (Authentik) | Agent executable/toolset authority |
| AgentIdentity | Stable logical identity, runtime binding, existing access grants, private bank/PVC | Provider OAuth credentials |
| AgentRuntimeProfile | Hermes engine/image, sizing, maximum toolset policy | Customer business roles |
| AgentFunctionalProfile (**proposed**) | Reusable approved role baseline and prerequisites | Toolsets, workspace mounts, integration grants, upstream credential access |
| Existing workspace/skill sources | Authorized shared skills and files | New toolsets or provider access merely because files are visible |
| IntegrationBinding | Per-agent integration operations/scopes | Business-role identity |
| LiteLLM/CPA | Per-consumer model authorization, backend credentials | Cross-tenant provider use |

An agent selects at most one tenant-local functional profile in the first
slice. Missing profiles, a foreign tenantRef, missing required skill or binding,
an unavailable/unauthorized logical model, or a forbidden/inactive toolset
produce a visible non-secret incompatible/not-ready state; **never** silently
fall back to broader access.

A functional role's required `delegation` cannot activate a runtime
`AllowedOff` toolset without the explicit existing AgentIdentity activation.
A requirement for `computer_use` with a runtime `Off` must fail. Platform
toolset policy and network/credential controls continue to apply regardless
of role instructions or a user-writable private config file.

The functional baseline is managed and read-only from the consumer's
perspective. Local `/opt/data` skills and personal settings remain writable;
role reconciliation must not overwrite them, clear cron, replace the private
PVC, or rename the Hindsight bank. A common role update produces a stable
effective revision/controlled rollout; no-op reconciliation does not.

## Product walkthrough (acceptance slice)

1. Declare one approved Indiba `sales` functional profile without changing
   Sam/Alex IDs or existing runtime, bank, access grants or PVC.
2. Bind both Sam and Alex to it. Their role baseline matches; each retains
   independent private settings/skills and Hindsight bank.
3. Use the authorized shared group `sales` skill library. A required but
   inaccessible skill is not automatically mounted.
4. Change one approved common sales instruction: both agents adopt the same
   revision without rewriting private content or reissuing provider secrets.
5. Customize Sam locally: Alex is unaffected. Rename Sam's display label:
   neither bank nor PVC/resource identity changes.
6. Check negative cases: incompatible forbidden toolset; missing binding;
   foreign-tenant profile reference; unauthorized model. All fail closed.
7. For a real conversational Indiba acceptance, add a separately authorized
   non-Codex chat provider route under a narrow #3869 slice. Do not enroll
   Codex OAuth merely to exercise this role-profile feature.

No population add/remove until #3824's Retain PVC lifecycle risk is fixed.
#3849 subsequently owns template expansion and stable population materialization;
this proposal governs only what each created agent may **reference**.

## Implementation slices after ADR review

1. Confirm Hermes' actual supported instruction/inclusion interfaces and
   define managed baseline vs private personalization precedence without
   mutating existing private config.
2. Introduce the smallest CRD/reference, indexed tenant/profile reconciliation,
   effective revision and fail-closed status with unit/reconciliation tests.
3. Add an Indiba sales baseline fixture only in a controlled GitOps PR; test
   compatibility without a direct production promotion.
4. Prove two real agents' shared baseline and private-state independence; then
   consider merging this design into an Accepted ADR after evidence.
5. Feed the reference contract into population work #3849; keep owner/group
   ontology in #3852/#3856 and authorization semantics in #3802.

## Related decisions and work

- [ADR-031](../../../docs/adr/031-agentidentity-is-the-fabric-runtime-identity.md)
- [ADR-033](../../../docs/adr/033-separate-agent-state-workspace-and-skills.md)
- [ADR-037](../../../docs/adr/037-skills-vs-platform-governed-executable-capabilities.md)
- [ADR-038 proposed](../../../docs/adr/038-reusable-agent-functional-configuration.md)
- #3851, #3852, #3856, #3802, #3849, #3824, #3869
