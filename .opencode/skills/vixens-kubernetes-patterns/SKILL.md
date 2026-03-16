---
name: vixens-kubernetes-patterns
description: >-
  Kubernetes patterns and best practices for Vixens cluster.
  Resource management, volume patterns, probes, init containers,
  and GitOps-compatible configurations. Use when: designing deployments,
  troubleshooting volume issues, configuring probes, or ensuring cluster
  bootstrap reproductibility.
argument-hint: "[pattern-name or component-type]"
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes
  audience: homelab-operators
---

# Vixens Kubernetes Patterns

**Production-tested patterns for reliable, reproducible Kubernetes deployments.**

**Focus:** $ARGUMENTS

---

## 🎯 Resource Management Patterns

### Pattern: Defense-in-Depth Resource Sizing

**Problem:** Kyverno policy mutations (like `sizing-v2-mutate`) are applied AFTER pod creation starts. Race conditions during cluster bootstrap or recovery can result in pods without resources.

**Solution:** Always specify BOTH hardcoded resources AND sizing labels.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    vixens.io/sizing-v2: "true"  # Enable VPA
spec:
  template:
    metadata:
      labels:
        vixens.io/sizing.app: V-medium  # Target sizing tier
    spec:
      containers:
      - name: app
        resources:  # ✅ Hardcoded fallback (REQUIRED)
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 1Gi
```

**Why both?**
- **Hardcoded resources** → Guaranteed minimum during bootstrap/failover
- **Sizing labels** → VPA can adjust dynamically in steady-state
- **Prevents OOMKilled** → Even if Kyverno policy not ready yet

**Validation:**
```bash
# Check if pod has BOTH
kubectl get pod $POD -o yaml | grep 'vixens.io/sizing'  # Should find label
kubectl get pod $POD -o jsonpath='{.spec.containers[0].resources}'  # Should have values
```

---

## 🔧 Init Container Patterns

### Pattern: Dynamic Config Generation

**Problem:** ConfigMaps are static. Environment variables in YAML config files (like `$BUCKET_NAME`) are not interpolated by applications.

**Solution:** Use init container to generate config from environment variables.

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      initContainers:
      - name: generate-config
        image: busybox:1.37.0
        command: ["sh", "-c"]
        args:
          - |
            cat > /config/app.yml <<EOF
            database:
              host: ${DB_HOST}
              port: ${DB_PORT}
            storage:
              bucket: ${S3_BUCKET}
              endpoint: ${S3_ENDPOINT}
            EOF
        envFrom:
          - secretRef:
              name: app-secrets
        volumeMounts:
          - name: generated-config
            mountPath: /config

      containers:
      - name: app
        args: ["-config", "/etc/app-config/app.yml"]
        volumeMounts:
          - name: generated-config
            mountPath: /etc/app-config  # ✅ Mount directory

      volumes:
      - name: generated-config
        emptyDir: {}
```

**Key points:**
1. Init container writes to emptyDir
2. Main container reads from same emptyDir
3. Shell interpolates `${VARS}` from secrets/configmaps
4. Config is generated fresh on every pod restart

**Anti-pattern:**
```yaml
# ❌ WRONG - Most apps don't interpolate shell vars in YAML
configMap:
  app.yml: |
    bucket: $S3_BUCKET  # Will be literal "$S3_BUCKET"
```

---

## 📦 Volume Mount Patterns

### Pattern: emptyDir Directory Mount (Not subPath)

**Problem:** `subPath` mounts require the file/directory to exist when the volume is first mounted. If an init container creates the file AFTER the main container's volume mount is initialized, the mount fails.

**Solution:** Mount entire directory, update config path in container args.

```yaml
# ❌ WRONG - subPath fails if file created by init container
initContainers:
- name: generate-config
  volumeMounts:
    - name: config
      mountPath: /config  # Writes /config/app.yml

containers:
- name: app
  args: ["-config", "/etc/app.yml"]
  volumeMounts:
    - name: config
      mountPath: /etc/app.yml  # ❌ FAILS - file doesn't exist at mount time
      subPath: app.yml

# ✅ CORRECT - Mount directory
initContainers:
- name: generate-config
  volumeMounts:
    - name: config
      mountPath: /config  # Writes /config/app.yml

containers:
- name: app
  args: ["-config", "/etc/config/app.yml"]  # Update path
  volumeMounts:
    - name: config
      mountPath: /etc/config  # ✅ Mount whole directory
```

**Why this works:**
- Directory mount is created immediately (empty)
- Init container populates it
- Main container sees populated directory
- No race condition on file existence

**When to use subPath:**
- Static ConfigMap/Secret (file exists before pod creation)
- Never with emptyDir populated by init containers

---

## 🏥 Probe Patterns

### Pattern: Check Persistent Process (Not Intermittent Command)

**Problem:** Liveness probe fails if it checks for a process that only runs intermittently.

**Example - Broken:**
```yaml
# Container runs: `sh -c "while true; do rclone sync ...; sleep 60; done"`

livenessProbe:
  exec:
    command: ["pgrep", "rclone"]  # ❌ FAILS 58/60 seconds
  periodSeconds: 30
```

**Why it fails:**
- rclone runs for ~2 seconds
- Rest of time only `sleep 60` is running
- Probe fails → container killed → CrashLoopBackOff

**Solution - Check shell wrapper:**
```yaml
livenessProbe:
  exec:
    command: ["pgrep", "-f", "sh -c"]  # ✅ Checks persistent shell
  periodSeconds: 30
```

**Alternative - Check pidfile/socket:**
```yaml
livenessProbe:
  exec:
    command: ["test", "-f", "/var/run/app.pid"]
  periodSeconds: 30
```

**Rule of thumb:**
- Liveness probe → Check **persistent process** (shell, supervisor, daemon)
- Readiness probe → Check **application functionality** (HTTP /health, TCP port)

---

## 🔐 Secret Management Patterns

### Pattern: Validate InfisicalSecret Sync

**Problem:** InfisicalSecret CRD shows `Synced` status but secret count is 0 (path empty in Infisical).

**Diagnosis:**
```bash
# Check sync status
kubectl get infisicalsecret $NAME -o jsonpath='{.status.conditions[?(@.type=="secrets.infisical.com/ReadyToSyncSecrets")].message}'

# Expected: "Last reconcile synced N secrets"
# Problem: "Last reconcile synced 0 secrets"

# Verify path exists in Infisical
curl -H "Authorization: Bearer $TOKEN" \
  "http://infisical:8085/api/v3/secrets/raw?workspaceId=$ID&environment=prod&secretPath=/path"

# Response: {"secrets": [], "imports": []}  # Empty = problem!
```

**Prevention:**
1. Create Infisical secrets BEFORE deploying InfisicalSecret CR
2. Monitor `status.conditions` in CI/CD
3. Add validation step in ArgoCD pre-sync hook

```yaml
# ArgoCD hook to validate secret count
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    argocd.argoproj.io/hook: PreSync
spec:
  template:
    spec:
      containers:
      - name: validate-secrets
        image: curlimages/curl
        command:
        - sh
        - -c
        - |
          # Wait for InfisicalSecret to sync
          sleep 10
          COUNT=$(kubectl get secret $SECRET_NAME -o json | jq '.data | length')
          if [ "$COUNT" -eq 0 ]; then
            echo "ERROR: Secret $SECRET_NAME is empty"
            exit 1
          fi
          echo "OK: Secret has $COUNT keys"
```

---

## 🔄 YAML Field Order Patterns

### Pattern: Application-Specific Config Structure

**Problem:** Some applications expect specific YAML field order (even though YAML spec says order doesn't matter).

**Example - Litestream:**
```yaml
# ✅ CORRECT - addr before dbs
addr: ":9090"
dbs:
  - path: /data/app.db
    replicas:
      - url: s3://bucket/app.db

# ❌ WRONG - May fail with some versions
dbs:
  - path: /data/app.db
addr: ":9090"
```

**Best practice:**
1. Follow official examples exactly
2. Test config locally before deploying
3. Document quirks in comments

```yaml
# Litestream requires 'addr' field BEFORE 'dbs' section
# See: https://github.com/benbjohnson/litestream/issues/XXX
addr: ":9090"
dbs: ...
```

---

## 🎨 Sync Wave Patterns

### Pattern: Policy Before Consumers

**Problem:** Kyverno policies deployed in same wave as apps they mutate → race condition.

**Solution:** Use sync waves to enforce ordering.

```yaml
# apps/00-infra/kyverno/base/kustomization.yaml
resources:
  - policies/sizing-v2-mutate.yaml
  - policies/security-hardening.yaml

# argocd/overlays/prod/apps/kyverno.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "3"  # Early wave

# argocd/overlays/prod/apps/my-app.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "7"  # Later wave
```

**Sync wave guidelines:**
| Wave | Purpose | Examples |
|------|---------|----------|
| 0-2 | Core infrastructure | Namespaces, CRDs, cert-manager |
| 3-4 | Policies & operators | Kyverno, Infisical operator |
| 5-6 | Critical services | DNS, ingress, monitoring |
| 7-9 | Applications | Business apps |
| 10+ | Optional/experimental | Dev tools, dashboards |

**Gap between policy and apps:** Minimum 4 sync waves (policy wave + 30s readiness).

---

## 🧪 Testing Patterns

### Pattern: Validation Before Promotion

**Pre-deployment checks:**
```bash
# 1. YAML syntax
yamllint -c yamllint-config.yml apps/$APP/**/*.yaml

# 2. Kustomize build
kustomize build apps/$APP/overlays/dev

# 3. Kubernetes validation
kustomize build apps/$APP/overlays/dev | kubectl apply --dry-run=client -f -

# 4. Resource kinds diff (detect silent drops)
kustomize build apps/$APP/overlays/dev | grep '^kind:' | sort > /tmp/before.txt
# Make changes...
kustomize build apps/$APP/overlays/dev | grep '^kind:' | sort > /tmp/after.txt
diff /tmp/before.txt /tmp/after.txt  # Missing kind = regression!
```

**Post-deployment validation:**
```bash
# 1. ArgoCD sync status
kubectl -n argocd get application $APP -o jsonpath='{.status.sync.status}'

# 2. Pod ready
kubectl -n $NS get pods -l app=$APP -o jsonpath='{.items[*].status.phase}'

# 3. Liveness/Readiness probes passing
kubectl -n $NS get pods -l app=$APP -o jsonpath='{.items[*].status.containerStatuses[*].ready}'

# 4. No recent restarts
kubectl -n $NS get pods -l app=$APP -o jsonpath='{.items[*].status.containerStatuses[*].restartCount}'
```

---

## 🧩 Kustomize Component Patterns

### Pattern: Granular Components (WHAT not WHY)

**Problem:** Monolithic components that bundle multiple unrelated concerns violate single-responsibility principle and create maintenance burden.

**Anti-pattern - Monolithic Component:**
```yaml
# ❌ apps/_shared/components/gold-maturity/kustomization.yaml
# WRONG - Bundles 4 unrelated concerns
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
patches:
  - patch: |-
      metadata:
        annotations:
          argocd.argoproj.io/sync-wave: "10"        # Concern 1: Deployment order
          goldilocks.fairwinds.com/enabled: "true"  # Concern 2: VPA recommendations
          vpa.kubernetes.io/updateMode: "Off"       # Concern 3: VPA mode
      spec:
        revisionHistoryLimit: 3                     # Concern 4: etcd optimization
```

**Why this is bad:**
- **Cannot opt-out individually** - If you want sync-wave 10 but NOT goldilocks, impossible
- **Named after outcome** - "gold-maturity" describes WHY (Gold tier status), not WHAT (concrete config)
- **Tight coupling** - Changes to one concern affect all consumers
- **Hidden dependencies** - Not clear what each app is using

**Solution - Granular Components:**
```yaml
# ✅ apps/_shared/components/sync-wave/wave-10/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
patches:
  - patch: |-
      metadata:
        annotations:
          argocd.argoproj.io/sync-wave: "10"
    target:
      kind: Deployment

# ✅ apps/_shared/components/goldilocks/enabled/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
patches:
  - patch: |-
      metadata:
        annotations:
          goldilocks.fairwinds.com/enabled: "true"
          vpa.kubernetes.io/updateMode: "Off"
    target:
      kind: Deployment

# ✅ apps/_shared/components/revision-history-limit/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
patches:
  - patch: |-
      spec:
        revisionHistoryLimit: 3
    target:
      kind: Deployment
```

**App consumption:**
```yaml
# apps/my-app/overlays/prod/kustomization.yaml
components:
  - ../../../../_shared/components/sync-wave/wave-10
  - ../../../../_shared/components/goldilocks/enabled
  - ../../../../_shared/components/revision-history-limit
  # ✅ Explicit, opt-in per concern
```

---

### Component Design Principles

#### 1. Single Responsibility
**Each component configures ONE concern.**

✅ Good:
- `sync-wave/wave-10` - Only sync-wave annotation
- `goldilocks/enabled` - Only Goldilocks + VPA mode
- `priority/high` - Only priorityClassName

❌ Bad:
- `gold-maturity` - Sync-wave + Goldilocks + VPA + revisionHistoryLimit
- `base` - Security context + fsGroup + seccomp + revisionHistoryLimit
- `resources` - Goldilocks + VPA autoscaling mode

#### 2. Name What It Does (WHAT), Not Why (WHY)

| ❌ Outcome-focused (WHY) | ✅ Configuration-focused (WHAT) |
|-------------------------|--------------------------------|
| `gold-maturity` | `sync-wave/wave-10`, `goldilocks/enabled` |
| `security-baseline` | `security-context`, `seccomp-profile` |
| `high-availability` | `poddisruptionbudget/1`, `topology-spread` |
| `resource-optimization` | `goldilocks/enabled`, `vpa/auto` |

**Why this matters:**
- **WHAT** = Implementation detail (concrete config)
- **WHY** = Business goal (abstract outcome)
- Components are reusable building blocks → describe the block, not the building

#### 3. Opt-In Composition

Apps should compose granular components, not inherit monoliths:

```yaml
# ❌ WRONG - Monolithic inheritance
components:
  - ../../../../_shared/components/base  # What's in here? Who knows.

# ✅ CORRECT - Explicit composition
components:
  - ../../../../_shared/components/security-context
  - ../../../../_shared/components/seccomp-profile
  - ../../../../_shared/components/revision-history-limit
  # Clear what each app uses
```

#### 4. Minimal Duplication

If 3+ apps use the same patch, extract to component:

```yaml
# ❌ WRONG - Duplicated in 36 dev overlays
patches:
  - patch: |-
      spec:
        replicas: 0
    target:
      kind: Deployment

# ✅ CORRECT - Shared component
# apps/_shared/components/dev-disable-replicas/kustomization.yaml
# Then reference in each dev overlay
```

---

### Component Migration Checklist

When refactoring a monolithic component:

- [ ] **Identify concerns** - List all patches/configs bundled
- [ ] **Create granular components** - One component per concern
- [ ] **Update consumers** - Replace monolith with explicit list
- [ ] **Validate builds** - `kustomize build` before/after match
- [ ] **Check kind diff** - No missing resources after refactor
- [ ] **Delete monolith** - Remove old component after migration
- [ ] **Document** - Add comment explaining the split

**Example validation:**
```bash
# Before refactoring
kustomize build apps/my-app/overlays/prod | grep '^kind:' | sort > /tmp/before.txt

# After replacing component
kustomize build apps/my-app/overlays/prod | grep '^kind:' | sort > /tmp/after.txt

# Ensure no resources dropped
diff /tmp/before.txt /tmp/after.txt  # Should be identical
```

---

### Red Flags (When to Refactor)

🚩 **Component bundles 2+ unrelated concerns**
- Example: sync-wave + goldilocks + security context

🚩 **Component name describes outcome, not config**
- Example: `gold-maturity`, `production-ready`, `secure-baseline`

🚩 **Apps can't opt-out of individual features**
- Example: Want sync-wave but not VPA → impossible with monolith

🚩 **Component exists as both component AND patches**
- Example: `resources/` component + `goldilocks/enabled/` component (duplication)

🚩 **Same patch in 5+ app overlays**
- Example: `replicas: 0` duplicated in 36 dev overlays

---

### Real-World Example: gold-maturity Refactoring

**Before (Monolithic):**
```yaml
# apps/_shared/components/gold-maturity/kustomization.yaml
# 54 apps using this
patches:
  - patch: sync-wave + goldilocks + vpa + revisionHistoryLimit
```

**Problem discovered:**
- Apps stuck at wave-10 (couldn't change deployment order)
- Couldn't disable VPA without losing other features
- Name implies outcome (Gold tier) not configuration

**After (Granular):**
```yaml
# Created:
apps/_shared/components/
├── sync-wave/
│   ├── wave-0/
│   ├── wave-1/
│   ├── wave-2/
│   ├── wave-10/
│   └── ...
├── goldilocks/enabled/
└── revision-history-limit/

# Migration:
- 54 apps migrated from gold-maturity to explicit components
- Apps now choose sync-wave independently (0-20)
- Can opt-out of individual features
- Clear dependencies
```

**Result:**
- ✅ 48/50 apps now have coherent sync-waves
- ✅ Apps can evolve independently
- ✅ Components reusable across contexts
- ✅ No hidden coupling

**Lesson learned:** Components are configuration primitives, not maturity tier shortcuts.

---

## 📚 References

- [Vixens Troubleshoot Skill](../vixens-troubleshoot/SKILL.md) - Production incident lessons
- [Vixens ArgoCD Safety](../vixens-argocd-safety/SKILL.md) - GitOps safety rules
- [Vixens Maturity](../vixens-maturity/SKILL.md) - Maturity tier requirements
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/) - Official docs
