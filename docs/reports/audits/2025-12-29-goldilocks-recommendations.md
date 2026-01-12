# Goldilocks Resource Recommendations - Production Cluster

Generated: 2025-12-29 13:33:41

## Namespace: media

| Application | Target CPU | Target Memory | Lower CPU | Lower Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|-----------|--------------|----------------------|--------------------|
| booklore | 548m | 5545 MB | 15m | 1848 MB | cpu:15m mem:1848Mi | cpu:548m mem:5545Mi |
| booklore-mariadb | 49m | 362 MB | 15m | 334 MB | cpu:15m mem:334Mi | cpu:49m mem:362Mi |
| frigate | 35m | 776 MB | 22m | 523 MB | cpu:22m mem:523Mi | cpu:35m mem:776Mi |
| hydrus-client | 49m | 3877 MB | 34m | 2294 MB | cpu:34m mem:2294Mi | cpu:49m mem:3877Mi |
| hydrus-server | 0 MB | 0 MB |  |  | cpu: mem: | cpu:0 MB mem:0Mi |
| jellyfin | 15m | 2062 MB | 15m | 1567 MB | cpu:15m mem:1567Mi | cpu:15m mem:2062Mi |
| jellyseerr | 15m | 236 MB | 15m | 214 MB | cpu:15m mem:214Mi | cpu:15m mem:236Mi |
| lazylibrarian | 15m | 488 MB | 15m | 259 MB | cpu:15m mem:259Mi | cpu:15m mem:488Mi |
| lidarr | 15m | 214 MB | 15m | 214 MB | cpu:15m mem:214Mi | cpu:15m mem:214Mi |
| music-assistant | 15m | 283 MB | 15m | 283 MB | cpu:15m mem:283Mi | cpu:15m mem:283Mi |
| mylar | 15m | 104 MB | 15m | 104 MB | cpu:15m mem:104Mi | cpu:15m mem:104Mi |
| prowlarr | 15m | 174 MB | 15m | 155 MB | cpu:15m mem:155Mi | cpu:15m mem:174Mi |
| radarr | 35m | 362 MB | 22m | 334 MB | cpu:22m mem:334Mi | cpu:35m mem:362Mi |
| sabnzbd | 15m | 100 MB | 15m | 100 MB | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| sonarr | 15m | 236 MB | 15m | 236 MB | cpu:15m mem:236Mi | cpu:15m mem:236Mi |
| whisparr | 15m | 120 MB | 15m | 120 MB | cpu:15m mem:120Mi | cpu:15m mem:120Mi |

## Other Namespaces

### Namespace: auth (2 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| authentik-server | 23m | 599Mi | cpu:22m mem:599Mi | cpu:23m mem:599Mi |
| authentik-worker | 15m | 640Mi | cpu:15m mem:599Mi | cpu:15m mem:640Mi |

### Namespace: birdnet-go (1 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| birdnet-go | 23m | 174Mi | cpu:22m mem:174Mi | cpu:23m mem:174Mi |

### Namespace: databases (1 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| redis-shared | 23m | 100Mi | cpu:22m mem:100Mi | cpu:23m mem:100Mi |

### Namespace: homeassistant (1 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| homeassistant | 11m | 50Mi | cpu:10m mem:50Mi | cpu:11m mem:50Mi |

### Namespace: monitoring (13 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| alertmanager | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| goldilocks-controller | 23m | 100Mi | cpu:15m mem:100Mi | cpu:23m mem:100Mi |
| goldilocks-dashboard | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| grafana | 11m | 174Mi | cpu:10m mem:174Mi | cpu:11m mem:174Mi |
| loki | 23m | 259Mi | cpu:22m mem:236Mi | cpu:23m mem:259Mi |
| prometheus-alertmanager | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| prometheus-kube-state-metrics | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| prometheus-prometheus-node-exporter | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| prometheus-server | 49m | 2838Mi | cpu:22m mem:1656Mi | cpu:49m mem:2838Mi |
| promtail | 35m | 104Mi | cpu:34m mem:100Mi | cpu:35m mem:104Mi |
| vpa-admission-controller | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| vpa-recommender | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| vpa-updater | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |

### Namespace: mosquitto (1 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| mosquitto | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |

### Namespace: networking (3 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| adguard-home | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| netvisor-daemon | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| netvisor-server | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |

### Namespace: services (4 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| docspell-joex | 15m | 1051Mi | cpu:15m mem:1050Mi | cpu:15m mem:1051Mi |
| docspell-restserver | 15m | 488Mi | cpu:15m mem:487Mi | cpu:15m mem:488Mi |
| gluetun | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| vaultwarden | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |

### Namespace: synology-csi (2 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| synology-csi-controller | 11m | 34Mi | cpu:10m mem:34Mi | cpu:11m mem:34Mi |
| synology-csi-node | 11m | 50Mi | cpu:10m mem:50Mi | cpu:11m mem:50Mi |

### Namespace: tools (9 apps)

| Application | Target CPU | Target Memory | Recommended Requests | Recommended Limits |
|-------------|------------|---------------|----------------------|--------------------|
| argocd-image-updater-controller | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| changedetection | 11m | 137Mi | cpu:10m mem:137Mi | cpu:11m mem:137Mi |
| gitops-revision-controller | 15m | 137Mi | cpu:15m mem:137Mi | cpu:15m mem:137Mi |
| headlamp | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| homepage | 23m | 194Mi | cpu:15m mem:174Mi | cpu:23m mem:194Mi |
| linkwarden | 23m | 776Mi | cpu:15m mem:640Mi | cpu:23m mem:776Mi |
| netbox | 35m | 683Mi | cpu:22m mem:640Mi | cpu:35m mem:683Mi |
| reloader-reloader | 15m | 100Mi | cpu:15m mem:100Mi | cpu:15m mem:100Mi |
| renovate | 763m | 825Mi | cpu:203m mem:568Mi | cpu:763m mem:825Mi |

