# Napkin - Agent Memory & Rules

**Continuously curated runbook for AI agents working on Vixens.**

Last updated: 2026-03-11

---

## 🔴 CRITICAL - Security Rules (Priority 1)

### NEVER Write Credentials Anywhere

**RULE:** Credentials MUST NEVER appear in:
- ❌ Beads task descriptions
- ❌ Git commit messages
- ❌ PR descriptions or comments
- ❌ Code comments
- ❌ Documentation files
- ❌ Log messages or debug output

**WHERE credentials belong:**
- ✅ Infisical vault (production secrets)
- ✅ Kubernetes Secrets (managed by InfisicalSecret)
- ✅ MinIO admin CLI (ephemeral, for rotation only)
- ✅ `.secrets/` directory (gitignored, local only)

**WHEN rotating credentials:**
- ✅ "Rotated MinIO credentials for adguard-home"
- ❌ "New key: ABC123XYZ / secret456..."

**INCIDENT REFERENCE:** 2026-03-10 - Leaked credentials in PRs #1975-1977 via Beads task descriptions. Credentials rotated, PRs closed, `.beads/issues.jsonl` gitignored. **Root cause: Agent wrote credentials in task descriptions.**

---

## 🔄 GitOps Principles (Priority 2)

### NEVER Modify Cluster State Directly

**RULE:** In GitOps, Git is the single source of truth. NEVER use `kubectl patch/apply/edit` to fix ArgoCD drift.

**WRONG (bypasses GitOps):**
```bash
# Detected drift: Application targetRevision incorrect
kubectl -n argocd patch application <app> -p '{"spec":{"source":{"targetRevision":"prod-stable"}}}'
```

**CORRECT (GitOps workflow):**
```bash
# 1. Identify drift root cause
kubectl -n argocd get application <app> -o yaml > /tmp/app-cluster.yaml
cat argocd/overlays/prod/apps/<app>.yaml  # Compare with Git

# 2. Fix in Git (if needed)
vim argocd/overlays/prod/apps/<app>.yaml
git add argocd/overlays/prod/apps/<app>.yaml
git commit -m "fix(argocd): correct <app> targetRevision to prod-stable"
git push

# 3. Let ArgoCD sync (or force refresh)
kubectl -n argocd patch application argocd --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

**WHY:**
- Manual patches create MORE drift (cluster ≠ Git)
- Next ArgoCD sync may revert manual changes
- Breaks audit trail (change not in Git history)
- Violates GitOps principles

**EXCEPTIONS:** None for production. Emergency hotfixes must be committed immediately after.

**INCIDENT REFERENCE:** 2026-03-11 - AdGuard Home had `targetRevision: fix/renovate-oom-memory-limits` in cluster while Git showed `prod-stable`. Agent used `kubectl patch` to fix (wrong). Root cause: previous manual change bypassed Git. Correct approach: investigate why drift exists, ensure Git is correct, let ArgoCD self-heal.

---

## 📋 Workflow Standards (Priority 2)

### Probe Timeout Standards

**RULE:** Probe timeouts must match workload type:
- APIs (low-latency): 1-2s
- **DaemonSets/sidecars: 5s** (industry standard)
- Batch jobs: 10s+

**RATIONALE:** DaemonSets run on control-plane nodes under load. 1s timeout too strict → false failures → restarts.

**EXAMPLES:**
- ✅ Promtail: 5s timeout (PR #1980)
- ✅ BirdNet data-syncer: 5s timeout (PR #1981)

### Probe Logic for Periodic Containers

**RULE:** For containers with `while true; do work; sleep N; done` pattern:
- ❌ Check work process (`pgrep rclone`) - fails during sleep
- ✅ Check shell script process (`pgrep -f 'script_name'`)
- ✅ Set `periodSeconds` > work cycle duration

**EXAMPLE:** BirdNet data-syncer (PR #1981)
- Cycle: rclone sync (5s) + sleep 60s = 65s
- Probe: `pgrep -f 'sync_s3'` with `periodSeconds: 90`

---

## 🎯 Resource Sizing (Priority 3)

### CronJob Resource Limits

**RULE:** CronJobs need generous limits (VPA doesn't apply):
- Set explicit `requests` (normal operation)
- Set high `limits` (burst protection)
- Cost of OOM > cost of over-provisioning

**EXAMPLE:** Renovate (PR #1980)
- Request: 512Mi (normal)
- Limit: 2Gi (burst during large dependency scans)

---

## 📝 Documentation Requirements (Priority 4)

### Mandatory Updates

**RULE:** After EVERY deployment, update:
1. `docs/applications/<category>/<app>.md` - Deployment table + Known Issues
2. `docs/STATUS.md` - Application status + restart counts

**NO EXCEPTIONS.** Documentation is code.

---

## 🔧 Known Patterns (Priority 5)

### DNS Dependency Cycles

**PATTERN:** Service provides DNS but needs DNS to bootstrap (e.g., AdGuard Home).

**SOLUTION:**
- Use IP addresses instead of DNS names in init containers
- Example: `LITESTREAM_ENDPOINT=http://192.168.111.69:9000` (not DNS)

**REFERENCE:** AdGuard Home (2026-03-10) - Litestream needed MinIO via DNS, but AdGuard provides cluster DNS.

### Production Deployment Workflow

**CRITICAL:** Production uses Git tag `prod-stable`, NOT main branch.

**WORKFLOW:**
1. Merge PR to `main`
2. Update tag: `git tag -f prod-stable main && git push -f origin prod-stable`
3. ArgoCD detects tag change → syncs production apps

**COMMON MISTAKE:** Forgetting to update `prod-stable` tag after merge → prod not updated.

**REFERENCE:** All production ArgoCD apps have `targetRevision: prod-stable` in `argocd/overlays/prod/apps/`

### InfisicalSecret Ownership Pattern

**PATTERN:** Switching from static Secret to InfisicalSecret management.

**CRITICAL STEP:** DELETE the static Secret FIRST, then InfisicalSecret recreates it with ownerReferences.

```bash
# 1. Deploy InfisicalSecret (via ArgoCD)
kubectl apply -f infisical-secret.yaml

# 2. Delete static secret
kubectl delete secret <name> -n <namespace>

# 3. Wait 60s for InfisicalSecret to recreate it
sleep 60

# 4. Verify ownerReferences
kubectl get secret <name> -o jsonpath='{.metadata.ownerReferences[0].kind}'
# Should show: InfisicalSecret
```

**WHY:** If static Secret exists without ownerReferences, InfisicalSecret won't manage it.

**REFERENCE:** AdGuard Home switch (2026-03-11) - Had to delete static secret for InfisicalSecret takeover.

### Kustomize Regression Check (CRITICAL)

**RULE:** After ANY `kustomization.yaml` change, verify resource kinds before/after.

```bash
# Before change
kustomize build apps/<app>/overlays/<env> | grep '^kind:' | sort > /tmp/before.txt

# Make change to kustomization.yaml

# After change
kustomize build apps/<app>/overlays/<env> | grep '^kind:' | sort > /tmp/after.txt

# Compare
diff /tmp/before.txt /tmp/after.txt
```

**WHY:** Kustomize silently drops resources when:
- Resource removed from `resources:` list
- Component removed from `components:`
- Patch target doesn't match anymore

**EXAMPLE:** IT-Tools ingress accidentally removed → service inaccessible (silent failure).

**REFERENCE:** AGENTS.md Step 3b - Mandatory kustomize kinds diff check.

---

## 🚫 Anti-Patterns (Priority 6)

### Things to NEVER Do

1. **Suppress type errors:** No `as any`, `@ts-ignore`, `@ts-expect-error`
2. **Commit without request:** Never `git commit` unless explicitly asked
3. **Shotgun debugging:** Random changes hoping something works
4. **Delete failing tests:** Fix the code, not the test
5. **Skip validation:** Always run linters/tests before marking complete

---

**This napkin is continuously curated. Remove outdated items, keep only recurring high-value guidance.**
