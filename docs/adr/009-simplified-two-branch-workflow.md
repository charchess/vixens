# ADR 009: Simplified Two-Branch Workflow (dev → main)

**Date:** 2025-12-29
**Status:** Superseded by [ADR-017](017-pure-trunk-based-single-branch.md)
**Deciders:** System Architect, DevOps
**Tags:** gitops, workflow

---

## Context

Previously, we maintained a four-branch workflow: `dev` → `test` → `staging` → `main`, each corresponding to a dedicated Kubernetes cluster environment. This was documented in ADR-008 (Trunk-based GitOps Workflow).

However, in practice:
- **test** and **staging** branches were rarely used for actual validation
- Only **dev** and **prod** clusters were actively maintained and tested
- The intermediate branches added complexity without providing real value
- PRs were being merged directly from `dev` to `main` in practice, bypassing test/staging

On 2025-12-29, we formally archived the `test` and `staging` branches:
- Branches deleted from remote repository
- Archive tags created: `archive/test-20251229`, `archive/staging-20251229`
- Git history preserved for audit purposes

---

## Decision

We will adopt a **simplified two-branch trunk-based workflow**:

### Active Branches
- **`dev`**: Development and testing environment
  - Maps to dev Kubernetes cluster (obsy, onyx, opale)
  - Auto-deploys via ArgoCD watching `dev` branch
  - All development work happens here

- **`main`**: Production environment
  - Maps to prod Kubernetes cluster (physical nodes)
  - Auto-deploys via ArgoCD watching `main` branch
  - Receives changes via Pull Request from `dev`

### Archived Branches
- **`test`**: Deleted, preserved as `archive/test-20251229` tag
- **`staging`**: Deleted, preserved as `archive/staging-20251229` tag

### Terraform Environments
Terraform definitions remain for all four environments (`dev`, `test`, `staging`, `prod`):
- `dev` and `prod`: Actively deployed clusters
- `test` and `staging`: Retained for future infrastructure testing but not deployed

### Kustomize Overlays
Application overlays remain for all four environments:
- `apps/*/overlays/dev/`: Active (dev cluster)
- `apps/*/overlays/prod/`: Active (prod cluster)
- `apps/*/overlays/test/`: Retained but not deployed
- `apps/*/overlays/staging/`: Retained but not deployed

---

## Workflow

### Development Cycle
1. All changes are committed to `dev` branch
2. Push to `dev` → ArgoCD auto-deploys to dev cluster
3. Validate changes in dev environment
4. When ready for production: Create PR `dev` → `main`
5. Review and merge PR
6. ArgoCD auto-deploys to prod cluster

### Promotion Command
```bash
# Compare changes before promoting
git diff dev..main -- apps/

# Create Pull Request for production promotion
gh pr create --base main --head dev \
  --title "chore: promote dev to prod" \
  --body "Validated changes ready for production deployment"
```

---

## Consequences

### Positive
- **Simplified workflow**: Only two active branches to maintain
- **Faster deployment**: Removes intermediate testing stages
- **Reduced complexity**: Fewer merge conflicts, clearer history
- **Aligned with reality**: Formalizes the actual workflow being used
- **Lower maintenance**: No need to sync test/staging branches
- **Clear separation**: dev = testing, main = production

### Negative
- **Less granular testing**: No dedicated test/staging environments
  - **Mitigation**: Dev cluster serves as comprehensive test environment
  - **Mitigation**: Resource limits and validation in dev before prod
- **Higher risk per deployment**: Changes go directly from dev to prod
  - **Mitigation**: Mandatory PR review before merging to main
  - **Mitigation**: ArgoCD health checks and rollback capabilities
  - **Mitigation**: Goldilocks, monitoring, and observability in place

### Neutral
- Terraform definitions retained for test/staging (future flexibility)
- Kustomize overlays retained for test/staging (documentation value)
- Infisical secrets retained for test/staging environments

---

## Implementation

### Completed (2025-12-29)
- ✅ Created archive tags: `archive/test-20251229`, `archive/staging-20251229`
- ✅ Deleted remote branches: `origin/test`, `origin/staging`
- ✅ Updated CLAUDE.md with new workflow documentation
- ✅ Updated branch strategy sections across documentation

### Retained Infrastructure
- Terraform environment definitions (test, staging) - not deployed
- Kustomize overlays (test, staging) - available if needed
- Infisical secrets (test, staging) - retained for completeness

---

## Validation

### Pre-Archive Validation
- All critical services running in dev and prod clusters
- ArgoCD configurations using correct branch references
- No active deployments on test/staging branches

### Post-Archive Validation
- Dev cluster auto-sync functioning (`dev` branch)
- Prod cluster auto-sync functioning (`main` branch)
- PR workflow tested: `dev` → `main` promotion successful
- Documentation updated and committed

---

## References

- ADR-008: Trunk-based GitOps Workflow (superseded by this ADR)
- CLAUDE.md: Updated workflow documentation
- Git tags: `archive/test-20251229`, `archive/staging-20251229`
- Commit: bd81d01 (CLAUDE.md workflow update)

---

## Future Considerations

If we need intermediate testing stages in the future:
1. Restore from archive tags: `git checkout archive/test-20251229 -b test`
2. Deploy test/staging clusters using existing Terraform definitions
3. Update ArgoCD to watch test/staging branches
4. Resume four-branch workflow if justified by team size or complexity
