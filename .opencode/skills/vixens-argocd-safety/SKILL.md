---
name: vixens-argocd-safety
description: >-
  ArgoCD safety rules - CRITICAL lessons from past incidents.
  ALWAYS LOAD when: touching ArgoCD apps, syncing, refreshing,
  restarting argocd-repo-server, updating prod-stable tag,
  or ANY ArgoCD troubleshooting. Trigger on: "argocd", "sync",
  "refresh", "prod-stable", "targetRevision", "prune".
---

# ArgoCD Safety Rules

**STOP AND READ THIS before any ArgoCD operation.**

## Critical Incident: 2026-03-09

### What Happened
User asked to fix ONE app (openclaw). Agent panicked when ArgoCD didn't pick up changes, and:
1. Restarted argocd-repo-server multiple times
2. Deleted ArgoCD applications
3. Forced syncs with `prune: true` while repo-server was broken
4. Result: **Entire cluster cascaded into failure**

### Root Cause
A **branch** `prod-stable` existed alongside the **tag** `prod-stable`. ArgoCD was using the branch (old commit) instead of the tag (new commit). Agent didn't diagnose this calmly — instead panicked and made destructive changes.

---

## NEVER DO (Hard Rules)

### 1. NEVER force sync with prune while repo-server has issues
```bash
# DANGEROUS - can delete everything if repo state is incomplete
kubectl patch application X --type merge -p '{"operation":{"sync":{"prune":true}}}'
```

### 2. NEVER delete ArgoCD applications to "force refresh"
```bash
# DANGEROUS - finalizer will delete all managed resources
kubectl delete application X  # NO!
```

### 3. NEVER restart argocd-repo-server repeatedly
Each restart clears cache but if the underlying issue isn't fixed, you're just making things worse.

### 4. NEVER make multiple aggressive changes without diagnosing first
If ArgoCD isn't syncing, STOP and diagnose WHY before taking action.

---

## ALWAYS DO (Required Steps)

### Before ANY ArgoCD troubleshooting:

#### 1. Check for branch/tag conflicts
```bash
# If targetRevision is a tag name, check for conflicts
git show-ref | grep <targetRevision>

# If you see both refs/heads/ AND refs/tags/ — THAT'S THE PROBLEM
# Fix: delete the branch, keep the tag
```

#### 2. Check repo-server health FIRST
```bash
kubectl -n argocd logs deployment/argocd-repo-server --tail=20
# Look for: git lock errors, fetch failures, OOM
```

#### 3. Check application sync status calmly
```bash
kubectl -n argocd get application X -o jsonpath='{.status.sync.revision}'
kubectl -n argocd get application X -o jsonpath='{.status.operationState.message}'
```

#### 4. If revision is wrong, check the source
```bash
# What revision does ArgoCD THINK it should use?
kubectl -n argocd get application X -o jsonpath='{.spec.source.targetRevision}'

# What does that resolve to on GitHub?
git ls-remote origin refs/tags/<tag>
git ls-remote origin refs/heads/<branch>
```

---

## Safe Refresh Procedure

### Step 1: Diagnose (don't act yet)
```bash
# Check app status
kubectl -n argocd get application X -o yaml | grep -A20 status:

# Check what revision it's using
kubectl -n argocd get application X -o jsonpath='{.status.sync.revision}'

# Compare to expected
git ls-remote origin refs/tags/prod-stable
```

### Step 2: Soft refresh (safe)
```bash
# This just refreshes the cache, doesn't change anything
kubectl -n argocd annotate application X argocd.argoproj.io/refresh=normal --overwrite
```

### Step 3: Hard refresh (still safe)
```bash
# Forces re-clone of repo
kubectl -n argocd annotate application X argocd.argoproj.io/refresh=hard --overwrite
```

### Step 4: If still not working, check for conflicts
```bash
# Branch vs tag conflict?
git show-ref | grep prod-stable

# If conflict exists, fix it (delete branch, keep tag)
git push origin --delete prod-stable  # delete the BRANCH
```

### Step 5: Only then, restart repo-server (ONCE)
```bash
kubectl -n argocd rollout restart deployment argocd-repo-server
kubectl -n argocd rollout status deployment argocd-repo-server --timeout=120s
```

---

## Red Flags (STOP immediately)

| Symptom | Meaning | Action |
|---------|---------|--------|
| `refname 'X' is ambiguous` | Branch AND tag with same name | Delete the branch |
| `git lock` in repo-server logs | Concurrent git operations | Wait, don't restart repeatedly |
| `ComparisonError` in app status | Repo-server can't read repo | Fix repo-server first, don't sync |
| Multiple apps showing `Unknown` | Repo-server is broken | DO NOT force sync anything |

---

## Recovery Checklist

If you've already broken things:

1. **STOP all manual interventions**
2. **Check CSI driver** — storage is often the cascade trigger
   ```bash
   kubectl get pods -n synology-csi
   ```
3. **Wait for repo-server to stabilize** (2-3 minutes after restart)
4. **Let ArgoCD self-heal** — it will reconcile automatically
5. **Do NOT force prune** — let apps come back naturally

---

## Key Lesson

> **When ArgoCD doesn't behave as expected, the problem is almost always a simple misconfig (wrong ref, branch vs tag, cache). DIAGNOSE FIRST. Aggressive actions like deleting apps or forcing prune will make things 10x worse.**
