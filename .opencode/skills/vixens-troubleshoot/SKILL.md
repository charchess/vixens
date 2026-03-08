---
name: vixens-troubleshoot
description: >-
  Vixens cluster troubleshooting expert. ALWAYS USE when: apps broken, pods crashing,
  CrashLoopBackOff, ImagePullBackOff, OOMKilled, deployments failing, service not accessible,
  ingress issues, certificate problems, secrets missing, PVC stuck, ArgoCD OutOfSync,
  maturity not updating, pod pending, high restarts. Trigger on: "broken", "not working",
  "crash", "error", "stuck", "failing", "debug", "fix", "troubleshoot", "why is X".
argument-hint: "[app-name or namespace]"
license: MIT
compatibility: opencode
metadata:
  domain: kubernetes
  audience: homelab-operators
---

# Vixens Troubleshooting Expert

You are an expert at debugging Vixens cluster issues **following GitOps principles**.

**Focus:** $ARGUMENTS

## ⚠️ GitOps Philosophy (CRITICAL)

### The Golden Rule

> **DIAGNOSIS = read-only kubectl commands**
> **FIX = Git commit → ArgoCD sync**

### Why This Matters

| Cluster | ArgoCD Mode | Manual Changes |
|---------|-------------|----------------|
| **Prod** | Self-heal ON | ❌ Overwritten in ~3 min |
| **Dev** | Self-heal ON | ❌ Overwritten in ~3 min |

**ArgoCD continuously reconciles cluster state to Git.** Any `kubectl apply/patch` is temporary and will be reverted.

### Proper Troubleshooting Flow

```
1. DIAGNOSE (read-only)     → kubectl get/describe/logs
2. HYPOTHESIZE              → "I think X is causing Y"
3. TEST IN DEV (throwaway)  → Deploy test app, tune, delete
4. FIX VIA GIT              → Edit manifest, commit, push
5. VERIFY                   → Watch ArgoCD sync, check pods
```

### Dev as Throwaway Sandbox

**Use dev cluster for experiments:**
```bash
# Switch to dev
export KUBECONFIG=.secrets/dev/kubeconfig-dev

# Deploy a test copy with different settings
kubectl create namespace test-jellyfin
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jellyfin-test
  namespace: test-jellyfin
spec:
  # ... test configuration
EOF

# Experiment, tune, observe
kubectl -n test-jellyfin logs -f deployment/jellyfin-test

# When done, DELETE everything
kubectl delete namespace test-jellyfin
```

**Never experiment on prod. Dev exists for this purpose.**

---

## Quick Diagnostics (Read-Only)

### 1. Check Overall Health
```bash
export KUBECONFIG=.secrets/prod/kubeconfig-prod

# Nodes
kubectl get nodes

# Unhealthy ArgoCD apps
kubectl -n argocd get applications | grep -v "Synced.*Healthy"

# Pods not running
kubectl get pods -A | grep -v Running | grep -v Completed

# High restart pods (>10)
kubectl get pods -A -o json | jq -r '.items[] | select(.status.containerStatuses != null) | select(([.status.containerStatuses[].restartCount] | add) > 10) | "\(.metadata.namespace)/\(.metadata.name): \([.status.containerStatuses[].restartCount] | add)"'

# Recent warning events
kubectl get events -A --field-selector type=Warning --sort-by='.lastTimestamp' | tail -15
```

---

## Common Issues

### Issue: App Shows "OutOfSync" in ArgoCD

**Causes:**
1. Git ref (prod-stable) hasn't been updated
2. Kustomize build differs from cluster state
3. Resource was manually modified (will self-heal)

**Diagnosis:**
```bash
# Check app revision
kubectl -n argocd get application $APP -o jsonpath='{.status.sync.revision}'

# Compare with prod-stable
git rev-parse prod-stable

# Check diff
kubectl -n argocd get application $APP -o jsonpath='{.status.sync.comparedTo}'
```

**Fix (GitOps way):**
```bash
# If revision mismatch, update tag via vixens-gitops skill
# DO NOT use kubectl patch on prod - use Git!
git tag -f prod-stable main
git push origin refs/tags/prod-stable --force

# ArgoCD will auto-sync within 3 minutes
# Or force refresh (read-only, just triggers fetch):
kubectl -n argocd annotate application $APP argocd.argoproj.io/refresh=hard --overwrite
```

### Issue: Pod in CrashLoopBackOff

**Diagnosis (read-only):**
```bash
# Get pod events
kubectl -n $NS describe pod $POD | grep -A30 Events

# Get container logs
kubectl -n $NS logs $POD -c $CONTAINER --previous

# Check termination reason
kubectl -n $NS get pod $POD -o jsonpath='{.status.containerStatuses[*].lastState.terminated.reason}'
```

**Common Causes & GitOps Fixes:**

| Reason | Cause | GitOps Fix |
|--------|-------|------------|
| **OOMKilled** | Memory limit too low | Edit `resources.limits.memory` in Git |
| **Error** | Application crash | Check logs, fix app config in Git |
| **ContainerConfigError** | Missing secret/configmap | Check InfisicalSecret path in Git |
| **ImagePullBackOff** | Wrong image | Fix image tag in Git |

**Testing a theory (use dev):**
```bash
# Switch to dev and test increased memory
export KUBECONFIG=.secrets/dev/kubeconfig-dev
kubectl -n $NS set resources deployment/$DEPLOY -c $CONTAINER --limits=memory=1Gi

# Watch if it helps (dev will self-heal in ~3 min anyway)
kubectl -n $NS get pods -w

# If it works, apply the REAL fix in Git:
# Edit apps/<app>/base/deployment.yaml, commit, push
```

### Issue: Pod Pending (Not Scheduling)

**Diagnosis:**
```bash
kubectl -n $NS describe pod $POD | grep -A10 Events
kubectl -n $NS get pod $POD -o jsonpath='{.status.conditions}' | jq .
```

**Common Causes & GitOps Fixes:**

| Event Message | Cause | GitOps Fix |
|---------------|-------|------------|
| Insufficient cpu/memory | Cluster full | Reduce requests in Git, or scale down other apps |
| node selector mismatch | Wrong labels | Fix nodeSelector in Git |
| persistentvolumeclaim not found | PVC missing | Add PVC to kustomization.yaml |
| 0/5 nodes are available | Taints/tolerations | Add toleration in Git |

**Add control-plane toleration (in Git):**
```yaml
# apps/<app>/base/deployment.yaml
spec:
  tolerations:
    - key: node-role.kubernetes.io/control-plane
      operator: Exists
      effect: NoSchedule
```

### Issue: Service Not Accessible

**Diagnosis:**
```bash
# Check service
kubectl -n $NS get svc $SVC

# Check endpoints (should list pod IPs)
kubectl -n $NS get endpoints $SVC

# Check ingress
kubectl -n $NS get ingress

# Test from inside cluster (ephemeral debug pod)
kubectl run debug --rm -it --image=busybox --restart=Never -- wget -qO- http://$SVC.$NS.svc.cluster.local
```

**Common Causes:**
- No endpoints = selector doesn't match pods → Fix selector in Git
- Empty endpoints = pods not ready → Fix probes in Git
- Service exists but ingress missing → Add ingress in Git

### Issue: Ingress Not Working

**Diagnosis:**
```bash
# Check ingress
kubectl -n $NS describe ingress $INGRESS

# Check Traefik logs
kubectl -n traefik logs -l app.kubernetes.io/name=traefik --tail=50 | grep -i error

# Check certificate
kubectl -n $NS get certificate
kubectl -n $NS describe certificate $CERT

# Check cert-manager
kubectl -n cert-manager logs -l app=cert-manager --tail=50 | grep -i error
```

**Common Causes & GitOps Fixes:**

| Symptom | Cause | GitOps Fix |
|---------|-------|------------|
| 404 | Wrong ingress path | Fix `rules.http.paths` in Git |
| 502 | Backend not ready | Fix probes/health in Git |
| Certificate not ready | ACME challenge failed | Check DNS, fix ingress host in Git |
| SSL error | TLS secret missing | Add `spec.tls` block in Git |

### Issue: Secrets Not Available

**Diagnosis:**
```bash
# Check InfisicalSecret
kubectl -n $NS get infisicalsecret
kubectl -n $NS describe infisicalsecret $SECRET

# Check if Secret was created
kubectl -n $NS get secrets | grep $SECRET

# Check Infisical operator
kubectl -n infisical-operator-system logs -l app.kubernetes.io/name=infisical-operator --tail=50
```

**Common Causes & GitOps Fixes:**
- InfisicalSecret path wrong → Fix `secretsPath` in Git
- Infisical token expired → Renew in Infisical UI, update in Git
- Secret not in Infisical → Add via Infisical UI (secrets managed externally)

### Issue: PVC Stuck Pending

**Diagnosis:**
```bash
# Check PVC
kubectl -n $NS describe pvc $PVC

# Check StorageClass
kubectl get storageclass

# Check Synology CSI controller
kubectl -n synology-csi logs -l app=synology-csi-controller --tail=50 | grep -i error
```

**Common Causes & GitOps Fixes:**

| Event | Cause | GitOps Fix |
|-------|-------|------------|
| waiting for volume | CSI provisioning | Check Synology connection (infra issue) |
| no persistent volumes available | Wrong access mode | Fix to `ReadWriteOnce` in Git |
| storageclass not found | Typo in class name | Fix to `synelia-iscsi-retain` in Git |

### Issue: Maturity Not Updating

**Diagnosis:**
```bash
# Check maturity controller
kubectl -n kyverno get cronjob maturity-controller

# Check last job
kubectl -n kyverno get jobs -l app=maturity-controller --sort-by='.metadata.creationTimestamp' | tail -3

# Check job logs
JOB=$(kubectl -n kyverno get jobs -l app=maturity-controller --sort-by='.metadata.creationTimestamp' -o jsonpath='{.items[-1].metadata.name}')
kubectl -n kyverno logs job/$JOB

# Check remaining violations
kubectl get policyreport -n $NS -o json | jq '[.items[].results[] | select(.result == "fail") | .policy] | unique'
```

**Fix:** Address violations in Git (see `vixens-maturity` skill for requirements).

**Manual Trigger (safe, just runs the controller):**
```bash
kubectl -n kyverno create job maturity-manual --from=cronjob/maturity-controller
```

---

## Recovery Procedures

### Restart All Pods of an App
```bash
# This is SAFE - just triggers a rolling restart
kubectl -n $NS rollout restart deployment/$DEPLOY
```

### Force ArgoCD Refresh (Read-Only)
```bash
# Just triggers Git fetch, no changes applied
kubectl -n argocd annotate application $APP argocd.argoproj.io/refresh=hard --overwrite
```

### Reset Stuck ArgoCD App (stuck deleting)
```bash
# ⚠️ EXCEPTION: This is a valid kubectl patch (ArgoCD control plane, not app state)
kubectl -n argocd patch application $APP -p '{"metadata":{"finalizers":null}}' --type=merge
```

### Drain and Reboot Node
```bash
kubectl drain $NODE --ignore-daemonsets --delete-emptydir-data
talosctl reboot -n $NODE
kubectl uncordon $NODE
```

### Emergency: Delete and Recreate Pod
```bash
# SAFE - Deployment will recreate automatically
kubectl -n $NS delete pod $POD
```

---

## ⚠️ What NOT To Do

### Never on Prod
```bash
# ❌ WRONG - Will be overwritten by ArgoCD self-heal
kubectl apply -f deployment.yaml
kubectl patch deployment $DEPLOY -p '...'
kubectl set image deployment/$DEPLOY ...
kubectl edit deployment $DEPLOY

# ✅ RIGHT - Change in Git, let ArgoCD sync
vim apps/<app>/base/deployment.yaml
git add . && git commit -m "fix(...): ..."
git push
```

### Exception: Temporary Theory Testing
If you MUST test a hypothesis on the cluster:

1. **Use dev, not prod**
2. **Expect it to be overwritten in ~3 minutes**
3. **If it works, immediately commit the real fix to Git**

```bash
# Switch to dev
export KUBECONFIG=.secrets/dev/kubeconfig-dev

# Test theory (will be reverted by ArgoCD)
kubectl -n $NS set resources deployment/$DEPLOY -c main --limits=memory=1Gi

# Watch quickly
kubectl -n $NS get pods -w

# If it works → Edit Git, commit, push
# The dev cluster will self-heal to match Git (with your fix)
```

---

## Log Locations

| Component | How to Access |
|-----------|---------------|
| ArgoCD | `kubectl -n argocd logs -l app.kubernetes.io/name=argocd-application-controller` |
| Kyverno | `kubectl -n kyverno logs -l app=kyverno` |
| Traefik | `kubectl -n traefik logs -l app.kubernetes.io/name=traefik` |
| Cert-manager | `kubectl -n cert-manager logs -l app=cert-manager` |
| External-DNS | `kubectl -n networking logs -l app.kubernetes.io/name=external-dns-gandi` |
| Synology CSI | `kubectl -n synology-csi logs -l app=synology-csi-controller` |
| Infisical | `kubectl -n infisical-operator-system logs -l app.kubernetes.io/name=infisical-operator` |
| Maturity Controller | `kubectl -n kyverno logs -l app=maturity-controller` |

---

## Escalation

If standard troubleshooting doesn't work:

1. **Check Talos health**: `talosctl health`
2. **Check etcd**: `talosctl etcd members`
3. **Check kubelet logs**: `talosctl logs kubelet`
4. **Reboot problematic node**: `talosctl reboot -n $NODE`
5. **Last resort**: Full cluster diagnostics with `talosctl dashboard`
