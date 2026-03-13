# GitHub Merge Queue Configuration

## Overview

GitHub Merge Queue ensures that PRs are tested together before merging to prevent conflicts and maintain a clean main branch.

## Benefits

- ✅ **Prevents broken main**: PRs are tested with latest main before merging
- ✅ **Serialized merges**: One PR at a time, preventing race conditions
- ✅ **Automatic conflict detection**: Detects conflicts before merge
- ✅ **Faster CI**: Queue runs only essential checks

## Configuration Steps

### 1. Enable Merge Queue

**Via GitHub UI:**

1. Go to repository **Settings** → **General** → **Pull Requests**
2. Check ✅ **Enable merge queue**
3. Configure merge method:
   - Merge method: **Squash and merge** (recommended)
   - Require status checks: ✅ (enabled)

### 2. Branch Protection Rules

**For `main` branch:**

```
Settings → Branches → main → Edit protection rules
```

**Required settings:**

```yaml
Require status checks to pass before merging: ✅
  Strict: ✅ (require branches to be up to date)
  Required checks:
    - Validate Merge Queue Entry
    - Security Check
    - Merge Queue Summary

Require merge queue: ✅
  Merge method: Squash
  Build concurrency: 1
  Minimum PRs in queue: 1
  Maximum PRs in queue: 5
  Merge timeout: 60 minutes
  
Require pull request reviews: ✅ (optional)
  Required approvals: 1
  
Dismiss stale reviews: ✅ (recommended)

Do not allow bypassing: ✅
```

### 3. Merge Queue Settings

**Build concurrency**: 1 (test one PR at a time)  
**Min/Max PRs**: 1-5 (balance speed vs queue length)  
**Timeout**: 60 min (fail if checks take too long)

## How It Works

### Normal PR Workflow

```
1. Create PR
2. PR checks run (validate.yaml)
3. Get approval
4. Add to merge queue
5. Merge queue checks run (merge-queue.yaml - faster)
6. Automatic merge to main
```

### Merge Queue Flow

```
PR #1 ready → Queue → Test with main → Merge → Done
                ↓
PR #2 ready → Wait → Test with main+PR1 → Merge → Done
                       ↓
PR #3 ready → Wait → Test with main+PR1+PR2 → Merge → Done
```

### What Happens

1. **PR approved** → Click "Merge when ready"
2. **Added to queue** → Creates temporary merge commit
3. **Queue validation** → Runs `merge-queue.yaml` (fast checks)
4. **Success** → Automatically merges to main
5. **Failure** → Removed from queue, PR author notified

## Workflow Comparison

| Check | PR Workflow | Merge Queue | Why Different |
|-------|-------------|-------------|---------------|
| YAML Lint | ✅ Full | ✅ Full | Same (fast) |
| Kustomize Build | ✅ All overlays | ✅ Sample (5) | Queue optimized |
| Security Scan | ✅ Full | ✅ Full | Same (required) |
| ArgoCD Validate | ✅ Full | ❌ Skipped | Queue optimization |
| Image Scanning | ✅ Warning | ❌ Skipped | Queue optimization |
| PR Preview | ✅ Yes | ❌ No | Not needed |
| PR Size Check | ✅ Yes | ❌ No | Not needed |

**Philosophy**: Merge queue runs **essential checks only** for speed.

## Usage

### For Developers

**Merging a PR:**

1. Wait for all PR checks to pass
2. Get required approvals (if configured)
3. Click **"Merge when ready"** button
4. GitHub automatically adds to merge queue
5. Wait for queue validation (usually < 2 minutes)
6. PR automatically merges when ready

**If queue validation fails:**

- PR is removed from queue
- Fix the issue
- Push new commit
- Wait for PR checks
- Try merge queue again

### Monitoring Queue

**View queue status:**
- Repository → **Insights** → **Merge queue**
- Shows: position, status, estimated time

**GitHub CLI:**
```bash
# View merge queue
gh pr list --state open --json mergeStateStatus

# Check if PR is in queue
gh pr view <number> --json mergeStateStatus
```

## Troubleshooting

### PR stuck in queue

**Symptoms:** PR in queue for > 10 minutes

**Solutions:**
1. Check workflow runs: `gh run list --branch gh-readonly-queue/main/pr-<number>`
2. View logs: `gh run view <run-id> --log`
3. Cancel and retry: Remove from queue → Fix → Re-add

### Queue validation failing

**Common causes:**
- Merge conflict with main
- New commit on main broke compatibility
- Flaky tests

**Solutions:**
1. Rebase PR on latest main
2. Fix conflicts
3. Re-add to queue

### All PRs removed from queue

**Cause:** Main branch had a commit that breaks compatibility

**Solution:**
1. Fix main branch
2. Re-add PRs to queue in order

## Best Practices

### ✅ DO

- Keep PRs small (< 500 lines) for faster queue processing
- Rebase on main before adding to queue
- Monitor queue status if urgent merge needed
- Use "Merge when ready" for all PRs

### ❌ DON'T

- Don't bypass queue (defeats the purpose)
- Don't add broken PRs to queue (slows down everyone)
- Don't merge directly to main (use queue)
- Don't force-push after adding to queue

## Concurrency Settings

Workflows include `concurrency` to prevent redundant runs:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true
```

**Benefits:**
- Cancels old workflow runs when new push happens
- Saves CI resources (~30-50% reduction)
- Faster feedback (no waiting for old runs)

**Where applied:**
- `validate.yaml` - Main validation workflow
- `pr-preview.yaml` - PR preview comments
- `pr-quality-checks.yaml` - PR size/metrics
- `merge-queue.yaml` - Merge queue validation

## Example Timeline

**Without merge queue:**
```
10:00 - PR #1 merged to main (no testing with main)
10:01 - PR #2 merged to main (not tested with PR #1)
10:02 - Main broken (PR #2 conflicted with PR #1)
10:03 - Emergency fix needed
```

**With merge queue:**
```
10:00 - PR #1 → Queue → Test → Merge (10:02)
10:01 - PR #2 → Queue → Wait for PR #1
10:02 - PR #2 → Test (with PR #1 merged) → Merge (10:04)
10:04 - Main always working ✅
```

## Metrics

**Expected improvements:**
- 📉 **50-70% fewer broken main commits**
- 📉 **30-40% fewer emergency fixes**
- 📈 **10-15% slower individual PR merges** (acceptable tradeoff)
- 📈 **Overall faster development** (fewer rollbacks)

## Advanced Configuration

### Custom Merge Queue Workflow

Edit `.github/workflows/merge-queue.yaml` to customize:

```yaml
# Add custom checks
- name: Custom validation
  run: |
    # Your custom logic
    ./scripts/custom-check.sh
```

### Different Queue Per Branch

```yaml
# In branch protection rules
main:
  merge_queue: enabled
  concurrency: 1

develop:
  merge_queue: enabled
  concurrency: 3  # Allow 3 concurrent
```

### Emergency Bypass (NOT RECOMMENDED)

If absolutely needed:

```bash
# Temporarily disable branch protection
# Merge directly
# Re-enable immediately
```

**Warning:** Only for critical hotfixes!

## References

- [GitHub Docs: Merge Queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue)
- [GitHub Blog: Merge Queue](https://github.blog/2023-02-08-improving-merge-queue/)
- [Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)

## Migration Plan

### Phase 1: Test with merge-queue.yaml
- ✅ Create workflow
- ✅ Test on non-protected branch
- ✅ Verify checks run correctly

### Phase 2: Enable on main (soft)
- Configure branch protection
- Make merge queue optional initially
- Monitor for issues

### Phase 3: Enforce merge queue
- Make merge queue required
- Update team documentation
- Monitor metrics

---

**Last Updated:** 2026-03-13  
**Owner:** CI/CD Team
