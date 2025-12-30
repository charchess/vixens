# GitOps Workflow

This guide explains the trunk-based GitOps workflow for promoting changes from development to production.

---

## Overview

**Workflow:** `dev` (development) → `main` (production via `prod-stable` tag)

**Key Principle:** All changes start in `dev`, get validated, then promoted to `main` for production deployment.

---

## Branch Strategy

| Branch | Purpose | ArgoCD Target | Cluster |
|--------|---------|---------------|---------|
| `dev` | Development & testing | `dev` | Dev cluster |
| `main` | Production-ready code | `prod-stable` | Prod cluster |

**Archived:** `test` and `staging` branches (no longer used)

---

## Standard Workflow

### 1. Make Changes in Dev

```bash
# Ensure you're on dev branch
git checkout dev
git pull origin dev

# Make your changes
# Edit files...

# Stage changes
git add .

# Commit with conventional format
git commit -m "feat: add new feature

Detailed description of changes.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

# Push to dev
git push origin dev
```

### 2. Validate in Dev Cluster

```bash
# Set dev kubeconfig
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Check ArgoCD sync status
kubectl -n argocd get applications

# Verify pods
kubectl get pods -A

# Test functionality (web UI, APIs, etc.)
```

**Validation Checklist:**
- [ ] ArgoCD apps synced and healthy
- [ ] All pods running
- [ ] Web UI accessible (if applicable)
- [ ] Functionality works as expected
- [ ] No errors in logs
- [ ] Secrets syncing correctly (if using Infisical)

### 3. Promote to Production

**IMPORTANT:** Do NOT merge dev into main directly!

#### Option A: Manual Promotion (Recommended)

```bash
# Checkout main
git checkout main
git pull origin main

# Cherry-pick commits from dev
git log dev --oneline -10  # Find commit hash
git cherry-pick <commit-hash>

# OR: Merge specific changes
git checkout main
git checkout dev -- apps/path/to/changed/files
git add .
git commit -m "feat: promote feature to production

Original commit: <commit-hash>

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

# Push to main
git push origin main
```

#### Option B: Tag-based Promotion

```bash
# Tag the validated commit in dev
git checkout dev
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# Merge to main
git checkout main
git merge v1.2.3
git push origin main
```

### 4. Verify Production Deployment

```bash
# Set prod kubeconfig
export KUBECONFIG=/root/vixens/terraform/environments/prod/kubeconfig-prod

# Check ArgoCD sync
kubectl -n argocd get applications

# Verify pods
kubectl get pods -A

# Test production URLs
curl -I https://app.truxonline.com
```

---

## Commit Message Format

Follow conventional commits format:

```
<type>[(<scope>)]: <subject>

[optional body]

[optional footer]
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `refactor:` - Code refactoring
- `docs:` - Documentation
- `chore:` - Maintenance
- `infra:` - Infrastructure changes
- `security:` - Security improvements

**Examples:**
```
feat(media): add jellyseerr application

feat: deploy prtg in tools namespace

fix: resolve alertmanager crashloop

refactor(arch): centralize http redirect middleware

docs: update deployment guide

infra: upgrade talos to v1.11.0
```

---

## Hotfix Workflow

For urgent production fixes:

```bash
# Create hotfix branch from main
git checkout main
git pull origin main
git checkout -b hotfix/critical-fix

# Make fix
# Edit files...
git add .
git commit -m "fix: critical production issue

Description of the fix.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

# Push hotfix
git push origin hotfix/critical-fix

# Merge to main
git checkout main
git merge hotfix/critical-fix
git push origin main

# Backport to dev
git checkout dev
git merge hotfix/critical-fix
git push origin dev

# Delete hotfix branch
git branch -d hotfix/critical-fix
git push origin --delete hotfix/critical-fix
```

---

## Rollback Procedure

If production deployment fails:

### Option 1: Revert Commit

```bash
git checkout main
git revert <bad-commit-hash>
git push origin main
```

### Option 2: Hard Reset (Dangerous!)

```bash
# ONLY if safe (no other changes since)
git checkout main
git reset --hard <last-good-commit>
git push origin main --force  # ⚠️ Requires force push
```

### Option 3: Redeploy Previous Version

```bash
# Find previous working commit
git log main --oneline -10

# Cherry-pick good commit
git checkout main
git revert HEAD
git cherry-pick <last-good-commit>
git push origin main
```

---

## Pull Request Workflow (Optional)

For team collaboration:

```bash
# Create feature branch from dev
git checkout dev
git checkout -b feature/new-feature

# Make changes
git add .
git commit -m "feat: new feature"
git push origin feature/new-feature

# Create PR: feature/new-feature → dev
gh pr create --base dev --head feature/new-feature \
  --title "feat: new feature" \
  --body "## Summary
...

## Test Plan
...

🤖 Generated with [Claude Code](https://claude.com/claude-code)"

# After approval & merge
git checkout dev
git pull origin dev
git branch -d feature/new-feature
```

---

## Best Practices

### DO ✅
- Test thoroughly in dev before promoting to main
- Use conventional commit messages
- Cherry-pick specific commits to main
- Validate ArgoCD sync status
- Keep dev and main synchronized
- Document changes in commit messages

### DON'T ❌
- Push directly to main without testing in dev
- Merge dev → main (creates merge commits)
- Skip validation steps
- Force push to main (unless rollback emergency)
- Commit secrets to Git
- Use vague commit messages

---

## Troubleshooting

### ArgoCD not syncing after push

```bash
# Check ArgoCD application
kubectl -n argocd get application <app-name> -o yaml

# Force refresh
kubectl -n argocd patch application <app-name> \
  --type merge \
  --patch '{"operation":{"initiatedBy":{"username":"manual"}}}'

# Manual sync
argocd app sync <app-name>
```

### Git conflicts during cherry-pick

```bash
# Resolve conflicts manually
git status
# Edit conflicted files
git add .
git cherry-pick --continue
```

### Wrong branch deployed

```bash
# Check ArgoCD targetRevision
kubectl -n argocd get application vixens-app-of-apps -o yaml | grep targetRevision

# Should be:
# - dev cluster: targetRevision: dev
# - prod cluster: targetRevision: prod-stable
```

---

## Related Documentation

- [Adding New Application](adding-new-application.md) - Deploy new apps
- [Task Management](task-management.md) - Archon workflow
- [ADR-008: Trunk-Based GitOps](../adr/008-trunk-based-gitops-workflow.md)
- [ADR-009: Two-Branch Workflow](../adr/009-simplified-two-branch-workflow.md)

---

**Last Updated:** 2025-12-30
