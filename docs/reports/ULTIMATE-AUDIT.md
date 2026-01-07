# Audit de Production Total (75 Apps)
| App | Namespace | Prod Req | Prod Lim | VPA Target | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| adguard-home | networking | 100m/256Mi | 500m/512Mi | 11m / 128.0Mi | 🟢 OK |
| amule | downloads | 50m/128Mi | N/A/512Mi | 11m / 128.0Mi | 🔴 GIT-ERROR |
| argocd | argocd | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| authentik | auth | 200m/512Mi | 500m/1024Mi | 323m / 1.4Gi | 🔴 OOM RISK |
| birdnet-go | birdnet-go | 200m/256Mi | 1000m/1Gi | 23m / 214.9Mi | 🟢 OK |
| booklore | media | 15m/334Mi | 49m/362Mi | 163m / 2.6Gi | 🔴 CPU BRIDÉ |
| cert-manager | cert-manager | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| cert-manager-config | cert-manager | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| cert-manager-secrets | cert-manager | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| cert-manager-webhook-gandi | cert-manager | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| changedetection | tools | N/A/N/A | N/A/N/A | 11m / 120.9Mi | 🔴 GIT-ERROR |
| cilium-lb | kube-system | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| cloudnative-pg | cnpg-system | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| cloudnative-pg-crds | cnpg-system | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| contacts | contacts | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| descheduler | kube-system | 50m/64Mi | 200m/128Mi | No VPA | 🔴 GIT-ERROR |
| docspell-native | services | 500m/2048Mi | 2000m/4096Mi | No VPA | 🟢 OK / NO VPA |
| external-dns-gandi | networking | N/A/N/A | N/A/N/A | 11m / 128.0Mi | ⚪ NO LIMITS |
| external-dns-gandi-secrets | networking | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| external-dns-unifi | networking | N/A/N/A | N/A/N/A | 11m / 64.0Mi | ⚪ NO LIMITS |
| external-dns-unifi-secrets | networking | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| frigate | media | 500m/1Gi | 2000m/8Gi | 2406m / 3.6Gi | 🔴 GIT-ERROR |
| gitops-revision-controller | tools | 50m/128Mi | 200m/512Mi | 11m / 137.9Mi | 🔴 GIT-ERROR |
| gluetun | services | N/A/N/A | N/A/N/A | 11m / 128.0Mi | ⚪ NO LIMITS |
| goldilocks | monitoring | N/A/N/A | N/A/N/A | 23m / 128.0Mi | 🔴 GIT-ERROR |
| grafana | monitoring | 100m/256Mi | N/A/512Mi | 23m / 214.9Mi | 🔴 GIT-ERROR |
| grafana-ingress | monitoring | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| headlamp | tools | 100m/128Mi | 200m/256Mi | 11m / 128.0Mi | 🔴 GIT-ERROR |
| homeassistant | homeassistant | 300m/1024Mi | 1000m/2048Mi | 11m / 64.0Mi | 🟢 OK |
| homepage | tools | N/A/N/A | N/A/N/A | 11m / 214.9Mi | 🔴 GIT-ERROR |
| hubble-ui | monitoring | 100m/128Mi | 200m/256Mi | No VPA | 🔴 GIT-ERROR |
| hydrus-client | media | 34m/2294Mi | 49m/3877Mi | 63m / 1.9Gi | 🔴 CPU BRIDÉ |
| infisical-operator | infisical-operator-system | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| it-tools | tools | 10m/32Mi | 100m/128Mi | 11m / 128.0Mi | 🟠 WARNING |
| it-tools-ingress | tools | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| jellyfin | media | 15m/1567Mi | 15m/2062Mi | 23m / 825.8Mi | 🔴 CPU BRIDÉ |
| jellyseerr | media | 50m/128Mi | 200m/256Mi | 78m / 422.3Mi | 🔴 OOM RISK |
| lazylibrarian | media | 15m/259Mi | 15m/488Mi | 23m / 174.6Mi | 🔴 CPU BRIDÉ |
| lidarr | media | 15m/214Mi | 15m/214Mi | 11m / 236.7Mi | 🔴 OOM RISK |
| linkwarden | tools | 100m/1Gi | 1000m/2Gi | 11m / 560.6Mi | 🔴 GIT-ERROR |
| loki | monitoring | 100m/256Mi | 500m/1024Mi | 23m / 259.5Mi | 🔴 GIT-ERROR |
| mail-gateway | mail-gateway | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| mariadb-shared | databases | 200m/512Mi | 1000m/1024Mi | No VPA | 🟢 OK |
| mealie | mealie | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| metrics-server | kube-system | 100m/200Mi | 500m/500Mi | No VPA | 🔴 GIT-ERROR |
| mosquitto | mosquitto | 50m/64Mi | 200m/256Mi | 11m / 128.0Mi | 🟢 OK |
| music-assistant | media | 15m/283Mi | 15m/283Mi | 11m / 214.9Mi | 🟢 OK |
| mylar | media | 15m/104Mi | 15m/104Mi | 11m / 128.0Mi | 🔴 OOM RISK |
| netbox | tools | N/A/N/A | N/A/N/A | 11m / 640.5Mi | ⚪ NO LIMITS |
| netvisor | networking | N/A/N/A | N/A/N/A | 11m / 128.0Mi | 🔴 GIT-ERROR |
| nfs-storage | media-stack | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| postgresql-shared | databases | 100m/256Mi | 500m/512Mi | No VPA | 🔴 GIT-ERROR |
| priority-classes | kube-system | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| prometheus | monitoring | 300m/1Gi | 100m/2Gi | 11m / 128.0Mi | 🔴 GIT-ERROR |
| prometheus-ingress | monitoring | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| promtail | monitoring | 50m/100Mi | 100m/256Mi | 49m / 128.0Mi | 🔴 GIT-ERROR |
| prowlarr | media | 15m/155Mi | 15m/174Mi | 11m / 194.3Mi | 🔴 OOM RISK |
| pyload | downloads | 50m/128Mi | N/A/512Mi | 11m / 128.0Mi | 🔴 GIT-ERROR |
| qbittorrent | downloads | 50m/256Mi | N/A/1Gi | 11m / 128.0Mi | 🔴 GIT-ERROR |
| radarr | media | 22m/334Mi | 35m/362Mi | 49m / 560.6Mi | 🔴 CPU BRIDÉ |
| redis-shared | databases | N/A/N/A | N/A/N/A | 23m / 128.0Mi | 🔴 GIT-ERROR |
| reloader | tools | 10m/128Mi | 100m/256Mi | 11m / 128.0Mi | 🔴 GIT-ERROR |
| renovate | tools | N/A/N/A | N/A/N/A | 977m / 990.6Mi | 🔴 GIT-ERROR |
| sabnzbd | media | 50m/256Mi | 500m/1Gi | 23m / 236.7Mi | 🟢 OK |
| sonarr | media | 15m/236Mi | 15m/236Mi | 23m / 259.5Mi | 🔴 GIT-ERROR |
| stirling-pdf | tools | 100m/256Mi | 1000m/1Gi | 23m / 362.6Mi | 🟢 OK |
| stirling-pdf-ingress | tools | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| synology-csi | synology-csi | N/A/N/A | N/A/N/A | 11m / 32.0Mi | ⚪ NO LIMITS |
| synology-csi-secrets | synology-csi | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| traefik | traefik | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| traefik-dashboard | traefik | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |
| vaultwarden | services | 50m/256Mi | 500m/512Mi | 11m / 128.0Mi | 🟢 OK |
| vixens-app-of-apps | argocd | N/A/N/A | N/A/N/A | No VPA | ⚪ NO LIMITS / NO VPA |
| vpa | vpa | 50m/100Mi | 200m/500Mi | No VPA | 🔴 GIT-ERROR |
| whisparr | media | 15m/120Mi | 15m/120Mi | 11m / 137.9Mi | 🔴 OOM RISK |
| whoami | whoami | N/A/N/A | N/A/N/A | No VPA | 🔴 GIT-ERROR |