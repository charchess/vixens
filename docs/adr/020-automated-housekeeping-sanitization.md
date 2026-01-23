# ADR-020: Automated Housekeeping (Sanitization)

**Date:** 2026-01-17
**Status:** Accepted
**Deciders:** User, Coding Agent
**Tags:** infra, sanitization, housekeeping

---

## Context
Kubernetes maintains by default up to 10 old ReplicaSets for each Deployment/StatefulSet to allow rollbacks. In a GitOps environment with high commit frequency, this leads to hundreds of orphan/inactive resources, causing API server bloat and dashboard clutter (e.g., Home Assistant having 10+ ReplicaSets).

## Decision
Enforce a global limit on revision history for all controllers.

1. **Standard Value:** `revisionHistoryLimit` set to `3` (balance between safety/rollback and cleanliness).
2. **Implementation Pattern:** 
   - A shared Kustomize **Component** in `apps/_shared/patches/`.
   - Included via the `components` section in application overlays.
   - Separate patches for `Deployment` and `StatefulSet` to avoid schema issues.
3. **Enforcement:** A Kyverno ClusterPolicy `require-revision-history-limit` audits this standard.

## Consequences
### Positives ✅
- Cleaner cluster (less orphan resources).
- Reduced load on ArgoCD and Kubernetes API.
- Better visibility during troubleshooting.
### Négatives ⚠️
- Rollback via `kubectl rollout undo` limited to 3 previous versions (irrelevant in GitOps where we rollback via Git).

## References
- Ticket Beads: vixens-wkrp
- [Application Deployment Standard](../reference/application-deployment-standard.md)
