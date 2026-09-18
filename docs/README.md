# Vixens documentation

The documentation is intentionally small. Git remains the desired-state source of truth.

## Current documentation

- [Architecture](architecture.md) — current system boundaries and GitOps architecture
- [Workflow](../WORKFLOW.md) — branch, PR, validation and promotion rules
- [ADR index](adr/000-index.md) — architecture decisions and their status
- [GitOps workflow guide](guides/gitops-workflow.md)
- [Production promotion guide](guides/promotion-workflow.md)
- [Secret management guide](guides/secret-management.md)
- [Adding an application](guides/adding-new-application.md)
- [Repository contract](reference/repository-contract.md)
- [Validation reference](reference/validation.md)
- [Application structure](reference/application-structure.md)

## Application documentation

`applications/` contains application-specific runbooks and design notes. The manifests in
`apps/` and `argocd/` remain authoritative if an application document disagrees with them.

## Historical documentation

- `adr/` records architectural decisions. Existing decision records are not rewritten when the architecture changes.
- `post-mortems/` records incidents as they were understood at the time.

Historical documents may legitimately mention retired technologies, storage systems, branches or workflows.

## What is deliberately not stored here

Vixens does not keep hand-maintained copies of live cluster status, generated conformity dashboards,
agent framework manuals, Terraform documentation, or task databases. Those became stale too easily
and duplicated authoritative systems.
