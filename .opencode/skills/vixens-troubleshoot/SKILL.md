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

You are an expert at debugging Vixens cluster issues.

**Focus:** $ARGUMENTS

## Quick Diagnostics

### 1. Check Overall Health
```bash
export KUBECONFIG=/home/charchess/vixens/.secrets/prod/kubeconfig-prod

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

## Common Issues

### Issue: App Shows "OutOfSync" in ArgoCD

**Causes:**
1. Git ref (prod-stable) hasn't been updated
2. Kustomize build differs from cluster state
3. Resource was manually modified

**Diagnosis:**
```bash
# Check app revision
kubectl -n argocd get application $APP -o jsonpath='{.status.sync.revision}'

# Compare with prod-stable
git rev-parse prod-stable

# Check diff
kubectl -n argocd get application $APP -o jsonpath='{.status.sync.comparedTo}'
```

**Fix:**
```bash
# If revision mismatch, update tag
git tag -f prod-stable main
git push origin refs/tags/prod-stable --force

# Force sync
kubectl -n argocd patch application $APP --type merge -p '{"operation":{"initiatedBy":{"automated":true},"sync":{"revision":"HEAD"}}}'
```

### Issue: Pod in CrashLoopBackOff

**Diagnosis:**
```bash
# Get pod events
kubectl -n $NS describe pod $POD | grep -A30 Events

# Get container logs
kubectl -n $NS logs $POD -c $CONTAINER --previous

# Check termination reason
kubectl -n $NS get pod $POD -o jsonpath='{.status.containerStatuses[*].lastState.terminated.reason}'
```

**Common Causes & Fixes:**

| Reason | Cause | Fix |
|--------|-------|-----|
| **OOMKilled** | Memory limit too low | Increase `resources.limits.memory` |
| **Error** | Application crash | Check logs for stack trace |
| **ContainerConfigError** | Missing secret/configmap | Check InfisicalSecret sync |
| **ImagePullBackOff** | Wrong image or registry auth | Check image name and pull secrets |

### Issue: Pod Pending (Not Scheduling)

**Diagnosis:**
```bash
kubectl -n $NS describe pod $POD | grep -A10 Events
kubectl -n $NS get pod $POD -o jsonpath='{.status.conditions}' | jq .
```

**Common Causes & Fixes:**

| Event Message | Cause | Fix |
|---------------|-------|-----|
| Insufficient cpu/memory | Cluster full | Scale down other pods or use control-plane toleration |
| node selector mismatch | Wrong labels | Check node selectors match existing nodes |
| persistentvolumeclaim not found | PVC missing | Check PVC exists and is bound |
| 0/5 nodes are available | Taints/tolerations | Add control-plane toleration |

**Add control-plane toleration:**
```yaml
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

# Test from inside cluster
kubectl run debug --rm -it --image=busybox -- wget -qO- http://$SVC.$NS.svc.cluster.local
```

**Common Causes:**
- No endpoints = selector doesn't match pods
- Empty endpoints = pods not ready
- Service exists but ingress missing

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

**Common Causes:**

| Symptom | Cause | Fix |
|---------|-------|-----|
| 404 | Wrong ingress path or missing backend | Check `rules.http.paths` |
| 502 | Backend not ready | Check pod health |
| Certificate not ready | ACME challenge failed | Check DNS and external-dns |
| SSL error | TLS secret missing | Check cert-manager Certificate resource |

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

**Common Causes:**
- InfisicalSecret path wrong (check `secretsPath`)
- Infisical token expired
- Secret not in Infisical at all

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

**Common Causes:**

| Event | Cause | Fix |
|-------|-------|-----|
| waiting for volume | CSI provisioning | Check Synology connection |
| no persistent volumes available | Wrong access mode | Use ReadWriteOnce |
| storageclass not found | Typo in class name | Use `synelia-iscsi-retain` or `synelia-iscsi-delete` |

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

**Manual Trigger:**
```bash
kubectl -n kyverno create job maturity-manual --from=cronjob/maturity-controller
```

## Recovery Procedures

### Restart All Pods of an App
```bash
kubectl -n $NS rollout restart deployment/$DEPLOY
```

### Force ArgoCD Resync
```bash
kubectl -n argocd patch application $APP --type merge -p '{"operation":{"initiatedBy":{"automated":true},"sync":{"revision":"HEAD","prune":true}}}'
```

### Reset Stuck ArgoCD App (stuck deleting)
```bash
# Remove finalizers
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
kubectl -n $NS delete pod $POD
# Deployment will recreate it automatically
```

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

## Escalation

If standard troubleshooting doesn't work:

1. **Check Talos health**: `talosctl health`
2. **Check etcd**: `talosctl etcd members`
3. **Check kubelet logs**: `talosctl logs kubelet`
4. **Reboot problematic node**: `talosctl reboot -n $NODE`
5. **Last resort**: Full cluster diagnostics with `talosctl dashboard`
