# Application Status Dashboard

**Last Updated:** 2026-01-10
**Cluster Environments:** dev, prod

---

## Overview

| Category              | Dev     | Prod    | Total   |
|:----------------------|:--------|:--------|:--------|
| **🟢 OK**             | -       | 45      | 45      |
| **🔴 NOK**            | -       | 12      | 12      |
| **🟡 Hibernate**      | -       | 0       | 0       |
| **⚫ Absent**          | -       | 18      | 18      |
| **Total**             | -       | 75      | 75      |

---

## Application Status Matrix

| Application                    | Dev       | Prod      | Last Prod Change | Conformity Score | Notes                           |
|:-------------------------------|:---------:|:---------:|:-----------------|:----------------:|:--------------------------------|
| **adguard-home**               | 🟢        | 🟢        | 2026-01-08       | 100/100          | Elite - Full compliance         |
| **alertmanager**               | 🟢        | 🟢        | 2026-01-07       | 90/100           | Gold - No backup                |
| **amule**                      | 🟢        | 🟢        | 2025-12-30       | 40/100           | Low - Needs consolidation       |
| **argocd**                     | 🟢        | 🟢        | 2026-01-07       | 90/100           | Gold - QoS degraded             |
| **authentik**                  | 🟢        | 🟢        | 2026-01-07       | 100/100          | Elite - Fixed ingress           |
| **birdnet-go**                 | 🟢        | 🟢        | 2025-12-20       | 65/100           | Low - GitOps issues             |
| **booklore**                   | 🟢        | 🔴        | 2025-12-15       | 40/100           | CPU throttled                   |
| **cert-manager**               | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure - No limits      |
| **cert-manager-config**        | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure                  |
| **cert-manager-secrets**       | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure                  |
| **cert-manager-webhook-gandi** | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure                  |
| **changedetection**            | 🟢        | 🟢        | 2025-12-18       | 70/100           | Valid - No backup               |
| **cilium-lb**                  | 🟢        | 🟢        | 2026-01-07       | 85/100           | Infrastructure                  |
| **cloudnative-pg**             | 🟢        | 🟢        | 2026-01-05       | 100/100          | Elite - Operator                |
| **contacts**                   | 🟢        | 🟢        | 2025-11-20       | 40/100           | Low - Needs consolidation       |
| **descheduler**                | 🟢        | 🟢        | 2025-12-28       | 40/100           | Infrastructure                  |
| **docspell**                   | 🟢        | 🟢        | 2025-12-22       | 80/100           | Valid - No backup               |
| **external-dns**               | 🟢        | 🟢        | 2026-01-08       | 90/100           | Gold - No backup                |
| **frigate**                    | 🟢        | 🔴        | 2025-12-30       | 90/100           | CPU throttled, QoS lost         |
| **gitops-revision-controller** | 🟢        | 🟢        | 2025-12-10       | 40/100           | Infrastructure                  |
| **gluetun**                    | 🟢        | 🟢        | 2025-11-25       | 50/100           | No limits                       |
| **goldilocks**                 | 🟢        | 🟢        | 2025-12-29       | 40/100           | Monitoring                      |
| **grafana**                    | 🟢        | 🟢        | 2026-01-07       | 40/100           | QoS lost                        |
| **grafana-ingress**            | 🟢        | 🟢        | 2025-11-15       | 20/100           | Legacy - To remove              |
| **headlamp**                   | 🟢        | 🟢        | 2025-12-05       | 40/100           | Infrastructure                  |
| **homeassistant**              | 🟢        | 🟢        | 2026-01-08       | 100/100          | Elite - Full compliance         |
| **homepage**                   | 🟢        | 🟢        | 2025-11-18       | 40/100           | Low - Needs consolidation       |
| **hubble-ui**                  | 🟢        | 🟢        | 2025-12-02       | 40/100           | Monitoring                      |
| **hydrus-client**              | 🟢        | 🔴        | 2025-12-28       | 80/100           | CPU throttled, needs review     |
| **infisical-operator**         | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure - Operator       |
| **it-tools**                   | 🟢        | 🔴        | 2025-11-22       | 60/100           | Resource warning                |
| **it-tools-ingress**           | 🟢        | 🟢        | 2025-11-15       | 20/100           | Legacy - To remove              |
| **jellyfin**                   | 🟢        | 🔴        | 2025-12-20       | 70/100           | CPU throttled                   |
| **jellyseerr**                 | 🟢        | 🔴        | 2025-12-18       | 75/100           | OOM risk                        |
| **lazylibrarian**              | 🟢        | 🔴        | 2025-12-10       | 40/100           | CPU throttled                   |
| **lidarr**                     | 🟢        | 🔴        | 2025-12-25       | 100/100          | OOM risk despite Elite score    |
| **linkwarden**                 | 🟢        | 🟢        | 2025-12-15       | 70/100           | Valid - No backup               |
| **loki**                       | 🟢        | 🟢        | 2026-01-07       | 70/100           | QoS lost                        |
| **mail-gateway**               | 🟢        | 🟢        | 2025-11-28       | 40/100           | Infrastructure                  |
| **mariadb-shared**             | 🟢        | 🟢        | 2026-01-07       | 100/100          | Elite - Fixed duplication       |
| **mealie**                     | 🟢        | 🟢        | 2025-12-30       | 95/100           | Gold - Minor backup gap         |
| **metrics-server**             | 🟢        | 🟢        | 2026-01-07       | 85/100           | Infrastructure                  |
| **mosquitto**                  | 🟢        | 🟢        | 2025-11-25       | 85/100           | Infrastructure - MQTT           |
| **music-assistant**            | 🟢        | 🟢        | 2025-12-08       | 40/100           | Low - Needs consolidation       |
| **mylar**                      | 🟢        | 🔴        | 2025-12-22       | 100/100          | OOM risk despite Elite score    |
| **netbox**                     | 🟢        | 🟢        | 2025-12-12       | 70/100           | Valid - No limits               |
| **netvisor**                   | 🟢        | 🟢        | 2026-01-08       | 40/100           | Low - Needs consolidation       |
| **nfs-storage**                | 🟢        | 🟢        | 2025-11-22       | 40/100           | Infrastructure                  |
| **postgresql-shared**          | 🟢        | 🟢        | 2026-01-07       | 90/100           | Gold - QoS lost                 |
| **priority-classes**           | 🟢        | 🟢        | 2025-12-15       | 85/100           | Infrastructure                  |
| **prometheus**                 | 🟢        | 🟢        | 2026-01-08       | 70/100           | QoS lost                        |
| **prometheus-ingress**         | 🟢        | 🟢        | 2025-11-15       | 20/100           | Legacy - To remove              |
| **promtail**                   | 🟢        | 🟢        | 2025-12-28       | 40/100           | Monitoring                      |
| **prowlarr**                   | 🟢        | 🔴        | 2025-12-25       | 100/100          | OOM risk despite Elite score    |
| **pyload**                     | 🟢        | 🟢        | 2025-12-05       | 40/100           | Low - Needs consolidation       |
| **qbittorrent**                | 🟢        | 🟢        | 2025-12-08       | 40/100           | Low - Needs consolidation       |
| **radarr**                     | 🟢        | 🔴        | 2025-12-28       | 100/100          | CPU throttled                   |
| **redis-shared**               | 🟢        | 🟢        | 2026-01-07       | 90/100           | Gold - QoS lost                 |
| **reloader**                   | 🟢        | 🟢        | 2025-12-10       | 40/100           | Infrastructure                  |
| **renovate**                   | 🟢        | 🟢        | 2025-12-18       | 40/100           | Infrastructure                  |
| **sabnzbd**                    | 🟢        | 🟢        | 2025-12-30       | 100/100          | Elite - Full compliance         |
| **sonarr**                     | 🟢        | 🟢        | 2025-12-30       | 100/100          | Elite - Full compliance         |
| **stirling-pdf**               | 🟢        | 🟢        | 2025-11-20       | 60/100           | Low - Needs consolidation       |
| **stirling-pdf-ingress**       | 🟢        | 🟢        | 2025-11-15       | 20/100           | Legacy - To remove              |
| **synology-csi**               | 🟢        | 🟢        | 2026-01-07       | 90/100           | Gold - Infrastructure           |
| **synology-csi-secrets**       | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure                  |
| **traefik**                    | 🟢        | 🟢        | 2026-01-08       | 90/100           | Gold - Infrastructure           |
| **traefik-dashboard**          | 🟢        | 🟢        | 2025-11-28       | 85/100           | Infrastructure                  |
| **vaultwarden**                | 🟢        | 🟢        | 2026-01-07       | 100/100          | Elite - Fixed health check      |
| **vixens-app-of-apps**         | 🟢        | 🟢        | 2026-01-05       | 85/100           | Infrastructure - ArgoCD root    |
| **vpa**                        | 🟢        | 🟢        | 2026-01-07       | 40/100           | Infrastructure                  |
| **whisparr**                   | 🟢        | 🔴        | 2025-12-25       | 100/100          | OOM risk despite Elite score    |
| **whoami**                     | 🟢        | 🟢        | 2025-11-15       | 40/100           | Test app                        |

---

## Status Legend

| Status      | Symbol | Description                                          |
|:------------|:-------|:-----------------------------------------------------|
| **OK**      | 🟢     | Application running and healthy                      |
| **NOK**     | 🔴     | Application degraded (OOM risk, CPU throttled, etc.) |
| **Hibernate**| 🟡    | Application intentionally stopped                    |
| **Absent**  | ⚫     | Application not deployed in this environment         |

---

## Conformity Score Breakdown

| Score Range | Status              | Count | % of Total |
|:-----------:|:--------------------|:-----:|:----------:|
| **90-100**  | 🏆 Elite / 🥇 Gold  | 28    | 37%        |
| **70-89**   | ✅ Valid            | 8     | 11%        |
| **40-69**   | ⚠️ To Consolidate   | 35    | 47%        |
| **0-39**    | ❌ Legacy           | 4     | 5%         |

---

## Priority Actions

### 🔴 Critical Issues (12 applications)

Applications with NOK status requiring immediate attention:

1. **booklore** - CPU throttled
2. **frigate** - CPU throttled + QoS lost
3. **hydrus-client** - CPU throttled
4. **it-tools** - Resource warning
5. **jellyfin** - CPU throttled
6. **jellyseerr** - OOM risk
7. **lazylibrarian** - CPU throttled
8. **lidarr** - OOM risk
9. **mylar** - OOM risk
10. **prowlarr** - OOM risk
11. **radarr** - CPU throttled
12. **whisparr** - OOM risk

### ⚠️ QoS Recovery (28 applications)

Applications affected by 2026-01-07 GitOps repair (resources-patch.yaml removal):
- ArgoCD, Traefik, Synology-CSI, Redis-shared, PostgreSQL-shared, Frigate
- Grafana, Loki, Prometheus, VPA, Metrics-Server, and 17 others

**Action Required:** Recreate resource patches with validated Kustomize selectors

### 🗑️ Legacy Cleanup (4 applications)

Standalone ingress resources to be removed:
- grafana-ingress
- prometheus-ingress
- stirling-pdf-ingress
- it-tools-ingress

**Action Required:** Migrate to consolidated middleware pattern

---

**Data Sources:**
- Production cluster state (kubectl)
- APP_AUDIT.md (scoring model)
- ULTIMATE-AUDIT.md (resource analysis)
- Git history (last change dates)
