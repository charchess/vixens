# Infrastructure Dependencies

**Last Updated:** 2026-02-07
**Status:** Active

---

## 🎯 Purpose

This document maps the infrastructure dependencies in the Vixens Kubernetes cluster, with particular focus on critical paths that can cause cascade failures. Understanding these dependencies is essential for:

- **Incident Response:** Quickly identify root cause of failures
- **Change Management:** Assess blast radius before making changes
- **Capacity Planning:** Understand resource constraints
- **DR Planning:** Prioritize recovery order

---

## 📊 Dependency Hierarchy

### Level 0: Foundation (Hardware/Network)

```
Physical Infrastructure
├── Synology NAS (192.168.111.69)
│   ├── iSCSI Target Service
│   ├── NFS Service
│   └── Infisical (http://192.168.111.69:8085)
├── Network Infrastructure
│   ├── VLAN 111 (Internal) - 192.168.111.0/24
│   └── VLAN 20X (Services) - 192.168.20X.0/24
└── Talos Nodes
    ├── Control Planes (3x HA)
    └── Worker Nodes
```

**Critical Failure Impact:** Complete cluster outage

---

### Level 1: Core Infrastructure Services

```
Kubernetes Core
├── etcd (Control Plane)
├── kube-apiserver
├── kube-controller-manager
├── kube-scheduler
└── kubelet (all nodes)

CNI (Cilium)
├── cilium-operator
├── cilium-agent (DaemonSet)
└── Cilium L2 IPAM + LB

Ingress Controller (Traefik)
├── traefik-controller
└── LoadBalancer Service (192.168.201.70)

Storage Drivers
├── Synology CSI Driver ⚠️ CRITICAL
│   ├── synology-csi-controller
│   └── synology-csi-node (DaemonSet)
└── NFS Provisioner
    └── nfs-subdir-external-provisioner
```

**Critical Failure Impact:** Cannot schedule pods, no network, no storage, no ingress

---

### Level 2: Platform Services

```
Secret Management
└── Infisical Operator ⚠️ CRITICAL
    ├── infisical-controller-manager
    └── Universal Auth Secret (namespace: argocd)

GitOps (ArgoCD)
├── argocd-server
├── argocd-application-controller
├── argocd-repo-server
└── argocd-redis

Policy Enforcement
└── Kyverno ⚠️ CRITICAL (Webhook)
    ├── kyverno-admission-controller
    ├── kyverno-background-controller
    ├── kyverno-cleanup-controller
    └── kyverno-reports-controller

Monitoring
├── Prometheus
├── Grafana
├── Loki
└── Promtail
```

**Critical Failure Impact:**
- Infisical: Cannot sync secrets → Apps fail
- ArgoCD: Cannot deploy/sync apps
- Kyverno: Cannot validate resources → Sync blocked
- Monitoring: Blind operations (no impact on apps)

---

### Level 3: Shared Services (Databases)

```
PostgreSQL (CloudNativePG)
└── postgresql-shared ⚠️ CRITICAL
    ├── Uses: iSCSI PVC (RWO)
    └── Dependents: 10+ applications

MariaDB
└── mariadb-shared ⚠️ CRITICAL
    ├── Uses: iSCSI PVC (RWO)
    └── Dependents: 5+ applications

Redis
└── redis-shared
    └── Dependents: Various caching needs
```

**Critical Failure Impact:** All dependent applications fail

---

### Level 4: Application Services

Applications are categorized by their storage dependencies:

#### Category A: iSCSI-dependent (HIGH RISK)

These apps use ReadWriteOnce (RWO) volumes from Synology CSI:

```
Home Automation
├── Home Assistant (iSCSI + strategy: Recreate)
├── Mosquitto (StatefulSet, iSCSI)
└── Music Assistant

Media Stack
├── Jellyfin (iSCSI)
├── Prowlarr (iSCSI)
├── Radarr (iSCSI)
├── Sonarr (iSCSI)
├── Lidarr (iSCSI)
├── LazyLibrarian (iSCSI)
├── Whisparr (iSCSI)
├── qBittorrent (iSCSI)
├── SABnzbd (iSCSI)
└── Pyload (iSCSI)

Tools
├── Penpot (iSCSI)
├── NocoDB (iSCSI)
├── Vikunja (iSCSI)
├── Linkwarden (iSCSI)
└── Stirling-PDF (iSCSI)

Networking
├── AdGuard Home (iSCSI + Litestream)
└── NetBird (iSCSI)
```

**Failure Cascade Path:**
```
CSI credentials invalid
    ↓
iSCSI login failures
    ↓
Volume attach/detach errors
    ↓
Multi-Attach errors
    ↓
Pods stuck ContainerCreating
    ↓
Applications Degraded/Progressing
```

#### Category B: Database-dependent (MEDIUM RISK)

These apps depend on shared PostgreSQL/MariaDB:

```
PostgreSQL-dependent
├── Authentik (auth + postgresql-shared)
├── Mealie (postgresql-shared)
├── Firefly III (postgresql-shared)
├── NetBox (postgresql-shared)
├── Docspell (postgresql-shared)
└── Contacts (postgresql-shared)

MariaDB-dependent
├── Booklore (mariadb-shared)
├── Vikunja (mariadb-shared)
└── [others]
```

**Failure Cascade Path:**
```
CSI credentials invalid
    ↓
Database PVC cannot mount
    ↓
Database pods stuck Init
    ↓
Dependent apps cannot connect
    ↓
Applications Progressing/Degraded
```

#### Category C: NFS-dependent (LOW RISK)

These apps use NFS from Synology (no authentication needed):

```
├── Velero (backup storage)
├── Media shared storage (NFS)
└── Various apps with NFS volumes
```

**Failure Impact:** Minimal, NFS more resilient than iSCSI

#### Category D: Stateless (NO RISK)

These apps have no persistent storage:

```
├── whoami
├── IT-Tools
├── Headlamp
└── Various monitoring components
```

**Failure Impact:** None, can restart anywhere

---

## 🔥 Critical Failure Scenarios

### Scenario 1: CSI Credentials Invalid (2026-02-07 Incident)

**Trigger:** Synology DSM password changed without updating Infisical secret

**Cascade Path:**
```
DSM password change
    ↓
synology-csi-credentials-sync (Infisical) outdated
    ↓
CSI driver authentication failures
    ↓
iSCSI login errors: "Failed to login with target iqn"
    ↓
Volume attachment failures + Multi-Attach errors
    ↓
Pods stuck: ContainerCreating, Init:0/1, Pending
    ↓
Databases cannot start (postgresql-shared, mariadb-shared)
    ↓
Database-dependent apps fail (authentik, mealie, etc.)
    ↓
Resource contention from cascading restarts
    ↓
Kyverno webhook temporarily unavailable (collateral)
    ↓
ArgoCD sync failures → apps OutOfSync
    ↓
Cluster-wide degradation
```

**Affected:** 50+ applications, 2+ hours recovery time

**Prevention:** Follow [DSM Password Change Procedure](../procedures/dsm-password-change.md)

---

### Scenario 2: Infisical Operator Failure

**Trigger:** Infisical operator crashes or loses connection to Infisical server

**Cascade Path:**
```
Infisical operator down
    ↓
InfisicalSecrets not synced
    ↓
Kubernetes secrets stale/missing
    ↓
Apps using these secrets fail (CSI, cert-manager, external-dns, etc.)
    ↓
Cascading failures based on affected secrets
```

**Affected:** All apps using InfisicalSecret

**Mitigation:**
- Existing secrets persist (not deleted)
- Manual secret creation possible as emergency workaround

---

### Scenario 3: Kyverno Webhook Unavailable

**Trigger:** Kyverno pods restart during cluster disruption

**Cascade Path:**
```
Kyverno admission-controller unavailable
    ↓
Webhook validation failures
    ↓
ArgoCD cannot apply resources
    ↓
Applications stuck OutOfSync
    ↓
Recovery blocked until Kyverno stabilizes
```

**Affected:** All ArgoCD applications during incident window

**Mitigation:**
- Kyverno has FailurePolicy: Fail (safe default)
- Wait for Kyverno to recover (usually <5 min)
- Temporary: Scale down Kyverno (emergency only!)

---

### Scenario 4: Shared Database Corruption

**Trigger:** I/O errors, node crash, improper shutdown

**Cascade Path:**
```
Database corruption (PostgreSQL/MariaDB)
    ↓
Pod CrashLoopBackOff or Init:0/1
    ↓
All dependent apps cannot connect
    ↓
Apps Degraded (waiting for DB)
```

**Affected:** 10-15 applications per database

**Recovery:**
1. Delete pod (force reschedule)
2. If PVC corrupted: Restore from Velero backup
3. If I/O error: Check Synology NAS health

---

## 📈 Dependency Impact Matrix

| Component | Direct Dependents | Indirect Dependents | MTTR | Blast Radius |
|-----------|-------------------|---------------------|------|--------------|
| **Synology CSI** | 40+ apps (iSCSI PVCs) | All cluster (critical) | 30-60 min | 🔴 CRITICAL |
| **Infisical Operator** | 20+ apps (secrets) | Dependent apps | 10-30 min | 🟠 HIGH |
| **postgresql-shared** | 10+ apps | None | 5-15 min | 🟡 MEDIUM |
| **mariadb-shared** | 5+ apps | None | 5-15 min | 🟡 MEDIUM |
| **Kyverno** | ArgoCD (webhook) | All apps (indirect) | 2-5 min | 🟠 HIGH |
| **ArgoCD** | Deployment workflow | None (apps keep running) | 10-20 min | 🟢 LOW |
| **Traefik** | Ingress access | None (pods run) | 5-10 min | 🟢 LOW |

**MTTR:** Mean Time To Recovery (estimated)
**Blast Radius:** Scope of impact on cluster

---

## 🛡️ Best Practices

### 1. Change Management

**ALWAYS assess dependencies before making changes:**

```bash
# Before changing DSM password
1. Check CSI dependency: kubectl get pods -n synology-csi
2. List affected PVCs: kubectl get pvc -A | grep synology
3. Review procedure: docs/procedures/dsm-password-change.md
4. Plan maintenance window: 30-60 min

# Before upgrading shared database
1. List dependent apps: See Level 4 dependency tree
2. Test upgrade in dev first
3. Have rollback plan (Velero backup)
4. Communicate downtime window
```

### 2. Incident Response

**Follow dependency tree from bottom-up:**

```
1. Identify failing apps (Level 4)
2. Check databases (Level 3)
3. Verify platform services (Level 2)
4. Validate core infrastructure (Level 1)
5. Inspect foundation (Level 0)
```

**Use dependency knowledge to narrow investigation:**
- Multiple apps failing → Check shared database
- All apps with PVCs failing → Check CSI driver
- ArgoCD sync failures → Check Kyverno webhook
- No new pods scheduling → Check core Kubernetes

### 3. Monitoring Priorities

**Alert on these critical paths:**

```yaml
Critical Alerts (P0):
  - Synology CSI authentication failures
  - Kyverno webhook unavailable
  - Infisical operator down
  - etcd unhealthy
  - Control plane node down

High Alerts (P1):
  - Shared database unhealthy
  - ArgoCD sync failures (>5 apps)
  - Traefik LoadBalancer down

Medium Alerts (P2):
  - Individual app failures
  - PVC mount delays
  - Resource limits reached
```

### 4. Testing Resilience

**Chaos engineering scenarios:**

```bash
# Test CSI failure recovery
1. Intentionally break CSI credentials
2. Observe cascade
3. Follow recovery procedure
4. Document actual vs. expected behavior

# Test database failure recovery
1. Delete database pod
2. Verify dependent apps handle gracefully
3. Confirm auto-recovery

# Test Kyverno webhook failure
1. Scale Kyverno to 0 temporarily
2. Attempt ArgoCD sync
3. Verify FailurePolicy behavior
4. Scale back and confirm recovery
```

---

## 🔍 Troubleshooting Checklist

When investigating failures, follow this systematic approach:

### Step 1: Identify Symptoms
```bash
# Applications unhealthy
kubectl get applications -n argocd | grep -v "Synced.*Healthy"

# Pods not running
kubectl get pods -A --field-selector status.phase!=Running,status.phase!=Succeeded

# PVC issues
kubectl get pvc -A | grep -v Bound

# Volume attachment problems
kubectl get volumeattachments | grep -v true
```

### Step 2: Check Critical Components
```bash
# CSI Driver
kubectl get pods -n synology-csi
kubectl logs -n synology-csi synology-csi-controller-0 -c synology-csi-plugin --tail=50

# Infisical Operator
kubectl get pods -n infisical-operator-system
kubectl get infisicalsecret -A

# Kyverno
kubectl get pods -n kyverno
kubectl logs -n kyverno -l app.kubernetes.io/component=admission-controller --tail=50

# Databases
kubectl get pods -n databases
```

### Step 3: Review Events
```bash
# Recent cluster events
kubectl get events -A --sort-by='.lastTimestamp' | tail -50

# Namespace-specific events
kubectl get events -n {namespace} --sort-by='.lastTimestamp'
```

### Step 4: Consult Runbooks
- **CSI issues:** [DSM Password Change Procedure](../procedures/dsm-password-change.md)
- **Cascade failures:** [Cascade Failure Recovery Runbook](../troubleshooting/cascade-failure-recovery.md)
- **Database issues:** Check Level 3 dependency tree above

---

## 📚 Related Documentation

- **[DSM Password Change Procedure](../procedures/dsm-password-change.md)** - Step-by-step for credential updates
- **[Cascade Failure Recovery](../troubleshooting/cascade-failure-recovery.md)** - Generic recovery runbook
- **[Post-Mortem 2026-02-07](../troubleshooting/post-mortems/2026-02-07-dsm-password-cascade-failure.md)** - Real incident analysis
- **[Synology CSI Documentation](../applications/01-storage/synology-csi.md)** - CSI driver details
- **[Application Deployment Standard](application-deployment-standard.md)** - Deployment best practices

---

**Maintained by:** Infrastructure Team
**Review Frequency:** After each major incident or architecture change
