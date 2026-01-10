# Application State - Desired (Reference Standard)

**Last Updated:** 2026-01-10
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
| **adguard-home**               | networking                 | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-high   | 0         | Standard       | 100          |
| **alertmanager**               | monitoring                 | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-high   | 0         | Relaxed        | 100          |
| **amule**                      | downloads                  | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-low    | 0         | None           | 85           |
| **argocd**                     | argocd                     | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-critical| -2        | Relaxed        | 100          |
| **authentik**                  | auth                       | 500m     | 1000m    | 1536Mi   | 2Gi      | Medium  | vixens-critical| 0         | Standard       | 100          |
| **birdnet-go**                 | birdnet-go                 | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Relaxed        | 85           |
| **booklore**                   | media                      | 200m     | 1000m    | 512Mi    | 3Gi      | Medium  | vixens-medium | 0         | Relaxed        | 85           |
| **cert-manager**               | cert-manager               | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **cert-manager-config**        | cert-manager               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **cert-manager-secrets**       | cert-manager               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **cert-manager-webhook-gandi** | cert-manager               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -4        | None           | 100          |
| **changedetection**            | tools                      | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | Relaxed        | 85           |
| **cilium-lb**                  | kube-system                | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| -2        | None           | 100          |
| **cloudnative-pg**             | cnpg-system                | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-critical| -4        | None           | 100          |
| **contacts**                   | contacts                   | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 85           |
| **descheduler**                | kube-system                | 50m      | 200m     | 128Mi    | 256Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **docspell**                   | services                   | 500m     | 2000m    | 2048Mi   | 4096Mi   | Large   | vixens-medium | 0         | Standard       | 100          |
| **external-dns-gandi**         | networking                 | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| 0         | Relaxed        | 100          |
| **external-dns-gandi-secrets** | networking                 | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **external-dns-unifi**         | networking                 | 50m      | 500m     | 64Mi     | 128Mi    | Small   | vixens-critical| 0         | Relaxed        | 100          |
| **external-dns-unifi-secrets** | networking                 | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **frigate**                    | media                      | 2500m    | 4000m    | 4Gi      | 8Gi      | XLarge  | vixens-high   | 0         | Standard       | 100          |
| **gitops-revision-controller** | tools                      | 50m      | 200m     | 128Mi    | 512Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **gluetun**                    | services                   | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **goldilocks**                 | monitoring                 | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **grafana**                    | monitoring                 | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-high   | 0         | Relaxed        | 100          |
| **headlamp**                   | tools                      | 100m     | 200m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **homeassistant**              | homeassistant              | 300m     | 1000m    | 1024Mi   | 2048Mi   | Medium  | vixens-high   | 0         | Critical       | 100          |
| **homepage**                   | tools                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **hubble-ui**                  | monitoring                 | 100m     | 200m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **hydrus-client**              | media                      | 100m     | 1000m    | 2Gi      | 4Gi      | Large   | vixens-medium | 0         | Mixed          | 100          |
| **infisical-operator**         | infisical-operator-system  | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **it-tools**                   | tools                      | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **jellyfin**                   | media                      | 100m     | 2000m    | 1Gi      | 4Gi      | Large   | vixens-medium | 0         | Relaxed        | 100          |
| **jellyseerr**                 | media                      | 100m     | 500m     | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **lazylibrarian**              | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-low    | 0         | Standard       | 85           |
| **lidarr**                     | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **linkwarden**                 | tools                      | 100m     | 1000m    | 1Gi      | 2Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **loki**                       | monitoring                 | 100m     | 500m     | 512Mi    | 1024Mi   | Small   | vixens-high   | 0         | Relaxed        | 100          |
| **mail-gateway**               | mail-gateway               | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-critical| 0         | Relaxed        | 100          |
| **mariadb-shared**             | databases                  | 200m     | 1000m    | 512Mi    | 1024Mi   | Medium  | vixens-critical| -1        | Standard       | 100          |
| **mealie**                     | mealie                     | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **metrics-server**             | kube-system                | 100m     | 500m     | 200Mi    | 500Mi    | Small   | vixens-critical| -2        | None           | 100          |
| **mosquitto**                  | mosquitto                  | 50m      | 200m     | 128Mi    | 256Mi    | Micro   | vixens-high   | 0         | Relaxed        | 100          |
| **music-assistant**            | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 85           |
| **mylar**                      | media                      | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **netbox**                     | tools                      | 100m     | 1000m    | 1Gi      | 2Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **netvisor**                   | networking                 | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-medium | 0         | None           | 85           |
| **nfs-storage**                | media-stack                | 50m      | 200m     | 128Mi    | 256Mi    | Micro   | vixens-critical| -2        | None           | 100          |
| **postgresql-shared**          | databases                  | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-critical| -1        | Standard       | 100          |
| **priority-classes**           | kube-system                | N/A      | N/A      | N/A      | N/A      | N/A     | N/A           | -5        | None           | 100          |
| **prometheus**                 | monitoring                 | 500m     | 2000m    | 1Gi      | 2Gi      | Medium  | vixens-high   | 0         | Relaxed        | 100          |
| **promtail**                   | monitoring                 | 50m      | 100m     | 128Mi    | 256Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **prowlarr**                   | media                      | 50m      | 500m     | 200Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **pyload**                     | downloads                  | 50m      | 500m     | 128Mi    | 512Mi    | Small   | vixens-low    | 0         | None           | 85           |
| **qbittorrent**                | downloads                  | 50m      | 500m     | 256Mi    | 1Gi      | Small   | vixens-low    | 0         | None           | 85           |
| **radarr**                     | media                      | 100m     | 500m     | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | Standard       | 100          |
| **redis-shared**               | databases                  | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| -1        | Relaxed        | 100          |
| **reloader**                   | tools                      | 10m      | 100m     | 128Mi    | 256Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **renovate**                   | tools                      | 1000m    | 2000m    | 1Gi      | 2Gi      | Large   | vixens-medium | 0         | None           | 85           |
| **sabnzbd**                    | media                      | 100m     | 500m     | 256Mi    | 1Gi      | Small   | vixens-low    | 0         | Standard       | 100          |
| **sonarr**                     | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
| **stirling-pdf**               | tools                      | 100m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-medium | 0         | None           | 85           |
| **synology-csi**               | synology-csi               | 50m      | 500m     | 128Mi    | 256Mi    | Small   | vixens-critical| -4        | None           | 100          |
| **synology-csi-secrets**       | synology-csi               | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -3        | None           | 100          |
| **traefik**                    | traefik                    | 200m     | 1000m    | 512Mi    | 1Gi      | Medium  | vixens-critical| -2        | Relaxed        | 100          |
| **traefik-dashboard**          | traefik                    | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-medium | 0         | None           | 85           |
| **vaultwarden**                | services                   | 100m     | 500m     | 256Mi    | 512Mi    | Small   | vixens-high   | 0         | Standard       | 100          |
| **vixens-app-of-apps**         | argocd                     | 10m      | 100m     | 64Mi     | 128Mi    | Micro   | vixens-critical| -5        | None           | 100          |
| **vpa**                        | vpa                        | 50m      | 200m     | 200Mi    | 500Mi    | Small   | vixens-medium | -2        | None           | 85           |
| **whisparr**                   | media                      | 50m      | 500m     | 256Mi    | 512Mi    | Small   | vixens-medium | 0         | Standard       | 100          |
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

### Planned Changes

**Resource Optimization (Priority 1)**:
- Increase frigate CPU request to 2500m (currently throttled)
- Increase authentik memory limit to 2Gi (OOM risk)
- Restore QoS for 28 apps affected by 2026-01-07 GitOps repair

**Backup Consolidation (Priority 2)**:
- Enable Prometheus metrics for all Litestream configs (vixens-wfxk)
- Review metrics after 1 week (vixens-yx42)
- Configure MinIO lifecycle policy (vixens-s5ch)

**Legacy Cleanup (Priority 3)**:
- Remove 4 standalone ingress resources
- Migrate to consolidated middleware pattern

---

## Maintenance Instructions

### When Making Decisions

1. **Update this document FIRST** with the new desired state
2. **Apply changes** to GitOps manifests (`apps/*/overlays/prod/`)
3. **Verify in production** using kubectl or ArgoCD UI
4. **Update STATE-ACTUAL.md** to reflect reality
5. **Update STATUS.md** conformity scores

### Editing Guidelines

- **Add date** to Decision History when making changes
- **Reference tasks** (Beads IDs) when relevant
- **Explain rationale** for non-obvious decisions
- **Keep alphabetical** order in main table
- **Validate** against reference standards before committing

### Automation Potential

Scripts can be developed to:
- Generate Kustomize patches from this table
- Compare STATE-ACTUAL vs STATE-DESIRED (conformity scoring)
- Auto-update STATUS.md from other reports
- Validate consistency with reference docs

---

**References:**
- docs/reference/RESOURCE_STANDARDS.md
- docs/reference/APPLICATION_SCORING_MODEL.md
- docs/reference/argocd-sync-waves.md
- docs/adr/014-litestream-backup-profiles-and-recovery-patterns.md
