# Workflow Concurrency Guide

## Overview

Concurrency controls prevent redundant workflow runs, saving CI resources and providing faster feedback.

## What is Concurrency?

**Without concurrency:**
```
Push 1 → Workflow run A (5 min)
Push 2 (30s later) → Workflow run B (5 min)
Push 3 (1min later) → Workflow run C (5 min)

Result: 3 workflows running, 15 minutes total CI time
Only Run C matters (latest code)
```

**With concurrency:**
```
Push 1 → Workflow run A (5 min)
Push 2 (30s later) → Run A cancelled, Run B starts (5 min)
Push 3 (1min later) → Run B cancelled, Run C starts (5 min)

Result: Only Run C completes, 5 minutes CI time saved
```

## Implementation

### Basic Syntax

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true
```

### Group Patterns

| Pattern | Use Case | Example |
|---------|----------|---------|
| `${{ github.workflow }}-${{ github.ref }}` | Per branch | `validate-main`, `validate-feature-x` |
| `${{ github.workflow }}-${{ github.event.pull_request.number }}` | Per PR | `validate-pr-123` |
| `${{ github.workflow }}-${{ github.event.pull_request.number \|\| github.ref }}` | **PR or branch** | **Recommended** |
| `${{ github.repository }}-${{ github.workflow }}` | Repo-wide | Limit total runs |

## Workflows with Concurrency

### 1. validate.yaml

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true
```

**Why:**
- PRs often have multiple commits pushed quickly
- Old validation runs become obsolete
- Cancel-in-progress saves ~3-5 minutes per PR

**Behavior:**
- PR #123, push 1 → Run A starts
- PR #123, push 2 → Run A cancelled, Run B starts
- PR #123, push 3 → Run B cancelled, Run C starts

### 2. pr-preview.yaml

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number }}
  cancel-in-progress: true
```

**Why:**
- PR preview only needs latest changes
- Old previews are useless
- Saves comment spam

**Behavior:**
- Cancels previous Kustomize diff generation
- Only latest diff posted to PR

### 3. pr-quality-checks.yaml

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number }}
  cancel-in-progress: true
```

**Why:**
- PR size metrics change with each push
- Old metrics are outdated
- Prevents duplicate comments

### 4. merge-queue.yaml

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.merge_group.head_sha }}
  cancel-in-progress: true
```

**Why:**
- Each merge queue entry is unique
- Prevents duplicate queue validations
- SHA-based ensures uniqueness

## Workflows WITHOUT Concurrency

### 1. k8s-version-matrix.yaml

**Why:** Scheduled workflow, one run per week

### 2. promote-prod.yaml

**Why:** Manual workflow, each promotion is intentional

### 3. renovate-discord-notify.yaml

**Why:** Event-driven, each event should notify

### 4. ci-failure-notify.yaml

**Why:** Event-driven, each failure should notify

## Benefits

### Resource Savings

**Before concurrency:**
- Average PR: 5 pushes
- Each workflow: 3 minutes
- Total CI time: 15 minutes per PR
- 20 PRs/week: **5 hours CI time**

**After concurrency:**
- Average PR: 5 pushes
- Only last workflow completes: 3 minutes
- Total CI time: 3 minutes per PR
- 20 PRs/week: **1 hour CI time**

**Savings: 80% CI time reduction**

### Faster Feedback

**Without concurrency:**
```
10:00 - Push commit A → Workflow starts
10:03 - Workflow still running
10:03 - Push commit B → Queued
10:03 - Push commit C → Queued
10:05 - Workflow A completes (obsolete)
10:05 - Workflow B starts
10:08 - Workflow B completes (obsolete)
10:08 - Workflow C starts
10:11 - Workflow C completes (13 minutes total)
```

**With concurrency:**
```
10:00 - Push commit A → Workflow starts
10:03 - Push commit B → Workflow A cancelled, B starts
10:04 - Push commit C → Workflow B cancelled, C starts
10:07 - Workflow C completes (7 minutes total)
```

**Result: 46% faster feedback**

## Best Practices

### ✅ DO

1. **Use concurrency for PR workflows**
   ```yaml
   # Good: Per-PR concurrency
   concurrency:
     group: ${{ github.workflow }}-${{ github.event.pull_request.number }}
     cancel-in-progress: true
   ```

2. **Use fallback for push events**
   ```yaml
   # Good: Handles both PR and push
   group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
   ```

3. **Cancel in-progress for development workflows**
   ```yaml
   cancel-in-progress: true  # Always true for dev workflows
   ```

### ❌ DON'T

1. **Don't use concurrency on production workflows**
   ```yaml
   # Bad: Production deployments should never cancel
   # promote-prod.yaml should NOT have concurrency
   ```

2. **Don't use repo-wide concurrency unnecessarily**
   ```yaml
   # Bad: Blocks all workflows
   concurrency:
     group: ${{ github.repository }}  # Too broad!
   ```

3. **Don't use concurrency on notification workflows**
   ```yaml
   # Bad: Every notification is important
   # ci-failure-notify.yaml should NOT have concurrency
   ```

## Advanced Patterns

### 1. Deployment Concurrency (QUEUE)

For production deployments that should **queue**, not cancel:

```yaml
concurrency:
  group: production-deployment
  cancel-in-progress: false  # Queue instead
```

**Behavior:**
- Deploy A starts
- Deploy B waits
- Deploy A completes
- Deploy B starts

### 2. Per-Environment Concurrency

```yaml
concurrency:
  group: deploy-${{ inputs.environment }}
  cancel-in-progress: false
```

**Behavior:**
- Deploy to dev: can run in parallel
- Deploy to prod: queued
- Independent per environment

### 3. Matrix Job Concurrency

```yaml
strategy:
  matrix:
    k8s-version: [1.28, 1.29, 1.30, 1.31]

# No workflow-level concurrency
# Let matrix run in parallel
```

## Monitoring

### Check Concurrency in Action

**GitHub UI:**
1. Go to **Actions** tab
2. Find a workflow with multiple runs
3. Look for "Cancelling" status

**GitHub CLI:**
```bash
# List cancelled runs
gh run list --status cancelled

# View specific run
gh run view <run-id>
```

### Metrics to Track

- **Cancelled runs**: Should increase after adding concurrency
- **Total CI time**: Should decrease 40-60%
- **Feedback time**: Should decrease 30-50%

## Troubleshooting

### Issue: Workflows constantly cancelling

**Cause:** Pushing too frequently

**Solution:**
- Wait for local tests before pushing
- Use `git commit --amend` + force push
- Enable pre-commit hooks

### Issue: Important run cancelled

**Cause:** Pushed new commit while testing

**Solution:**
- Wait for workflow to complete before pushing
- Or re-run the workflow manually

### Issue: Concurrency not working

**Cause:** Group pattern doesn't match

**Debug:**
```yaml
- name: Debug concurrency group
  run: |
    echo "Workflow: ${{ github.workflow }}"
    echo "PR: ${{ github.event.pull_request.number }}"
    echo "Ref: ${{ github.ref }}"
    echo "Group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}"
```

## Examples

### Example 1: PR with Multiple Pushes

```
11:00 - Create PR #456
11:01 - Push commit A → validate.yaml run #100 starts
11:03 - Push commit B → Run #100 cancelled, run #101 starts
11:04 - Push commit C → Run #101 cancelled, run #102 starts
11:07 - Run #102 completes ✅

Result: Only 1 workflow completed (run #102)
Time saved: 6 minutes (2 cancelled runs)
```

### Example 2: Multiple PRs (Independent)

```
11:00 - PR #456, push → validate.yaml run #100 starts
11:01 - PR #457, push → validate.yaml run #101 starts (parallel)
11:02 - PR #456, push → Run #100 cancelled, run #102 starts
11:03 - PR #457, push → Run #101 cancelled, run #103 starts

Result: Run #102 and #103 both complete (different PRs)
Concurrency groups: "validate-456" and "validate-457"
```

### Example 3: Push to Branch (No PR)

```
11:00 - Push to feature-x → validate.yaml run #100 starts
11:02 - Push to feature-x → Run #100 cancelled, run #101 starts
11:05 - Run #101 completes ✅

Concurrency group: "validate-refs/heads/feature-x"
```

## Migration Checklist

- [x] Add concurrency to validate.yaml
- [ ] Add concurrency to pr-preview.yaml (when Phase 3 merges)
- [ ] Add concurrency to pr-quality-checks.yaml (when Phase 5 merges)
- [x] Add concurrency to merge-queue.yaml
- [ ] Monitor cancelled runs for 1 week
- [ ] Measure CI time savings
- [ ] Update team documentation

## References

- [GitHub Docs: Concurrency](https://docs.github.com/en/actions/using-jobs/using-concurrency)
- [Concurrency Group Expressions](https://docs.github.com/en/actions/learn-github-actions/contexts#example-usage-of-a-context)
- [Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#concurrency)

---

**Last Updated:** 2026-03-13  
**Owner:** CI/CD Team
