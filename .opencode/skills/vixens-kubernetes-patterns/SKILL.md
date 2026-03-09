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

## 📚 References

- [Vixens Troubleshoot Skill](../vixens-troubleshoot/SKILL.md) - Production incident lessons
- [Vixens ArgoCD Safety](../vixens-argocd-safety/SKILL.md) - GitOps safety rules
- [Vixens Maturity](../vixens-maturity/SKILL.md) - Maturity tier requirements
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/) - Official docs
