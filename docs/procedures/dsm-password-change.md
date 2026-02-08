# Synology DSM Password Change

**Date:** 2026-02-07
**Status:** VALIDATED
**Owner:** Infrastructure Team

---

## 🎯 Purpose

This procedure documents how to safely change the Synology DSM (DiskStation Manager) password and update all dependent Kubernetes services, specifically the Synology CSI driver that manages persistent storage volumes.

**Critical:** Failure to follow this procedure correctly will cause storage failures across the entire cluster, potentially leading to cascading application failures.

## 📋 Prerequisites

- Administrative access to Synology DSM (192.168.111.69)
- Access to Infisical (http://192.168.111.69:8085)
- `kubectl` configured for target cluster (dev/prod)
- ArgoCD access to force application sync
- Downtime window planned (recommended: 30-60 minutes)

## 🚨 Impact Analysis

**Affected Systems:**
- Synology CSI Driver (all nodes)
- All applications using RWO PersistentVolumes (iSCSI)
- Databases: PostgreSQL, MariaDB (dependent apps cascade)
- Backup systems using NFS storage

**Expected Symptoms if not done correctly:**
- `Failed to login with target iqn` errors in CSI logs
- `Multi-Attach error` for volumes
- Pods stuck in `ContainerCreating` state
- Applications in `Progressing` or `Degraded` state
- Potential Kyverno webhook failures (temporary)

## 🚀 Step-by-Step Instructions

### Step 1: Pre-Change Validation

**Verify cluster health BEFORE making changes:**

```bash
# Set environment
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod

# Check all applications are healthy
kubectl get applications -n argocd | grep -v "Synced.*Healthy"

# Check CSI driver status
kubectl get pods -n synology-csi
kubectl get volumeattachments | grep -v "true"

# Check no pods are pending/failing
kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded | wc -l

# Document current state for comparison
kubectl get applications -n argocd -o json > /tmp/apps-before.json
```

**Expected:** All checks should be clean before proceeding.

### Step 2: Change DSM Password

1. Log in to Synology DSM: https://192.168.111.69:5001
2. Navigate to **Control Panel → User & Group**
3. Select the CSI service account (usually `csi-admin` or similar)
4. Click **Edit** and change password
5. **DOCUMENT** the new password securely

⚠️ **DO NOT close this window until Step 3 is complete!**

### Step 3: Update Infisical Secrets

**Update credentials in Infisical IMMEDIATELY:**

1. Log in to Infisical: http://192.168.111.69:8085
2. Navigate to **Project: vixens → Environment: {dev|prod}**
3. Go to path: `/apps/01-storage/synology-csi`
4. Edit the `client-info.yml` secret:
   ```yaml
   ---
   host: 192.168.111.69  # Synology NAS IP
   port: 5000             # DSM API port
   https: false           # Use HTTPS
   username: csi-admin    # CSI service account
   password: {NEW_PASSWORD_HERE}  # ← UPDATE THIS
   ```
5. **Save** the changes

### Step 4: Force Infisical Secret Sync

**Trigger immediate sync to Kubernetes:**

```bash
# Force reconciliation of InfisicalSecret
kubectl annotate infisicalsecret synology-csi-credentials-sync \
  -n synology-csi \
  --overwrite \
  reconcile="$(date +%s)"

# Wait 30 seconds for sync
sleep 30

# Verify secret was updated
kubectl get secret -n synology-csi client-info-secret -o jsonpath='{.data.client-info\.yml}' | base64 -d

# Check InfisicalSecret status
kubectl describe infisicalsecret synology-csi-credentials-sync -n synology-csi | grep -A5 "Status:"
```

**Expected:** Status should show `ReadyToSyncSecrets: True`

### Step 5: Restart CSI Driver

**Force CSI pods to reload new credentials:**

```bash
# Restart CSI controller
kubectl rollout restart statefulset/synology-csi-controller -n synology-csi

# Restart CSI node DaemonSet
kubectl rollout restart daemonset/synology-csi-node -n synology-csi

# Wait for rollout to complete
kubectl rollout status statefulset/synology-csi-controller -n synology-csi --timeout=5m
kubectl rollout status daemonset/synology-csi-node -n synology-csi --timeout=5m

# Verify all CSI pods are Running
kubectl get pods -n synology-csi
```

**Expected:** All CSI pods should be `Running` with recent restart times.

### Step 6: Verify Storage Operations

**Test that CSI can create/attach volumes:**

```bash
# Check CSI driver logs for authentication errors
kubectl logs -n synology-csi synology-csi-controller-0 -c synology-csi-plugin --tail=50 | grep -i "error\|failed"

# Check volume attachments
kubectl get volumeattachments | head -10

# Monitor for new attach/detach events
kubectl get events -n synology-csi --sort-by='.lastTimestamp' | tail -20
```

**Expected:** No authentication errors, all volume attachments should show `ATTACHED=true`

### Step 7: Handle Affected Applications

**Applications may need intervention to recover:**

```bash
# Identify apps in non-healthy state
kubectl get applications -n argocd -o json | \
  jq -r '.items[] | select(.status.health.status != "Healthy") | .metadata.name'

# Force sync apps that are OutOfSync
for app in $(kubectl get applications -n argocd -o json | \
  jq -r '.items[] | select(.status.sync.status == "OutOfSync") | .metadata.name'); do
  echo "Syncing $app..."
  kubectl annotate application $app -n argocd \
    --overwrite argocd.argoproj.io/refresh=normal
done

# Clean up failed/terminated pods
kubectl delete pod -A --field-selector status.phase=Failed
```

### Step 8: Database Recovery

**If databases are stuck (common issue):**

```bash
# Check database status
kubectl get pods -n databases

# If PostgreSQL has I/O errors
kubectl delete pod -n databases postgresql-shared-1
kubectl wait --for=condition=ready pod -n databases -l app.kubernetes.io/name=postgresql-shared --timeout=5m

# If MariaDB has Multi-Attach errors
kubectl delete pod -n databases mariadb-shared-0
kubectl wait --for=condition=ready pod -n databases -l app.kubernetes.io/name=mariadb-shared --timeout=5m

# Verify databases are healthy
kubectl get pods -n databases
```

### Step 9: Monitor Recovery

**Watch cluster stabilize (10-30 minutes):**

```bash
# Watch application health
watch -n 10 'kubectl get applications -n argocd | grep -v "Synced.*Healthy"'

# Monitor pod recovery
watch -n 5 'kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded | wc -l'

# Check for cascading issues
kubectl get events -A --sort-by='.lastTimestamp' | tail -50
```

**Expected:** Progressive improvement, counts should decrease steadily.

## ✅ Verification

Procedure is successful when:

1. ✅ **CSI Driver:** All synology-csi pods Running, no authentication errors in logs
2. ✅ **Volume Attachments:** All volumes show `ATTACHED=true`, no Multi-Attach errors
3. ✅ **Applications:** All ArgoCD apps `Synced` and `Healthy`
4. ✅ **Pods:** No pods stuck in `Pending`, `ContainerCreating`, or `Error` state
5. ✅ **Databases:** PostgreSQL and MariaDB pods Running, no I/O errors
6. ✅ **Services:** All critical services accessible via ingress

**Final validation commands:**

```bash
# Should return 0
kubectl get applications -n argocd | grep -v "Synced.*Healthy" | wc -l

# Should return 0 (or only Completed jobs)
kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded | wc -l

# Test storage create/delete
kubectl create -f /root/vixens/scripts/testing/test-pvc.yaml
kubectl delete -f /root/vixens/scripts/testing/test-pvc.yaml
```

## ⚠️ Troubleshooting

### Issue: CSI pods crash after restart

**Symptoms:** CSI pods in `CrashLoopBackOff`, logs show authentication failures

**Cause:** Infisical secret not yet synced or contains incorrect credentials

**Fix:**
```bash
# Verify secret content
kubectl get secret -n synology-csi client-info-secret -o jsonpath='{.data.client-info\.yml}' | base64 -d

# Check for typos in password, verify DSM credentials work
curl -k -X POST https://192.168.111.69:5001/webapi/auth.cgi \
  -d "api=SYNO.API.Auth" \
  -d "method=login" \
  -d "version=3" \
  -d "account=csi-admin" \
  -d "passwd={NEW_PASSWORD}"

# If credentials wrong, update in Infisical and repeat Step 4-5
```

### Issue: Multi-Attach errors persist

**Symptoms:** Pods show `Multi-Attach error`, volumes attached to multiple nodes

**Cause:** Old volume attachments not cleaned up properly

**Fix:**
```bash
# Identify problematic volumes
kubectl get volumeattachments | grep -v true

# Force detach (DANGEROUS - ensure pods are stopped first)
kubectl delete volumeattachment {csi-hash-from-above}

# Delete affected pod to trigger reattach
kubectl delete pod -n {namespace} {pod-name}
```

### Issue: Kyverno webhook failures

**Symptoms:** ArgoCD sync failures with `failed calling webhook "validate.kyverno.svc-fail"`

**Cause:** Kyverno disrupted during cascade, webhook temporarily unavailable

**Fix:**
```bash
# Check Kyverno health
kubectl get pods -n kyverno

# Restart Kyverno admission controller
kubectl rollout restart deployment/kyverno-admission-controller -n kyverno

# Wait for stability
kubectl rollout status deployment/kyverno-admission-controller -n kyverno

# Retry failed ArgoCD syncs (they should succeed now)
```

### Issue: Databases won't start (I/O errors)

**Symptoms:** PostgreSQL logs show `input/output error`, MariaDB stuck in Init

**Cause:** Filesystem corruption or stale NFS/iSCSI handles

**Fix:**
```bash
# For PostgreSQL
kubectl delete pod -n databases postgresql-shared-1
# If still fails, check PVC events
kubectl describe pvc -n databases | grep -A10 Events

# For MariaDB with RWO volume
# Ensure StatefulSet has strategy: Recreate
kubectl get statefulset -n databases mariadb-shared -o yaml | grep -A5 strategy

# Last resort: delete PVC (DATA LOSS!)
# Only if backups exist and corruption confirmed
kubectl delete pvc -n databases data-mariadb-shared-0
# StatefulSet will recreate, restore from backup
```

### Issue: Applications stuck in Progressing for >30 minutes

**Symptoms:** Apps show `Progressing` in ArgoCD, pods `Pending` or `ContainerCreating`

**Cause:** Resource exhaustion or dependency deadlock

**Fix:**
```bash
# Check node resources
kubectl top nodes
kubectl describe nodes | grep -A10 "Allocated resources"

# Check for PodDisruptionBudgets blocking
kubectl get pdb -A

# Check pod events
kubectl describe pod -n {namespace} {pod-name} | grep -A20 Events

# If resource issue, scale down non-critical apps temporarily
kubectl scale deployment -n {namespace} {app} --replicas=0
```

## 🔄 Rollback Procedure

If the password change causes unrecoverable issues:

1. **Revert DSM password** to previous value
2. **Update Infisical** with old password
3. **Force sync** (Step 4)
4. **Restart CSI** (Step 5)
5. **Document** what went wrong for post-mortem

## 📚 Related Documentation

- [Synology CSI Documentation](../applications/01-storage/synology-csi.md)
- [Infisical Secret Management](../reference/infisical-secrets.md)
- [Infrastructure Dependencies](../reference/infrastructure-dependencies.md) ← See cascade diagram
- [Cascade Failure Recovery](../troubleshooting/cascade-failure-recovery.md) ← General recovery procedure

## 📝 Post-Incident Actions

After successful password change:

1. **Create post-mortem** if issues occurred (see incident 2026-02-07)
2. **Update monitoring** to alert on CSI authentication failures
3. **Review backup schedule** to ensure recovery capability
4. **Test DR procedure** with simulated DSM failure

---

**Lesson Learned (2026-02-07 Incident):**
Changing DSM password without immediately updating Infisical caused a 2-hour cluster-wide cascade failure affecting 50+ applications. This procedure was created to prevent recurrence.
