# Guide: Backup/Restore Pattern Implementation

**Version:** 1.0
**Date:** 2026-01-05
**Related:**
- [ADR-013: Layered Configuration & Disaster Recovery](../adr/013-layered-configuration-disaster-recovery.md)
- [Reference: Application Deployment Standard](../reference/application-deployment-standard.md)

---

## Overview

Ce guide explique comment implémenter le **pattern backup/restore** pour garantir la récupération automatique des applications après un incident (cluster reset, PVC loss, node failure).

**Objectifs:**
- 🔄 Recovery automatique en < 30 minutes
- 📊 RPO < 5 minutes (config) / < 1 minute (databases)
- 🎯 Zero intervention manuelle
- 🛡️ Zero secrets dans Git

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Architecture Pattern](#architecture-pattern)
3. [Implementation Steps](#implementation-steps)
4. [Code Templates](#code-templates)
5. [Testing & Validation](#testing--validation)
6. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

- MinIO déployé et accessible
- Credentials MinIO dans Infisical
- Application avec configuration persistante

### Basic Implementation (5 steps)

```bash
# 1. Créer ConfigMap vanilla base
# 2. Créer InitContainer script
# 3. Créer Backup CronJob
# 4. Créer Restore PreSync Job
# 5. Tester le cycle complet
```

**Temps estimé:** 2-3 heures par application

---

## Architecture Pattern

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│ Git Repository (apps/<app>/)                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ┌────────────────────┐  ┌────────────────────┐                │
│ │ ConfigMap          │  │ Restore Job        │                │
│ │ (Vanilla Config)   │  │ (ArgoCD PreSync)   │                │
│ └────────────────────┘  └────────────────────┘                │
│                                                                 │
│ ┌────────────────────┐  ┌────────────────────┐                │
│ │ Deployment         │  │ Backup CronJob     │                │
│ │ + InitContainer    │  │ (Every 5 min)      │                │
│ │ + Sidecar          │  │                    │                │
│ └────────────────────┘  └────────────────────┘                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │ MinIO/S3            │
                    │ (Backup Storage)    │
                    │                     │
                    │ /app/config/        │
                    │ /app/database/      │
                    └─────────────────────┘
```

### Data Flow

**Normal Operation:**
```
App writes config → PVC → Sidecar backup (5min) → MinIO
```

**Disaster Recovery:**
```
ArgoCD PreSync → Restore Job → Fetch MinIO → EmptyDir
    ↓
InitContainer → Layer 1 (Vanilla) → PVC
    ↓
InitContainer → Layer 2 (Restore) → Override PVC
    ↓
App starts → Full config available
```

---

## Implementation Steps

### Step 1: Analyze Application Data

**Identifier les 3 tiers:**

```bash
# Examiner les fichiers de l'app
kubectl exec -n <namespace> deploy/<app> -- ls -la /config

# Classifier chaque fichier/dossier:
# - Tier 1 (Git): Config statique, pas de secrets
# - Tier 2 (Backup): Config dynamique, secrets intégrés
# - Tier 3 (Backup continu): Databases, cache
```

**Exemple HomeAssistant:**

| Fichier | Tier | Raison |
|---------|------|--------|
| `configuration.yaml` | 2 | Secrets + info personnelle |
| `automations.yaml` | 2 | Créé par utilisateur |
| `scripts.yaml` | 2 | Créé par utilisateur |
| `home-assistant_v2.db` | 3 | État temps réel |
| `.storage/` | 2 | Configuration UI |

**Exemple Frigate:**

| Fichier | Tier | Raison |
|---------|------|--------|
| `config.yml` | 2 | RTSP URLs avec credentials |
| `clips/` | 3 | Cache temporaire (NFS mieux) |
| `recordings/` | - | NFS direct (volume externe) |

### Step 2: Create Vanilla ConfigMap

**Objectif:** Config minimale fonctionnelle pour premier boot.

```yaml
# apps/<app>/base/configmap-vanilla.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-vanilla-config
  namespace: app-namespace
data:
  configuration.yaml: |
    # Configuration de base SAFE pour Git
    # PAS de secrets, PAS d'info personnelle

    app:
      name: "Application"

    http:
      # Reverse proxy config (TOUJOURS nécessaire)
      use_x_forwarded_for: true
      trusted_proxies:
        - 192.168.201.0/24  # Traefik prod
        - 192.168.208.0/24  # Traefik dev

    # Structure minimale fonctionnelle
    # (reste sera restauré depuis backup)
```

**Validation:**
```bash
# Config vanilla doit permettre à l'app de démarrer
# Même sans backup disponible
kubectl apply -f configmap-vanilla.yaml
```

### Step 3: Create rclone Configuration

**Secret pour accès MinIO:**

```yaml
# apps/<app>/base/secret-rclone.yaml
apiVersion: infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: rclone-minio-config
  namespace: app-namespace
spec:
  hostAPI: http://infisical.infisical.svc:8085
  resyncInterval: 60
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens
        envSlug: prod  # ou dev
        secretsPath: /apps/<app>/backup
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: argocd
  managedSecretReference:
    secretName: rclone-minio-config
    secretType: Opaque
```

**Dans Infisical (`/apps/<app>/backup`):**
```
RCLONE_CONFIG_MINIO_TYPE=s3
RCLONE_CONFIG_MINIO_PROVIDER=Minio
RCLONE_CONFIG_MINIO_ENV_AUTH=false
RCLONE_CONFIG_MINIO_ACCESS_KEY_ID=<minio-access-key>
RCLONE_CONFIG_MINIO_SECRET_ACCESS_KEY=<minio-secret-key>
RCLONE_CONFIG_MINIO_ENDPOINT=http://minio.minio.svc:9000
```

**Alternative (rclone.conf file):**

```ini
[minio]
type = s3
provider = Minio
env_auth = false
access_key_id = <from-secret>
secret_access_key = <from-secret>
endpoint = http://minio.minio.svc:9000
```

### Step 4: Create Backup CronJob

**Continuous backup vers MinIO:**

```yaml
# apps/<app>/base/backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: app-backup
  namespace: app-namespace
  labels:
    app.kubernetes.io/name: app
    app.kubernetes.io/component: backup
spec:
  schedule: "*/5 * * * *"  # Toutes les 5 minutes
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            app.kubernetes.io/name: app
            app.kubernetes.io/component: backup
        spec:
          restartPolicy: OnFailure

          containers:
          - name: rclone-backup
            image: rclone/rclone:latest
            command:
            - /bin/sh
            - -c
            - |
              set -e
              echo "=== Starting backup $(date) ==="

              # Sync vers MinIO (one-way)
              rclone sync /config minio:vixens-backups/app/config \
                --config /rclone/rclone.conf \
                --exclude "*.log" \
                --exclude "*.tmp" \
                --exclude ".cache/**" \
                --verbose

              echo "✅ Backup complete $(date)"

            resources:
              requests:
                memory: "64Mi"
                cpu: "10m"
              limits:
                memory: "128Mi"
                cpu: "100m"

            volumeMounts:
            - name: config-pvc
              mountPath: /config
              readOnly: true
            - name: rclone-config
              mountPath: /rclone

          volumes:
          - name: config-pvc
            persistentVolumeClaim:
              claimName: app-config
          - name: rclone-config
            secret:
              secretName: rclone-minio-config
```

**Validation:**
```bash
# Créer un job manuel pour tester
kubectl create job -n app-namespace --from=cronjob/app-backup test-backup

# Vérifier logs
kubectl logs -n app-namespace job/test-backup

# Vérifier dans MinIO
# mc ls minio/vixens-backups/app/config
```

### Step 5: Create Restore PreSync Job

**Restore avant déploiement (ArgoCD hook):**

```yaml
# apps/<app>/base/restore-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: app-restore
  namespace: app-namespace
  annotations:
    # ArgoCD PreSync: s'exécute AVANT sync des ressources
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: BeforeHookCreation
spec:
  template:
    metadata:
      labels:
        app.kubernetes.io/name: app
        app.kubernetes.io/component: restore
    spec:
      restartPolicy: Never

      containers:
      - name: rclone-restore
        image: rclone/rclone:latest
        command:
        - /bin/sh
        - -c
        - |
          set -e
          echo "=== Starting restore $(date) ==="

          # Vérifier si backup existe
          if rclone ls minio:vixens-backups/app/config \
             --config /rclone/rclone.conf >/dev/null 2>&1; then

            echo "✅ Backup found, restoring..."
            rclone copy minio:vixens-backups/app/config /restore \
              --config /rclone/rclone.conf \
              --verbose

            echo "✅ Restore complete"
            touch /restore/.restore-success
          else
            echo "⚠️  No backup found, will use vanilla config"
            touch /restore/.no-backup
          fi

          echo "=== Restore job finished $(date) ==="

        resources:
          requests:
            memory: "128Mi"
            cpu: "50m"
          limits:
            memory: "256Mi"
            cpu: "200m"

        volumeMounts:
        - name: restore-volume
          mountPath: /restore
        - name: rclone-config
          mountPath: /rclone

      volumes:
      - name: restore-volume
        emptyDir: {}
      - name: rclone-config
        secret:
          secretName: rclone-minio-config
```

**Note:** EmptyDir partagé avec InitContainer du Deployment (étape suivante).

### Step 6: Create InitContainer Script

**Layered initialization:**

```yaml
# apps/<app>/base/configmap-init.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-init-script
  namespace: app-namespace
data:
  init.sh: |
    #!/bin/sh
    set -e

    echo "=== Smart Layered Init ==="

    # LAYER 1: Vanilla base (TOUJOURS)
    echo "Layer 1: Deploying vanilla configuration..."
    if [ -f /defaults/configuration.yaml ]; then
      cp /defaults/configuration.yaml /config/configuration.yaml
      echo "✅ Vanilla config deployed"
    else
      echo "❌ ERROR: Vanilla config missing!"
      exit 1
    fi

    # LAYER 2: Restore backup (OVERRIDE vanilla)
    if [ -f /restore/.restore-success ]; then
      echo "Layer 2: Restoring backup..."
      cp -r /restore/* /config/ 2>/dev/null || true
      echo "✅ Backup restored"
    elif [ -f /restore/.no-backup ]; then
      echo "⚠️  No backup available, using vanilla only"
    else
      echo "⚠️  Restore job not run, using vanilla only"
    fi

    # LAYER 3: Inject secrets (COMPLEMENT)
    if [ -d /secrets ]; then
      echo "Layer 3: Injecting secrets from Infisical..."
      # Export env vars depuis secrets
      export $(cat /secrets/*.env | xargs)

      # Remplacer tokens ${VAR} dans config
      envsubst < /config/configuration.yaml > /tmp/merged.yaml
      mv /tmp/merged.yaml /config/configuration.yaml

      echo "✅ Secrets injected"
    fi

    # Vérifications finales
    echo "=== Validation ==="
    ls -la /config

    echo "✅ Initialization complete"
```

### Step 7: Update Deployment

**Intégrer InitContainer + Sidecar:**

```yaml
# apps/<app>/base/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: app-namespace
  labels:
    app.kubernetes.io/name: app
  annotations:
    argocd.argoproj.io/sync-wave: "0"
    goldilocks.fairwinds.com/enabled: "true"
    vpa.kubernetes.io/updateMode: "Off"
spec:
  replicas: 1
  strategy:
    type: Recreate  # Si PVC RWO
  selector:
    matchLabels:
      app.kubernetes.io/name: app
  template:
    metadata:
      labels:
        app.kubernetes.io/name: app
    spec:
      priorityClassName: medium-priority

      # INIT CONTAINER: Layered config
      initContainers:
      - name: init-config
        image: alpine:latest
        command: ["/bin/sh", "/scripts/init.sh"]
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        volumeMounts:
        - name: init-script
          mountPath: /scripts
        - name: vanilla-config
          mountPath: /defaults
        - name: restore-volume
          mountPath: /restore
        - name: config-pvc
          mountPath: /config

      # MAIN CONTAINER
      containers:
      - name: app
        image: app:latest
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        volumeMounts:
        - name: config-pvc
          mountPath: /config
        livenessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 10

      # BACKUP SIDECAR (alternative au CronJob)
      - name: backup-sidecar
        image: rclone/rclone:latest
        command:
        - /bin/sh
        - -c
        - |
          # Backup continu toutes les 5 minutes
          while true; do
            echo "Backing up config..."
            rclone sync /config minio:vixens-backups/app/config \
              --config /rclone/rclone.conf \
              --exclude "*.log"
            sleep 300
          done
        resources:
          requests:
            memory: "64Mi"
            cpu: "10m"
          limits:
            memory: "128Mi"
            cpu: "100m"
        volumeMounts:
        - name: config-pvc
          mountPath: /config
          readOnly: true
        - name: rclone-config
          mountPath: /rclone

      # VOLUMES
      volumes:
      - name: init-script
        configMap:
          name: app-init-script
          defaultMode: 0755
      - name: vanilla-config
        configMap:
          name: app-vanilla-config
      - name: restore-volume
        emptyDir: {}  # Partagé avec PreSync job
      - name: config-pvc
        persistentVolumeClaim:
          claimName: app-config
      - name: rclone-config
        secret:
          secretName: rclone-minio-config
```

---

## Code Templates

### Template: SQLite Backup (Litestream)

**Pour bases de données SQLite uniquement:**

```yaml
# Sidecar Litestream
- name: litestream
  image: litestream/litestream:latest
  args:
  - replicate
  env:
  - name: LITESTREAM_ACCESS_KEY_ID
    valueFrom:
      secretKeyRef:
        name: minio-credentials
        key: access-key
  - name: LITESTREAM_SECRET_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: minio-credentials
        key: secret-key
  volumeMounts:
  - name: data-pvc
    mountPath: /data
  - name: litestream-config
    mountPath: /etc/litestream.yml
    subPath: litestream.yml

---
# ConfigMap litestream.yml
apiVersion: v1
kind: ConfigMap
metadata:
  name: litestream-config
data:
  litestream.yml: |
    dbs:
    - path: /data/app.db
      replicas:
      - type: s3
        bucket: vixens-backups
        path: app/database
        endpoint: http://minio.minio.svc:9000
        force-path-style: true
```

### Template: Multi-File Backup

**Si plusieurs fichiers/dossiers à backuper:**

```bash
# init.sh
# Backup sélectif
rclone sync /config minio:backups/app/config \
  --include "*.yaml" \
  --include "*.json" \
  --include ".storage/**" \
  --exclude "*.log" \
  --exclude "*.tmp"
```

---

## Testing & Validation

### Test 1: Backup Functionality

```bash
# 1. Modifier un fichier dans l'app
kubectl exec -n app-namespace deploy/app -- \
  sh -c 'echo "test-backup" > /config/test.txt'

# 2. Attendre 5 minutes (ou trigger manuel)
kubectl create job -n app-namespace --from=cronjob/app-backup manual-backup

# 3. Vérifier dans MinIO
# Web UI: http://minio-console.minio.svc:9001
# CLI: mc ls minio/vixens-backups/app/config/test.txt
```

### Test 2: Restore Functionality

```bash
# 1. Noter état actuel
kubectl exec -n app-namespace deploy/app -- ls -la /config

# 2. Détruire le PVC
kubectl delete pvc -n app-namespace app-config

# 3. Re-sync ArgoCD (trigger PreSync + Deployment)
argocd app sync app-namespace

# 4. Vérifier restore
kubectl logs -n app-namespace job/app-restore
kubectl logs -n app-namespace deploy/app -c init-config

# 5. Valider contenu
kubectl exec -n app-namespace deploy/app -- ls -la /config
kubectl exec -n app-namespace deploy/app -- cat /config/test.txt
# Devrait contenir "test-backup"
```

### Test 3: Disaster Recovery Complet

```bash
# Simulation incident cluster reset

# 1. Backup avant test
kubectl create job -n app-namespace --from=cronjob/app-backup pre-test-backup

# 2. Détruire TOUT
kubectl delete namespace app-namespace

# 3. Re-créer via ArgoCD
argocd app sync app-namespace

# 4. Vérifier recovery automatique
kubectl get pods -n app-namespace -w
kubectl logs -n app-namespace deploy/app -c init-config

# 5. Valider fonctionnement
curl https://app.dev.truxonline.com
```

**Success Criteria:**
- ✅ Namespace recréé
- ✅ PreSync job exécuté
- ✅ Config restaurée
- ✅ App fonctionnelle
- ✅ Aucune intervention manuelle

---

## Troubleshooting

### Issue: "No backup found"

**Symptôme:**
```
⚠️  No backup found, will use vanilla config
```

**Causes possibles:**
1. Premier déploiement (pas encore de backup)
2. MinIO inaccessible
3. Credentials incorrects
4. Bucket/path incorrect

**Debug:**
```bash
# Vérifier MinIO accessible
kubectl exec -n app-namespace deploy/app -c backup-sidecar -- \
  rclone ls minio: --config /rclone/rclone.conf

# Vérifier credentials
kubectl get secret -n app-namespace rclone-minio-config -o yaml

# Créer backup initial manuel
kubectl create job -n app-namespace --from=cronjob/app-backup initial-backup
```

### Issue: "Permission denied" lors restore

**Symptôme:**
```
cp: can't create '/config/configuration.yaml': Permission denied
```

**Cause:** Problème de permissions PVC

**Fix:**
```yaml
# InitContainer avec fsGroup
spec:
  securityContext:
    fsGroup: 1000
  initContainers:
  - name: fix-permissions
    image: alpine:latest
    command: ["chown", "-R", "1000:1000", "/config"]
    volumeMounts:
    - name: config-pvc
      mountPath: /config
```

### Issue: Backup trop volumineux

**Symptôme:**
```
Backup job timeout after 10 minutes
```

**Cause:** Fichiers volumineux (logs, cache)

**Fix: Améliorer excludes**
```bash
rclone sync /config minio:backups/app/config \
  --exclude "*.log" \
  --exclude "*.tmp" \
  --exclude ".cache/**" \
  --exclude "*.db-wal" \
  --exclude "*.db-shm" \
  --max-size 100M
```

### Issue: Restore écrase modifications

**Symptôme:** Config utilisateur écrasée par backup ancien

**Cause:** Backup obsolète restauré

**Fix: Versionning**
```bash
# Backup avec timestamp
rclone sync /config minio:backups/app/$(date +%Y%m%d-%H%M%S) \
  --config /rclone/rclone.conf

# Restore depuis latest
rclone copy minio:backups/app/latest /restore
```

---

## Best Practices

### 1. Retention Policy

```bash
# Garder backups 7 jours
rclone delete minio:backups/app \
  --min-age 7d \
  --rmdirs
```

### 2. Monitoring

```yaml
# Prometheus metrics (custom)
- name: backup-metrics
  image: prom/pushgateway
  # Push backup success/failure
```

### 3. Encryption

```yaml
# rclone avec encryption
[minio-crypt]
type = crypt
remote = minio:vixens-backups
password = <encrypted-password>
```

### 4. Multi-Destination

```bash
# Backup vers MinIO + S3
rclone sync /config minio:backups/app/config
rclone sync /config s3:offsite-backups/app/config
```

---

## Next Steps

1. **Implémenter pattern sur apps critiques:**
   - HomeAssistant
   - Frigate
   - Mosquitto

2. **Créer Helm chart réutilisable:**
   - Template générique
   - Values par app

3. **Automatiser monitoring:**
   - Prometheus alerts
   - Grafana dashboard

4. **Documentation applicative:**
   - Update `docs/applications/<app>.md`
   - Procédure recovery spécifique

---

## References

- [ADR-013: Layered Configuration](../adr/013-layered-configuration-disaster-recovery.md)
- [Application Deployment Standard](../reference/application-deployment-standard.md)
- [rclone Documentation](https://rclone.org/docs/)
- [Litestream](https://litestream.io/)
- [ArgoCD Sync Hooks](https://argo-cd.readthedocs.io/en/stable/user-guide/resource_hooks/)

---

**Document Owner:** Infrastructure Team
**Last Updated:** 2026-01-05
**Next Review:** 2026-04-05
