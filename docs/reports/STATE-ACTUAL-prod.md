# Application State - Actual (Prod Reality)

**Last Updated:** 2026-01-11
**Environment:** Prod Cluster
**Data Source:** Live cluster state + GitOps manifests

---

## Production Application Inventory

| App                            | NS                        | CPU Req | CPU Lim | Mem Req | Mem Lim | VPA Target        | Profile | Priority                | Sync Wave | Backup Profile | Score |
| ------------------------------ | ------------------------- | ------- | ------- | ------- | ------- | ----------------- | ------- | ----------------------- | --------- | -------------- | ----- |
| **adguard-home**               | networking                | 1       | 1       | 1Gi     | 1Gi     | 143m / 126805489  | Medium  | vixens-critical         | 10        | Standard       | 0     |
| **amule**                      | downloads                 | 50m     | N/A     | 128Mi   | 512Mi   | 11m / 128Mi       | Small   | N/A                     | 10        | None           | 0     |
| **argocd**                     | argocd                    | N/A     | N/A     | N/A     | N/A     | 323m / 865936536  | Unknown | N/A                     | 0         | None           | 0     |
| **authentik**                  | auth                      | N/A     | N/A     | N/A     | N/A     | 49m / 628694953   | Unknown | N/A                     | 10        | None           | 0     |
| **birdnet-go**                 | birdnet-go                | 200m    | 1       | 256Mi   | 1Gi     | 476m / 813749082  | Medium  | N/A                     | 10        | None           | 0     |
| **booklore**                   | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 1038683533  | Unknown | N/A                     | 10        | None           | 0     |
| **cert-manager**               | cert-manager              | 100m    | 500m    | 128Mi   | 512Mi   | No VPA            | Small   | N/A                     | 0         | None           | 0     |
| **cert-manager-config**        | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 2         | None           | 0     |
| **cert-manager-secrets**       | cert-manager              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **cert-manager-webhook-gandi** | cert-manager              | 50m     | 200m    | 64Mi    | 256Mi   | No VPA            | Small   | N/A                     | 1         | None           | 0     |
| **changedetection**            | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **cilium-lb**                  | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -2        | None           | 0     |
| **cloudnative-pg**             | cnpg-system               | 100m    | 500m    | 128Mi   | 512Mi   | No VPA            | Small   | homelab-critical        | 3         | None           | 0     |
| **contacts**                   | contacts                  | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **descheduler**                | kube-system               | 50m     | 200m    | 64Mi    | 128Mi   | No VPA            | Micro   | system-cluster-critical | 10        | None           | 0     |
| **docspell-native**            | services                  | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 7         | None           | 0     |
| **external-dns-gandi**         | networking                | 20m     | 100m    | 64Mi    | 128Mi   | 11m / 128Mi       | Micro   | N/A                     | 5         | None           | 0     |
| **external-dns-gandi-secrets** | networking                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 4         | None           | 0     |
| **external-dns-unifi**         | networking                | 20m     | 100m    | 64Mi    | 128Mi   | 11m / 93633096    | Micro   | N/A                     | 5         | None           | 0     |
| **external-dns-unifi-secrets** | networking                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 4         | None           | 0     |
| **frigate**                    | media                     | 500m    | 2       | 1Gi     | 4Gi     | 11m / 109814751   | Large   | vixens-high             | 10        | Standard       | 0     |
| **gitops-revision-controller** | tools                     | 50m     | 200m    | 128Mi   | 512Mi   | No VPA            | Small   | N/A                     | 10        | None           | 0     |
| **gluetun**                    | services                  | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 10        | None           | 0     |
| **goldilocks**                 | monitoring                | 25m     | N/A     | 256Mi   | N/A     | 23m / 128Mi       | Unknown | N/A                     | 0         | None           | 0     |
| **grafana**                    | monitoring                | N/A     | N/A     | N/A     | N/A     | 11m / 144645763   | Unknown | N/A                     | 6         | None           | 0     |
| **grafana-ingress**            | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 7         | None           | 0     |
| **headlamp**                   | tools                     | 100m    | 200m    | 128Mi   | 256Mi   | No VPA            | Small   | N/A                     | 10        | None           | 0     |
| **homeassistant**              | homeassistant             | 300m    | 1       | 1Gi     | 2Gi     | 1238m / 248153480 | Medium  | vixens-high             | 10        | Standard       | 0     |
| **homepage**                   | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **hubble-ui**                  | monitoring                | 100m    | 200m    | 128Mi   | 256Mi   | 11m / 64Mi        | Small   | N/A                     | 10        | None           | 0     |
| **hydrus-client**              | media                     | 500m    | 2       | 1Gi     | 4Gi     | 35m / 865936536   | Large   | vixens-medium           | 10        | Standard       | 0     |
| **infisical-operator**         | infisical-operator-system | 100m    | 500m    | 128Mi   | 256Mi   | No VPA            | Small   | N/A                     | -3        | None           | 0     |
| **it-tools**                   | tools                     | 10m     | 100m    | 32Mi    | 128Mi   | No VPA            | Micro   | N/A                     | 5         | None           | 0     |
| **it-tools-ingress**           | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 6         | None           | 0     |
| **jellyfin**                   | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 272061154   | Unknown | homelab-important       | 10        | None           | 0     |
| **jellyseerr**                 | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 297164212   | Unknown | N/A                     | 10        | None           | 0     |
| **kubernetes-dashboard**       | kubernetes-dashboard      | 100m    | 250m    | 200Mi   | 400Mi   | No VPA            | Small   | N/A                     | 5         | None           | 0     |
| **lazylibrarian**              | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 144645763   | Unknown | N/A                     | 10        | None           | 0     |
| **lidarr**                     | media                     | 100m    | 1       | 256Mi   | 1Gi     | 23m / 109814751   | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **linkwarden**                 | tools                     | 100m    | 1       | 1Gi     | 2Gi     | No VPA            | Medium  | N/A                     | 5         | None           | 0     |
| **loki**                       | monitoring                | 100m    | 500m    | 256Mi   | 1Gi     | 23m / 297164212   | Medium  | N/A                     | 10        | None           | 0     |
| **mail-gateway**               | mail-gateway              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **mariadb-shared**             | databases                 | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 4         | None           | 0     |
| **mariadb-shared-config**      | databases                 | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 3         | None           | 0     |
| **mealie-prod**                | mealie                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **media-namespace**            | media                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -1        | None           | 0     |
| **metrics-server**             | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 10        | None           | 0     |
| **mosquitto**                  | mosquitto                 | 50m     | 200m    | 64Mi    | 256Mi   | 11m / 128Mi       | Small   | N/A                     | 10        | None           | 0     |
| **music-assistant**            | media                     | N/A     | N/A     | N/A     | N/A     | 11m / 144645763   | Unknown | N/A                     | 10        | None           | 0     |
| **mylar**                      | media                     | 100m    | 1       | 256Mi   | 1Gi     | 11m / 78221997    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **netbox**                     | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 5         | None           | 0     |
| **netvisor**                   | networking                | N/A     | N/A     | N/A     | N/A     | 11m / 128Mi       | Unknown | N/A                     | 10        | None           | 0     |
| **nfs-storage**                | media-stack               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -1        | None           | 0     |
| **postgresql-shared**          | databases                 | 100m    | 500m    | 256Mi   | 512Mi   | No VPA            | Small   | homelab-critical        | 4         | None           | 0     |
| **priority-classes**           | kube-system               | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -4        | None           | 0     |
| **prometheus**                 | monitoring                | 100m    | N/A     | 128Mi   | 512Mi   | 11m / 128Mi       | Small   | N/A                     | 5         | None           | 0     |
| **prometheus-ingress**         | monitoring                | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 6         | None           | 0     |
| **promtail**                   | monitoring                | 50m     | 100m    | 100Mi   | 256Mi   | 35m / 128Mi       | Small   | N/A                     | 10        | None           | 0     |
| **prowlarr**                   | media                     | 100m    | 1       | 256Mi   | 1Gi     | 11m / 78221997    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **pyload**                     | downloads                 | 50m     | N/A     | 128Mi   | 512Mi   | 11m / 128Mi       | Small   | N/A                     | 10        | None           | 0     |
| **qbittorrent**                | downloads                 | 50m     | N/A     | 256Mi   | 1Gi     | 11m / 128Mi       | Medium  | N/A                     | 10        | None           | 0     |
| **radarr**                     | media                     | 100m    | 1       | 256Mi   | 1Gi     | 23m / 109814751   | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **redis-shared**               | databases                 | N/A     | N/A     | N/A     | N/A     | 23m / 128Mi       | Unknown | N/A                     | -1        | None           | 0     |
| **reloader**                   | tools                     | 10m     | 100m    | 128Mi   | 256Mi   | No VPA            | Small   | N/A                     | 10        | None           | 0     |
| **renovate**                   | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 20        | None           | 0     |
| **sabnzbd**                    | media                     | 50m     | 2       | 256Mi   | 4Gi     | 11m / 93633096    | Large   | vixens-low              | 10        | Standard       | 0     |
| **sonarr**                     | media                     | 100m    | 1       | 256Mi   | 1Gi     | 126m / 109814751  | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **stirling-pdf**               | tools                     | 100m    | 1       | 256Mi   | 1Gi     | No VPA            | Medium  | N/A                     | 5         | None           | 0     |
| **stirling-pdf-ingress**       | tools                     | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 6         | None           | 0     |
| **synology-csi**               | synology-csi              | N/A     | N/A     | N/A     | N/A     | 11m / 32Mi        | Unknown | N/A                     | 0         | None           | 0     |
| **synology-csi-secrets**       | synology-csi              | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | -1        | None           | 0     |
| **traefik**                    | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **traefik-dashboard**          | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 1         | None           | 0     |
| **traefik-middlewares**        | traefik                   | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 1         | None           | 0     |
| **vaultwarden**                | services                  | 50m     | 500m    | 256Mi   | 512Mi   | 35m / 64Mi        | Small   | vixens-high             | 10        | Standard       | 0     |
| **vixens-app-of-apps**         | argocd                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | N/A                     | 0         | None           | 0     |
| **vpa**                        | vpa                       | 50m     | 200m    | 100Mi   | 500Mi   | No VPA            | Small   | N/A                     | 0         | None           | 0     |
| **whisparr**                   | media                     | 100m    | 1       | 256Mi   | 1Gi     | 11m / 78221997    | Medium  | vixens-medium           | 10        | Standard       | 0     |
| **whoami**                     | whoami                    | N/A     | N/A     | N/A     | N/A     | No VPA            | Unknown | vixens-low              | 10        | None           | 0     |
