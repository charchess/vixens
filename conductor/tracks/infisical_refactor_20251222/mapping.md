# Infisical Secret Path Mapping

This table maps existing Infisical secret paths to the new standardized hierarchy mirroring the Git repository structure.

## Mapping Table

| Application | Git Path | Current Infisical Path | Target Infisical Path |
|-------------|----------|------------------------|-----------------------|
| cert-manager-webhook-gandi | `apps/00-infra/cert-manager-webhook-gandi` | `/cert-manager` | `/apps/00-infra/cert-manager-webhook-gandi` |
| synology-csi | `apps/01-storage/synology-csi` | `/synology-csi` | `/apps/01-storage/synology-csi` |
| alertmanager | `apps/02-monitoring/alertmanager` | `/alertmanager` | `/apps/02-monitoring/alertmanager` |
| goldilocks | `apps/02-monitoring/goldilocks` | `/goldilocks` | `/apps/02-monitoring/goldilocks` |
| grafana | `apps/02-monitoring/grafana` | `/grafana` | `/apps/02-monitoring/grafana` |
| hubble-ui | `apps/02-monitoring/hubble-ui` | `/hubble-ui` | `/apps/02-monitoring/hubble-ui` |
| loki | `apps/02-monitoring/loki` | `/loki` | `/apps/02-monitoring/loki` |
| prometheus | `apps/02-monitoring/prometheus` | `/prometheus` | `/apps/02-monitoring/prometheus` |
| promtail | `apps/02-monitoring/promtail` | `/promtail` | `/apps/02-monitoring/promtail` |
| authentik | `apps/03-security/authentik` | `/authentik` | `/apps/03-security/authentik` |
| postgresql-shared (admin) | `apps/04-databases/postgresql-shared` | `/postgresql-shared` | `/apps/04-databases/postgresql-shared` |
| redis-shared | `apps/04-databases/redis-shared` | `/redis-shared` | `/apps/04-databases/redis-shared` |
| homeassistant | `apps/10-home/homeassistant` | `/homeassistant` | `/apps/10-home/homeassistant` |
| mosquitto | `apps/10-home/mosquitto` | `/mosquitto` | `/apps/10-home/mosquitto` |
| birdnet-go | `apps/20-media/birdnet-go` | `/birdnet-go` | `/apps/20-media/birdnet-go` |
| booklore | `apps/20-media/booklore` | `/booklore` | `/apps/20-media/booklore` |
| frigate | `apps/20-media/frigate` | `/frigate` | `/apps/20-media/frigate` |
| hydrus-client | `apps/20-media/hydrus-client` | `/hydrus-client` | `/apps/20-media/hydrus-client` |
| hydrus-server | `apps/20-media/hydrus-server` | `/hydrus-server` | `/apps/20-media/hydrus-server` |
| lazylibrarian | `apps/20-media/lazylibrarian` | `/lazylibrarian` | `/apps/20-media/lazylibrarian` |
| lidarr | `apps/20-media/lidarr` | `/lidarr` | `/apps/20-media/lidarr` |
| mylar | `apps/20-media/mylar` | `/mylar` | `/apps/20-media/mylar` |
| prowlarr | `apps/20-media/prowlarr` | `/prowlarr` | `/apps/20-media/prowlarr` |
| radarr | `apps/20-media/radarr` | `/radarr` | `/apps/20-media/radarr` |
| sabnzbd | `apps/20-media/sabnzbd` | `/sabnzbd` | `/apps/20-media/sabnzbd` |
| sonarr | `apps/20-media/sonarr` | `/sonarr` | `/apps/20-media/sonarr` |
| whisparr | `apps/20-media/whisparr` | `/whisparr` | `/apps/20-media/whisparr` |
| adguard-home | `apps/40-network/adguard-home` | `/adguard-home` | `/apps/40-network/adguard-home` |
| netvisor | `apps/40-network/netvisor` | `/netvisor` | `/apps/40-network/netvisor` |
| docspell | `apps/60-services/docspell` | `/docspell` | `/apps/60-services/docspell` |
| docspell-native | `apps/60-services/docspell-native` | `/docspell` | `/apps/60-services/docspell-native` |
| gluetun | `apps/60-services/gluetun` | `/gluetun` | `/apps/60-services/gluetun` |
| vaultwarden | `apps/60-services/vaultwarden` | `/vaultwarden` | `/apps/60-services/vaultwarden` |
| argocd-image-updater | `apps/70-tools/argocd-image-updater` | `/argocd-image-updater` | `/apps/70-tools/argocd-image-updater` |
| changedetection | `apps/70-tools/changedetection` | `/changedetection` | `/apps/70-tools/changedetection` |
| headlamp | `apps/70-tools/headlamp` | `/headlamp` | `/apps/70-tools/headlamp` |
| homepage | `apps/70-tools/homepage` | `/homepage` | `/apps/70-tools/homepage` |
| linkwarden | `apps/70-tools/linkwarden` | `/linkwarden` | `/apps/70-tools/linkwarden` |
| netbox | `apps/70-tools/netbox` | `/netbox` | `/apps/70-tools/netbox` |

## Shared Secrets Duplication

Shared secrets must be duplicated to the application's specific path.

| Shared Secret | Source Path | Consumer Application | Target Path |
|---------------|-------------|----------------------|-------------|
| PostgreSQL Shared Credentials | `/postgresql-shared` | `authentik` | `/apps/03-security/authentik` |
| PostgreSQL Shared Credentials | `/postgresql-shared` | `docspell` | `/apps/60-services/docspell` |
| PostgreSQL Shared Credentials | `/postgresql-shared` | `netbox` | `/apps/70-tools/netbox` |
| PostgreSQL Shared Credentials | `/postgresql-shared` | `linkwarden` | `/apps/70-tools/linkwarden` |
| PostgreSQL Shared Credentials | `/postgresql-shared` | `homeassistant` | `/apps/10-home/homeassistant` |
| Redis Shared Credentials | `/redis-shared` | `authentik` | `/apps/03-security/authentik` |
| Redis Shared Credentials | `/redis-shared` | `netbox` | `/apps/70-tools/netbox` |
| MQTT Credentials | `/mosquitto` | `homeassistant` | `/apps/10-home/homeassistant` |
| MQTT Credentials | `/mosquitto` | `frigate` | `/apps/20-media/frigate` |
| S3 Backup Credentials | `/postgresql-shared` | `postgresql-shared` | `/apps/04-databases/postgresql-shared` |
