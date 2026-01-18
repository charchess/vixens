# ADR-007: Renovate Trunk-Based Workflow

**Date:** 2026-01-11
**Status:** Accepted
**Deciders:** Architecture Team
**Tags:** renovate, gitops, trunk-based

---

## Context

After migrating to pure trunk-based development (ADR-017), the 4-branch promotion workflow (`dev` → `test` → `staging` → `main`) has been replaced by a single-branch workflow with production releases based on the `prod-stable` Git tag.

The previous strategy of Renovate targeting the `dev` branch is no longer applicable as `dev` has been archived. We need to define how Renovate interacts with the new single-branch architecture.

## Decision

**Renovate Bot will target the `main` branch directly via Pull Requests.**

### Configuration applied:

```json
{
  "baseBranches": ["main"],
  "repositories": ["charchess/vixens"],
  "platform": "github",
  "dependencyDashboard": true,
  "automerge": false
}
```

### New Workflow:

1.  **Renovate detects an update.**
2.  **Renovate creates a Pull Request targeting `main`.**
3.  **GitHub Actions runs validation checks** (YAML syntax, Kubernetes structure, ArgoCD validation).
4.  **Manual review** of the Pull Request.
5.  **Merge to `main`** (Squash merge recommended).
6.  **ArgoCD auto-deploys to Dev cluster** (watching `main` HEAD).
7.  **Validation in Dev cluster.**
8.  **Promotion to Prod cluster** via `gh workflow run promote-prod.yaml` (moving `prod-stable` tag).

## Consequences

### Positives

✅ **Simplified Dependency Management**
-   Aligns with the pure trunk-based workflow (ADR-017).
-   Reduces the number of PRs needed to propagate an update (1 PR to `main` instead of 1 + 3 promote PRs).

✅ **Continuous Integration**
-   Every dependency update is automatically validated by CI checks before merging to `main`.

✅ **Fast Dev Deployment**
-   Updates reach the dev cluster immediately after merge, allowing for rapid testing.

✅ **Full Control**
-   No automatic merging to `main`. A human (or advanced agent) must review and approve all dependency changes.

### Négatives

⚠️ **Manual Promotion**
-   Requires manual triggering of the promotion workflow to reach production.

⚠️ **Risk of Main Pollution**
-   If an update breaks the dev cluster, it is already on `main`. However, since prod tracks `prod-stable`, production remains safe until the issue is identified and reverted on `main`.

## Implementation

**File:** `apps/70-tools/renovate/base/configmap.yaml` (and Renovate dashboard settings)

```json
"baseBranches": ["main"]
```

## References

-   [ADR-017: Pure Trunk-Based Development](017-pure-trunk-based-single-branch.md)
-   [Workflow GitOps Vixens](../../WORKFLOW.md)
-   [Renovate Documentation](https://docs.renovatebot.com/)

---

**Decision Owner:** System Architecture
**Implementation Date:** 2026-01-11
**Review Date:** 2026-04-11 (after 3 months of trunk-based usage)
