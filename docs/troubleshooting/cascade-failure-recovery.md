# Cascade Failure Recovery Runbook

**Purpose:** Generic procedure for detecting, diagnosing, and recovering from cascade failures in Kubernetes cluster.

**Last Updated:** 2026-02-07
**Status:** Active

---

## 🎯 What is a Cascade Failure?

A **cascade failure** occurs when the failure of one component triggers failures in dependent components, creating a domino effect that amplifies the initial problem.

**Common triggers:**
- Storage backend failures (CSI, NFS)
- Shared database crashes
- Network disruptions
- Resource exhaustion
- Credential/secret issues
- Admission webhook failures

**Typical cascade path:**
```
Single component failure
    ↓
Direct dependents fail
    ↓
Indirect dependents fail
    ↓
Resource contention from restarts
    ↓
Cluster-wide instability
```

---

## 🚨 Detection

### Symptoms of Cascade Failure

**Multiple indicators simultaneously:**
- 📊 **ArgoCD:** 10+ applications OutOfSync/Degraded/Progressing
- 🔴 **Pods:** 20+ pods in non-Running state
- ⚠️ **Events:** High volume of FailedMount, CrashLoopBackOff, ImagePullBackOff
- 🔄 **Restarts:** Continuous pod restart cycles
- 📈 **Metrics:** Resource usage spikes (CPU/memory)

### Quick Detection Commands

```bash
# Set environment (adjust for dev/prod)
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod

# Count unhealthy applications
kubectl get applications -n argocd | grep -v "Synced.*Healthy" | wc -l
# Normal: 0-5 | Warning: 5-10 | Critical: 10+

# Count non-running pods
kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded | wc -l
# Normal: 0-10 | Warning: 10-20 | Critical: 20+

# Check for high error rate in events
kubectl get events -A --sort-by='.lastTimestamp' | grep -i "error\|failed" | wc -l
# Normal: <50/min | Warning: 50-100/min | Critical: 100+/min

# Identify cascade pattern (multiple failures starting at same time)
kubectl get events -A --sort-by='.lastTimestamp' | grep "Error\|Failed" | head -50
```

**🔴 CASCADE CONFIRMED if:**
- Multiple namespaces affected simultaneously
- Failures follow dependency pattern (storage → database → apps)
- Time correlation between failures (within 5-10 min window)

---

## 🔍 Diagnosis Phase

### Step 1: Identify Root Cause Layer

Follow the dependency hierarchy from **bottom-up**:

#### Layer 0: Foundation (Hardware/Network)
```bash
# Check node health
kubectl get nodes
kubectl top nodes

# Check for node-level issues
kubectl describe nodes | grep -A5 "Conditions:"
```

**Symptoms:** All nodes NotReady, network timeouts, storage unavailable

#### Layer 1: Core Infrastructure
```bash
# Check CSI driver
kubectl get pods -n synology-csi
kubectl logs -n synology-csi synology-csi-controller-0 -c synology-csi-plugin --tail=50 | grep -i error

# Check CNI (Cilium)
kubectl get pods -n kube-system -l k8s-app=cilium

# Check ingress
kubectl get pods -n traefik
kubectl get svc -n traefik traefik -o wide
```

**Symptoms:** Volume mount failures, network policy issues, ingress unavailable

#### Layer 2: Platform Services
```bash
# Check Infisical operator
kubectl get pods -n infisical-operator-system
kubectl get infisicalsecret -A | grep -v "True.*True.*True"

# Check Kyverno
kubectl get pods -n kyverno
kubectl logs -n kyverno -l app.kubernetes.io/component=admission-controller --tail=50 | grep -i error

# Check ArgoCD
kubectl get pods -n argocd
kubectl get applications -n argocd -o json | jq -r '.items[].status.conditions[] | select(.type=="SyncError") | .message' | head -20
```

**Symptoms:** Secret sync failures, admission webhook errors, ArgoCD sync blocked

#### Layer 3: Shared Services (Databases)
```bash
# Check shared databases
kubectl get pods -n databases

# PostgreSQL
kubectl logs -n databases postgresql-shared-1 --tail=50 | grep -i error
kubectl describe pod -n databases postgresql-shared-1 | grep -A10 Events

# MariaDB
kubectl logs -n databases mariadb-shared-0 --tail=50 | grep -i error
kubectl describe pod -n databases mariadb-shared-0 | grep -A10 Events
```

**Symptoms:** Database pods Init/CrashLoop, I/O errors, connection refused

### Step 2: Identify Failure Pattern

**Pattern A: Storage Cascade**
```
CSI authentication failure
    ↓
iSCSI mount errors
    ↓
PVC mount failures
    ↓
Pods stuck ContainerCreating
```

**Root causes:** DSM password changed, CSI misconfigured, NAS unreachable

**Pattern B: Database Cascade**
```
Database corruption/failure
    ↓
Database pod crashes
    ↓
Dependent apps cannot connect
    ↓
Apps Degraded/Progressing
```

**Root causes:** I/O errors, OOMKilled, node crash, improper shutdown

**Pattern C: Webhook Cascade**
```
Kyverno/admission webhook down
    ↓
Resource validation fails
    ↓
ArgoCD cannot sync
    ↓
Apps stuck OutOfSync
```

**Root causes:** Kyverno pod crash, network policy, resource limits

**Pattern D: Secret Cascade**
```
Infisical operator failure
    ↓
Secrets not synced
    ↓
Apps missing credentials
    ↓
Apps CrashLoopBackOff
```

**Root causes:** Infisical server down, network issues, misconfigured InfisicalSecret

### Step 3: Document Current State

**Capture baseline for comparison:**
```bash
# Save current state
kubectl get applications -n argocd -o json > /tmp/apps-during-incident.json
kubectl get pods -A -o json > /tmp/pods-during-incident.json
kubectl get events -A --sort-by='.lastTimestamp' > /tmp/events-during-incident.txt

# Count by status
kubectl get pods -A --no-headers | awk '{print $4}' | sort | uniq -c

# Identify most affected namespaces
kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded --no-headers | \
  awk '{print $1}' | sort | uniq -c | sort -rn | head -10
```

---

## 🛠️ Recovery Phase

### General Recovery Sequence

**Principle:** Fix from bottom-up (infrastructure → platform → databases → apps)

#### Phase 1: Stop the Bleeding (5 min)

**Objective:** Prevent further damage

```bash
# 1. Identify and isolate the root cause
#    - For storage: Verify CSI credentials, check NAS health
#    - For database: Check PVC status, node health
#    - For webhook: Verify Kyverno pod status

# 2. Clean up zombie pods (stuck in Failed/Unknown state)
kubectl delete pod -A --field-selector status.phase=Failed
kubectl delete pod -A --field-selector status.phase=Unknown

# 3. If resource contention is severe, scale down non-critical apps
# (Only if cluster is truly overwhelmed)
kubectl scale deployment -n {namespace} {app} --replicas=0
```

#### Phase 2: Fix Root Cause (10-30 min)

**Follow pattern-specific procedure:**

##### For Storage Cascade (Pattern A)

1. **Verify CSI credentials:**
   ```bash
   # Check Infisical secret sync
   kubectl get infisicalsecret -n synology-csi synology-csi-credentials-sync -o yaml
   kubectl get secret -n synology-csi client-info-secret -o jsonpath='{.data.client-info\.yml}' | base64 -d

   # If outdated, follow DSM Password Change Procedure
   # See: docs/procedures/dsm-password-change.md
   ```

2. **Restart CSI driver:**
   ```bash
   kubectl rollout restart statefulset/synology-csi-controller -n synology-csi
   kubectl rollout restart daemonset/synology-csi-node -n synology-csi
   kubectl rollout status statefulset/synology-csi-controller -n synology-csi --timeout=5m
   ```

3. **Clean up volume attachments:**
   ```bash
   # Identify problematic volumes
   kubectl get volumeattachments | grep -v true

   # If Multi-Attach errors, force detach (CAREFUL!)
   kubectl delete volumeattachment {csi-hash}
   ```

##### For Database Cascade (Pattern B)

1. **Check database pod status:**
   ```bash
   kubectl get pods -n databases
   kubectl logs -n databases {db-pod} --tail=100
   kubectl describe pod -n databases {db-pod} | grep -A20 Events
   ```

2. **Recover database pod:**
   ```bash
   # If Init/CrashLoop, delete pod (StatefulSet will recreate)
   kubectl delete pod -n databases {db-pod}

   # If I/O errors, check PVC
   kubectl describe pvc -n databases | grep -A10 Events

   # If PVC issue, may need to restore from backup (Velero)
   velero restore create --from-backup {backup-name}
   ```

3. **Verify database health:**
   ```bash
   kubectl wait --for=condition=ready pod -n databases -l app.kubernetes.io/name={db} --timeout=5m
   kubectl logs -n databases {db-pod} --tail=50 | grep -i "ready\|listening"
   ```

##### For Webhook Cascade (Pattern C)

1. **Check Kyverno status:**
   ```bash
   kubectl get pods -n kyverno
   kubectl logs -n kyverno -l app.kubernetes.io/component=admission-controller --tail=50
   ```

2. **Restart Kyverno:**
   ```bash
   kubectl rollout restart deployment/kyverno-admission-controller -n kyverno
   kubectl rollout status deployment/kyverno-admission-controller -n kyverno --timeout=5m
   ```

3. **Verify webhook:**
   ```bash
   kubectl get validatingwebhookconfigurations | grep kyverno
   kubectl describe validatingwebhookconfigurations kyverno-resource-validating-webhook-cfg
   ```

##### For Secret Cascade (Pattern D)

1. **Check Infisical operator:**
   ```bash
   kubectl get pods -n infisical-operator-system
   kubectl logs -n infisical-operator-system -l control-plane=controller-manager --tail=100
   ```

2. **Check InfisicalSecrets:**
   ```bash
   kubectl get infisicalsecret -A
   kubectl describe infisicalsecret -n {namespace} {secret-name}
   ```

3. **Force reconciliation:**
   ```bash
   kubectl annotate infisicalsecret {secret-name} -n {namespace} \
     --overwrite reconcile="$(date +%s)"
   ```

#### Phase 3: Recover Applications (20-60 min)

**Objective:** Restore application health progressively

1. **Force ArgoCD sync for OutOfSync apps:**
   ```bash
   # List OutOfSync apps
   kubectl get applications -n argocd -o json | \
     jq -r '.items[] | select(.status.sync.status == "OutOfSync") | .metadata.name'

   # Force refresh
   for app in $(kubectl get applications -n argocd -o json | \
     jq -r '.items[] | select(.status.sync.status == "OutOfSync") | .metadata.name'); do
     echo "Refreshing $app..."
     kubectl annotate application $app -n argocd \
       --overwrite argocd.argoproj.io/refresh=normal
   done
   ```

2. **Restart stuck database-dependent apps:**
   ```bash
   # After databases are healthy, restart dependent apps
   # See: docs/reference/infrastructure-dependencies.md for list

   kubectl rollout restart deployment -n {namespace} {app}
   ```

3. **Clean up Completed/Failed jobs:**
   ```bash
   kubectl delete job -A --field-selector status.successful=1
   kubectl delete pod -A --field-selector status.phase=Succeeded
   ```

#### Phase 4: Monitor Recovery (30-60 min)

**Objective:** Confirm cluster stability

```bash
# Watch application recovery
watch -n 10 'kubectl get applications -n argocd | grep -v "Synced.*Healthy"'

# Monitor pod recovery
watch -n 5 'kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded | wc -l'

# Check for new errors
kubectl get events -A --sort-by='.lastTimestamp' | tail -50

# Resource usage stabilizing?
kubectl top nodes
kubectl top pods -A --sort-by=cpu | head -20
```

**Recovery successful when:**
- Application count stabilizes (no new failures)
- Pod count decreases steadily
- Events show normal operations (no errors)
- Resource usage returns to baseline

---

## ✅ Validation

### Final Health Check

```bash
# 1. All applications healthy
kubectl get applications -n argocd | grep -v "Synced.*Healthy" | wc -l
# Expected: 0

# 2. All pods running (except completed jobs)
kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded | wc -l
# Expected: 0

# 3. No volume attachment issues
kubectl get volumeattachments | grep -v true | wc -l
# Expected: 0

# 4. Critical services accessible
curl -I https://argocd.truxonline.com
curl -I https://traefik.truxonline.com/dashboard/
# Expected: HTTP 200 or 30X

# 5. Databases healthy
kubectl get pods -n databases
# Expected: All Running

# 6. No recent errors in events
kubectl get events -A --sort-by='.lastTimestamp' | grep -i error | tail -20
# Expected: No new errors (<5 min old)
```

---

## 📊 Post-Incident Actions

### Immediate (Within 1 hour)

1. **Document the incident:**
   - Create post-mortem: `docs/troubleshooting/post-mortems/YYYY-MM-DD-{incident}.md`
   - Use template from existing post-mortems
   - Include timeline, root cause, impact, recovery steps

2. **Communicate status:**
   - Update stakeholders (users, ops team, management)
   - Provide ETA for full resolution
   - Document any data loss or ongoing issues

### Short-term (Within 1 week)

1. **Update runbooks:**
   - If new failure pattern discovered, add to this runbook
   - Update pattern-specific procedures as needed
   - Document any workarounds used

2. **Improve monitoring:**
   - Add alerts for detected failure pattern
   - Improve detection of early warning signs
   - Add dashboard for cascade failure indicators

3. **Test recovery:**
   - Validate recovery procedure in dev environment
   - Simulate failure with chaos engineering
   - Document actual vs. expected recovery time

### Long-term (Within 1 month)

1. **Architectural improvements:**
   - Identify and fix Single Points of Failure
   - Add circuit breakers to prevent cascades
   - Improve retry logic and backoff strategies
   - Consider redundancy for critical components

2. **Automation:**
   - Automate detection (alerts)
   - Automate diagnosis (scripts)
   - Consider auto-recovery for known patterns

3. **Training:**
   - Share lessons learned with team
   - Update runbooks based on feedback
   - Conduct incident response drills

---

## 🔧 Advanced Recovery Techniques

### Nuclear Options (Last Resort)

**⚠️ USE ONLY WHEN STANDARD RECOVERY FAILS**

#### 1. Temporary Disable Kyverno
```bash
# If Kyverno is blocking recovery and cannot be fixed quickly
kubectl scale deployment -n kyverno kyverno-admission-controller --replicas=0

# DO RECOVERY WORK

# Re-enable Kyverno
kubectl scale deployment -n kyverno kyverno-admission-controller --replicas=3
```

**Risk:** Removes admission validation temporarily

#### 2. Force Delete Stuck Pods
```bash
# If pods won't terminate normally
kubectl delete pod {pod-name} -n {namespace} --grace-period=0 --force

# If still stuck
kubectl patch pod {pod-name} -n {namespace} -p '{"metadata":{"finalizers":null}}'
```

**Risk:** Can leave orphaned resources

#### 3. Delete and Recreate PVC
```bash
# ⚠️ DATA LOSS! Only if PVC is corrupted and backups exist
kubectl scale deployment/statefulset -n {namespace} {app} --replicas=0
kubectl delete pvc -n {namespace} {pvc-name}
# Recreate will happen automatically or via ArgoCD
# Then restore from Velero backup
```

**Risk:** Permanent data loss if no backup

#### 4. Node Drain and Reboot
```bash
# If node is misbehaving
kubectl drain {node-name} --ignore-daemonsets --delete-emptydir-data
# Reboot node (Talos)
talosctl reboot --nodes {node-ip}
# Wait for node Ready
kubectl uncordon {node-name}
```

**Risk:** Downtime for pods on that node

---

## 📚 Related Documentation

- **[Infrastructure Dependencies](../reference/infrastructure-dependencies.md)** - Dependency mapping and cascade patterns
- **[DSM Password Change Procedure](../procedures/dsm-password-change.md)** - Specific procedure for CSI credential update
- **[Post-Mortems](post-mortems/)** - Historical incidents and lessons learned
- **[Application Deployment Standard](../reference/application-deployment-standard.md)** - Best practices to prevent failures

---

## 📞 Emergency Contacts

**During cascade failure:**
- Check #alerts channel for automated alerts
- Escalate to on-call if MTTR > 2 hours
- Document all actions in incident channel

**Severity Assessment:**
- **S0 (Critical):** Production cluster down, >50% apps affected
- **S1 (High):** 20-50% apps affected, critical services impacted
- **S2 (Medium):** <20% apps affected, non-critical services
- **S3 (Low):** Single app failure, no cascade detected

---

**Last Updated:** 2026-02-07
**Maintained by:** Infrastructure Team
**Review Frequency:** After each cascade failure incident
