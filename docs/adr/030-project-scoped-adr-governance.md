# ADR-030: Project-scoped ADR governance

**Date:** 2026-10-02  
**Status:** Accepted  
**Scope:** Repository  
**Related Project:** vixens core / vixens roadmap / vixens perso  
**Related Issues:** #3707, #3546, #3690  
**Deciders:** Vixens maintainers  
**Tags:** architecture, adr, governance, documentation

## Context

Vixens is a monorepo with three GitHub Projects that serve different planning domains:

- `vixens core` for shared infrastructure and platform foundations;
- `vixens roadmap` for the tenantized TXO Fabric / AI-as-a-Service product;
- `vixens perso` for personal applications that consume the shared foundation.

Architecture Decision Records predate that split. The repository already has useful historical ADRs, but the catalog mixes current and superseded decisions, uses several status spellings, and contains historical numbering/header collisions. GitHub Projects and ADRs also solve different problems: Projects track work, while ADRs preserve durable architectural decisions and their rationale.

A governance rule is needed so the ADR catalog can grow with all three Projects without becoming either three disconnected histories or a second operational handbook.

## Decision

### One repository-wide sequence

All new ADRs use one global monotonically increasing `ADR-NNN` sequence under `docs/adr/`.

New ADR numbers must be unique. Historical numbering/header collisions are preserved and documented rather than renumbered, because renumbering old records would rewrite history and break links. The catalog identifies such anomalies explicitly.

### Scope and Project metadata

New ADRs record:

```text
Status: Proposed | Accepted | Superseded | Deprecated
Scope: Repository | Core | AIaaS | Personal
Related Project: <GitHub Project name, or N/A>
Related Issues: #...
Supersedes: ADR-...       # when applicable
Superseded by: ADR-...    # when applicable
```

Scope means:

- **Repository** — decisions that genuinely span the monorepo or its development/governance model;
- **Core** — shared infrastructure, security, storage, networking and platform foundations usable independently of one product tenant;
- **AIaaS** — TXO Fabric and its tenant/agent product contracts;
- **Personal** — personal/application workloads and architecture that consume Core without becoming prerequisites for Core.

`Related Project` is planning ownership, not a second architectural hierarchy. A Repository-scoped ADR may relate to more than one Project.

Existing ADRs do not need to be rewritten merely to add metadata. The index may provide their retrospective scope/project classification where changing the original record would add no historical value.

### Status semantics

For new ADRs:

- **Proposed** — a candidate decision; not authoritative architecture;
- **Accepted** — the decision has been made and is current unless superseded;
- **Superseded** — replaced by a newer explicit decision;
- **Deprecated** — abandoned/no longer recommended without a direct replacement requirement.

Historical `Active` and `Implemented` labels remain understandable in old records, but new ADRs use the four canonical values above. The index normalizes their meaning without silently editing historical wording.

### ADRs are decision records, not living runbooks

An ADR records the context, decision, alternatives/trade-offs and durable consequences. It may record the implementation that proved the decision, but mutable commands, image tags, workflow details, inventories and operating procedures belong in living documentation such as `WORKFLOW.md`, guides, runbooks or application docs.

Accepted ADRs are historical records. A material change in architectural direction gets a new ADR that explicitly supersedes the old one. Small factual corrections, broken-link fixes or explicit current-state/supersession notes may be applied to an old ADR without pretending the original decision was different.

When an ADR and current executable behavior disagree unexpectedly, the implementation is not silently rewritten into the old record: the discrepancy is investigated and either the implementation is corrected or a new decision is recorded.

### Retrospective ADRs

Important architectural contracts that were implemented before an ADR existed may be backfilled. Such an ADR must say that it is retrospective and link to the issues, PRs or physical acceptance that established the behavior.

Retrospective ADRs describe the accepted implementation; they do not manufacture prior intent or make an unresolved proposal appear accepted.

### GitHub Projects and ADRs remain separate

A GitHub issue/Project item tracks work, acceptance and release targeting. An ADR is created only when the result constitutes a durable architectural decision worth preserving. There is no one-issue-one-ADR rule.

## Consequences

### Positive

- Core, AIaaS and Personal decisions remain discoverable without splitting repository history.
- Work tracking and architecture history have clear, non-overlapping responsibilities.
- Supersession is explicit and old decisions remain auditable.
- Retrospective records can capture already-proven architecture without rewriting history.
- New ADR numbering cannot create additional collisions.

### Negative

- Historical records keep some non-canonical statuses and numbering/header anomalies.
- The catalog must be maintained when ADRs are added or superseded.
- A small amount of metadata is required on each new ADR.

## Alternatives considered

### One ADR namespace per GitHub Project

Rejected. Cross-cutting decisions would become ambiguous, numbering would fragment, and moving planning responsibility between Projects could imply a false architecture move.

### Rewrite all historical ADRs into the new format

Rejected. That would create noisy diffs and risk changing the historical meaning of accepted/superseded decisions.

### Use GitHub issues only

Rejected. Issues are excellent work/acceptance records but are not a stable, reviewable architecture history once implementation work closes.

## References

- #3707 — ADR governance and catalog reconciliation
- #3546 — active documentation drift audit
- #3690 — GitHub Project roadmap synchronization
- `docs/adr/000-index.md` — authoritative ADR catalog
- `docs/adr/README.md` — ADR usage guide
