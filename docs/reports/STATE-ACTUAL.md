# 📊 État Réel du Cluster - 2026-02-08 13:38:45

## 🖥️ Node Summary

| Node Name | Role          | CPU Cap | RAM Cap    | OS              | Kernel       |
| --------- | ------------- | ------- | ---------- | --------------- | ------------ |
| peach     | worker        | 8       | 8096984Ki  | Talos (v1.12.2) | 6.18.5-talos |
| pearl     | worker        | 4       | 8002644Ki  | Talos (v1.12.2) | 6.18.5-talos |
| phoebe    | control-plane | 4       | 7969528Ki  | Talos (v1.12.2) | 6.18.5-talos |
| poison    | control-plane | 4       | 16094408Ki | Talos (v1.12.2) | 6.18.5-talos |
| powder    | control-plane | 4       | 16128584Ki | Talos (v1.12.2) | 6.18.5-talos |

## 📦 Application Details

| App                                          | NS                        | CPU Req | CPU Lim | Mem Req | Mem Lim | VPA Target   | Priority          | Wave | Backup | QoS        | Score |
| -------------------------------------------- | ------------------------- | ------- | ------- | ------- | ------- | ------------ | ----------------- | ---- | ------ | ---------- | ----- |
| **argocd**                                   | argocd                    | N/A     | N/A     | N/A     | N/A     | None         | N/A               | 0    | None   | BestEffort | 0     |
| **authentik**                                | auth                      | 500m    | 1       | 1Gi     | 2Gi     | None         | vixens-critical   | 0    | None   | Burstable  | 55    |
| **birdnet-go**                               | birdnet-go                | 200m    | 1       | 256Mi   | 1Gi     | 35m / 214Mi  | N/A               | 0    | None   | Burstable  | 35    |
| **cert-manager**                             | cert-manager              | 100m    | 100m    | 128Mi   | 128Mi   | None         | vixens-high       | 0    | None   | Guaranteed | 80    |
| **cert-manager-webhook-gandi**               | cert-manager              | 50m     | 50m     | 64Mi    | 64Mi    | None         | N/A               | 0    | None   | Guaranteed | 60    |
| **cloudnative-pg**                           | cnpg-system               | 100m    | 500m    | 128Mi   | 512Mi   | None         | homelab-critical  | 0    | None   | Burstable  | 55    |
| **mariadb-shared**                           | databases                 | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **postgresql**                               | databases                 | 500m    | 500m    | 512Mi   | 512Mi   | None         | vixens-critical   | 0    | None   | Guaranteed | 80    |
| **redis-shared**                             | databases                 | N/A     | N/A     | N/A     | N/A     | 23m / 128Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **pyload**                                   | downloads                 | 50m     | N/A     | 128Mi   | 512Mi   | 11m / 128Mi  | N/A               | 0    | None   | Burstable  | 15    |
| **firefly-iii**                              | finance                   | 200m    | 200m    | 512Mi   | 512Mi   | 23m / 64Mi   | vixens-medium     | 0    | None   | Guaranteed | 80    |
| **firefly-iii-importer**                     | finance                   | 10m     | 500m    | 128Mi   | 512Mi   | 11m / 214Mi  | vixens-low        | 0    | None   | Burstable  | 55    |
| **homeassistant**                            | homeassistant             | 500m    | 2       | 2Gi     | 4Gi     | 247m / 104Mi | vixens-high       | 0    | Active | Burstable  | 75    |
| **secrets-operator**                         | infisical-operator-system | 10m     | 500m    | 64Mi    | 256Mi   | None         | N/A               | 0    | None   | Burstable  | 35    |
| **kyverno**                                  | kyverno                   | 100m    | 200m    | 128Mi   | 384Mi   | None         | N/A               | 0    | None   | Burstable  | 35    |
| **mealie**                                   | mealie                    | 100m    | 500m    | 256Mi   | 512Mi   | 23m / 42Mi   | N/A               | 0    | Active | Burstable  | 55    |
| **amule**                                    | media                     | 50m     | N/A     | 128Mi   | 512Mi   | 11m / 128Mi  | N/A               | 0    | None   | Burstable  | 15    |
| **booklore**                                 | media                     | 10m     | 100m    | 64Mi    | 128Mi   | 11m / 2553Mi | N/A               | 0    | None   | Burstable  | 35    |
| **frigate**                                  | media                     | 500m    | 8       | 4Gi     | 8Gi     | 23m / 60Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **hydrus-client**                            | media                     | 1       | 1       | 2Gi     | 2Gi     | 11m / 599Mi  | vixens-medium     | 0    | Active | Burstable  | 75    |
| **jellyfin**                                 | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 683Mi  | homelab-important | 0    | None   | BestEffort | 20    |
| **jellyseerr**                               | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 422Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **lazylibrarian**                            | media                     | 10m     | 100m    | 64Mi    | 128Mi   | 11m / 42Mi   | N/A               | 0    | Active | Burstable  | 55    |
| **lidarr**                                   | media                     | 100m    | 1       | 256Mi   | 1Gi     | 78m / 74Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **music-assistant**                          | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 488Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **mylar**                                    | media                     | 100m    | 500m    | 256Mi   | 512Mi   | 11m / 42Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **prowlarr**                                 | media                     | 100m    | 500m    | 256Mi   | 512Mi   | 23m / 42Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **qbittorrent**                              | media                     | 50m     | N/A     | 256Mi   | 1Gi     | 11m / 128Mi  | N/A               | 0    | None   | Burstable  | 15    |
| **radarr**                                   | media                     | 100m    | 1       | 256Mi   | 1Gi     | 126m / 47Mi  | vixens-medium     | 0    | Active | Burstable  | 75    |
| **sabnzbd**                                  | media                     | 100m    | 1       | 256Mi   | 1Gi     | 11m / 89Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **sonarr**                                   | media                     | 100m    | 1       | 256Mi   | 1Gi     | 93m / 47Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **whisparr**                                 | media                     | 100m    | 1       | 256Mi   | 1Gi     | 11m / 89Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **alertmanager**                             | monitoring                | 200m    | 200m    | 512Mi   | 512Mi   | None         | vixens-critical   | 0    | None   | Guaranteed | 80    |
| **goldilocks**                               | monitoring                | 25m     | N/A     | 256Mi   | N/A     | None         | N/A               | 0    | None   | Burstable  | 15    |
| **grafana**                                  | monitoring                | 200m    | 200m    | 512Mi   | 512Mi   | 11m / 259Mi  | vixens-critical   | 0    | None   | Burstable  | 55    |
| **kube-state-metrics**                       | monitoring                | 200m    | 200m    | 256Mi   | 256Mi   | None         | vixens-critical   | 0    | None   | Guaranteed | 80    |
| **loki**                                     | monitoring                | 500m    | 500m    | 1Gi     | 1Gi     | 35m / 599Mi  | vixens-critical   | 0    | None   | Guaranteed | 80    |
| **prometheus**                               | monitoring                | 1       | 1       | 2Gi     | 2Gi     | None         | vixens-critical   | 0    | None   | Burstable  | 55    |
| **prometheus-node-exporter**                 | monitoring                | N/A     | N/A     | N/A     | N/A     | None         | N/A               | 0    | None   | BestEffort | 0     |
| **promtail**                                 | monitoring                | 50m     | 200m    | 256Mi   | 256Mi   | 49m / 155Mi  | vixens-critical   | 0    | None   | Burstable  | 55    |
| **mosquitto**                                | mosquitto                 | 50m     | 200m    | 64Mi    | 256Mi   | 23m / 89Mi   | N/A               | 0    | None   | Burstable  | 35    |
| **adguard-home**                             | networking                | 200m    | 200m    | 512Mi   | 512Mi   | 93m / 104Mi  | vixens-high       | 0    | Active | Guaranteed | 100   |
| **external-dns**                             | networking                | 20m     | 100m    | 64Mi    | 128Mi   | None         | vixens-high       | 0    | None   | Burstable  | 55    |
| **netbird-dashboard**                        | networking                | 100m    | 100m    | 256Mi   | 256Mi   | 11m / 128Mi  | vixens-high       | 0    | None   | Guaranteed | 80    |
| **netbird-management**                       | networking                | 200m    | 200m    | 512Mi   | 512Mi   | 11m / 128Mi  | vixens-high       | 0    | None   | Guaranteed | 80    |
| **netbird-relay**                            | networking                | 100m    | 100m    | 256Mi   | 256Mi   | 11m / 128Mi  | vixens-high       | 0    | None   | Guaranteed | 80    |
| **netbird-signal**                           | networking                | 100m    | 100m    | 256Mi   | 256Mi   | 11m / 128Mi  | vixens-high       | 0    | None   | Guaranteed | 80    |
| **netvisor**                                 | networking                | N/A     | N/A     | N/A     | N/A     | None         | N/A               | 0    | None   | BestEffort | 0     |
| **netvisor-daemon**                          | networking                | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **policy-reporter**                          | policy-reporter           | 50m     | 200m    | 128Mi   | 256Mi   | None         | N/A               | 0    | None   | Burstable  | 35    |
| **holmes**                                   | robusta                   | N/A     | N/A     | N/A     | N/A     | None         | N/A               | 0    | None   | BestEffort | 0     |
| **robusta-forwarder**                        | robusta                   | 10m     | N/A     | 512Mi   | 512Mi   | None         | N/A               | 0    | None   | Burstable  | 15    |
| **robusta-runner**                           | robusta                   | 250m    | 500m    | 1Gi     | 1Gi     | None         | N/A               | 0    | None   | Burstable  | 35    |
| **node-collector**                           | security                  | 100m    | 500m    | 100M    | 500M    | None         | N/A               | 0    | None   | Burstable  | 35    |
| **scan-vulnerabilityreport-786fb9585-p7qfg** | security                  | 100m    | 500m    | 100M    | 500M    | None         | N/A               | 0    | None   | Burstable  | 35    |
| **scan-vulnerabilityreport-c84ff9879-zz9xn** | security                  | 100m    | 500m    | 100M    | 500M    | None         | N/A               | 0    | None   | Burstable  | 35    |
| **trivy-operator**                           | security                  | 200m    | 1       | 1Gi     | 1Gi     | None         | vixens-medium     | 0    | None   | Burstable  | 55    |
| **docspell**                                 | services                  | 200m    | 1       | 1Gi     | 2Gi     | None         | vixens-medium     | 0    | None   | Burstable  | 55    |
| **gluetun**                                  | services                  | N/A     | N/A     | N/A     | N/A     | 11m / 236Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **vaultwarden**                              | services                  | 50m     | 500m    | 256Mi   | 512Mi   | 23m / 89Mi   | vixens-medium     | 0    | Active | Burstable  | 75    |
| **synology-csi**                             | synology-csi              | N/A     | N/A     | N/A     | N/A     | None         | N/A               | 0    | None   | BestEffort | 0     |
| **changedetection**                          | tools                     | 50m     | 500m    | 256Mi   | 512Mi   | 11m / 334Mi  | N/A               | 0    | None   | Burstable  | 35    |
| **headlamp**                                 | tools                     | 100m    | 200m    | 128Mi   | 256Mi   | 11m / 128Mi  | vixens-medium     | 0    | None   | Burstable  | 55    |
| **homepage**                                 | tools                     | N/A     | N/A     | N/A     | N/A     | 11m / 137Mi  | N/A               | 0    | None   | BestEffort | 0     |
| **it-tools**                                 | tools                     | 10m     | 100m    | 32Mi    | 128Mi   | 11m / 128Mi  | N/A               | 0    | None   | Burstable  | 35    |
| **linkwarden**                               | tools                     | 100m    | 1       | 1Gi     | 2Gi     | 11m / 932Mi  | N/A               | 0    | None   | Burstable  | 35    |
| **netbox**                                   | tools                     | N/A     | N/A     | N/A     | N/A     | 11m / 1051Mi | N/A               | 0    | None   | BestEffort | 0     |
| **nocodb**                                   | tools                     | 100m    | 500m    | 256Mi   | 1Gi     | 11m / 362Mi  | vixens-low        | 0    | None   | Burstable  | 55    |
| **penpot-backend**                           | tools                     | 200m    | 1       | 512Mi   | 1536Mi  | 23m / 825Mi  | vixens-medium     | 0    | None   | Burstable  | 55    |
| **penpot-exporter**                          | tools                     | 100m    | 1       | 512Mi   | 1Gi     | 11m / 174Mi  | vixens-medium     | 0    | None   | Burstable  | 55    |
| **penpot-frontend**                          | tools                     | 50m     | 200m    | 128Mi   | 256Mi   | 11m / 128Mi  | vixens-medium     | 0    | None   | Burstable  | 55    |
| **radar**                                    | tools                     | 100m    | 500m    | 128Mi   | 512Mi   | 78m / 728Mi  | N/A               | 0    | None   | Burstable  | 35    |
| **reloader**                                 | tools                     | 100m    | 100m    | 256Mi   | 256Mi   | None         | vixens-critical   | 0    | None   | Guaranteed | 80    |
| **stirling-pdf-chart**                       | tools                     | 100m    | 1       | 256Mi   | 1Gi     | None         | N/A               | 0    | None   | Burstable  | 35    |
| **vikunja**                                  | tools                     | 100m    | N/A     | 256Mi   | 512Mi   | 11m / 128Mi  | N/A               | 0    | None   | Burstable  | 15    |
| **traefik**                                  | traefik                   | 250m    | 2       | 256Mi   | 2Gi     | None         | vixens-high       | 0    | None   | Burstable  | 55    |
| **velero**                                   | velero                    | 100m    | 500m    | 256Mi   | 512Mi   | None         | N/A               | 0    | None   | Burstable  | 35    |
| **vertical-pod-autoscaler**                  | vpa                       | 200m    | 200m    | 500Mi   | 500Mi   | None         | vixens-critical   | 0    | None   | Guaranteed | 80    |
| **whoami**                                   | whoami                    | 50m     | 50m     | 128Mi   | 128Mi   | 11m / 128Mi  | vixens-medium     | 0    | None   | Guaranteed | 80    |
