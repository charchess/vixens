# GitOps Workflow

This guide explains the **pure trunk-based GitOps workflow** for the Vixens infrastructure.

---

## Overview

**Workflow:** Feature branch → `main` → Auto-deploy dev → Manual promotion to prod

**Key Principle:** Single source of truth (`main` branch). Feature branches for all changes, direct merge to `main` via PR.

**Reference:** [ADR-017: Pure Trunk-Based Development](../adr/017-pure-trunk-based-single-branch.md)

---

## Branch Strategy

| Branch | Purpose | ArgoCD Target | Cluster | Auto-Deploy |
|--------|---------|---------------|---------|-------------|
| `main` | Single source of truth | `main` (HEAD) | Dev cluster | ✅ Yes |
| `prod-stable` (tag) | Production release | `prod-stable` | Prod cluster | ✅ Yes (after tag) |
| `feature/*` | Development work | N/A | N/A | ❌ No |

**Archived:** `dev`, `test`, and `staging` branches (ADR-017 migration)

---

## Standard Workflow

### 1. Create Feature Branch

```bash
# Start from latest main
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/add-new-service
# OR: git checkout -b fix/resolve-issue
# OR: git checkout -b refactor/cleanup-code

# Make your changes
# Edit files...

# Stage changes
git add .

# Commit with conventional format
git commit -m "feat(media): add jellyfin application

Deploy Jellyfin media server with:
- PVC for media storage
- Ingress with HTTPS
- Infisical secrets integration

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### 2. Push and Create Pull Request

```bash
# Push feature branch
git push origin feature/add-new-service

# Create PR to main
gh pr create --base main --head feature/add-new-service \
  --title "feat(media): add jellyfin application" \
  --body "## Summary

Deploy Jellyfin media server for media streaming.

## Changes

- \`apps/20-media/jellyfin/\` - New application
- ArgoCD Application manifest
- Kustomize overlays for dev/prod

## Test Plan

- [ ] ArgoCD sync successful
- [ ] Pod running
- [ ] Ingress accessible
- [ ] Storage mounted correctly

## Validation

\`\`\`bash
kubectl -n media get pods
curl -I https://jellyfin.dev.truxonline.com
\`\`\`

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
```

### 3. Wait for CI Checks and Review

GitHub Actions will run validation checks:
- YAML syntax and style
- Kubernetes structure validation
- ArgoCD application validation
- Security best practices
- Branch flow compliance

**DO NOT** merge until all checks pass ✅

### 4. Merge to Main (Auto-Deploy Dev)

```bash
# Merge PR (squash recommended)
gh pr merge <pr-number> --squash --delete-branch

# OR: via GitHub UI
```

**Result:** ArgoCD automatically syncs to **dev cluster** within 1-3 minutes.

### 5. Validate in Dev Cluster

```bash
# Set dev kubeconfig
export KUBECONFIG=/root/vixens/.secrets/dev/kubeconfig-dev

# Check ArgoCD sync status
kubectl -n argocd get applications | grep jellyfin

# Verify pods
kubectl -n media get pods -l app=jellyfin

# Test functionality (web UI, APIs, etc.)
curl -I https://jellyfin.dev.truxonline.com

# Check logs
kubectl -n media logs -l app=jellyfin --tail=50
```

**Validation Checklist:**
- [ ] ArgoCD app synced and healthy
- [ ] Pods running (no crashes/restarts)
- [ ] Web UI accessible (if applicable)
- [ ] Functionality works as expected
- [ ] No errors in logs
- [ ] Secrets syncing correctly (if using Infisical)
- [ ] Ingress HTTPS certificate valid

### 6. Promote to Production

**IMPORTANT:** Production uses the `prod-stable` Git tag, NOT a branch!

```bash
# After successful dev validation, promote to prod
gh workflow run promote-prod.yaml -f version=v1.2.3

# This GitHub Action will:
# 1. Create/move prod-stable tag to current main HEAD
# 2. Push tag to GitHub
# 3. ArgoCD auto-syncs prod cluster from prod-stable tag
```

**Manual promotion (if workflow unavailable):**

```bash
# Tag current main as prod-stable
git checkout main
git pull origin main
git tag -f prod-stable
git push origin prod-stable --force

# Verify ArgoCD syncs in prod
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod
kubectl -n argocd get applications
```

### 7. Verify Production Deployment

```bash
# Set prod kubeconfig
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod

# Check ArgoCD sync
kubectl -n argocd get applications | grep jellyfin

# Verify pods
kubectl -n media get pods -l app=jellyfin

# Test production URLs
curl -I https://jellyfin.truxonline.com

# Monitor for 5-10 minutes
kubectl -n media get events --sort-by='.lastTimestamp' | tail -20
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
git checkout -b hotfix/critical-security-fix

# Make fix
# Edit files...
git add .
git commit -m "fix(security): patch CVE-2024-12345

Critical security patch for exposed endpoint.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

# Push and create PR
git push origin hotfix/critical-security-fix
gh pr create --base main --head hotfix/critical-security-fix \
  --title "fix(security): patch CVE-2024-12345" \
  --body "## URGENT SECURITY FIX

Description of vulnerability and fix.

## Validation

Tested in dev cluster."

# Fast-track approval and merge
gh pr merge <pr-number> --squash --delete-branch

# Validate in dev
# ... validation steps ...

# Fast promotion to prod (skip staging validation if critical)
gh workflow run promote-prod.yaml -f version=v1.2.3-hotfix
```

---

## Rollback Procedure

If production deployment fails:

### Option 1: Revert via Git

```bash
# Identify bad commit
git log main --oneline -10

# Create revert PR
git checkout main
git pull origin main
git checkout -b revert/bad-feature
git revert <bad-commit-hash>
git push origin revert/bad-feature

# Create PR and merge
gh pr create --base main --head revert/bad-feature \
  --title "revert: rollback bad feature" \
  --body "Reverts commit <hash> due to production issue."

gh pr merge <pr-number> --squash --delete-branch

# Promote reverted main to prod
gh workflow run promote-prod.yaml -f version=v1.2.3-rollback
```

### Option 2: Move prod-stable Tag Backward

```bash
# EMERGENCY ONLY: Move prod-stable to previous working commit
git checkout main
git log --oneline -20  # Find last working commit

git tag -f prod-stable <last-good-commit>
git push origin prod-stable --force

# ArgoCD will auto-sync prod to previous version
# Verify in prod cluster
```

### Option 3: Manual ArgoCD Rollback

```bash
# Via ArgoCD UI or CLI
argocd app rollback <app-name> <previous-revision>

# Verify
kubectl -n <namespace> get pods
```

---

## Best Practices

### DO ✅
- Always work in feature branches
- Create PR for ALL changes (no direct push to main)
- Wait for CI checks to pass before merging
- Test thoroughly in dev before promoting to prod
- Use conventional commit messages
- Squash merge PRs to keep main clean
- Document changes in PR description
- Monitor deployments for 5-10 minutes after promotion

### DON'T ❌
- Push directly to main (branch protection prevents this)
- Skip CI/CD validation
- Merge failing PRs
- Force push to main (emergency only)
- Promote to prod without dev validation
- Commit secrets to Git
- Use vague commit messages
- Deploy on Friday afternoon (unless hotfix)

---

## Troubleshooting

### ArgoCD not syncing after merge

```bash
# Check ArgoCD application
kubectl -n argocd get application <app-name> -o yaml

# Check sync status
kubectl -n argocd get application <app-name> -o jsonpath='{.status.sync.status}'

# Force refresh
argocd app get <app-name> --refresh

# Manual sync
argocd app sync <app-name>
```

### Wrong targetRevision

```bash
# Dev cluster should point to main (HEAD)
kubectl -n argocd get application vixens-app-of-apps -o jsonpath='{.spec.source.targetRevision}'
# Output: main

# Prod cluster should point to prod-stable (tag)
kubectl -n argocd get application vixens-app-of-apps -o jsonpath='{.spec.source.targetRevision}'
# Output: prod-stable
```

### PR merge blocked by branch protection

```bash
# Check required status checks
gh pr checks <pr-number>

# Wait for all checks to pass
# If check is stuck, re-run via GitHub UI or:
gh pr checks <pr-number> --watch

# Check branch protection rules
gh repo view --web
# Navigate to: Settings > Branches > main
```

### Production deployment different from dev

```bash
# Compare commits
git log main...prod-stable --oneline

# Check if prod-stable tag is behind
git log --oneline --graph --all

# Promote latest main to prod
gh workflow run promote-prod.yaml -f version=v1.2.4
```

---

## Emergency Procedures

### Complete Production Outage

1. **Immediate rollback:**
   ```bash
   git tag -f prod-stable <last-known-good-commit>
   git push origin prod-stable --force
   ```

2. **Verify recovery:**
   ```bash
   kubectl -n argocd get applications
   kubectl get pods -A | grep -v Running
   ```

3. **Create incident report** in Beads
4. **Root cause analysis** before re-deploying

### ArgoCD Down

```bash
# If ArgoCD itself is down, manual kubectl needed
kubectl apply -k apps/<app-path>/overlays/<env>/

# Restart ArgoCD components
kubectl -n argocd rollout restart deployment argocd-server
kubectl -n argocd rollout restart deployment argocd-repo-server
```

---

## Related Documentation

- [Adding New Application](adding-new-application.md) - Deploy new apps
- [Task Management](task-management.md) - Beads workflow
- [ADR-017: Pure Trunk-Based Development](../adr/017-pure-trunk-based-single-branch.md) ⭐
- [ADR-008: Trunk-Based GitOps](../adr/008-trunk-based-gitops-workflow.md) (Superseded)
- [ADR-009: Two-Branch Workflow](../adr/009-simplified-two-branch-workflow.md) (Superseded)

---

**Last Updated:** 2026-01-11 (ADR-017 migration)
