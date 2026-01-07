# Audit de Production Total (75 Apps)

| App                          | Namespace                  | Prod Req     | Prod Lim      | VPA Target      | Status                |
| :--------------------------- | :------------------------- | :----------- | :------------ | :-------------- | :-------------------- |
| adguard-home                 | networking                 | 100m/256Mi   | 500m/512Mi    | 11m / 128Mi     | 🟢 OK                 |
| amule                        | downloads                  | 50m/128Mi    | N/A/512Mi     | 11m / 128Mi     | 🟢 OK                 |
| argocd                       | argocd                     | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| authentik                    | auth                       | 200m/512Mi   | 500m/1024Mi   | 323m / 1.4Gi    | 🔴 OOM RISK           |
| birdnet-go                   | birdnet-go                 | 200m/256Mi   | 1000m/1Gi     | 23m / 215Mi     | 🟢 OK                 |
| booklore                     | media                      | 15m/334Mi    | 49m/362Mi     | 163m / 2.6Gi    | 🔴 CPU BRIDÉ          |
| cert-manager                 | cert-manager               | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| cert-manager-config          | cert-manager               | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| cert-manager-secrets         | cert-manager               | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| cert-manager-webhook-gandi   | cert-manager               | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| changedetection              | tools                      | N/A/N/A      | N/A/N/A       | 11m / 121Mi     | 🟢 OK                 |
| cilium-lb                    | kube-system                | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| cloudnative-pg               | cnpg-system                | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| contacts                     | contacts                   | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| descheduler                  | kube-system                | 50m/64Mi     | 200m/128Mi    | No VPA          | 🟢 OK / NO VPA        |
| docspell-native              | services                   | 500m/2048Mi  | 2000m/4096Mi  | No VPA          | 🟢 OK / NO VPA        |
| external-dns-gandi           | networking                 | N/A/N/A      | N/A/N/A       | 11m / 128Mi     | ⚪ NO LIMITS          |
| external-dns-gandi-secrets   | networking                 | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| external-dns-unifi           | networking                 | N/A/N/A      | N/A/N/A       | 11m / 64Mi      | ⚪ NO LIMITS          |
| external-dns-unifi-secrets   | networking                 | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| frigate                      | media                      | 500m/1Gi     | 2000m/8Gi     | 2406m / 3.6Gi   | 🔴 CPU BRIDÉ          |
| gitops-revision-controller   | tools                      | 50m/128Mi    | 200m/512Mi    | 11m / 138Mi     | 🟢 OK                 |
| gluetun                      | services                   | N/A/N/A      | N/A/N/A       | 11m / 128Mi     | ⚪ NO LIMITS          |
| goldilocks                   | monitoring                 | N/A/N/A      | N/A/N/A       | 23m / 128Mi     | 🟢 OK                 |
| grafana                      | monitoring                 | 100m/256Mi   | N/A/512Mi     | 23m / 215Mi     | 🟢 OK                 |
| grafana-ingress              | monitoring                 | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| headlamp                     | tools                      | 100m/128Mi   | 200m/256Mi    | 11m / 128Mi     | 🟢 OK                 |
| homeassistant                | homeassistant              | 300m/1024Mi  | 1000m/2048Mi  | 11m / 64Mi      | 🟢 OK                 |
| homepage                     | tools                      | N/A/N/A      | N/A/N/A       | 11m / 215Mi     | 🟢 OK                 |
| hubble-ui                    | monitoring                 | 100m/128Mi   | 200m/256Mi    | No VPA          | 🟢 OK / NO VPA        |
| hydrus-client                | media                      | 34m/2294Mi   | 49m/3877Mi    | 63m / 1.9Gi     | 🔴 CPU BRIDÉ          |
| infisical-operator           | infisical-operator-system  | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| it-tools                     | tools                      | 10m/32Mi     | 100m/128Mi    | 11m / 128Mi     | 🟠 WARNING            |
| it-tools-ingress             | tools                      | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| jellyfin                     | media                      | 15m/1567Mi   | 15m/2062Mi    | 23m / 826Mi     | 🔴 CPU BRIDÉ          |
| jellyseerr                   | media                      | 50m/128Mi    | 200m/256Mi    | 78m / 422Mi     | 🔴 OOM RISK           |
| lazylibrarian                | media                      | 15m/259Mi    | 15m/488Mi     | 23m / 175Mi     | 🔴 CPU BRIDÉ          |
| lidarr                       | media                      | 15m/214Mi    | 15m/214Mi     | 11m / 237Mi     | 🔴 OOM RISK           |
| linkwarden                   | tools                      | 100m/1Gi     | 1000m/2Gi     | 11m / 561Mi     | 🟢 OK                 |
| loki                         | monitoring                 | 100m/256Mi   | 500m/1024Mi   | 23m / 260Mi     | 🟢 OK                 |
| mail-gateway                 | mail-gateway               | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| mariadb-shared               | databases                  | 200m/512Mi   | 1000m/1024Mi  | No VPA          | 🟢 OK                 |
| mealie                       | mealie                     | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| metrics-server               | kube-system                | 100m/200Mi   | 500m/500Mi    | No VPA          | 🟢 OK / NO VPA        |
| mosquitto                    | mosquitto                  | 50m/64Mi     | 200m/256Mi    | 11m / 128Mi     | 🟢 OK                 |
| music-assistant              | media                      | 15m/283Mi    | 15m/283Mi     | 11m / 215Mi     | 🟢 OK                 |
| mylar                        | media                      | 15m/104Mi    | 15m/104Mi     | 11m / 128Mi     | 🔴 OOM RISK           |
| netbox                       | tools                      | N/A/N/A      | N/A/N/A       | 11m / 641Mi     | ⚪ NO LIMITS          |
| netvisor                     | networking                 | N/A/N/A      | N/A/N/A       | 11m / 128Mi     | 🟢 OK                 |
| nfs-storage                  | media-stack                | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| postgresql-shared            | databases                  | 100m/256Mi   | 500m/512Mi    | No VPA          | 🟢 OK / NO VPA        |
| priority-classes             | kube-system                | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| prometheus                   | monitoring                 | 300m/1Gi     | 100m/2Gi      | 11m / 128Mi     | 🟢 OK                 |
| prometheus-ingress           | monitoring                 | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| promtail                     | monitoring                 | 50m/100Mi    | 100m/256Mi    | 49m / 128Mi     | 🟢 OK                 |
| prowlarr                     | media                      | 15m/155Mi    | 15m/174Mi     | 11m / 194Mi     | 🔴 OOM RISK           |
| pyload                       | downloads                  | 50m/128Mi    | N/A/512Mi     | 11m / 128Mi     | 🟢 OK                 |
| qbittorrent                  | downloads                  | 50m/256Mi    | N/A/1Gi       | 11m / 128Mi     | 🟢 OK                 |
| radarr                       | media                      | 22m/334Mi    | 35m/362Mi     | 49m / 561Mi     | 🔴 CPU BRIDÉ          |
| redis-shared                 | databases                  | N/A/N/A      | N/A/N/A       | 23m / 128Mi     | 🟢 OK                 |
| reloader                     | tools                      | 10m/128Mi    | 100m/256Mi    | 11m / 128Mi     | 🟢 OK                 |
| renovate                     | tools                      | N/A/N/A      | N/A/N/A       | 977m / 991Mi    | 🟢 OK                 |
| sabnzbd                      | media                      | 50m/256Mi    | 500m/1Gi      | 23m / 237Mi     | 🟢 OK                 |
| sonarr                       | media                      | 15m/236Mi    | 15m/236Mi     | 23m / 260Mi     | 🟢 OK                 |
| stirling-pdf                 | tools                      | 100m/256Mi   | 1000m/1Gi     | 23m / 363Mi     | 🟢 OK                 |
| stirling-pdf-ingress         | tools                      | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| synology-csi                 | synology-csi               | N/A/N/A      | N/A/N/A       | 11m / 32Mi      | ⚪ NO LIMITS          |
| synology-csi-secrets         | synology-csi               | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| traefik                      | traefik                    | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| traefik-dashboard            | traefik                    | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |
| vaultwarden                  | services                   | 50m/256Mi    | 500m/512Mi    | 11m / 128Mi     | 🟢 OK                 |
| vixens-app-of-apps           | argocd                     | N/A/N/A      | N/A/N/A       | No VPA          | ⚪ NO LIMITS / NO VPA |
| vpa                          | vpa                        | 50m/100Mi    | 200m/500Mi    | No VPA          | 🟢 OK / NO VPA        |
| whisparr                     | media                      | 15m/120Mi    | 15m/120Mi     | 11m / 138Mi     | 🔴 OOM RISK           |
| whoami                       | whoami                     | N/A/N/A      | N/A/N/A       | No VPA          | 🟢 OK                 |