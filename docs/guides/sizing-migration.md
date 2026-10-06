# Resource sizing migration — retired guide

> **Status: historical compatibility stub.**
>
> The previous contents of this file documented the February 2026 v1 sizing
> migration, including `vixens.io/sizing`, empty `resources: {}` blocks and
> direct `kubectl patch clusterpolicy` operations. Those instructions are no
> longer the current Vixens contract and must not be used as an operational
> runbook.
>
> Historical details remain available in Git history.

## Current sources

Use these sources instead:

- [RESOURCE_STANDARDS.md](../reference/RESOURCE_STANDARDS.md) — current resource,
  sizing and priority conventions;
- [ADR-023](../adr/023-7-tier-goldification-system-v2.md) — maturity model;
- [ADR-029](../adr/029-align-maturity-with-current-platform.md) — current
  maturity/sizing alignment and removal of obsolete assumptions;
- root [WORKFLOW.md](../../WORKFLOW.md) / [AGENTS.md](../../AGENTS.md) — change
  lifecycle and GitOps rules;
- current Kyverno manifests under
  `apps/00-infra/kyverno/base/policies/` — executable policy state.

Current workloads commonly use per-container sizing labels such as
`vixens.io/sizing.<container>` together with explicit bootstrap-safe Kubernetes
resource requests/limits where required by current standards. Do not infer policy
from the retired v1 examples.

## Changing sizing policy

Persistent policy changes belong in Git:

1. inspect the current policy manifests and comparable workloads;
2. create an issue/branch when appropriate;
3. modify the desired state in Git;
4. run applicable local validation and PR CI;
5. merge to `main`;
6. let ArgoCD reconcile dev and validate behavior;
7. promote an explicitly approved immutable candidate if production change is
   required.

Do not enable/disable sizing policy through a persistent live
`kubectl patch clusterpolicy` workaround.
