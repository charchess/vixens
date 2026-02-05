# Application State - Desired (Reference Standard)

**Last Updated:** 2026-02-05 (Stabilization Milestone)
**Purpose:** Reference configuration serving as single source of truth for application profiles
**Maintenance:** Updated manually or by agent during architectural decisions

---

## 📋 How to Use This Document

This document defines the **desired state** for all applications. It serves as:

1. **Reference for new deployments** - Copy these settings when adding apps
2. **Conformity baseline** - Compare STATE-ACTUAL.md against this document
3. **Decision tracking** - Changes reflect architectural decisions made during sessions
4. **Automation input** - Scripts can generate patches from this data

**Workflow Example:**
```
Decision: "Upgrade homeassistant to Critical backup profile"
→ Update this document (STATE-DESIRED.md)
→ Apply change to GitOps manifests
→ Verify in production
→ Update STATE-ACTUAL.md to reflect reality
```

---

## Production Application Standards

| App                            | NS                         | CPU Req  | CPU Lim  | Mem Req  | Mem Lim  | Profile | Priority      | Sync Wave | Backup Profile | Target Score |
|:-------------------------------|:---------------------------|:---------|:---------|:---------|:---------|:--------|:--------------|:---------:|:---------------|:------------:|
| **adguard-home**               | networking                 | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-high   | 0         | Standard       | 100          |
| **amule**                      | media                      | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-low    | 0         | None           | 85           |
| **argocd**                     | argocd                     | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-critical| -2        | Relaxed        | 100          |
| **authentik**                  | auth                       | 300m     | 1000m    | 1536Mi   | 2Gi      | Medium  | vixens-critical| 0         | Standard       | 100          |
| **birdnet-go**                 | birdnet-go                 | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Relaxed        | 85           |
| **booklore**                   | media                      | 200m     | 1000m    | 1536Mi   | 3Gi      | Medium  | vixens-medium | 0         | Relaxed        | 85           |
| **cert-manager**               | cert-manager               | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **cert-manager-config**        | cert-manager               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **cert-manager-secrets**       | cert-manager               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **cert-manager-webhook-gandi** | cert-manager               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -4        | None           | 100          |
| **changedetection**            | tools                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Relaxed        | 85           |
| **cilium**                     | kube-system                | 100m     | 1000m    | 256Mi    | 512Mi    | Small   | vixens-critical| -2        | None           | 100          |
| **cilium-lb**                  | kube-system                | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| -2        | None           | 100          |
| **cloudnative-pg**             | cnpg-system                | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-critical| -4        | None           | 100          |
| **contacts**                   | networking                 | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 85           |
| **descheduler**                | kube-system                | 50m      | 200m     | 128Mi    | 256Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **docspell-native**            | services                   | 500m     | 2000m    | 2048Mi   | 4096Mi   | Large   | vixens-medium | 0         | Standard       | 100          |
| **external-dns-gandi**         | networking                 | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| 0         | Relaxed        | 100          |
| **external-dns-unifi**         | networking                 | 50m      | 500m     | 64Mi     | 128Mi    | Small   | vixens-critical| 0         | Relaxed        | 100          |
| **firefly-iii**                | finance                    | 200m     | 500m     | 256Mi    | 512Mi    | Medium  | vixens-medium | 0         | Standard       | 100          |
| **frigate**                    | media                      | 3000m    | 8000m    | 4Gi      | 8Gi      | XLarge  | vixens-medium | 0         | Standard       | 100          |
| **gluetun**                    | services                   | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **goldilocks**                 | monitoring                 | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **grafana**                    | monitoring                 | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-high   | 0         | Relaxed        | 100          |
| **headlamp**                   | tools                      | 100m     | 200m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 100          |
| **homeassistant**              | homeassistant              | 300m     | 1000m    | 1536Mi   | 3072Mi   | Medium  | vixens-high   | 0         | Critical       | 100          |
| **homepage**                   | tools                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | None           | 85           |

---

## Decision History

Record of architectural decisions reflected in this document:

### 2026-01-10 - Initial Reference Creation
- Established baseline desired state from production analysis
- Aligned backup profiles with ADR-014 (Litestream profiles)
- Set target scores to 85+ for all applications (100 for Elite/Gold)
- Identified legacy ingress resources for removal

### 🏆 Elite Compliance Standards (Kyverno & Policies)

To achieve a "Gold" status and 100% conformity, applications must adhere to these policy-driven standards:

### 1. Metadata & Labels
- **Managed By:** Every resource must have the label `app.kubernetes.io/managed-by: argocd`.
- **Environment:** Namespaces must have the label `vixens.lab/environment: shared` (or prod/dev).
- **Goldilocks:** Namespaces must have `goldilocks.fairwinds.com/enabled: "true"` for VPA visibility.

### 2. Reliability (Probes)
- **Primary Container:** Must have both `livenessProbe` and `readinessProbe` (HTTP or TCP).
- **Sidecars (Standard):** Sidecars like `config-syncer` (rclone) or `litestream` **MUST** have a `readinessProbe` to ensure sync is operational.
- **Sidecars (Graceful):** Ensure `terminationGracePeriodSeconds` is set to at least 30s for database-heavy sidecars.

### 3. Resources (Elite QoS)
- **QoS Guaranteed:** CPU/Mem Requests **MUST** equal Limits for all "Critical" and "High" priority apps.
- **Priority Class:** Every Deployment/StatefulSet must have a valid `priorityClassName` (defaulting to `vixens-medium`).

---

## Decision History

### 2026-02-05 - Stabilization Milestone (v3.1.536)
- Adjusted Frigate CPU requests to 3000m based on live usage (Prometheus).
- Increased Home Assistant Mem requests to 1536Mi to ensure Guaranteed QoS.
- Standardized Booklore RAM at 1.5Gi (Medium profile) after VPA observations.
- Formalized Kyverno sidecar probe requirements in Elite Standards.
- Aligned namespaces for media apps (amule, pyload, qbittorrent moved to media).
- Added Trivy, Radar, and Cilium to the baseline inventory.

### 2026-01-10 - Initial Reference Creation
- Established baseline desired state from production analysis
- Aligned backup profiles with ADR-014 (Litestream profiles)
- Set target scores to 85+ for all applications (100 for Elite/Gold)
- Identified legacy ingress resources for removal.md
