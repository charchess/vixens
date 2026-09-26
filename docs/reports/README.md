# Reports Directory

This directory contains current operational reports and historical point-in-time analyses for Vixens.

## Current reports

Living documents may include:

- `STATUS.md` — high-level application state;
- `AUDIT-CONFORMITY.md` — conformity/audit results;
- `STATE-ACTUAL.md` — observed runtime state when maintained;
- `STATE-DESIRED.md` — documented desired-state decisions when maintained;
- `validation/` — functional and technical validation outputs;
- `audits/` — focused technical audits.

Git manifests and ArgoCD remain authoritative for desired runtime configuration. Reports are supporting documentation, not an alternate source of truth.

## Historical reports

Dated reports are point-in-time snapshots. Git history is the default archive for reports that are no longer useful in the current tree. Only historical documents with durable provenance value and intentional active references should remain under `docs/archive/`.

Archived documents may reference retired tooling such as Beads, Archon, Serena, Just or Infisical. Preserve those references when they explain historical decisions, but do not use them as current instructions.

## Updating living reports

When a living report is still in use:

1. derive facts from the current Git desired state and, where relevant, current cluster observations;
2. distinguish desired state from observed runtime state;
3. link non-obvious decisions to an ADR;
4. link planned work or incidents to GitHub Issues;
5. include the observation/update date when state can become stale.

Do not create a parallel task database in report files.

## Task tracking

GitHub Issues are the canonical task and incident tracker. References in current reports should use GitHub issue/PR numbers rather than Beads or Archon IDs.

## Automation

Reporting scripts must be optional consumers of repository/cluster/GitHub data. A report generator must not become a prerequisite for normal GitOps operation.

Current helper scripts live under `scripts/reports/`. The management report generator uses Kubernetes observations plus GitHub Issues; it does not depend on Beads.

## Naming

Point-in-time reports should prefer:

```text
YYYY-MM-DD-<topic>.md
```

Living documents should have stable descriptive names and clearly state whether they are authoritative, derived, or observational.

## Canonical workflow references

- [`WORKFLOW.md`](../../WORKFLOW.md)
- [`AGENTS.md`](../../AGENTS.md)
- [`docs/guides/gitops-workflow.md`](../guides/gitops-workflow.md)
