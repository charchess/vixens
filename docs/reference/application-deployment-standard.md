# Application Deployment Standard

**Version:** 1.0
**Date:** 2026-01-05
**Status:** Mandatory for all new deployments
**Related:** [ADR-013: Layered Configuration & Disaster Recovery](../adr/013-layered-configuration-disaster-recovery.md)

---

## Purpose

Ce document définit le **standard obligatoire** pour tous les déploiements d'applications dans le cluster Vixens. Le respect de ce standard garantit:
- 🔄 **Disaster Recovery automatique** (recovery < 30min)
- 📊 **Resource management optimal** (VPA, Goldilocks)
- 🎯 **Priority-based scheduling** (apps critiques protégées)
- 🛡️ **Security & compliance** (secrets, network policies)

---

## 🚨 Mandatory Requirements

### 1. Resource Limits & Requests

**RÈGLE:** Tous les containers DOIVENT définir `resources.requests` et `resources.limits`.

#### Pourquoi?

- ❌ **Sans requests:** Pod peut être schedulé sur nœud surchargé
- ❌ **Sans limits:** Pod peut consommer toutes les ressources du nœud
- ❌ **Sans les deux:** Impossibilité d'optimiser avec VPA/Goldilocks

#### Configuration Minimale

```yaml
containers:
- name: app
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "512Mi"
      cpu: "500m"
```

#### Guidelines par Type d'Application

| Type | Requests (CPU) | Requests (Memory) | Limits (CPU) | Limits (Memory) |
|------|----------------|-------------------|--------------|-----------------|
| **Web UI léger** | 50m | 128Mi | 200m | 256Mi |
| **API backend** | 100m | 256Mi | 500m | 512Mi |
| **Base de données** | 250m | 512Mi | 1000m | 2Gi |
| **Media processing** | 500m | 1Gi | 2000m | 4Gi |
| **Sidecar (backup)** | 10m | 64Mi | 100m | 128Mi |
| **InitContainer** | 50m | 128Mi | 200m | 256Mi |

#### Cas Particuliers

**GPU workloads:**
```yaml
resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "8Gi"
    cpu: "4000m"
    # GPU via securityContext privileged, pas via resources
```

**Bursty workloads** (CPU utilisation sporadique):
```yaml
resources:
  requests:
    cpu: "100m"      # Baseline faible
    memory: "256Mi"
  limits:
    cpu: "2000m"     # Burst élevé
    memory: "512Mi"  # Memory = stable
```

---

### 2. VPA & Goldilocks Activation

**RÈGLE:** Tous les Deployments/StatefulSets DOIVENT avoir les annotations VPA/Goldilocks.

#### Pourquoi?

- 📊 Goldilocks analyse l'utilisation réelle
- 🎯 VPA recommande les valeurs optimales
- 💰 Économie de ressources (over-provisioning évité)
- 🚀 Performance améliorée (under-provisioning détecté)

#### Annotations Obligatoires

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  labels:
    app.kubernetes.io/name: app
  annotations:
    # VPA: Mode recommandation (JAMAIS auto sur prod!)
    vpa.kubernetes.io/updateMode: "Off"

    # Goldilocks: Analyse activée
    goldilocks.fairwinds.com/enabled: "true"

    # Goldilocks: VPA update mode (Off = recommandation seulement)
    goldilocks.fairwinds.com/vpa-update-mode: "off"
spec:
  template:
    metadata:
      labels:
        app.kubernetes.io/name: app
```

#### Workflow d'Optimisation

```bash
# 1. Déployer avec estimates initiaux
kubectl apply -f app/

# 2. Attendre 24-48h (Goldilocks collecte métriques)

# 3. Consulter dashboard Goldilocks
# https://goldilocks.dev.truxonline.com

# 4. Appliquer recommandations VPA
# Copier valeurs depuis dashboard → kustomization.yaml

# 5. Redéployer avec valeurs optimisées
kubectl apply -f app/

# 6. Répéter tous les 3 mois (workload évolue)
```

#### Exceptions

**VPA désactivé si:**
- Application avec HPA (Horizontal Pod Autoscaler)
- Workload avec resource requirements fixes (GPU, huge pages)
- Apps legacy sans métriques Prometheus

---

### 3. Priority Classes

**RÈGLE:** Tous les Pods DOIVENT définir une `priorityClassName`.

#### Pourquoi?

- 🎯 **Eviction prévisible:** Apps critiques protégées
- 🚀 **Scheduling optimal:** Haute priorité = placement préférentiel
- 💥 **Disaster scenario:** Ressources limitées = apps critiques survivent

#### Priority Classes Disponibles

```yaml
# Infrastructure critique (cluster-critical)
priorityClassName: system-cluster-critical  # Reserved for k8s components

# Applications critiques niveau 1 (high-priority)
priorityClassName: high-priority            # Production apps critiques
# Exemples: HomeAssistant, Mosquitto, DNS, Auth

# Applications standard (medium-priority)
priorityClassName: medium-priority          # Production apps standard
# Exemples: Jellyfin, Frigate, Immich

# Applications best-effort (low-priority)
priorityClassName: low-priority             # Dev, test, batch jobs
# Exemples: Renovate, backup jobs, CI/CD
```

#### Définitions (créées dans cluster)

```yaml
# high-priority
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000000
globalDefault: false
description: "Critical production applications"

---
# medium-priority (DEFAULT)
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: medium-priority
value: 500000
globalDefault: true
description: "Standard production applications"

---
# low-priority
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: low-priority
value: 100000
globalDefault: false
description: "Best-effort workloads"
```

#### Assignment Guidelines

**High Priority (1M):**
- Infrastructure critique: DNS, Auth (Authentik), MQTT
- Applications domotique: HomeAssistant, Node-RED
- Monitoring: Prometheus, Grafana, Alertmanager

**Medium Priority (500k) - DEFAULT:**
- Applications média: Jellyfin, Immich, Frigate
- Services utilisateur: Nextcloud, Paperless-NGX
- Databases: PostgreSQL, Redis

**Low Priority (100k):**
- Batch jobs: Renovate, backup jobs
- CI/CD: Workflows GitHub Actions
- Development: Apps de test

#### Configuration

```yaml
# apps/homeassistant/base/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: homeassistant
spec:
  template:
    spec:
      priorityClassName: high-priority  # CRITIQUE: domotique
      containers:
      - name: homeassistant
        # ...
```

---

### 4. Layered Configuration & Disaster Recovery

**RÈGLE:** Applications avec configuration persistante DOIVENT implémenter le pattern layered configuration.

#### Pattern Overview

Voir [ADR-013](../adr/013-layered-configuration-disaster-recovery.md) pour détails complets.

**3 Tiers obligatoires:**

1. **Tier 1: Configuration Statique (Git)**
   - ConfigMap avec base vanilla fonctionnelle
   - Reverse proxy, network settings
   - Zero secrets

2. **Tier 2: Configuration Dynamique (Backup)**
   - Full config utilisateur depuis MinIO/S3
   - Contient secrets/info personnelle OK
   - Restore automatique au boot

3. **Tier 3: État Applicatif (Backup Continu)**
   - SQLite → Litestream sidecar
   - Config files → rclone sidecar (5min interval)
   - Media files → NFS direct

#### Implémentation Minimale

```yaml
# ConfigMap vanilla
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-vanilla
data:
  config.yaml: |
    # Configuration de base fonctionnelle
    # ...

---
# Deployment avec InitContainer
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      initContainers:
      - name: init-config
        image: alpine:latest
        command: ["/scripts/init.sh"]
        volumeMounts:
        - name: init-script
          mountPath: /scripts
        - name: vanilla-config
          mountPath: /defaults
        - name: config-pvc
          mountPath: /config

      containers:
      - name: app
        volumeMounts:
        - name: config-pvc
          mountPath: /config

      - name: backup-sidecar
        image: rclone/rclone:latest
        command: ["/scripts/backup.sh"]
        volumeMounts:
        - name: config-pvc
          mountPath: /config
          readOnly: true

      volumes:
      - name: vanilla-config
        configMap:
          name: app-vanilla
      - name: config-pvc
        persistentVolumeClaim:
          claimName: app-config
```

#### Quand Appliquer?

**Obligatoire pour:**
- Applications avec config modifiable (HomeAssistant, Frigate)
- Applications avec secrets intégrés (RTSP URLs, API keys)
- Applications avec état critique (bases de données)

**Optionnel pour:**
- Applications stateless (Whoami, simple web servers)
- Applications 100% configurées par env vars
- Applications read-only config (Traefik depuis values.yaml)

---

### 5. Security & Best Practices

#### Network Policies (Recommandé)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: app-netpol
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: app
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: traefik
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 53  # DNS
```

#### Pod Security Standards (Obligatoire)

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: app
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true  # Si possible
          capabilities:
            drop:
            - ALL
```

**Exceptions (avec justification):**
- `privileged: true` → GPU access (Frigate, Jellyfin)
- `runAsUser: 0` → Legacy apps (documenter why)

#### Secrets Management

**INTERDIT:**
```yaml
# ❌ Secrets en clair dans Git
env:
- name: API_KEY
  value: "sk-1234567890abcdef"
```

**OBLIGATOIRE:**
```yaml
# ✅ Secrets depuis Infisical
env:
- name: API_KEY
  valueFrom:
    secretKeyRef:
      name: app-secrets  # InfisicalSecret
      key: api-key
```

---

### 6. Deployment Strategy

#### Strategy Type

```yaml
spec:
  strategy:
    # RollingUpdate (DEFAULT) - Pour apps multi-replicas
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1

  # OU

  strategy:
    # Recreate - Pour apps avec PVC RWO (ReadWriteOnce)
    type: Recreate
```

**Choisir Recreate si:**
- PVC avec accessMode: ReadWriteOnce
- Application non-compatible multi-instances
- Base de données SQLite locale

#### Replicas

```yaml
spec:
  replicas: 1  # DEFAULT pour apps avec état

  # OU

  replicas: 3  # Pour apps stateless critiques
```

**High Availability (HA):**
- Apps critiques sans état → 3 replicas
- Databases → 1 replica (ou cluster HA dédié)
- Media apps → 1 replica (PVC RWO limitation)

---

### 7. Health Checks

**OBLIGATOIRE:** Tous les containers doivent définir liveness & readiness probes.

```yaml
containers:
- name: app
  livenessProbe:
    httpGet:
      path: /healthz
      port: 8080
    initialDelaySeconds: 30
    periodSeconds: 10
    timeoutSeconds: 5
    failureThreshold: 3

  readinessProbe:
    httpGet:
      path: /ready
      port: 8080
    initialDelaySeconds: 10
    periodSeconds: 5
    timeoutSeconds: 3
    failureThreshold: 3
```

**Alternatives:**
- `exec:` pour apps sans HTTP endpoint
- `tcpSocket:` pour databases
- `grpc:` pour gRPC services

---

### 8. Labels & Annotations

#### Labels Obligatoires (Kubernetes Recommended)

```yaml
metadata:
  labels:
    app.kubernetes.io/name: app-name
    app.kubernetes.io/instance: app-name-dev
    app.kubernetes.io/version: "1.2.3"
    app.kubernetes.io/component: backend
    app.kubernetes.io/part-of: system-name
    app.kubernetes.io/managed-by: argocd
```

#### Annotations Standards

```yaml
metadata:
  annotations:
    # ArgoCD sync wave
    argocd.argoproj.io/sync-wave: "0"

    # VPA/Goldilocks (voir section 2)
    goldilocks.fairwinds.com/enabled: "true"
    vpa.kubernetes.io/updateMode: "Off"

    # Prometheus scraping (si applicable)
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    prometheus.io/path: "/metrics"
```

---

### 9. Storage (PVC)

#### Access Modes

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-data
spec:
  accessModes:
  - ReadWriteOnce  # RWO - 1 node only (DEFAULT)
  # OU
  - ReadWriteMany  # RWX - Multiple nodes (NFS)

  resources:
    requests:
      storage: 10Gi

  storageClassName: synology-iscsi-retain  # Prod
  # OU
  storageClassName: synology-iscsi-delete  # Dev
```

#### Storage Classes

| Environment | StorageClass | Reclaim Policy | Usage |
|-------------|--------------|----------------|-------|
| **Prod** | `synology-iscsi-retain` | Retain | Data critique |
| **Dev** | `synology-iscsi-delete` | Delete | Dev/Test |
| **Shared** | `nfs-storage` | Retain | Fichiers partagés |

#### Sizing Guidelines

```yaml
# Config (petite)
storage: 1Gi      # HomeAssistant config, Mosquitto config

# Databases (moyenne)
storage: 10Gi     # PostgreSQL, Redis

# Media cache (grande)
storage: 100Gi    # Frigate clips, Jellyfin cache

# Media library (très grande)
# → NFS direct (pas de PVC)
```

---

### 10. Tolerations (Control Plane)

**OBLIGATOIRE pour apps d'infrastructure** déployées sur control plane.

```yaml
spec:
  template:
    spec:
      tolerations:
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule

      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
```

**Apps concernées:**
- Cilium (CNI)
- CoreDNS
- Monitoring (Prometheus, Grafana)
- Infisical Operator
- cert-manager

---

## 📋 Deployment Checklist

Avant de déployer une nouvelle application, vérifier:

### Configuration
- [ ] `resources.requests` définis (CPU + Memory)
- [ ] `resources.limits` définis (CPU + Memory)
- [ ] `priorityClassName` assigné (high/medium/low)
- [ ] Annotations VPA/Goldilocks présentes
- [ ] Labels Kubernetes recommended appliqués

### Security
- [ ] Secrets via InfisicalSecret (pas en clair)
- [ ] `securityContext` configuré (runAsNonRoot)
- [ ] Network Policy créée (si nécessaire)
- [ ] Image tag fixe (pas `:latest`)

### Reliability
- [ ] Health checks définis (liveness + readiness)
- [ ] Strategy appropriée (RollingUpdate vs Recreate)
- [ ] PVC storageClass correct (retain vs delete)
- [ ] Backup/restore pattern implémenté (si config persistante)

### Observability
- [ ] Prometheus metrics endpoint (si applicable)
- [ ] Logs structurés (JSON preferred)
- [ ] Documentation créée dans `docs/applications/<category>/<app>.md`

### GitOps
- [ ] ArgoCD sync-wave configurée
- [ ] Kustomize overlays (dev, prod)
- [ ] ConfigMap vanilla créé (Tier 1)
- [ ] Testé en dev avant prod

---

## 🧪 Testing & Validation

### Disaster Recovery Test

**Fréquence:** Mensuel (cluster dev)

```bash
# 1. Prendre note de l'état actuel
kubectl -n app get all
kubectl -n app exec deploy/app -- ls -la /config

# 2. Détruire le PVC
kubectl -n app delete pvc app-config

# 3. Redéployer (ArgoCD sync ou kubectl apply)
argocd app sync app

# 4. Vérifier recovery automatique
kubectl -n app get pods -w
kubectl -n app exec deploy/app -- ls -la /config

# 5. Valider fonctionnement
curl https://app.dev.truxonline.com
```

**Success criteria:**
- ✅ Pod démarre sans erreur
- ✅ Config restaurée depuis backup
- ✅ Application fonctionnelle
- ✅ Pas d'intervention manuelle

### Resource Optimization Test

**Fréquence:** Trimestriel

```bash
# 1. Consulter Goldilocks dashboard
# https://goldilocks.dev.truxonline.com

# 2. Comparer requests actuels vs recommandés
# QoS: Guaranteed, Burstable, ou BestEffort?

# 3. Identifier over/under provisioning
# Over: requests >> utilisation réelle
# Under: limits atteints régulièrement

# 4. Ajuster valeurs dans kustomization.yaml

# 5. Redéployer et monitorer 7 jours
```

---

## 📚 Templates & Examples

### Template Complet

Voir: `docs/templates/application-deployment-template.yaml`

### Exemples Réels

- **HomeAssistant:** `apps/homeassistant/` (Layered config, High priority, GPU)
- **Frigate:** `apps/20-media/frigate/` (Backup/restore, Medium priority, GPU)
- **Mosquitto:** `apps/mosquitto/` (Persistence DB, High priority)
- **Whoami:** `apps/whoami/` (Minimal stateless, Low priority)

---

## 🔗 References

- [ADR-013: Layered Configuration & Disaster Recovery](../adr/013-layered-configuration-disaster-recovery.md)
- [Guide: Adding a New Application](../guides/adding-new-application.md)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [VPA Documentation](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
- [Goldilocks](https://goldilocks.docs.fairwinds.com/)

---

## 🔄 Maintenance

**Document Owner:** Infrastructure Team
**Review Frequency:** Quarterly
**Last Updated:** 2026-01-05
**Next Review:** 2026-04-05

**Change Log:**
- 2026-01-05: Initial version (post-incident 2026-01-05)
