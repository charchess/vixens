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
| **hubble-ui**                  | monitoring                 | 100m     | 200m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 100          |
| **hydrus-client**              | media                      | 100m     | 1000m    | 2Gi      | 4Gi      | Large   | vixens-medium | 0         | Mixed          | 100          |
| **infisical-operator**         | infisical-operator-system  | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **it-tools**                   | tools                      | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **jellyfin**                   | media                      | 100m     | 2000m    | 1Gi      | 4Gi      | Large   | vixens-medium | 0         | Relaxed        | 100          |
| **jellyseerr**                 | media                      | 100m     | 500m     | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **kyverno**                    | kyverno                    | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **lazylibrarian**              | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-low    | 0         | Standard       | 85           |
| **lidarr**                     | media                      | 100m     | 1000m    | 256Mi    | 1Gi      | Small   | vixens-medium | 0         | Standard       | 100          |
| **linkwarden**                 | tools                      | 100m     | 1000m    | 1Gi      | 2Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **loki**                       | monitoring                 | 100m     | 500m     | 512Mi    | 1024Mi   | Small   | vixens-high   | 0         | Relaxed        | 100          |
| **mail-gateway**               | services                   | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| 0         | Relaxed        | 100          |
| **mariadb-shared**             | databases                  | 200m     | 1000m    | 512Mi    | 1024Mi   | Medium  | vixens-critical| -1        | Standard       | 100          |
| **mealie**                     | mealie                     | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **metrics-server**             | kube-system                | 100m     | 500m     | 200Mi    | 500Mi    | Small   | vixens-critical| -2        | None           | 100          |
| **mosquitto**                  | mosquitto                  | 50m      | 200m     | 128Mi    | 256Mi    | Micro   | vixens-high   | 0         | Relaxed        | 100          |
| **music-assistant**            | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 85           |
| **mylar**                      | media                      | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **netbird**                    | networking                 | 100m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-high   | 0         | Standard       | 100          |
| **netbox**                     | tools                      | 100m     | 1000m    | 1Gi      | 2Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **netvisor**                   | networking                 | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **nfs-storage**                | media-stack                | 50m      | 200m     | 128Mi    | 256Mi    | Micro   | vixens-critical| -2        | None           | 100          |
| **nocodb**                     | tools                      | 100m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **penpot**                     | tools                      | 200m     | 1000m    | 1Gi      | 2Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **postgresql-shared**          | databases                  | 200m     | 1000m    | 1024Mi   | 2Gi      | Medium  | vixens-critical| -1        | Standard       | 100          |
| **priority-classes**           | kube-system                | N/A      | N/A      | N/A      | N/A      | N/A     | N/A           | -5        | None           | 100          |
| **prometheus**                 | monitoring                 | 500m     | 2000m    | 1Gi      | 2Gi      | Medium  | vixens-high   | 0         | Relaxed        | 100          |
| **promtail**                   | monitoring                 | 50m      | 100m     | 128Mi    | 256Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **prowlarr**                   | media                      | 50m      | 500m     | 200Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **pyload**                     | media                      | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-low    | 0         | None           | 85           |
| **qbittorrent**                | media                      | 50m      | 500m     | 256Mi    | 1Gi      | Small   | vixens-low    | 0         | None           | 85           |
| **radar**                      | tools                      | 100m     | 500m     | 128Mi    | 512Mi    | Small   | vixens-medium | 0         | None           | 100          |
| **radarr**                     | media                      | 100m     | 500m     | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **redis-shared**               | databases                  | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| -1        | Relaxed        | 100          |
| **reloader**                   | tools                      | 10m      | 100m     | 128Mi    | 256Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **robusta**                    | robusta                    | 250m     | 500m     | 1Gi      | 1Gi      | Medium  | vixens-high   | 0         | None           | 100          |
| **renovate**                   | tools                      | 1000m    | 2000m    | 1Gi      | 2Gi      | Large   | vixens-medium | 0         | None           | 85           |
| **sabnzbd**                    | media                      | 100m     | 500m     | 256Mi    | 1Gi      | Small   | vixens-low    | 0         | Standard       | 100          |
| **sonarr**                     | media                      | 100m     | 1000m    | 512Mi    | 1Gi      | Small   | vixens-medium | 0         | Standard       | 100          |
| **stirling-pdf**               | tools                      | 100m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | None           | 85           |
| **synology-csi**               | synology-csi               | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **traefik**                    | traefik                    | 250m     | 2000m    | 512Mi    | 2Gi      | Medium  | vixens-critical| -2        | Relaxed        | 100          |
| **trivy**                      | security                   | 200m     | 1000m    | 1Gi      | 1Gi      | Medium  | vixens-medium | 0         | None           | 100          |
| **vaultwarden**                | services                   | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-high   | 0         | Standard       | 100          |
| **velero**                     | velero                     | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **vikunja**                    | tools                      | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **vpa**                        | vpa                        | 50m      | 200m     | 200Mi    | 500Mi    | Small   | vixens-medium | -2        | None           | 85           |
| **whisparr**                   | media                      | 100m     | 1000m    | 512Mi    | 1Gi      | Small   | vixens-medium | 0         | Standard       | 100          |
| **whoami**                     | whoami                     | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-low    | 0         | None           | 85           |

---

## 🚫 Applications to Remove

These applications should be removed from production:

| App                      | Reason                                      | Replacement                        |
|:-------------------------|:--------------------------------------------|:-----------------------------------|
| **grafana-ingress**      | Legacy standalone ingress                   | Consolidated middleware in Grafana |
| **prometheus-ingress**   | Legacy standalone ingress                   | Consolidated middleware in Prometheus |
| **stirling-pdf-ingress** | Legacy standalone ingress                   | Consolidated middleware in stirling-pdf |
| **it-tools-ingress**     | Legacy standalone ingress                   | Consolidated middleware in it-tools |

---

## Profile Definitions

### Resource Profiles (T-Shirt Sizing)

Based on **docs/reference/RESOURCE_STANDARDS.md**:

| Profile   | CPU Req  | CPU Lim  | Mem Req  | Mem Lim  | Use Case                                    |
|:----------|:---------|:---------|:---------|:---------|:--------------------------------------------|
| **Micro** | 10m      | 100m     | 64Mi     | 128Mi    | Sidecars, exporters, config syncs           |
| **Small** | 50m      | 500m     | 256Mi    | 512Mi    | Optimized apps (Go/Rust), static tools      |
| **Medium**| 200m     | 1000m    | 512Mi    | 1Gi      | Standard web apps (Python/Node), monitoring |
| **Large** | 1000m    | 2000m    | 2Gi      | 4Gi      | Databases, heavy apps (Jellyfin)            |
| **XLarge**| 2000m    | 4000m    | 4Gi      | 8Gi      | AI processing (Frigate), large indexers     |

### Priority Classes

Based on **docs/reference/RESOURCE_STANDARDS.md**:

| Priority           | Value  | Preemption | Use Case                           |
|:-------------------|:-------|:-----------|:-----------------------------------|
| **vixens-critical**| 100000 | Never      | Infrastructure core (Ingress, CSI) |
| **vixens-high**    | 50000  | Rare       | Vital services (HA, Monitoring)    |
| **vixens-medium**  | 10000  | Standard   | User-facing apps                   |
| **vixens-low**     | 0      | First      | Background tasks (downloads)       |

### Sync Waves

Based on **docs/reference/argocd-sync-waves.md**:

| Wave | Purpose                    | Examples                                    |
|:----:|:---------------------------|:--------------------------------------------|
| **-5**| CRDs                      | priority-classes, vixens-app-of-apps        |
| **-4**| Operators                 | cloudnative-pg, cert-manager, synology-csi  |
| **-3**| Secrets & Configuration   | InfisicalSecrets, cert-manager-secrets      |
| **-2**| Infrastructure            | cilium-lb, traefik, metrics-server          |
| **-1**| Shared Services           | postgresql-shared, redis-shared, mariadb    |
| **0** | Applications (default)    | All user-facing apps                        |

### Backup Profiles (Litestream)

Based on **docs/adr/014-litestream-backup-profiles-and-recovery-patterns.md**:

| Profile      | Snapshot Interval | Retention | Sync Interval | Use Case                                 |
|:-------------|:------------------|:----------|:--------------|:-----------------------------------------|
| **Critical** | 1h                | 14d       | 1s            | High-activity DBs (Home Assistant)       |
| **Standard** | 6h                | 7d        | 1s            | Most apps with SQLite                    |
| **Relaxed**  | 24h               | 3d        | 1s            | Config files, low-activity DBs           |
| **Ephemeral**| None              | None      | N/A           | Caches, disposable data (skip backup)    |
| **Mixed**    | Varies            | Varies    | 1s            | Multi-DB apps (Hydrus: 3 profiles)       |

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
