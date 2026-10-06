# Resource & Priority Reference Map — historical snapshot

> **Status: historical compatibility stub.**
>
> The previous contents were a point-in-time production inventory captured on
> 2026-01-03. It included Pod resource observations, priority recommendations and
> components that have since been migrated or retired (including the old Infisical
> operator). It must not be used as a current resource or scheduling reference.
>
> The original table remains available in Git history.

## Current sources

Use:

- [RESOURCE_STANDARDS.md](RESOURCE_STANDARDS.md) for sizing conventions;
- [quality-standards.md](quality-standards.md) for maturity/resource expectations;
- ADR-029 for the current interpretation of maturity/resource policy;
- current manifests under `apps/` for declared requests, limits, sizing labels and
  PriorityClasses;
- current VPA/Goldilocks/cluster metrics for observed runtime recommendations;
- generated current reports under `docs/reports/` only when their timestamp is
  relevant to the question.

Do not infer current capacity, Pod status, priority or resource values from a dated
inventory table.
