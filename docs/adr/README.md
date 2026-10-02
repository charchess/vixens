# Architecture Decision Records (ADRs)

Architecture Decision Records preserve durable architecture decisions and their rationale for the Vixens monorepo.

The authoritative catalog is [`000-index.md`](000-index.md). Governance is defined by [ADR-030](030-project-scoped-adr-governance.md).

## What belongs in an ADR

Use an ADR when a decision establishes a durable boundary, ownership model or architectural trade-off that future changes need to understand.

Do **not** use ADRs as a second copy of mutable operational documentation. Current commands, image tags, release procedures, application inventories and runbooks belong in `WORKFLOW.md`, guides, procedures, reference docs or application docs. ADRs should link to those living sources when useful.

GitHub Projects/issues track work and acceptance. ADRs record the architectural result. There is no one-issue-one-ADR rule.

## Numbering

New ADRs use one repository-wide monotonically increasing `ADR-NNN` sequence, regardless of GitHub Project.

Historical numbering/header collisions already exist (notably the old `013` and `018` identities). They are preserved to avoid rewriting history and breaking links. They are called out explicitly in the catalog. **New numbers must be unique.**

Do not infer chronology or precedence from the number alone; use date, status and explicit supersession links.

## Canonical status values

New ADRs use:

| Status | Meaning |
|---|---|
| Proposed | Candidate decision; not authoritative architecture |
| Accepted | Decision made/current unless explicitly superseded |
| Superseded | Replaced by a newer explicit decision |
| Deprecated | Abandoned/no longer recommended without requiring a direct replacement |

Older ADRs may contain `Active` or `Implemented`. Preserve their historical wording; the index may normalize them to the closest canonical meaning.

## Scope and GitHub Project ownership

New ADRs declare both architectural **Scope** and planning **Related Project**.

| Scope | Meaning |
|---|---|
| Repository | Cross-monorepo development/governance/architecture decisions |
| Core | Shared infrastructure/platform foundations independent of a specific product tenant |
| AIaaS | TXO Fabric tenant/agent product contracts |
| Personal | Personal/application architecture consuming Core |

Current Project names are `vixens core`, `vixens roadmap`, and `vixens perso`. A Repository-scoped decision can relate to more than one Project.

Existing ADRs do not need noisy rewrites just to add metadata; their retrospective classification is maintained in the index when that is sufficient.

## Retrospective ADRs

A durable contract may be documented after implementation when the architecture is already established. Mark the ADR explicitly as **retrospective** and link to the issues/PRs/physical acceptance that made the behavior real.

A retrospective ADR records the accepted implementation. It must not make an unresolved proposal look accepted or pretend the record existed before the decision.

## Important current decisions

### Repository workflow and governance

- [ADR-017: Pure Trunk-Based Development](017-pure-trunk-based-single-branch.md) — single `main` architecture; use `WORKFLOW.md` for current promotion mechanics.
- [ADR-030: Project-scoped ADR governance](030-project-scoped-adr-governance.md) — global numbering, scopes, Projects, status and lifecycle rules.

### Application maturity

- [ADR-023: 7-Tier Goldification System v2](023-7-tier-goldification-system-v2.md) — maturity model.
- [ADR-029: Align application maturity with the current platform](029-align-maturity-with-current-platform.md) — current OpenBao/ESO and resource interpretation amendment.

### Core secrets and platform foundations

- [ADR-018: OpenBao / External Secrets and NAS FQDN](018-openbao-external-secrets-and-nas-fqdn.md) — current secret architecture; this number is a documented historical collision.
- [ADR-021: Netbird Native Manifests](021-netbird-native-manifests.md) — current Netbird deployment architecture.
- [ADR-025: Local Path Provisioner](025-local-path-provisioner.md)
- [ADR-028: Retire DataAngel Restore Init from Production Workloads](028-retire-dataangel-prod-zfs.md)

### TXO Fabric / AIaaS

- [ADR-031: AgentIdentity is the Fabric runtime agent identity](031-agentidentity-is-the-fabric-runtime-identity.md)
- [ADR-032: TXO Fabric is tenant-neutral and valid with zero clients](032-fabric-is-tenant-neutral.md)
- [ADR-033: Separate private agent state, shared business data and shared skills](033-separate-agent-state-workspace-and-skills.md)
- [ADR-034: Build Hermes extensions into an immutable TXO runtime image](034-immutable-hermes-runtime-supply-chain.md)
- [ADR-035: Centralize Fabric external inference behind the TXO AI Gateway](035-centralized-fabric-inference-boundary.md)
- [ADR-036: Keep durable agent memory behind the tenant Hindsight service boundary](036-hindsight-external-durable-memory-boundary.md)
- [ADR-037: Separate agent-extensible skills from platform-governed executable capabilities](037-skills-vs-platform-governed-executable-capabilities.md)

Future Fabric decisions are added only after their owning implementation/acceptance establishes a durable contract. Unsettled candidates remain in their owning issues rather than appearing here as premature Accepted architecture.

## Creating a new ADR

1. Start from current `main` and check concurrent PRs.
2. Read [ADR-030](030-project-scoped-adr-governance.md).
3. Allocate the next unused global number; do not renumber historical ADRs.
4. Start from `docs/templates/adr-template.md`.
5. State `Status`, `Scope`, `Related Project` and `Related Issues`.
6. Explain context, decision, consequences and meaningful alternatives.
7. Add explicit supersession links when a previous durable decision changes.
8. Update `000-index.md` and this README when the set of important current decisions changes.
9. Submit through the normal branch → PR → CI workflow.
