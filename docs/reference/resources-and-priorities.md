# Resource & Priority Reference Map (Prod)

**Last Updated:** 2026-01-04

## 🚨 Critical Issues

### Blocked Pods (8 total)

| Namespace | Pod | Status | Duration | Root Cause |
|-----------|-----|--------|----------|------------|
| cert-manager | cert-manager | **OutOfcpu** | 13h | Node poison saturated (98% CPU requests) |
| vpa | vpa-updater | **OutOfcpu** | 13h | Node poison saturated (98% CPU requests) |
| networking | external-dns-unifi | **OutOfmemory** | 13h | Node poison saturated (91% RAM requests) |
| tools | changedetection | **OutOfmemory** | 13h | Node poison saturated (91% RAM requests) |
| tools | headlamp | **OutOfmemory** | 13h | Node poison saturated (91% RAM requests) |
| cert-manager | cert-manager-webhook | Error | 13h | Dependent on cert-manager (OutOfcpu) |
| birdnet-go | birdnet-go | Init:ContainerStatusUnknown | 13h | Investigation needed |
| downloads | amule | Init:ContainerStatusUnknown | 13h | Investigation needed |

### Node Resource Allocation

| Node | CPU Allocatable | CPU Requests | CPU % | Memory Allocatable | Memory Requests | Memory % | Status |
|------|-----------------|--------------|-------|--------------------|-----------------|----------|--------|
| **poison** | 3950m | **3884m** | **98%** ⚠️ | 15138Mi | **13808Mi** | **91%** ⚠️ | **CRITICAL** |
| **powder** | 3950m | 3117m | 78% | 15100Mi | 8703Mi | 57% | OK |
| **phoebe** | 3950m | 2164m | 54% | 7178Mi | **5303Mi** | **73%** ⚠️ | WARNING |

**Note**: phoebe has ~50% less RAM than poison/powder

### CPU Limits Over-Allocation

| Node | CPU Limits | Over-Allocation |
|------|------------|-----------------|
| **poison** | **20272m** | **513%** (5.1x capacity) |
| **powder** | 12273m | 310% (3.1x capacity) |
| **phoebe** | 8800m | 222% (2.2x capacity) |

### Memory Limits Over-Allocation

| Node | Memory Limits | Over-Allocation |
|------|---------------|-----------------|
| **poison** | **27976Mi** | **185%** (1.85x capacity) |
| **phoebe** | 12338Mi | 172% (1.72x capacity) |
| **powder** | 13680Mi | 90% (within capacity) |

## 📊 Top Resource Consumers

### Top 10 CPU Requests

| Rank | Namespace | Pod | CPU Request | % of Node | Node |
|------|-----------|-----|-------------|-----------|------|
| 1 | media | **frigate** | **2406m** | **60%** | powder |
| 2 | traefik | traefik (replica 1) | 500m | 12% | poison |
| 3 | traefik | traefik (replica 2) | 500m | 12% | phoebe |
| 4 | traefik | traefik (replica 3) | 500m | 12% | poison |
| 5 | monitoring | prometheus-server | 300m | 7% | phoebe |
| 6 | kube-system | kube-apiserver (x3) | 200m each | 5% each | all nodes |
| 7 | birdnet-go | birdnet-go (x2) | 200m each | 5% each | phoebe, poison |
| 8 | auth | authentik-server | 200m | 5% | poison |
| 9 | homeassistant | homeassistant | 154m | 3% | poison |
| 10 | media | booklore | 100m | 2% | powder |

### Top 10 Memory Requests

| Rank | Namespace | Pod | Memory Request | % of Node | Node |
|------|-----------|-----|----------------|-----------|------|
| 1 | media | **frigate** | **3682Mi** | **24%** | powder |
| 2 | media | hydrus-client | 1953Mi | 12% | powder |
| 3 | homeassistant | homeassistant | 2017Mi | 13% | poison |
| 4 | monitoring | prometheus-server | 1Gi | 14% | phoebe |
| 5 | services | docspell-joex | 1Gi | 6% | poison |
| 6 | media | booklore | 1Gi | 6% | powder |
| 7 | tools | linkwarden | 512Mi | 7% | phoebe |
| 8 | media | jellyfin | 825Mi | 5% | poison |
| 9 | tools | netbox | 640Mi | 4% | poison |
| 10 | media | radarr | 560Mi | 3% | poison |

### Pods Without Resource Requests (30+ pods)

**Critical Infrastructure (needs immediate attention):**
- **argocd** (6 pods): application-controller, applicationset-controller, notifications-controller, redis, repo-server, server
- **cilium** (5 pods): operator, envoy (x3), hubble-relay
- **monitoring** (4 pods): prometheus node-exporter (x3), grafana
- **synology-csi** (4 pods): controller (4 containers), node (2 containers per node)

**Other:**
- auth/authentik-worker
- databases/redis-shared
- networking/netvisor (daemon, server)
- tools/renovate (cronjobs)
- whoami

## 📋 Detailed Resource Map

| Namespace | App | Container | Req CPU | Req RAM | Lim CPU | Lim RAM | Status | Current Prio | Rec. Prio |
|-----------|-----|-----------|---------|---------|---------|---------|--------|--------------|-----------|
| argocd | argocd-application-controller | application-controller | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| argocd | argocd-applicationset-controller | applicationset-controller | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| argocd | argocd-notifications-controller | notifications-controller | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| argocd | argocd-redis | redis | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| argocd | argocd-repo-server | repo-server | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| argocd | argocd-server | server | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| auth | authentik-server | authentik-server | 200m | 512Mi | 1072m | 3058Mi | Running | homelab-critical | High (Prod-Critical) |
| auth | authentik-worker | authentik-worker | **0** | **0** | 0 | 0 | Running | homelab-critical | High (Prod-Critical) |
| birdnet-go | birdnet-go | birdnet-go | 200m | 256Mi | 1000m | 1Gi | **Init:Unknown** | default (0) | Medium (Prod-Standard) |
| cert-manager | cert-manager | cert-manager-controller | 100m | 128Mi | 500m | 512Mi | **OutOfcpu** | default (0) | High (Prod-Critical) |
| cert-manager | cert-manager-cainjector | cert-manager-cainjector | 50m | 64Mi | 200m | 256Mi | Running | default (0) | High (Prod-Critical) |
| cert-manager | cert-manager-webhook | cert-manager-webhook | 50m | 64Mi | 200m | 256Mi | **Error** | default (0) | High (Prod-Critical) |
| cert-manager | cert-manager-webhook-gandi | cert-manager-webhook-gandi | 50m | 64Mi | 200m | 256Mi | Running | default (0) | High (Prod-Critical) |
| databases | postgresql-shared | postgres | 100m | 256Mi | 500m | 512Mi | Running | default (0) | High (Prod-Critical) |
| databases | redis-shared | redis | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| downloads | amule | amule | 50m | 128Mi | 0 | 512Mi | **Init:Unknown** | default (0) | Low (Prod-BestEffort) |
| downloads | pyload | pyload | 50m | 128Mi | 0 | 512Mi | Running | default (0) | Low (Prod-BestEffort) |
| downloads | qbittorrent | qbittorrent | 50m | 256Mi | 0 | 512Mi | Running | default (0) | Low (Prod-BestEffort) |
| homeassistant | homeassistant | filebrowser | 11m | 64Mi | 500m | 128Mi | Running | homelab-critical | Medium (Prod-Standard) |
| homeassistant | homeassistant | homeassistant | 154m | 2017Mi | 1072m | 3058Mi | Running | homelab-critical | Medium (Prod-Standard) |
| infisical-operator-system | infisical-opera-controller-manager | manager | 100m | 128Mi | 500m | 256Mi | Running | default (0) | Medium (Prod-Standard) |
| kube-system | cilium | cilium-agent | **0** | 10Mi | 0 | 0 | Running | system-node-critical | High (Prod-Critical) |
| kube-system | cilium-envoy | cilium-envoy | **0** | **0** | 0 | 0 | Running | system-node-critical | High (Prod-Critical) |
| kube-system | cilium-operator | cilium-operator | **0** | **0** | 0 | 0 | Running | system-cluster-critical | High (Prod-Critical) |
| kube-system | coredns | coredns | 100m | 70Mi | 200m | 170Mi | Running | system-cluster-critical | High (Prod-Critical) |
| kube-system | hubble-relay | hubble-relay | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| kube-system | kube-apiserver | kube-apiserver | 200m | 512Mi | 0 | 0 | Running | system-cluster-critical | High (Prod-Critical) |
| kube-system | kube-controller-manager | kube-controller-manager | 50m | 256Mi | 0 | 0 | Running | system-cluster-critical | High (Prod-Critical) |
| kube-system | kube-scheduler | kube-scheduler | 10m | 64Mi | 0 | 0 | Running | system-cluster-critical | High (Prod-Critical) |
| kube-system | metrics-server | metrics-server | 100m | 200Mi | 500m | 500Mi | Running | system-cluster-critical | High (Prod-Critical) |
| mealie | mealie | mealie | 100m | 256Mi | 500m | 512Mi | Running | default (0) | Medium (Prod-Standard) |
| media | booklore | booklore | 100m | 1Gi | 1000m | 2Gi | Running | default (0) | Low (Prod-BestEffort) |
| media | booklore-mariadb | mariadb | 15m | 334Mi | 49m | 362Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | frigate | frigate | **2406m** | **3682Mi** | **9624m** | **5524Mi** | Running | homelab-important | Low (Prod-BestEffort) |
| media | hydrus-client | hydrus-client | 63m | 1953Mi | 500m | 2930Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | jellyfin | jellyfin | 23m | 825Mi | 500m | 1689Mi | Running | homelab-important | Low (Prod-BestEffort) |
| media | jellyseerr | jellyseerr | 78m | 422Mi | 500m | 633Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | lazylibrarian | lazylibrarian | 23m | 174Mi | 500m | 261Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | lidarr | lidarr | 11m | 236Mi | 500m | 354Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | music-assistant | music-assistant | 11m | 214Mi | 500m | 322Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | mylar | mylar | 11m | 128Mi | 500m | 192Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | prowlarr | prowlarr | 11m | 194Mi | 500m | 291Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | radarr | radarr | 49m | 560Mi | 500m | 840Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | sabnzbd | sabnzbd | 23m | **512Mi** | 500m | **2Gi** | Running ✅ | default (0) | Low (Prod-BestEffort) |
| media | sonarr | sonarr | 23m | 259Mi | 500m | 389Mi | Running | default (0) | Low (Prod-BestEffort) |
| media | whisparr | whisparr | 11m | 137Mi | 500m | 206Mi | Running | default (0) | Low (Prod-BestEffort) |
| monitoring | goldilocks-controller | goldilocks | 25m | 256Mi | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| monitoring | goldilocks-dashboard | goldilocks | 25m | 256Mi | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| monitoring | grafana | grafana | **0** | **0** | 0 | 512Mi | Running | default (0) | High (Prod-Critical) |
| monitoring | loki | loki | 100m | 256Mi | 500m | 1Gi | Running | default (0) | High (Prod-Critical) |
| monitoring | prometheus-alertmanager | alertmanager | 100m | 128Mi | 0 | 512Mi | Running | default (0) | High (Prod-Critical) |
| monitoring | prometheus-kube-state-metrics | kube-state-metrics | 100m | 128Mi | 200m | 256Mi | Running | default (0) | High (Prod-Critical) |
| monitoring | prometheus-node-exporter | node-exporter | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| monitoring | prometheus-server | prometheus-server | 300m | 1Gi | 0 | 2Gi | Running | default (0) | High (Prod-Critical) |
| monitoring | promtail | promtail | 50m | 100Mi | 100m | 256Mi | Running | default (0) | High (Prod-Critical) |
| mosquitto | mosquitto | mosquitto | 50m | 64Mi | 200m | 256Mi | Running | default (0) | Medium (Prod-Standard) |
| networking | adguard-home | adguard-home | 11m | 128Mi | 500m | 192Mi | Running | homelab-critical | High (Prod-Critical) |
| networking | external-dns-gandi | external-dns | 20m | 64Mi | 100m | 128Mi | Running | default (0) | High (Prod-Critical) |
| networking | external-dns-unifi | external-dns | 20m | 64Mi | 100m | 128Mi | **OutOfmemory** | default (0) | High (Prod-Critical) |
| networking | external-dns-unifi | unifi-webhook | **0** | **0** | 0 | 0 | **OutOfmemory** | default (0) | High (Prod-Critical) |
| networking | netvisor-daemon | daemon | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| networking | netvisor-server | server | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| services | docspell-joex | joex | 100m | 1Gi | 1000m | 1536Mi | Running | default (0) | Medium (Prod-Standard) |
| services | docspell-restserver | restserver | 100m | 512Mi | 1000m | 1Gi | Running | default (0) | Medium (Prod-Standard) |
| services | gluetun | gluetun | 11m | 128Mi | 500m | 192Mi | Running | default (0) | Medium (Prod-Standard) |
| services | vaultwarden | vaultwarden | 11m | 128Mi | 500m | 192Mi | Running | homelab-important | Medium (Prod-Standard) |
| synology-csi | synology-csi-controller | csi-provisioner | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| synology-csi | synology-csi-controller | csi-attacher | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| synology-csi | synology-csi-controller | csi-resizer | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| synology-csi | synology-csi-controller | synology-csi-plugin | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| synology-csi | synology-csi-node | csi-node-driver-registrar | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| synology-csi | synology-csi-node | synology-csi-plugin | **0** | **0** | 0 | 0 | Running | default (0) | High (Prod-Critical) |
| tools | changedetection | browserless | 11m | 120Mi | 1000m | 256Mi | **OutOfmemory** | default (0) | Medium (Prod-Standard) |
| tools | changedetection | changedetection | 11m | 89Mi | 1000m | 128Mi | **OutOfmemory** | default (0) | Medium (Prod-Standard) |
| tools | gitops-revision-controller | controller | 50m | 128Mi | 200m | 512Mi | Running | default (0) | Medium (Prod-Standard) |
| tools | headlamp | headlamp | 11m | 128Mi | 500m | 192Mi | **OutOfmemory** | default (0) | Medium (Prod-Standard) |
| tools | homepage | homepage | 11m | 214Mi | 500m | 394Mi | Running | default (0) | Medium (Prod-Standard) |
| tools | it-tools | it-tools | 10m | 32Mi | 100m | 128Mi | Running | default (0) | Medium (Prod-Standard) |
| tools | linkwarden | linkwarden | 100m | 512Mi | 1000m | 2Gi | Running | default (0) | Medium (Prod-Standard) |
| tools | netbox | netbox | 11m | 640Mi | 500m | 1173Mi | Running | default (0) | Medium (Prod-Standard) |
| tools | reloader-reloader | reloader-reloader | 10m | 128Mi | 100m | 256Mi | Running | default (0) | Medium (Prod-Standard) |
| tools | stirling-pdf | stirling-pdf-chart | 100m | 256Mi | 1000m | 1Gi | Running | default (0) | Medium (Prod-Standard) |
| traefik | traefik | traefik | 500m | 512Mi | 2000m | 2Gi | Running | homelab-critical | High (Prod-Critical) |
| vpa | vpa-vertical-pod-autoscaler-admission-controller | admission-controller | 50m | 100Mi | 200m | 500Mi | Running | default (0) | High (Prod-Critical) |
| vpa | vpa-vertical-pod-autoscaler-recommender | recommender | 50m | 250Mi | 200m | 1Gi | Running | default (0) | High (Prod-Critical) |
| vpa | vpa-vertical-pod-autoscaler-updater | updater | 50m | 100Mi | 200m | 500Mi | **OutOfcpu** | default (0) | High (Prod-Critical) |
| whoami | whoami | whoami | **0** | **0** | 0 | 0 | Running | default (0) | Medium (Prod-Standard) |

## 🎯 Immediate Actions Required

### 1. Critical - Unblock Poison Node (98% CPU, 91% RAM)

**Move heavy workloads off poison:**
- Move 1x traefik replica to phoebe/powder (frees 500m CPU, 512Mi RAM)
- Move homeassistant to phoebe (frees 154m CPU, 2017Mi RAM)

**Total freed on poison**: 654m CPU (16%), 2529Mi RAM (16%)

### 2. Critical - Define Resource Requests for Infrastructure Pods

**Priority 1 (Critical Infrastructure - must have requests):**
- ArgoCD (6 pods) - recommend: 50m CPU, 128Mi RAM each
- Cilium operator - recommend: 50m CPU, 128Mi RAM
- Cilium envoy (x3) - recommend: 10m CPU, 64Mi RAM each
- Hubble relay - recommend: 10m CPU, 64Mi RAM
- Synology CSI controller (4 containers) - recommend: 10m CPU, 32Mi RAM each
- Synology CSI node (2 containers per node) - recommend: 10m CPU, 64Mi RAM each

**Priority 2 (Monitoring):**
- Prometheus node-exporter (x3) - recommend: 10m CPU, 64Mi RAM each
- Grafana - recommend: 50m CPU, 128Mi RAM

### 3. High Priority - Optimize Over-Provisioned Apps

**Frigate** (most critical):
- Current: 2406m CPU request, 3682Mi RAM
- Investigation needed: Is 2406m CPU (60% of a node) justified?
- Recommendation: Analyze actual usage and reduce to realistic values

**Docspell**:
- Current: joex 100m/1Gi, restserver 100m/512Mi
- Investigation needed: Are 1Gi+ memory requests justified?

### 4. Medium Priority - Implement PriorityClasses

Create 4 priority tiers:
1. **Prod-Critical** (1000): kube-system, argocd, traefik, monitoring, networking
2. **Prod-Standard** (500): homeassistant, databases, services
3. **Prod-BestEffort** (100): media apps, downloads, tools
4. **Prod-Batch** (0): cronjobs, temporary workloads

### 5. Medium Priority - Enable Resource Autoscaling

Use Goldilocks VPA recommendations to right-size:
- Media apps (overprovisioned with 500m CPU limits for 11m usage)
- Tools apps (many with 1Gi limits but <256Mi actual usage)

## 📝 Notes

- **sabnzbd fixed**: Memory increased from 354Mi → 2Gi to resolve OOMKilled (2026-01-04)
- **phoebe constraint**: Only 7Gi RAM vs 15Gi on other nodes - avoid memory-intensive workloads
- **poison saturation**: Most heavily loaded node - needs immediate rebalancing
- **Over-allocation acceptable**: CPU/memory limits can exceed capacity as k8s uses QoS classes for eviction
