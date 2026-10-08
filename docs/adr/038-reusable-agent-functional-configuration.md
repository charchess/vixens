# ADR-038: Reusable functional configuration separate from agent identity and runtime

**Date:** 2026-10-08  
**Status:** Proposed  
**Scope:** AIaaS  
**Related Project:** vixens roadmap  
**Related Issues:** #3851, #3852, #3856, #3802, #3849, #3848, #3869  
**Deciders:** Vixens maintainers  
**Tags:** txo-fabric, agentidentity, functional-configuration, skills, policy, reusable-profiles

> **Partially implemented / not physically accepted:** the minimal tenant-owned
> `AgentFunctionalProfile.spec.instructions` and optional per-agent reference now
> exist. The first implementation deliberately uses a native read-only Hermes
> skill, *not* a compulsory system prompt. The richer requirements and role
> structure below remain proposed and are not current CRD fields. No live-agent
> adoption or production promotion is authorized by this ADR.

## Context

Fabric already separates immutable logical agents (`AgentIdentity`) from
platform-owned engines and runtime security ceilings (`AgentRuntimeProfile`).
Existing distinct, durable tenant resources must remain stable if an agent's
business role changes. The four Indiba identities are declared separately today:
Sam, Alex, Clover and Jerry all use `hermes-default`.

Reusable commercial/HR/support behavior is currently scattered across agent-local
private state, authorized shared skill mounts, workspace access, integration grants,
and capability requests. Making one `hermes-sales` runtime profile would mix
business intent with executable privileges and immutable images. Repeating the
same prompts, required skills and expected integration classes in every
`AgentIdentity` would prevent safe shared updates. Waiting for population
templates (#3849) alone would also leave pre-existing independently declared
agents without a reusable functional binding.

The proposal must preserve ADR-031 (one logical identity), ADR-033 (private and
shared skill/storage domains) and ADR-037 (platform toolset admission), and must
not introduce a second Persona identity or bypass tenant authorization.

## Proposed decision

### Three independent axes

1. **Identity and ownership** — `AgentIdentity` is the only logical agent,
   with immutable tenant/agent technical identifiers, independent bank and PVC,
   and mutable display name. Owner scope and workspace grants are separate;
   #3852/#3856/#3802 define the durable ownership/IAM model.
2. **Execution** — `AgentRuntimeProfile` chooses the engine/image, storage
   policy, sizing and maximum executable capability admission. A functional
   profile cannot alter the image, pod policy, model-access credentials or
   executable ceiling.
3. **Behavior** — a reusable, versioned, tenant-owned **functional profile**
   describes a non-secret baseline role/instructions and its *requirements* for
   skills, logical model capabilities and external integrations, independently of
   the runtime engine. Agent-local personalization remains private/writable.

### Minimal first-class reference

For the v0.1 implementation slice, favor a cluster-scoped
`AgentFunctionalProfile` carrying an immutable `spec.tenantRef.name`, and
an optional `AgentIdentity.spec.functional.profileRef`.

* The profile is reusable by any number of agents **within its tenant**.
* For v0.1 an agent selects **at most one** profile. An absent binding preserves
  the existing runtime/AgentIdentity behavior; there is no implicit business role.
* A profile name is unique globally as a Kubernetes resource; resolution must
  verify the profile's immutable tenantRef matches the caller's TenantBundle.
  Reject missing, foreign, stale or ambiguous profile references fail closed.
* Platform-curated reusable templates may later be *published/copy-provisioned*
  into a tenant-owned profile; this is **not** a cross-tenant reference to a
  mutable shared authority or automatic entitlement.
* Do not introduce additional identity, user or group CRDs in this slice.
  Population templates (#3849) will default a functional reference when available
  but do not themselves define the meaning of a business role.
* Keep the new CRD/field design **provisional** pending #3851 approval and
  explicit compatibility review; the YAML sample in the living design is
  intentionally non-deployable.

### Fields and precedence

Keep the v0.1 profile minimal:

* A role label/description and declarative **non-secret instruction source**
  authorized for this tenant. Do not silently reinterpret arbitrary mutable
  business files or skill text as trusted managed system instructions.
* Names of **required** skills from already-authorized mounted sources,
  resolved by Fabric against the actual effective user/group/org skill scopes.
  A profile cannot create mounts, rewrite skill names or grant access.
* Names of **required** logical model capabilities; model selection and actual
  provider credentials remain with the gateway and #3869/#3868.
* Names/types of **required** integration capabilities only; authorization
  continues to require per-agent `IntegrationBinding`, never copied credentials.
* Optional **requested** toolsets are expectations, **not activation grants**:
  `AgentRuntimeProfile` remains the admission ceiling, and a
  `AllowedOff` toolset still requires explicit
  `AgentIdentity.spec.runtime.capabilities.enableToolsets`. An `Off`,
  undeclared or inactive requirement makes the configuration incompatible.

Precedence is *not* "merge and last writer wins". Security-admission and access
constraints are intersected first; profile prerequisites must be met or the
binding becomes not-ready without elevating rights. The tenant-approved role
baseline remains read-only and reproducible. The agent's existing private
`/opt/data` behavior and writable local skills may add personalization and
narrow choices but cannot overwrite protected platform policy or acquire a
permission through instructional content. For contradictory business
instructions, choose an explicit source and precedence in the runtime adapter
rather than concatenating arbitrary files or destructive rewrites.

A missing/incompatible profile exposes a specific non-secret
`FunctionalConfigurationReady=False` condition and **does not** roll out a
newly expanded runtime policy. An already-running agent must not be silently
converted to an unrestricted default on profile disappearance; the
implementation must define deterministic fail-closed withdrawal/recovery,
including a pod-replacement scenario.

### Revision, updates and durability

* Compute an observable **effective functional revision** from the approved
  profile's content/source revision and resolved requirements; on a real change,
  reconcile the affected agents with a controlled rollout when required.
* A no-op reconcile must not trigger a rollout or key regeneration.
* Updates apply to all agents referencing that tenant profile but do not rewrite
  or delete private `/opt/data`, Hindsight banks, retained PVCs, cron state,
  local skills or per-agent display names.
* Removing a binding returns the agent to its explicitly authorized baseline,
  without deleting local customization. Revoking a required skill/integration
  must not leave the functional profile reporting Ready.
* Authentication, IAM membership, file mounts and integration credentials
  remain enforced by their existing systems; prompts are not access controls.

### Compatibility and migration

Existing hAIrem and Indiba `AgentIdentity` resources without a functional
binding remain valid. No automatic re-keying of `agentKey`, metadata.name,
bankId or runtime PVC is permitted. The already-provisioned technical key
scheme from #3853 remains compatible; its mapping to long-term IAM/owner
identities is explicitly open under #3852/#3856/#3802.

Before changing existing populations, resolve #3824: a known Retain PVC
ownerReference/garbage-collection race risks loss of the PVC object during
agent deletion. Functional-profile changes themselves must not delete or
recreate AgentIdentity resources.

### Deferred deliberately

Multiple composable role overlays, platform-global mutable profiles, customer
self-service editors, arbitrary ConfigMap/Secret references, per-agent
privilege-bearing overrides, role-driven workspace grants, cross-tenant profile
consumption, and automated migration of private agent prompts/skills are
deferred until concrete users justify them. These cannot be treated as implied
effects of this ADR.

## Evidence needed before acceptance

A first implementation should prove with Sam and Alex in Indiba:

1. same `hermes-default` runtime and shared approved sales baseline, but
   distinct technical identities, user-facing names, Hindsight banks and
   retained private state;
2. same functional revision when the shared baseline is unchanged, updated
   revision/rollout after a legitimate common change, no rollout on no-op;
3. each agent retains separately editable local skills/preferences while
   consuming the authorized group skill library;
4. disallowed toolsets, absent authorizations, foreign-tenant profile refs and
   unauthorized models/integrations are rejected without privilege escalation;
5. removal of a profile or access grant fails closed and recovers on
   reconciliation without deleting private memory, skills or PVCs;
6. current unbound agents remain unchanged, including hAIrem;
7. no profile instruction text, credential or account material appears in
   status/events beyond explicitly log-safe metadata.

A real **Indiba conversational** scenario additionally requires a configured
and authorized chat provider (currently deliberately no Indiba Codex OAuth);
that is a separately scoped #3869 slice, not a reason to grant CodeX OAuth.

## Consequences

### Positive

- one reusable functional configuration for already existing and future agents;
- business roles do not explode the number of runtime profiles;
- local Hermes self-improvement and shared skill trust are preserved;
- stable agent/memory identity and tenant-specific authorization remain intact;
- #3849 can express concise populations without becoming an IAM or role engine.

### Negative

- one new tenant-owned declarative API introduces controller/status/revision
  logic, migrations and validation work;
- content/source compatibility with pinned Hermes must be verified before
  implementing instruction rendering;
- all agents referencing a role may roll during an approved common change;
- v0.1 deliberately does not implement complex overlay inheritance.

## Alternatives considered

### Use only population defaults from #3849

Not sufficient: Sam, Alex and other agents exist as independent AgentIdentity
resources today, and shared functional updates must not require rewriting or
replacing those identities.

### Put business roles into AgentRuntimeProfile

Rejected: image/sizing and executable capability admission must not become
business-role configuration. This would also create combinatorial runtime types.

### Use a second Persona/agent object

Rejected by ADR-031: duplicates lifecycle/ownership and destabilizes memory
and private state.

### Mount one writable organization-wide instructions/skills directory

Rejected: writable business files and skill libraries have different trust
semantics (ADR-033), and a profile must not acquire new privileges by
causing a broad workspace mount.

## References

- #3851 — functional-role contract and implementation decision
- #3852, #3856, #3802 — canonical ID, ownership and IAM/workspace separation
- #3849 — declarative agent populations
- #3824 — Retain PVC lifecycle safety prerequisite for population mutation
- ADR-031 — logical AgentIdentity
- ADR-033 — private/shared storage and skills
- ADR-037 — executable toolset admission
- [Functional configuration design](../../apps/60-services/txo-fabric/FUNCTIONAL-CONFIGURATION.md)
