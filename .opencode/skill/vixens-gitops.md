---
description: >-
  Vixens GitOps workflow expert. ALWAYS USE for: creating PRs, merging, pushing changes,
  promoting to production, updating prod-stable tag, branch protection, GitHub Actions,
  CI checks, squash merge, git workflow. Trigger on: "push", "merge", "PR", "pull request",
  "promote", "deploy to prod", "prod-stable", "release", "tag", "commit", "git".
---

# Vixens GitOps Workflow Expert

You are an expert in the Vixens GitOps workflow using GitHub, ArgoCD, and trunk-based development.

## Branch Strategy

```
feature-branch → main (Dev) → prod-stable tag (Production)
```

| Branch/Tag | Target | Auto-deploy |
|------------|--------|-------------|
| `main` | Dev cluster | Yes (ArgoCD watches main) |
| `prod-stable` | Prod cluster | Yes (ArgoCD watches tag) |
| `prod-working` | Reference | No (last known good state) |

## Quick Reference

### 1. Push Changes (Create PR)
```bash
# Create feature branch from current changes
git checkout -b feat/my-feature
git add .
git commit -m "feat(scope): description"
git push -u origin feat/my-feature

# Create PR
gh pr create --title "feat(scope): description" --body "## Summary
- Change 1
- Change 2"
```

### 2. Merge PR
```bash
# Check PR status
gh pr checks <PR_NUMBER>

# Merge with squash (preferred)
gh pr merge <PR_NUMBER> --squash

# Or auto-merge when checks pass
gh pr merge <PR_NUMBER> --squash --auto
```

### 3. Promote to Production
```bash
# Update prod-stable tag to main
git fetch origin main
git tag -f prod-stable origin/main
git push origin refs/tags/prod-stable --force

# Verify
git log --oneline -1 prod-stable
kubectl -n argocd get applications -o custom-columns='NAME:.metadata.name,REVISION:.status.sync.revision' | head -10
```

## Complete Workflow

### Feature Development → Dev
```bash
# 1. Start from main
git checkout main
git pull origin main

# 2. Create feature branch
git checkout -b feat/my-feature

# 3. Make changes, commit
git add .
git commit -m "feat(app): add new feature"

# 4. Push and create PR
git push -u origin feat/my-feature
gh pr create --title "feat(app): add new feature" --body "## Summary
- Added X
- Fixed Y"

# 5. Wait for checks, then merge
gh pr checks <PR_NUMBER> --watch
gh pr merge <PR_NUMBER> --squash --auto

# 6. Clean up
git checkout main
git pull origin main
git branch -d feat/my-feature
```

### Dev → Production Promotion
```bash
# 1. Verify dev is stable
kubectl --kubeconfig=.secrets/dev/kubeconfig-dev get pods -A | grep -v Running

# 2. Check what will be promoted
git log --oneline prod-stable..main

# 3. Update prod-stable tag
git tag -f prod-stable main
git push origin refs/tags/prod-stable --force

# 4. Verify ArgoCD picks up changes (wait ~30s)
export KUBECONFIG=.secrets/prod/kubeconfig-prod
kubectl -n argocd get applications | grep -v "Synced.*Healthy"

# 5. Update prod-working reference (after validation)
git tag -f prod-working prod-stable
git push origin refs/tags/prod-working --force
```

## Branch Protection Rules

Main branch has protection:
- ❌ Direct push forbidden
- ✅ PR required
- ✅ Status checks must pass
- ✅ Squash merge only

### Required Checks
- YAML Syntax & Style
- Kubernetes Structure
- ArgoCD Structure
- Security Best Practices
- Production Configuration Checks
- Validation Summary
- GitGuardian Security Checks

## Commit Message Convention

```
type(scope): description

Types:
- feat     New feature
- fix      Bug fix
- docs     Documentation
- chore    Maintenance
- refactor Code refactoring
- test     Tests
- ci       CI/CD changes

Examples:
- feat(jellyfin): add GPU transcoding support
- fix(traefik): correct TLS configuration
- docs(readme): update installation guide
- chore(deps): update helm chart to v2.0.0
```

## GitHub CLI Commands

### PR Management
```bash
# List open PRs
gh pr list

# View PR details
gh pr view <PR_NUMBER>

# Check PR status
gh pr checks <PR_NUMBER>

# Merge options
gh pr merge <PR_NUMBER> --squash           # Squash and merge
gh pr merge <PR_NUMBER> --squash --auto    # Auto-merge when ready
gh pr merge <PR_NUMBER> --squash --delete-branch  # Delete branch after

# Close without merging
gh pr close <PR_NUMBER>
```

### Workflow Runs
```bash
# List recent runs
gh run list --limit 5

# View run details
gh run view <RUN_ID>

# Watch run in progress
gh run watch <RUN_ID>

# Re-run failed checks
gh run rerun <RUN_ID>
```

## ArgoCD Sync After Promotion

### Force Refresh All Apps
```bash
# Refresh to pick up new tag
for app in $(kubectl -n argocd get applications -o name); do
  kubectl -n argocd patch $app --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
done
```

### Check Sync Status
```bash
# All apps
kubectl -n argocd get applications -o custom-columns='NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status,REVISION:.status.sync.revision'

# Only out-of-sync
kubectl -n argocd get applications | grep -v "Synced.*Healthy"
```

### Force Sync Specific App
```bash
kubectl -n argocd patch application $APP --type merge -p '{"operation":{"initiatedBy":{"automated":true},"sync":{"revision":"HEAD"}}}'
```

## Rollback Procedures

### Rollback Production to Previous State
```bash
# 1. Find previous good commit
git log --oneline prod-working

# 2. Reset prod-stable to prod-working
git tag -f prod-stable prod-working
git push origin refs/tags/prod-stable --force

# 3. Force ArgoCD refresh
kubectl -n argocd patch application <APP> --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

### Rollback to Specific Commit
```bash
# 1. Find the commit
git log --oneline -20

# 2. Update tag
git tag -f prod-stable <COMMIT_SHA>
git push origin refs/tags/prod-stable --force
```

## Troubleshooting

### PR Won't Merge
```bash
# Check which checks failed
gh pr checks <PR_NUMBER>

# View failed check logs
gh run view <RUN_ID> --log-failed
```

### ArgoCD Not Picking Up Changes
```bash
# Check app target revision
kubectl -n argocd get application <APP> -o jsonpath='{.spec.source.targetRevision}'

# Should show: prod-stable (for prod apps) or main (for dev apps)

# Force hard refresh
kubectl -n argocd patch application <APP> --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

### Tag Push Rejected
```bash
# Ensure you're using --force for tag updates
git push origin refs/tags/prod-stable --force

# If still failing, check GitHub branch protection
# Tags might need admin rights to force-push
```

## Safety Checklist

Before promoting to prod:
- [ ] All dev apps healthy: `kubectl -n argocd get applications | grep -v Healthy`
- [ ] No crashing pods: `kubectl get pods -A | grep -v Running`
- [ ] Changes tested in dev
- [ ] No pending migrations or breaking changes
- [ ] Team notified (if significant changes)

After promoting to prod:
- [ ] ArgoCD apps synced
- [ ] Pods healthy
- [ ] Services accessible
- [ ] Update prod-working tag
