# Napkin - Agent Memory & Rules

**Continuously curated runbook for AI agents working on Vixens.**

Last updated: 2026-03-28

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

## 🎛️ Sizing System (Priority 2)

1. **[2026-03-28] sizing-v2-mutate écrase TOUTES les resources à l'admission**
   La Kyverno policy `sizing-v2-mutate` injecte cpu/memory req+lim selon le sizing label (`vixens.io/sizing.<container>: V-nano`) et tourne EN DERNIER — après VPA. Les valeurs du manifest et les recommendations VPA sont ignorées. Seul le sizing label détermine les ressources finales.
   Do instead: pour changer les ressources d'un pod, changer son sizing label. V-nano=64Mi/256Mi, V-micro=128Mi/512Mi, V-small=256Mi/1Gi, V-medium=512Mi/2Gi, V-large=1Gi/4Gi.

2. **[2026-03-28] VPA mode: InPlaceOrRecreate sans minReplicas (PR #2554/#2555)**
   Tous les VPAs V-* sont en `InPlaceOrRecreate` (resize in-place sans éviction, fallback éviction si in-place impossible). Pas de `minReplicas:1` — le global `--min-replicas=2` du déploiement VPA bloque volontairement l'éviction des single-replica pods (safe pour RWO PVCs node-locked). In-place ne nécessite pas d'éviction → ça marche quand même pour les single-replica.
   Do instead: ne JAMAIS ajouter `minReplicas:1` sur les VPAs générés — l'éviction fallback sur un pod single-replica avec PVC RWO peut strander le service si le nœud est plein.

3. **[2026-03-28] VPA uncappedTarget << minAllowed → cache stale injecte mauvaises valeurs**
   Si le minAllowed VPA est augmenté (via annotation vixens.io/vpa.min-memory), l'admission controller continue d'injecter l'ancienne recommandation. `sizing-v2-mutate` écrase tout de toute façon → changer le sizing label est la seule vraie solution, pas kubectl restart.

## 🏗️ Infrastructure Patterns (Priority 2)

1. **[2026-03-28] Cilium eBPF egress cassé après incident nœud**
   Symptôme: pod sur le nœud ne peut plus joindre internet (context deadline exceeded), les autres nœuds OK.
   Do instead: `kubectl -n kube-system delete pod $(kubectl -n kube-system get pod -l k8s-app=cilium --field-selector spec.nodeName=<NODE> -o jsonpath='{.items[0].metadata.name}')` — le nouveau pod réinitialise l'eBPF state.

2. **[2026-03-28] iSCSI lock zombie — fuser bloque le cleanup indéfiniment**
   Symptôme: `iscsiadm: Timeout on acquiring lock on DB: /run/lock/iscsi/lock.write` sur un nœud spécifique, PVCs bloqués en ContainerCreating des heures.
   Do instead: vérifier `kubectl -n synology-csi exec iscsi-lock-cleanup-<NODE> -- fuser /host-lock/lock.write` — si vide mais fichier présent, le cleanup tournera au prochain cycle (FORCE_THRESHOLD=3600s depuis PR #2538). Si besoin immédiat: `kubectl -n synology-csi delete pod synology-csi-node-<NODE>`.

3. **[2026-03-28] Nœud saturé après rolling update → single-replica pod avec gros sizing**
   Symptôme: pods Pending sur un nœud après rolling update — un pod single-replica monopolise la majorité des requests (ex: frigate V-3xlarge=8Gi sur 15Gi). VPA InPlaceOrRecreate va réduire in-place progressivement, mais seulement si le pod tourne déjà. Attendre 15-30 min.
   Do instead: si le pod n'a pas encore démarré (Pending au restart), c'est le sizing label initial qui bloque. Baisser le sizing label via Git (ex: V-3xlarge→V-2xlarge). Ne jamais supprimer le pod manuellement si PVC RWO sur nœud plein.

4. **[2026-03-28] local-path PVC = node-lock permanent**
   Symptôme: pod Pending "didn't match PersistentVolume node affinity" même après fix sizing.
   Do instead: la seule vraie solution est de migrer la PVC vers iSCSI. Court terme: réduire les requests des autres pods sur ce nœud pour faire de la place.

5. **[2026-03-28] Kyverno mutation priorityClassName + spec.priority obligatoires ensemble**
   Si on injecte uniquement `priorityClassName` via Kyverno, `spec.priority` reste à 0 (PodPriority plugin tourne AVANT le webhook). Le PriorityAdmission validator voit une incohérence → pod rejeté pour classes non-zero.
   Do instead: toujours patcher les deux champs dans la même mutation. Pattern: apiCall vers `/apis/scheduling.k8s.io/v1/priorityclasses/<name>`, jmesPath `value || 0`, puis patch `/spec/priority` + `/spec/priorityClassName`.

6. **[2026-03-28] Chaîne de dépendance priority : redis-shared → authentik**
   redis-shared (databases) est le backing store d'authentik-server/worker (vixens-critical). Si redis est vixens-low, il peut être préempté et casser le SSO.
   Do instead: toujours aligner la priority d'un backing store sur la priority minimale de ses consommateurs les plus critiques. redis-shared = vixens-high.

---

## 🎛️ Kyverno Gotchas (Priority 3)

### Variables `element.*` scoped au foreach uniquement

**RULE:** `element.name`, `element.resources` etc. ne sont valides QUE dans le bloc `foreach`. Jamais dans `validate.message` au niveau règle.

**WRONG:**
```yaml
validate:
  message: "Container {{ element.name }} has no sizing"  # ❌ hors foreach
  foreach: [...]
```

**CORRECT:**
```yaml
validate:
  message: "Container has no sizing declaration."  # ✅ message générique
  foreach:
    - list: "request.object.spec.containers"
      deny: [...]
```

Do instead: utiliser message générique au niveau `validate:`, ou placer le message à l'intérieur du `foreach` si Kyverno le supporte.

### Changement de structure ClusterPolicy = Immutable → delete+recreate requis

**RULE:** En Kyverno, certains champs sont immutables :
- Changer `validate.pattern` → `validate.foreach+deny` → **IMMUTABLE**
- Changer `validate.foreach` → `validate.pattern` → **IMMUTABLE**
- Changer structure `generate` rule → **IMMUTABLE**

**CONSEQUENCE:** ArgoCD sync échoue avec `changes of immutable fields of a rule spec is disallowed`.

**SOLUTION OBLIGATOIRE avant merge quand la structure de règle change :**
```bash
# Ajouter dans le PR description ou pre-sync hook
kubectl delete clusterpolicy <policy-name>
# ArgoCD recreate automatiquement après sync
```

Do instead: identifier les changements de structure dans le PR review, noter explicitement "nécessite delete+recreate" dans la PR description. Jamais de surprise en prod.

### ArgoCD kyverno app Failed depuis plusieurs jours → invisible

**RULE:** Le sync kyverno peut échouer silencieusement (Helm hooks résiduels) pendant des jours. Les policies continuent de fonctionner (cache), l'impact est invisible.

**SYMPTÔME:** `kyverno:migrate-resources` ClusterRole/ClusterRoleBinding restés après post-upgrade hook.

**DETECTION:** Toujours vérifier au début de session :
```bash
kubectl -n argocd get applications | grep -v "Synced.*Healthy"
```

Do instead: inclure ce check dans la checklist d'entrée de session (`just resume`).

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

### Migrations YAML en masse — tester sur 2-3 fichiers d'abord

**RULE:** Toute migration automatisée de YAML sur 10+ fichiers = tester d'abord sur un subset.

```bash
# 1. Appliquer sur 2-3 fichiers
migrate_one_file.py apps/70-tools/headlamp/overlays/dev/kustomization.yaml

# 2. Vérifier avec kustomize build
kustomize build apps/70-tools/headlamp/overlays/dev

# 3. Seulement si OK → batch
find apps -path "*/overlays/dev/kustomization.yaml" | xargs migrate.py
```

Do instead: JAMAIS lancer un script de migration en batch sans validation préalable sur 2-3 fichiers + kustomize build.

**INCIDENT:** Session 2026-03-17 — script Python trop agressif a cassé 75 fichiers dev. Nécessité `git checkout --` + reprise propre.

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

### OpenClaw PVC Recovery Pattern

**PATTERN:** When openclaw.json is lost/corrupted, old Released PVs on Synology NAS contain the backup.

**PROCEDURE:**
```bash
# 1. Find Released PVs
kubectl get pv | grep Released | grep openclaw

# 2. Remove claimRef to make Available
kubectl patch pv <pv-name> --type json -p '[{"op":"remove","path":"/spec/claimRef"}]'

# 3. Create recovery PVC (must include volumeName + securityContext uid=1000)
# 4. Create recovery pod (uid=1000, labels: vixens.io/sizing: recovery)
# 5. kubectl exec and read /data/openclaw.json
# 6. IMPORTANT: Remove skills.load.paths if present (old key, causes crash)
# 7. Scale down openclaw, disable ArgoCD auto-sync, copy fixed config, re-enable
# 8. Cleanup: delete recovery pod, pvc
```

**KEY LESSON:** openclaw.json may have `skills.load.paths` from old versions. Remove before restoring.
Do instead: `python3 -c "import json,sys; d=json.load(sys.stdin); d.get('skills',{}).pop('load',None); print(json.dumps(d))" < original.json > fixed.json`

**REFERENCE:** 2026-03-11 recovery from pvc-7c8f2a5a (Feb 15 backup) - recovered 31 agents incl. Lisa/Meli/Aurelia

### ArgoCD Auto-Sync Temporary Disable (Emergency Only)

**WHEN:** Need to make manual changes without ArgoCD selfHeal reverting (e.g., config recovery).

```bash
# Disable auto-sync
kubectl patch application <app> -n argocd --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}'

# ... do manual work ...

# Re-enable (MANDATORY - never leave disabled)
kubectl patch application <app> -n argocd --type merge -p '{"spec":{"syncPolicy":{"automated":{"prune":true,"selfHeal":true}}}}'
```

Do instead: ALWAYS re-enable auto-sync immediately after emergency work.
**REFERENCE:** 2026-03-11 - Used to copy fixed openclaw.json without ArgoCD self-healing back

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
