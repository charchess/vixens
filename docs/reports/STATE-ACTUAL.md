# Application State - Actual (Production Reality)

**Last Updated:** 2026-01-10
**Environment:** Production Cluster
**Data Source:** Live cluster state + GitOps manifests

---

## Production Application Inventory

| App                            | NS                         | CPU Req  | CPU Lim  | Mem Req  | Mem Lim  | VPA Target       | Profile | Priority      | Sync Wave | Backup Profile | Score |
|:-------------------------------|:---------------------------|:---------|:---------|:---------|:---------|:-----------------|:--------|:--------------|:---------:|:---------------|:-----:|
| **adguard-home**               | networking                 | 100m     | 500m     | 256Mi    | 512Mi    | 11m / 128Mi      | Small   | vixens-high   | 0         | Standard       | 100   |
| **alertmanager**               | monitoring                 | 100m     | 500m     | 256Mi    | 512Mi    | -                | Small   | vixens-high   | 0         | None           | 90    |
| **amule**                      | downloads                  | 50m      | N/A      | 128Mi    | 512Mi    | 11m / 128Mi      | Small   | vixens-low    | 0         | None           | 40    |
| **argocd**                     | argocd                     | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -2        | None           | 90    |
| **authentik**                  | auth                       | 200m     | 500m     | 512Mi    | 1024Mi   | 323m / 1.4Gi     | Medium  | vixens-critical| 0         | Standard       | 100   |
| **birdnet-go**                 | birdnet-go                 | 200m     | 1000m    | 256Mi    | 1Gi      | 23m / 215Mi      | Medium  | vixens-medium | 0         | Relaxed        | 65    |
| **booklore**                   | media                      | 15m      | 49m      | 334Mi    | 362Mi    | 163m / 2.6Gi     | Custom  | vixens-medium | 0         | None           | 40    |
| **cert-manager**               | cert-manager               | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -4        | None           | 85    |
| **cert-manager-config**        | cert-manager               | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -3        | None           | 85    |
| **cert-manager-secrets**       | cert-manager               | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -3        | None           | 85    |
| **cert-manager-webhook-gandi** | cert-manager               | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -4        | None           | 85    |
| **changedetection**            | tools                      | N/A      | N/A      | N/A      | N/A      | 11m / 121Mi      | Micro   | vixens-medium | 0         | None           | 70    |
| **cilium-lb**                  | kube-system                | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -2        | None           | 85    |
| **cloudnative-pg**             | cnpg-system                | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -4        | None           | 100   |
| **contacts**                   | contacts                   | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-medium | 0         | None           | 40    |
| **descheduler**                | kube-system                | 50m      | 200m     | 64Mi     | 128Mi    | No VPA           | Micro   | vixens-medium | 0         | None           | 40    |
| **docspell**                   | services                   | 500m     | 2000m    | 2048Mi   | 4096Mi   | No VPA           | Large   | vixens-medium | 0         | None           | 80    |
| **external-dns-gandi**         | networking                 | N/A      | N/A      | N/A      | N/A      | 11m / 128Mi      | Micro   | vixens-critical| 0         | None           | 90    |
| **external-dns-gandi-secrets** | networking                 | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -3        | None           | 85    |
| **external-dns-unifi**         | networking                 | N/A      | N/A      | N/A      | N/A      | 11m / 64Mi       | Micro   | vixens-critical| 0         | None           | 90    |
| **external-dns-unifi-secrets** | networking                 | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -3        | None           | 85    |
| **frigate**                    | media                      | 500m     | 2000m    | 1Gi      | 8Gi      | 2406m / 3.6Gi    | XLarge  | vixens-high   | 0         | Standard       | 90    |
| **gitops-revision-controller** | tools                      | 50m      | 200m     | 128Mi    | 512Mi    | 11m / 138Mi      | Small   | vixens-medium | 0         | None           | 40    |
| **gluetun**                    | services                   | N/A      | N/A      | N/A      | N/A      | 11m / 128Mi      | Micro   | vixens-medium | 0         | None           | 50    |
| **goldilocks**                 | monitoring                 | N/A      | N/A      | N/A      | N/A      | 23m / 128Mi      | Micro   | vixens-medium | 0         | None           | 40    |
| **grafana**                    | monitoring                 | 100m     | N/A      | 256Mi    | 512Mi    | 23m / 215Mi      | Small   | vixens-high   | 0         | None           | 40    |
| **grafana-ingress**            | monitoring                 | N/A      | N/A      | N/A      | N/A      | No VPA           | N/A     | N/A           | 0         | None           | 20    |
| **headlamp**                   | tools                      | 100m     | 200m     | 128Mi    | 256Mi    | 11m / 128Mi      | Small   | vixens-medium | 0         | None           | 40    |
| **homeassistant**              | homeassistant              | 300m     | 1000m    | 1024Mi   | 2048Mi   | 11m / 64Mi       | Medium  | vixens-high   | 0         | Critical       | 100   |
| **homepage**                   | tools                      | N/A      | N/A      | N/A      | N/A      | 11m / 215Mi      | Micro   | vixens-medium | 0         | None           | 40    |
| **hubble-ui**                  | monitoring                 | 100m     | 200m     | 128Mi    | 256Mi    | No VPA           | Small   | vixens-medium | 0         | None           | 40    |
| **hydrus-client**              | media                      | 34m      | 49m      | 2294Mi   | 3877Mi   | 63m / 1.9Gi      | Custom  | vixens-medium | 0         | Mixed          | 80    |
| **infisical-operator**         | infisical-operator-system  | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -4        | None           | 85    |
| **it-tools**                   | tools                      | 10m      | 100m     | 32Mi     | 128Mi    | 11m / 128Mi      | Micro   | vixens-medium | 0         | None           | 60    |
| **it-tools-ingress**           | tools                      | N/A      | N/A      | N/A      | N/A      | No VPA           | N/A     | N/A           | 0         | None           | 20    |
| **jellyfin**                   | media                      | 15m      | 15m      | 1567Mi   | 2062Mi   | 23m / 826Mi      | Custom  | vixens-medium | 0         | None           | 70    |
| **jellyseerr**                 | media                      | 50m      | 200m     | 128Mi    | 256Mi    | 78m / 422Mi      | Small   | vixens-medium | 0         | None           | 75    |
| **lazylibrarian**              | media                      | 15m      | 15m      | 259Mi    | 488Mi    | 23m / 175Mi      | Custom  | vixens-low    | 0         | None           | 40    |
| **lidarr**                     | media                      | 15m      | 15m      | 214Mi    | 214Mi    | 11m / 237Mi      | Custom  | vixens-medium | 0         | Standard       | 100   |
| **linkwarden**                 | tools                      | 100m     | 1000m    | 1Gi      | 2Gi      | 11m / 561Mi      | Medium  | vixens-medium | 0         | None           | 70    |
| **loki**                       | monitoring                 | 100m     | 500m     | 256Mi    | 1024Mi   | 23m / 260Mi      | Small   | vixens-high   | 0         | None           | 70    |
| **mail-gateway**               | mail-gateway               | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| 0         | None           | 40    |
| **mariadb-shared**             | databases                  | 200m     | 1000m    | 512Mi    | 1024Mi   | No VPA           | Medium  | vixens-critical| -1        | Standard       | 100   |
| **mealie**                     | mealie                     | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-medium | 0         | Standard       | 95    |
| **metrics-server**             | kube-system                | 100m     | 500m     | 200Mi    | 500Mi    | No VPA           | Small   | vixens-critical| -2        | None           | 85    |
| **mosquitto**                  | mosquitto                  | 50m      | 200m     | 64Mi     | 256Mi    | 11m / 128Mi      | Micro   | vixens-high   | 0         | None           | 85    |
| **music-assistant**            | media                      | 15m      | 15m      | 283Mi    | 283Mi    | 11m / 215Mi      | Custom  | vixens-medium | 0         | None           | 40    |
| **mylar**                      | media                      | 15m      | 15m      | 104Mi    | 104Mi    | 11m / 128Mi      | Custom  | vixens-medium | 0         | Standard       | 100   |
| **netbox**                     | tools                      | N/A      | N/A      | N/A      | N/A      | 11m / 641Mi      | Unknown | vixens-medium | 0         | None           | 70    |
| **netvisor**                   | networking                 | N/A      | N/A      | N/A      | N/A      | 11m / 128Mi      | Micro   | vixens-medium | 0         | None           | 40    |
| **nfs-storage**                | media-stack                | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -2        | None           | 40    |
| **postgresql-shared**          | databases                  | 100m     | 500m     | 256Mi    | 512Mi    | No VPA           | Small   | vixens-critical| -1        | Standard       | 90    |
| **priority-classes**           | kube-system                | N/A      | N/A      | N/A      | N/A      | No VPA           | N/A     | N/A           | -5        | None           | 85    |
| **prometheus**                 | monitoring                 | 300m     | 100m     | 1Gi      | 2Gi      | 11m / 128Mi      | Medium  | vixens-high   | 0         | None           | 70    |
| **prometheus-ingress**         | monitoring                 | N/A      | N/A      | N/A      | N/A      | No VPA           | N/A     | N/A           | 0         | None           | 20    |
| **promtail**                   | monitoring                 | 50m      | 100m     | 100Mi    | 256Mi    | 49m / 128Mi      | Micro   | vixens-medium | 0         | None           | 40    |
| **prowlarr**                   | media                      | 15m      | 15m      | 155Mi    | 174Mi    | 11m / 194Mi      | Custom  | vixens-medium | 0         | Standard       | 100   |
| **pyload**                     | downloads                  | 50m      | N/A      | 128Mi    | 512Mi    | 11m / 128Mi      | Small   | vixens-low    | 0         | None           | 40    |
| **qbittorrent**                | downloads                  | 50m      | N/A      | 256Mi    | 1Gi      | 11m / 128Mi      | Small   | vixens-low    | 0         | None           | 40    |
| **radarr**                     | media                      | 22m      | 35m      | 334Mi    | 362Mi    | 49m / 561Mi      | Custom  | vixens-medium | 0         | Standard       | 100   |
| **redis-shared**               | databases                  | N/A      | N/A      | N/A      | N/A      | 23m / 128Mi      | Micro   | vixens-critical| -1        | None           | 90    |
| **reloader**                   | tools                      | 10m      | 100m     | 128Mi    | 256Mi    | 11m / 128Mi      | Micro   | vixens-medium | 0         | None           | 40    |
| **renovate**                   | tools                      | N/A      | N/A      | N/A      | N/A      | 977m / 991Mi     | Unknown | vixens-medium | 0         | None           | 40    |
| **sabnzbd**                    | media                      | 50m      | 500m     | 256Mi    | 1Gi      | 23m / 237Mi      | Small   | vixens-low    | 0         | Standard       | 100   |
| **sonarr**                     | media                      | 15m      | 15m      | 236Mi    | 236Mi    | 23m / 260Mi      | Custom  | vixens-medium | 0         | Standard       | 100   |
| **stirling-pdf**               | tools                      | 100m     | 1000m    | 256Mi    | 1Gi      | 23m / 363Mi      | Medium  | vixens-medium | 0         | None           | 60    |
| **stirling-pdf-ingress**       | tools                      | N/A      | N/A      | N/A      | N/A      | No VPA           | N/A     | N/A           | 0         | None           | 20    |
| **synology-csi**               | synology-csi               | N/A      | N/A      | N/A      | N/A      | 11m / 32Mi       | Unknown | vixens-critical| -4        | None           | 90    |
| **synology-csi-secrets**       | synology-csi               | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -3        | None           | 85    |
| **traefik**                    | traefik                    | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-critical| -2        | None           | 90    |
| **traefik-dashboard**          | traefik                    | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-medium | 0         | None           | 85    |
| **vaultwarden**                | services                   | 50m      | 500m     | 256Mi    | 512Mi    | 11m / 128Mi      | Small   | vixens-high   | 0         | Standard       | 100   |
| **vixens-app-of-apps**         | argocd                     | N/A      | N/A      | N/A      | N/A      | No VPA           | N/A     | vixens-critical| -5        | None           | 85    |
| **vpa**                        | vpa                        | 50m      | 200m     | 100Mi    | 500Mi    | No VPA           | Small   | vixens-medium | -2        | None           | 40    |
| **whisparr**                   | media                      | 15m      | 15m      | 120Mi    | 120Mi    | 11m / 138Mi      | Custom  | vixens-medium | 0         | Standard       | 100   |
| **whoami**                     | whoami                     | N/A      | N/A      | N/A      | N/A      | No VPA           | Unknown | vixens-low    | 0         | None           | 40    |

---

## Legend

### Resource Profile
- **Micro**: 10m/100m CPU, 64Mi/128Mi RAM (sidecars, exporters)
- **Small**: 50m/500m CPU, 256Mi/512Mi RAM (optimized apps)
- **Medium**: 200m/1000m CPU, 512Mi/1Gi RAM (standard web apps)
- **Large**: 1000m/2000m CPU, 2Gi/4Gi RAM (databases, heavy apps)
- **XLarge**: 2000m/4000m CPU, 4Gi/8Gi RAM (AI processing, large indexers)
- **Custom**: Non-standard profile (needs review)
- **Unknown**: No limits defined or VPA data unavailable

### Priority Classes
- **vixens-critical** (100000): Infrastructure core (never evicted)
- **vixens-high** (50000): Vital services (high availability)
- **vixens-medium** (10000): Standard applications
- **vixens-low** (0): Background tasks (sacrificable)

### Sync Waves
- **-5**: CRDs
- **-4**: Operators
- **-3**: Secrets & Configuration
- **-2**: Infrastructure
- **-1**: Shared Services
- **0**: Applications (default)

### Backup Profiles (Litestream)
- **Critical**: 1h snapshots, 14d retention, 1s sync
- **Standard**: 6h snapshots, 7d retention, 1s sync
- **Relaxed**: 24h snapshots, 3d retention, 1s sync
- **Ephemeral**: Skip backup (caches, disposable data)
- **Mixed**: Multiple databases with different profiles
- **None**: No backup configured

### Score (Conformity)
Based on APPLICATION_SCORING_MODEL.md:
- **90-100**: Elite/Gold (production-ready)
- **70-89**: Valid (acceptable quality)
- **40-69**: To Consolidate (needs improvement)
- **0-39**: Legacy (critical gaps)

---

## Critical Issues Detected

### 🔴 OOM Risk (6 applications)
Resource limits below VPA target recommendations:
- **authentik**: 1024Mi limit vs 1.4Gi target
- **jellyseerr**: 256Mi limit vs 422Mi target
- **lidarr**: 214Mi limit vs 237Mi target
- **mylar**: 104Mi limit vs 128Mi target
- **prowlarr**: 174Mi limit vs 194Mi target
- **whisparr**: 120Mi limit vs 138Mi target

**Action Required**: Increase memory limits to match VPA targets

### 🔴 CPU Throttled (6 applications)
CPU requests significantly below VPA target:
- **booklore**: 15m request vs 163m target
- **frigate**: 500m request vs 2406m target
- **hydrus-client**: 34m request vs 63m target
- **jellyfin**: 15m request vs 23m target
- **lazylibrarian**: 15m request vs 23m target
- **radarr**: 22m request vs 49m target

**Action Required**: Increase CPU requests to reduce throttling

### ⚪ Missing Resource Limits (28 applications)
Applications with N/A for requests/limits (affected by 2026-01-07 GitOps repair):
- Infrastructure: argocd, cert-manager, cilium-lb, synology-csi, traefik, metrics-server
- Databases: redis-shared, postgresql-shared
- Monitoring: grafana, prometheus, loki, vpa
- Others: changedetection, contacts, external-dns, gluetun, goldilocks, homepage, mail-gateway, mealie, netbox, netvisor, renovate, whoami

**Action Required**: Recreate resource patches with validated Kustomize selectors

---

**Data Sources:**
- Production cluster: kubectl get pods -A with resource inspection
- VPA recommendations: Goldilocks analysis
- GitOps manifests: apps/*/overlays/prod/
- Reference standards: docs/reference/RESOURCE_STANDARDS.md
