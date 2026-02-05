# Application State - Actual (Prod Reality)

**Last Updated:** 2026-02-05
**Environment:** Prod Cluster
**Data Source:** Live cluster state + GitOps manifests

---

## Production Application Inventory

| App                            | NS                        | CPU Req | CPU Lim | Mem Req | Mem Lim | VPA Target        | Profile | Priority                | Sync Wave | Backup Profile | Score |
| ------------------------------ | ------------------------- | ------- | ------- | ------- | ------- | ----------------- | ------- | ----------------------- | --------- | -------------- | ----- |
| **adguard-home**               | networking                | 100m    | 500m    | 256Mi   | 512Mi   | 93m / 126805489   | Small   | vixens-critical         | 10        | Standard       | 0     |
| **amule**                      | downloads                 | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **argocd**                     | argocd                    | N/A     | N/A     | N/A     | N/A     | 203m / 1038683533 | Unknown | N/A                     | 0         | None           | 0     |
| **authentik**                  | auth                      | N/A     | N/A     | N/A     | N/A     | 93m / 1389197403  | Unknown | N/A                     | 10        | None           | 0     |
| **birdnet-go**                 | birdnet-go                | 200m    | 1       | 256Mi   | 1Gi     | 35m / 323522422   | Medium  | N/A                     | 10        | None           | 0     |
| **booklore**                   | media                     | N/A     | N/A     | N/A     | N/A     | 35m / 2823238195  | Unknown | N/A                     | 10        | None           | 0     |
| **cert-manager**               | cert-manager              | 100m    | 500m    | 128Mi   | 512Mi   | No VPA            | Small   | N/A                     | 0         | None           | 0     |
| **cert-manager-config**        | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 2         | None           | 0     |
| **cert-manager-secrets**       | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **cert-manager-webhook-gandi** | cert-manager              | 50m     | 200m    | 64Mi    | 256Mi   | No VPA            | Small   | N/A                     | 1         | None           | 0     |
| **changedetection**            | tools                     | N/A     | N/A     | N/A     | N/A     | 11m / 144645763   | Unknown | N/A                     | 10        | None           | 0     |
| **cilium-lb**                  | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -2        | None           | 0     |
| **cloudnative-pg**             | cnpg-system               | 100m    | 500m    | 128Mi   | 512Mi   | No VPA            | Small   | homelab-critical        | 3         | None           | 0     |
| **contacts**                   | contacts                  | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **descheduler**                | kube-system               | 50m     | 200m    | 64Mi    | 128Mi   | No VPA            | Micro   | system-cluster-critical | 10        | None           | 0     |
| **docspell-native**            | services                  | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 7         | None           | 0     |
| **external-dns-gandi**         | networking                | 20m     | 100m    | 64Mi    | 128Mi   | 11m / 128Mi       | Micro   | N/A                     | 5         | None           | 0     |
| **external-dns-gandi-secrets** | networking                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 4         | None           | 0     |
| **external-dns-unifi**         | networking                | 20m     | 100m    | 64Mi    | 128Mi   | 11m / 64Mi        | Micro   | N/A                     | 5         | None           | 0     |
| **external-dns-unifi-secrets** | networking                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 4         | None           | 0     |
| **firefly-iii**                | finance                   | 100m    | 500m    | 256Mi   | 512Mi   | No VPA            | Small   | vixens-low              | 0         | None           | 0     |
| **firefly-iii-importer**       | finance                   | 10m     | 500m    | 128Mi   | 512Mi   | No VPA            | Small   | vixens-low              | 0         | None           | 0     |
| **frigate**                    | media                     | 500m    | 8       | 4Gi     | 8Gi     | 11m / 44739242    | Large   | vixens-medium           | 10        | Standard       | 0     |
| **gluetun**                    | services                  | N/A     | N/A     | N/A     | N/A     | 11m / 248153480   | Unknown | N/A                     | 10        | None           | 0     |
| **goldilocks**                 | monitoring                | 25m     | N/A     | 256Mi   | N/A     | 11m / 128Mi       | Unknown | N/A                     | 0         | None           | 0     |
| **grafana**                    | monitoring                | N/A     | N/A     | N/A     | N/A     | 11m / 272061154   | Unknown | vixens-critical         | 6         | None           | 0     |
| **grafana-ingress**            | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 7         | None           | 0     |
| **headlamp**                   | tools                     | 100m    | 200m    | 128Mi   | 256Mi   | 11m / 128Mi       | Small   | N/A                     | 10        | None           | 0     |
| **homeassistant**              | homeassistant             | 500m    | 2       | 2Gi     | 4Gi     | 247m / 78221997   | Large   | vixens-high             | 10        | Standard       | 0     |
| **homepage**                   | tools                     | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 10        | None           | 0     |
| **hydrus-client**              | media                     | 1       | 1       | 2Gi     | 2Gi     | 11m / 442809964   | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **infisical-operator**         | infisical-operator-system | 10m     | 500m    | 64Mi    | 256Mi   | No VPA            | Small   | N/A                     | -3        | None           | 0     |
| **it-tools**                   | tools                     | 10m     | 100m    | 32Mi    | 128Mi   | 11m / 128Mi       | Micro   | N/A                     | 5         | None           | 0     |
| **it-tools-ingress**           | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 6         | None           | 0     |
| **jellyfin**                   | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 203699302   | Unknown | homelab-important       | 10        | None           | 0     |
| **jellyseerr**                 | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 323522422   | Unknown | N/A                     | 10        | None           | 0     |
| **kyverno**                    | kyverno                   | 100m    | N/A     | 128Mi   | 384Mi   | No VPA            | Small   | N/A                     | 5         | None           | 0     |
| **lazylibrarian**              | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 44739242    | Unknown | N/A                     | 10        | Standard       | 0     |
| **lidarr**                     | media                     | 100m    | 1       | 256Mi   | 1Gi     | 93m / 63544758    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **linkwarden**                 | tools                     | 100m    | 1       | 1Gi     | 2Gi     | 671m / 813749082  | Medium  | N/A                     | 5         | None           | 0     |
| **loki**                       | monitoring                | 500m    | 500m    | 1Gi     | 1Gi     | 49m / 511772986   | Medium  | vixens-critical         | 10        | None           | 0     |
| **mail-gateway**               | mail-gateway              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **mariadb-shared**             | databases                 | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 4         | None           | 0     |
| **mealie**                     | mealie                    | 100m    | 500m    | 256Mi   | 512Mi   | 23m / 44739242    | Small   | N/A                     | 10        | Standard       | 0     |
| **media-shared-secrets**       | media                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 9         | None           | 0     |
| **metrics-server**             | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **mosquitto**                  | mosquitto                 | 50m     | 200m    | 64Mi    | 256Mi   | 23m / 64Mi        | Small   | N/A                     | 10        | None           | 0     |
| **music-assistant**            | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 163378051   | Unknown | N/A                     | 10        | None           | 0     |
| **mylar**                      | media                     | 100m    | 500m    | 256Mi   | 512Mi   | 11m / 49566436    | Small   | vixens-medium           | 10        | Standard       | 0     |
| **netbird**                    | networking                | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 15        | None           | 0     |
| **netbox**                     | tools                     | N/A     | N/A     | N/A     | N/A     | 11m / 1102117711  | Unknown | N/A                     | 5         | None           | 0     |
| **netvisor**                   | networking                | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 10        | None           | 0     |
| **nfs-storage**                | media-stack               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -1        | None           | 0     |
| **nocodb**                     | tools                     | 100m    | 500m    | 256Mi   | 1Gi     | 11m / 297164212   | Medium  | vixens-low              | 0         | None           | 0     |
| **penpot**                     | tools                     | N/A     | N/A     | N/A     | N/A     | 23m / 865936536   | Unknown | N/A                     | 0         | None           | 0     |
| **policy-reporter**            | policy-reporter           | 50m     | 200m    | 128Mi   | 256Mi   | No VPA            | Small   | N/A                     | 6         | None           | 0     |
| **postgresql-shared**          | databases                 | 500m    | 500m    | 512Mi   | 512Mi   | No VPA            | Small   | vixens-critical         | 4         | None           | 0     |
| **priority-classes**           | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -4        | None           | 0     |
| **prometheus**                 | monitoring                | 200m    | 200m    | 512Mi   | 512Mi   | 11m / 128Mi       | Small   | vixens-critical         | 5         | None           | 0     |
| **prometheus-ingress**         | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 6         | None           | 0     |
| **promtail**                   | monitoring                | 200m    | 200m    | 256Mi   | 256Mi   | 63m / 163378051   | Small   | vixens-critical         | 10        | None           | 0     |
| **prowlarr**                   | media                     | 100m    | 500m    | 256Mi   | 512Mi   | 23m / 44739242    | Small   | vixens-medium           | 10        | Standard       | 0     |
| **pyload**                     | downloads                 | 50m     | N/A     | 128Mi   | 512Mi   | 11m / 128Mi       | Small   | N/A                     | 10        | None           | 0     |
| **qbittorrent**                | downloads                 | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **radar**                      | tools                     | 100m    | 500m    | 128Mi   | 512Mi   | 49m / 764046746   | Small   | N/A                     | 0         | None           | 0     |
| **radarr**                     | media                     | 100m    | 1       | 256Mi   | 1Gi     | 23m / 44739242    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **redis-shared**               | databases                 | N/A     | N/A     | N/A     | N/A     | 23m / 128Mi       | Unknown | N/A                     | -1        | None           | 0     |
| **reloader**                   | tools                     | 100m    | 100m    | 256Mi   | 256Mi   | 11m / 128Mi       | Small   | vixens-critical         | 10        | None           | 0     |
| **renovate**                   | tools                     | 500m    | 1       | 512Mi   | 1Gi     | 1238m / 813749082 | Medium  | vixens-medium           | 20        | None           | 0     |
| **robusta**                    | robusta                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **sabnzbd**                    | media                     | 100m    | 1       | 256Mi   | 1Gi     | 23m / 44739242    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **shared-namespaces**          | argocd                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -3        | None           | 0     |
| **sonarr**                     | media                     | 100m    | 1       | 256Mi   | 1Gi     | 35m / 44739242    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **stirling-pdf**               | tools                     | 100m    | 1       | 256Mi   | 1Gi     | 11m / 1102117711  | Medium  | N/A                     | 5         | None           | 0     |
| **stirling-pdf-ingress**       | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 6         | None           | 0     |
| **synology-csi**               | synology-csi              | N/A     | N/A     | N/A     | N/A     | 11m / 49566436    | Unknown | N/A                     | 0         | None           | 0     |
| **synology-csi-secrets**       | synology-csi              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -1        | None           | 0     |
| **traefik**                    | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **traefik-dashboard**          | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 1         | None           | 0     |
| **traefik-middlewares**        | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 1         | None           | 0     |
| **trivy**                      | security                  | 200m    | 1       | 1Gi     | 1Gi     | No VPA            | Medium  | vixens-medium           | 0         | None           | 0     |
| **vaultwarden**                | services                  | 50m     | 500m    | 256Mi   | 512Mi   | 23m / 49566436    | Small   | vixens-medium           | 10        | Standard       | 0     |
| **velero**                     | velero                    | 100m    | 500m    | 256Mi   | 512Mi   | No VPA            | Small   | vixens-critical         | 1         | None           | 0     |
| **velero-maintenance-config**  | velero                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **velero-secrets**             | velero                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **vikunja**                    | tools                     | 100m    | N/A     | 256Mi   | 512Mi   | 11m / 128Mi       | Small   | N/A                     | 5         | None           | 0     |
| **vixens-app-of-apps**         | argocd                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **vpa**                        | vpa                       | 200m    | 200m    | 500Mi   | 500Mi   | No VPA            | Small   | vixens-critical         | 0         | None           | 0     |
| **whisparr**                   | media                     | 100m    | 1       | 256Mi   | 1Gi     | 23m / 49566436    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **whoami**                     | whoami                    | 50m     | 50m     | 128Mi   | 128Mi   | 11m / 128Mi       | Micro   | vixens-medium           | 10        | None           | 0     |
