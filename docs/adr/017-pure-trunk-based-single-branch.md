# ADR-017: Pure Trunk-Based Development (Single Branch main)

**Date:** 2026-01-11
**Status:** ✅ Accepted
**Deciders:** System Architect, DevOps
**Supersedes:** ADR-009 (Simplified Two-Branch Workflow)
**Related:** ADR-008 (Trunk-based GitOps Workflow)

---

## Context

### Current State (ADR-009)

Following ADR-009, we simplified from 4 branches (dev/test/staging/main) to 2 branches (dev/main):
- **`dev` branch** → dev cluster (auto-deploy)
- **`main` branch** → prod cluster (via PR from dev)
- **Promotion:** PR from `dev` to `main`

This reduced complexity but still maintains two long-lived branches with potential divergence.

### Industry Best Practice (2026)

**Pure Trunk-Based Development:**
- Single long-lived branch (`main`)
- Feature branches are short-lived (<24h)
- Environment differentiation via Git tags (not branches)
- ArgoCD `targetRevision` points to branch for dev, tags for prod

**Benefits:**
- No merge conflicts between environment branches
- Linear Git history
- Simplified CI/CD (one branch to watch)
- Faster development cycle (no PR bottleneck for dev)
- Aligned with Google, Netflix, Spotify practices

**Sources:**
- [Trunk-Based Development](https://trunkbaseddevelopment.com/)
- [GitOps Best Practices 2026](https://akuity.io/blog/gitops-best-practices-whitepaper)
- [ArgoCD Environment Promotion Patterns](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)

---

## Decision

**Migrate to pure trunk-based development with a single `main` branch.**

### New Workflow

```
Feature Branch → main (auto-deploy dev) → prod-stable tag (manual promotion)
                  ↓                          ↓
              HEAD (dev cluster)    prod-stable (prod cluster)
```

### Branch Strategy

**Active Branches:**
- **`main`** - Single source of truth for all code

**Deleted Branches:**
- ~~`dev`~~ - Removed (merged into main)
- ~~`test`~~ - Already archived (ADR-009)
- ~~`staging`~~ - Already archived (ADR-009)

### ArgoCD Configuration

**DEV Environment:**
```yaml
spec:
  source:
    repoURL: https://github.com/charchess/vixens
    targetRevision: main  # Track HEAD of main
    path: apps/...
```

**PROD Environment:**
```yaml
spec:
  source:
    repoURL: https://github.com/charchess/vixens
    targetRevision: prod-stable  # Track specific tag
    path: apps/...
```

### Git Tags

**Production Tags:**
- `prod-stable` - Current production version (moved manually)
- `prod-v1.2.3` - Specific production releases

**Dev Tags (optional):**
- `dev-v1.2.3` - Snapshots for rollback (automated)

---

## Workflow

### 1. Feature Development

```bash
# Create feature branch from main
git checkout main
git pull
git checkout -b feature/xyz

# Develop and commit
git add .
git commit -m "feat: add xyz"

# Push feature branch
git push -u origin feature/xyz

# Create PR to main
gh pr create --base main --head feature/xyz \
  --title "feat: add xyz" \
  --body "Description of changes"
```

### 2. Auto-Deploy to Dev

```bash
# After PR is merged to main
# → GitHub Action creates dev snapshot tag (optional)
# → ArgoCD dev (targetRevision: main) auto-syncs
# → Dev cluster deploys new version
```

### 3. Production Promotion

```bash
# Manual promotion after validation in dev
gh workflow run promote-prod.yaml -f version=v1.2.3

# This workflow:
# 1. Creates/moves prod-stable tag to main HEAD
# 2. Creates prod-v1.2.3 tag for audit trail
# 3. ArgoCD prod (targetRevision: prod-stable) auto-syncs
```

### 4. Rollback (if needed)

```bash
# Move prod-stable tag to previous version
git tag -f prod-stable prod-v1.2.2
git push origin prod-stable --force

# ArgoCD detects tag change and rolls back
```

---

## Consequences

### Positive

✅ **Maximum Simplicity:**
- Single branch to maintain (main)
- No merge conflicts between branches
- No PR bottleneck for dev deployments
- Linear Git history

✅ **Faster Development Cycle:**
- Commit to main → immediate dev deployment
- No waiting for PR approval to test in dev
- Faster iteration and feedback

✅ **Cleaner Git History:**
- No "merge dev to main" commits
- Feature branches merge directly to main
- Easy to understand project evolution

✅ **Better CI/CD:**
- Single branch to watch
- Simplified GitHub Actions
- Fewer redundant builds

✅ **Industry Standard:**
- Aligned with trunk-based development best practices
- Well-documented patterns
- Tooling support (Renovate, ArgoCD, etc.)

### Negative

⚠️ **Discipline Required:**
- Commits to main must be stable
- Broken commits immediately affect dev
- Requires good testing and CI

⚠️ **No "Safe" Dev Branch:**
- Experimental work must stay in feature branches
- Can't accumulate changes before testing

⚠️ **Initial Migration Effort:**
- Update 88 ArgoCD Applications
- Merge dev to main
- Delete dev branch
- Update documentation and CI/CD

### Mitigations

**For stability:**
- Mandatory PR review before merge to main
- CI/CD validation (linting, tests) on PRs
- Feature flags for risky changes
- Quick rollback via Git tags

**For experimentation:**
- Use long-lived feature branches if needed
- Deploy feature branches to personal namespaces
- Use dev cluster for validation before merge

---

## Migration Plan

### Phase 1: Preparation ✅

1. ✅ Create ADR-017 (this document)
2. ✅ Validate approach with team
3. 📝 Ensure prod-stable tag exists and is correct

### Phase 2: ArgoCD Migration (1 hour)

1. 🔄 Update all ArgoCD applications in `argocd/overlays/dev/`:
   - Change `targetRevision: dev` → `targetRevision: main`
   - 88 files to update
2. 🔄 Commit changes to main branch
3. 🔄 Verify ArgoCD dev applications sync correctly

### Phase 3: Branch Cleanup (30 minutes)

1. 🔄 Ensure dev branch is fully merged to main
2. 🔄 Create archive tag: `git tag archive/dev-20260111`
3. 🔄 Delete dev branch: `git push origin --delete dev`
4. 🔄 Verify dev cluster continues to sync from main

### Phase 4: Documentation Update (30 minutes)

1. 🔄 Update CLAUDE.md with new workflow
2. 🔄 Update WORKFLOW.md
3. 🔄 Update docs/guides/gitops-workflow.md
4. 🔄 Update GitHub Actions workflows (if needed)

### Phase 5: Validation (1 hour)

1. 🔄 Test feature branch workflow (create, merge, verify dev deploy)
2. 🔄 Test production promotion (move prod-stable tag)
3. 🔄 Test rollback (revert prod-stable tag)
4. 🔄 Verify all documentation is accurate

**Total Estimated Time:** 3 hours

---

## Rollback Strategy

If migration fails:

1. **Recreate dev branch:**
   ```bash
   git checkout archive/dev-20260111 -b dev
   git push -u origin dev
   ```

2. **Revert ArgoCD applications:**
   ```bash
   # Revert commit that changed targetRevision
   git revert <commit-sha>
   ```

3. **Document issues** in post-mortem

---

## Success Metrics

**After 1 month, success if:**
- ✅ 0 merge conflicts (no branches to merge)
- ✅ Dev deployment time < 2 minutes (commit → sync)
- ✅ Prod promotion time < 5 minutes
- ✅ Rollback time < 2 minutes
- ✅ 100% team satisfaction with workflow
- ✅ Git history remains linear and readable

---

## References

- [Trunk-Based Development](https://trunkbaseddevelopment.com/)
- [Google's Trunk-Based Development](https://cloud.google.com/architecture/devops/devops-tech-trunk-based-development)
- [GitOps Best Practices (Akuity)](https://akuity.io/blog/gitops-best-practices-whitepaper)
- [ArgoCD Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
- ADR-008: Trunk-based GitOps Workflow
- ADR-009: Simplified Two-Branch Workflow (superseded)

---

## Implementation Tracking

**Task:** vixens-4tt (refactor(gitops): migrate to trunk-based development workflow)
**Implementation Date:** 2026-01-11
**Review Date:** 2026-02-11 (after 1 month)
**Decision Owner:** System Architect
