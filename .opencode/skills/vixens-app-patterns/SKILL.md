---
name: vixens-app-patterns
description: |
  Master reference for Vixens K8s application patterns. Covers stateless (native/Helm), 
  stateful (SQLite/PostgreSQL), and complex (multi-deployment) apps. Includes templates, 
  patterns (NetworkPolicy, ServiceMonitor, VPA), decision matrix, and DoD checklist.
  
  Use when: deploying new apps, refactoring existing apps, or validating compliance with 
  Vixens standards (sizing labels, Litestream backup, Infisical secrets, ArgoCD overlays).
version: 1.0.0
author: Vixens Infrastructure Team
---

# Vixens Application Patterns

**Comprehensive guide for deploying applications to Vixens K8s cluster (Talos + ArgoCD + Kustomize).**

## 📚 Table of Contents

1. [Quick Start](#quick-start)
2. [Templates](#templates)
3. [Patterns](#patterns)
4. [Decision Matrix](#decision-matrix)
5. [Standards (DoD)](#standards-dod)
6. [Examples](#examples)
7. [Troubleshooting](#troubleshooting)

---

## Quick Start

### 1. Determine App Type

| Type | Characteristics | Template |
|------|-----------------|----------|
| **Stateless (Native)** | No persistence, K8s manifests | `templates/stateless-native/` |
| **Stateless (Helm)** | No persistence, Helm chart | `templates/stateless-helm/` |
| **Stateful (SQLite)** | SQLite + config files | `templates/stateful/` |
| **Complex** | Multi-deployment, NetworkPolicy, metrics | `templates/complex/` |

### 2. Copy Template

```bash
# Example: Deploy stateless app (native K8s)
cp -r .opencode/skills/vixens-app-patterns/templates/stateless-native apps/70-tools/my-new-app

# Adapt manifests
cd apps/70-tools/my-new-app
# Edit base/deployment.yaml, base/service.yaml, overlays/*/ingress.yaml
```

### 3. Validate & Deploy

```bash
# Validate YAML
yamllint apps/70-tools/my-new-app/**/*.yaml

# Test Kustomize build
kustomize build apps/70-tools/my-new-app/overlays/dev

# Check for kustomization.yaml changes (regression detection)
kustomize build apps/70-tools/my-new-app/overlays/dev | grep '^kind:' | sort

# Commit & push
git add apps/70-tools/my-new-app
git commit -m "feat(my-new-app): deploy to dev"
git push origin main

# ArgoCD will auto-sync (or manual sync via UI)
```

---

## Templates

### Stateless (Native K8s)

**Use for**: Simple apps without persistence (whoami, stirling-pdf)

**Structure**:
```
apps/70-tools/my-app/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml       # Sizing labels, tolerations, probes
│   ├── service.yaml
│   └── namespace.yaml         # Optional
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml # replicas: 0, components
    │   └── ingress.yaml       # dev.truxonline.com
    └── prod/
        ├── kustomization.yaml # components (gold-maturity)
        └── ingress.yaml       # truxonline.com
```

**Key Features**:
- Sizing labels (Kyverno injects resources)
- priorityClassName + CP toleration
- Probes (liveness, readiness, startup)
- Annotations: `vixens.io/nometrics`, `fast-start`, `no-long-connections`
- Overlays: dev (replicas: 0), prod (gold-maturity components)

**See**: `templates/stateless-native/`

---

### Stateless (Helm)

**Use for**: Apps deployed via Helm charts (it-tools, stirling-pdf)

**Structure**:
```
apps/70-tools/my-app/
├── base/
│   ├── kustomization.yaml     # Empty (Helm manages resources)
│   └── values.yaml            # Helm values (sizing, tolerations, probes)
└── overlays/
    ├── dev/
    │   └── kustomization.yaml # replicas: 0 patch, components
    └── prod/
        └── kustomization.yaml # gold-maturity patches (metadata only)
```

**Key Features**:
- `values.yaml` instead of `deployment.yaml`
- Sizing labels in `podLabels` (Helm values)
- Kustomize patches ONLY Deployment metadata (not pod template)
- Limitations: `revisionHistoryLimit` hardcoded in chart (can't patch)

**See**: `templates/stateless-helm/`

---

### Stateful

**Use for**: Apps with SQLite + config files (vaultwarden, trilium)

**Structure**:
```
apps/60-services/my-app/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml        # + init containers + sidecars
│   ├── service.yaml
│   ├── pvc.yaml               # RWO, synelia-iscsi-retain
│   ├── infisical-secret.yaml  # Secrets from Infisical
│   └── litestream-config.yaml # S3 backup config
└── overlays/
    ├── dev/
    │   └── kustomization.yaml # replicas: 0, infisical envSlug: dev
    └── prod/
        ├── kustomization.yaml # components, infisical envSlug: prod
        └── infisical-patch.yaml
```

**Key Features**:
- **Init containers**: `restore-config` (rclone), `restore-db` (litestream)
- **Sidecars**: `litestream` (SQLite backup), `config-syncer` (S3 sync)
- **PVC**: RWO volume for persistence
- **Infisical**: Secrets synced from Infisical vault
- **Sizing labels**: Per-container (main + sidecars + inits)

**See**: `templates/stateful/` + `examples/vaultwarden/`

---

### Complex

**Use for**: Multi-deployment apps (authentik: server+worker, homeassistant)

**Structure**:
```
apps/03-security/my-app/
├── base/
│   ├── kustomization.yaml
│   ├── deployment-server.yaml     # HTTP endpoint
│   ├── deployment-worker.yaml     # Background jobs
│   ├── service.yaml
│   ├── networkpolicy-server.yaml  # Ingress from Traefik
│   ├── networkpolicy-worker.yaml  # Egress only
│   ├── servicemonitor.yaml        # Prometheus metrics
│   ├── middleware.yaml            # Traefik forward-auth (optional)
│   └── vpa.yaml                   # VerticalPodAutoscaler (optional)
└── overlays/
    ├── dev/
    └── prod/
```

**Key Features**:
- Multiple Deployments (server/worker pattern)
- NetworkPolicy per-pod (server: ingress+egress, worker: egress only)
- ServiceMonitor (Prometheus scraping)
- VPA (VerticalPodAutoscaler)
- Traefik Middleware (custom CRDs)

**See**: `templates/complex/` + `examples/authentik/`

---

## Patterns

### NetworkPolicy

**Purpose**: Restrict pod network access (security)

**Pattern**: Ingress from Traefik, Egress allow-all (homelab)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: my-app
spec:
  podSelector:
    matchLabels:
      app: my-app
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: traefik
      ports:
        - protocol: TCP
          port: 80
  egress:
    - {}  # Allow all egress (homelab: DNS, APIs, inter-app)
```

**See**: `patterns/networkpolicy.yaml`

---

### ServiceMonitor

**Purpose**: Enable Prometheus metrics scraping

**Pattern**: Standard scrape config

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
    - port: metrics  # or "http"
      path: /metrics
      interval: 30s
```

**See**: `patterns/servicemonitor.yaml`

---

### VerticalPodAutoscaler

**Purpose**: VPA recommendations for resource tuning

**Pattern**: Off mode (manual review)

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: my-app
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  updatePolicy:
    updateMode: "Off"  # Manual review only
```

**See**: `patterns/vpa.yaml`

---

## Decision Matrix

### Which Template?

| Question | Answer | Template |
|----------|--------|----------|
| Does it persist data? | No | stateless-native or stateless-helm |
| Is it a Helm chart? | Yes | stateless-helm |
| Does it use SQLite? | Yes | stateful |
| Multiple deployments? | Yes | complex |
| Does it expose metrics? | Yes | Add ServiceMonitor |
| Needs network restrictions? | Yes | Add NetworkPolicy |

### Component Selection

| Component | Dev | Prod | Use When |
|-----------|-----|------|----------|
| `replicas: 0` patch | ✅ | ❌ | Always in dev |
| `gold-maturity` | ❌ | ✅ | Always in prod |
| `revision-history-limit` | ✅ | ✅ | Always |
| `priority/<tier>` | ⚠️ | ✅ | If not in base |
| `resources` | ⚠️ | ⚠️ | Only if explicit resources needed |
| `poddisruptionbudget/0` | ❌ | ✅ | High-availability apps |

---

## Standards (DoD)

### Mandatory Checklist

**Before deployment, ALL apps MUST have**:

- [ ] **1. Sizing labels** — `vixens.io/sizing.<container>: <tier>` (NO explicit `resources: {}`)
- [ ] **2. Priority class** — `priorityClassName: vixens-<critical|high|medium|low>`
- [ ] **3. CP toleration** — `node-role.kubernetes.io/control-plane` toleration present
- [ ] **4. Revision history** — `revisionHistoryLimit: 3` (reduces etcd growth)
- [ ] **5. Probes** — liveness, readiness, startup (HTTP/TCP/exec)
- [ ] **6. Overlays** — dev (replicas: 0), prod (gold-maturity components)
- [ ] **7. Ingress** — HTTPS only, cert-manager TLS, Traefik https-redirect middleware
- [ ] **8. Security context** — fsGroup, runAsNonRoot, capabilities drop ALL

**Stateful apps MUST ALSO have**:

- [ ] **9. Persistence** — PVC (RWO, synelia-iscsi-retain)
- [ ] **10. Backup** — Litestream (SQLite) OR external DB backup documented
- [ ] **11. Restore** — Init containers (restore-config, restore-db)
- [ ] **12. Secrets** — Infisical sync OR k8s Secret with rotation

**Complex apps MAY have**:

- [ ] **13. NetworkPolicy** — Ingress/egress rules defined
- [ ] **14. ServiceMonitor** — Prometheus metrics exposed
- [ ] **15. VPA** — VerticalPodAutoscaler configured

---

### Sizing Tiers

```
B-nano    (Build/Init): 10m/32Mi
B-small   (Build/Init): 50m/64Mi
V-nano    (Sidecar):    10m/128Mi
V-small   (Sidecar):    50m/256Mi
V-medium  (Sidecar):    100m/512Mi
V-large   (Sidecar):    500m/1Gi
G-nano    (Guaranteed): 10m/32Mi (requests=limits)
G-small   (Guaranteed): 100m/256Mi
G-medium  (Guaranteed): 500m/512Mi
G-large   (Guaranteed): 1000m/1Gi
G-xl      (Guaranteed): 2000m/2Gi

SB-*      (StatefulSet Burstable): Same as V-* but for StatefulSets
```

**Rule**: Kyverno injects resources via labels. NEVER add `resources: {}` blocks.

---

### Annotations

**Vixens-specific**:
```yaml
vixens.io/nometrics: "true"              # Disable Prometheus scraping
vixens.io/fast-start: "true"             # Fast startup (reduce probe delays)
vixens.io/service-binding: "false"       # No service binding (CNPG pattern)
vixens.io/no-long-connections: "true"    # Stateless hint
vixens.io/explicitly-allow-root: "true"  # Security override (dangerous)
vixens.io/health-bump: "5"               # Probe timeout multiplier
vixens.io/backup-profile: "critical"     # Backup strategy (critical|standard|relaxed)
```

**Standard K8s**:
```yaml
reloader.stakater.com/auto: "true"          # Auto-reload on ConfigMap change
prometheus.io/scrape: "true"                # Enable Prometheus scraping
prometheus.io/port: "9090"                  # Metrics port
prometheus.io/path: "/metrics"              # Metrics path
goldilocks.fairwinds.com/enabled: "true"    # Goldilocks VPA recommendations
autoscaling.k8s.io/vpa: "true"              # VPA enabled
```

---

## Examples

### 1. Whoami (Stateless Native)

**Location**: `examples/whoami/`

**Type**: Stateless, native K8s, minimal

**Features**:
- Single container (no sidecars)
- Sizing: V-nano
- Probes: HTTP /health
- Dev: replicas: 0 + 2 components
- Prod: 6 components (gold-maturity, base, resources, pdb/0, priority/low, revision-history-limit)

**Learn**: Simplest possible deployment, component usage pattern

---

### 2. Vaultwarden (Stateful)

**Location**: `examples/vaultwarden/`

**Type**: Stateful, SQLite + config files, full resilience

**Features**:
- 3 containers: main + litestream + config-syncer
- 3 init containers: fix-permissions + restore-config + restore-db
- PVC: 5Gi RWO
- Litestream: S3 backup every 6h, 7d retention
- Config-syncer: S3 sync every 60s
- Infisical: secrets sync
- NetworkPolicy: Traefik ingress only
- ServiceMonitor: Litestream metrics

**Learn**: Complete stateful app with zero-data-loss backup/restore

---

### 3. Authentik (Complex)

**Location**: `examples/authentik/`

**Type**: Complex, multi-deployment, auth provider

**Features**:
- 2 deployments: server (HTTP) + worker (background jobs)
- 2 NetworkPolicies: server (ingress+egress), worker (egress only)
- ServiceMonitor: server metrics only
- Traefik Middleware: forward-auth for SSO
- Config-syncer: blueprints backup
- Infisical: 2 secrets (redis + main)

**Learn**: Multi-deployment pattern, per-pod NetworkPolicy, custom CRDs

---

## Troubleshooting

### Kustomize build fails

**Error**: `accumulating resources: accumulation err='accumulating resources from '../../base': ...`

**Fix**: Check `kustomization.yaml` references. All paths must exist.

```bash
# Debug
kustomize build apps/my-app/overlays/dev --enable-alpha-plugins

# Common issues
# - Missing base/kustomization.yaml
# - Wrong relative path to base
# - Resource file doesn't exist
```

---

### ArgoCD sync fails

**Error**: `ComparisonError: Manifest generation error`

**Fix**: Validate YAML syntax + Kustomize build locally

```bash
yamllint apps/my-app/**/*.yaml
kustomize build apps/my-app/overlays/prod
```

---

### Pod stuck in Pending

**Error**: `0/5 nodes available: 5 Insufficient memory`

**Fix**: Check sizing labels + VPA recommendations

```bash
# Check pod events
kubectl describe pod -n <namespace> <pod-name>

# Check VPA recommendations
kubectl get vpa -n <namespace> <app-name> -o yaml

# Adjust sizing labels in deployment.yaml
vixens.io/sizing.<container>: V-small  # Increase tier
```

---

### Litestream restore fails

**Error**: `no database found at s3://bucket/app/db.sqlite3`

**Fix**: First-time deploy OR S3 bucket empty

```bash
# This is EXPECTED on first deploy
# Litestream creates initial backup after first write

# Verify S3 bucket access
kubectl exec -n <namespace> <pod> -c restore-db -- \
  env | grep LITESTREAM

# Check S3 bucket contents (from local machine)
mc ls prod/<bucket>/
```

---

### Infisical secret not syncing

**Error**: `InfisicalSecret reconciliation failed`

**Fix**: Check Infisical credentials + path

```bash
# Check InfisicalSecret status
kubectl get infisicalsecret -n <namespace> <secret-name> -o yaml

# Verify Infisical operator logs
kubectl logs -n infisical-operator-system -l app=infisical-operator

# Common issues
# - Wrong secretsPath (/apps/<category>/<app>)
# - Wrong envSlug (dev vs prod)
# - Infisical universal-auth secret expired
```

---

## References

- **Golden Standard**: `~/vixens/docs/reference/app-golden-standard.md`
- **Backup Patterns**: `~/vixens/docs/guides/backup-restore-pattern.md`
- **Config-Syncer**: `~/vixens/docs/guides/pattern-config-syncer.md`
- **Litestream ADR**: `~/vixens/docs/adr/014-litestream-backup-profiles-and-recovery-patterns.md`
- **Adding New App**: `~/vixens/docs/guides/adding-new-application.md`

---

**Version**: 1.0.0  
**Last Updated**: 2026-03-10  
**Maintainer**: Vixens Infrastructure Team
