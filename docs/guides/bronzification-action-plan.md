# Bronzification action plan — historical

> **Status: historical compatibility stub.**
>
> This path previously contained a point-in-time bronzification execution plan
> with old application counts, old chart versions, legacy policy assumptions and
> operational snippets that included direct pushes, live ArgoCD patches/edits and
> manual production-tag handling.
>
> It is **not** a current runbook. The original content remains available in Git
> history for incident/decision archaeology.

## Current sources

For current maturity work use:

- [ADR-023](../adr/023-7-tier-goldification-system-v2.md) — maturity model;
- [ADR-029](../adr/029-align-maturity-with-current-platform.md) — alignment with
  current platform conventions;
- [RESOURCE_STANDARDS.md](../reference/RESOURCE_STANDARDS.md) — resource/sizing
  standards;
- [app-golden-standard.md](../reference/app-golden-standard.md) — current reusable
  application standard;
- current GitHub Issues / `vixens roadmap` — live scope, priority and target;
- root [WORKFLOW.md](../../WORKFLOW.md) / [AGENTS.md](../../AGENTS.md) — GitOps
  lifecycle and validation rules.

## Operational rule

Diagnosis may use read-only cluster observations. Persistent fixes go through Git,
PR, CI and ArgoCD. Production is promoted only through
`.github/workflows/promote-prod.yaml` after explicit authorization of the exact
validated candidate.

Do not use historical snippets from this former action plan to:

- push directly to `main`;
- edit or patch ArgoCD Applications as a persistent fix;
- move `prod-stable` manually;
- recreate legacy sizing/policy patterns.
