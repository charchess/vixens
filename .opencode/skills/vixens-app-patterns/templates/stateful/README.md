# Stateful Template

Template pour applications stateful avec SQLite + Litestream (backup S3) + Config-Syncer (sync fichiers non-DB).

## Quand utiliser ce template

- Application avec données persistantes (SQLite)
- Besoin de backup automatique temps-réel (Litestream)
- Besoin de sync fichiers de config (Config-Syncer)
- Secrets gérés par Infisical
- Résilience critique (diamond tier)

## Exemples d'applications

- vaultwarden (password manager)
- trilium (notes)
- firefly-iii (finances)

## Architecture

### 3 containers principaux:
1. **App container** - Application principale
2. **Litestream sidecar** - Backup continu SQLite → S3
3. **Config-Syncer sidecar** - Sync fichiers config → S3 (1 min)

### 2 init containers:
1. **fix-permissions** - chown 1000:1000 /data
2. **restore-config** - Restore fichiers config depuis S3

### Volumes:
- **PVC RWO** - Volume principal (/data)
- **ConfigMap** - Configuration Litestream

### Secrets (Infisical):
- `LITESTREAM_ACCESS_KEY_ID` - S3 access key
- `LITESTREAM_SECRET_ACCESS_KEY` - S3 secret key
- `LITESTREAM_ENDPOINT` - S3 endpoint (Minio)
- `LITESTREAM_BUCKET` - S3 bucket name

## Structure

```
stateful/
├── base/
│   ├── deployment.yaml         # Deployment 3-containers
│   ├── service.yaml           # Service (http + metrics)
│   ├── pvc.yaml               # PVC 5Gi RWO
│   ├── infisical-secret.yaml  # InfisicalSecret
│   ├── litestream-config.yaml # Litestream ConfigMap
│   └── kustomization.yaml     # Base Kustomize
└── overlays/
    ├── dev/
    │   └── kustomization.yaml  # Dev: replicas=0 + envSlug=dev
    └── prod/
        └── kustomization.yaml  # Prod: gold-maturity + envSlug=prod
```

## Personnalisation

### 1. Remplacer les placeholders

Dans tous les fichiers:
- `APP_NAME` → nom de votre application
- `NAMESPACE_NAME` → namespace cible
- `CATEGORY` → catégorie app (ex: 60-services)
- `IMAGE_REGISTRY/IMAGE_NAME:IMAGE_TAG` → image Docker
- `ENV_SLUG` → `dev` ou `prod`

### 2. Configurer Infisical secrets

Dans Infisical UI (http://192.168.111.69:8085):

1. Projet: `vixens`
2. Environment: `dev` ou `prod`
3. Path: `/apps/CATEGORY/APP_NAME`
4. Secrets requis:
   ```
   LITESTREAM_ACCESS_KEY_ID=xxx
   LITESTREAM_SECRET_ACCESS_KEY=xxx
   LITESTREAM_ENDPOINT=http://192.168.111.69:9000
   LITESTREAM_BUCKET=vixens-ENV-APP_NAME
   ```

**Créer le bucket S3 (Minio):**
```bash
mc alias set minio http://192.168.111.69:9000 ACCESS_KEY SECRET_KEY
mc mb minio/vixens-prod-APP_NAME
mc mb minio/vixens-dev-APP_NAME
```

### 3. Ajuster les probes

Si votre application n'a pas `/health`:
```yaml
livenessProbe:
  httpGet:
    path: /api/healthz  # Votre endpoint
    port: http
```

Ou TCP probe:
```yaml
livenessProbe:
  tcpSocket:
    port: http
  initialDelaySeconds: 5
```

### 4. Ajuster le sizing

Labels `vixens.io/sizing.*` dans deployment.yaml:
- `fix-permissions: G-nano` (init, 10m/16Mi)
- `restore-config: B-nano` (init, 10m/16Mi)
- `APP_NAME: V-nano` (app, 25m/32Mi) ← **AJUSTER**
- `litestream: V-nano` (sidecar, 25m/32Mi)
- `config-syncer: V-nano` (sidecar, 25m/32Mi)

Tiers disponibles:
- **B-nano** - 10m CPU / 16Mi RAM
- **V-nano** - 25m CPU / 32Mi RAM
- **V-small** - 50m CPU / 64Mi RAM
- **V-medium** - 100m CPU / 128Mi RAM
- **G-nano** - 50m CPU / 64Mi RAM

### 5. Ajuster le PVC size

Dans `base/pvc.yaml`:
```yaml
resources:
  requests:
    storage: 5Gi  # Ajuster selon besoins
```

**ATTENTION:** `synelia-iscsi-retain` = volumes **NOT deleted** on PVC deletion.

### 6. Ajuster Litestream config

Dans `base/litestream-config.yaml`:
```yaml
dbs:
  - path: /data/db.sqlite3  # Votre chemin DB
    replicas:
      - url: s3://$LITESTREAM_BUCKET/db.sqlite3
        endpoint: $LITESTREAM_ENDPOINT
        retention: 24h  # Garder 24h de WAL
        snapshot-interval: 1h  # Snapshot toutes les 1h
```

### 7. Ajuster Config-Syncer excludes

Dans `deployment.yaml`, container `config-syncer`:
```yaml
args:
  - |
    rclone sync /data s3:$LITESTREAM_BUCKET/config \
      --exclude "*.sqlite3*" \
      --exclude "*.log" \
      --exclude "cache/**" \  # Ajouter vos exclusions
      --exclude "tmp/**"
```

Syncer **tout sauf**:
- DB SQLite (géré par Litestream)
- Logs (éphémères)
- Cache (reconstituable)

### 8. Variables d'environnement app

Dans `deployment.yaml`, container principal:
```yaml
env:
  - name: DATA_FOLDER
    value: /data
  - name: DATABASE_URL
    value: /data/db.sqlite3
  # Ajouter vos vars
  - name: CUSTOM_VAR
    valueFrom:
      secretKeyRef:
        name: APP_NAME-secrets
        key: CUSTOM_VAR
```

## Backup Strategy

### Litestream (SQLite backup)
- **Continuous replication** - WAL segments toutes les 1s
- **Snapshots** - Full backup toutes les 1h (configurable)
- **Retention** - 24h de WAL (configurable)
- **Recovery** - Automatic on pod restart (restore-db init container)

### Config-Syncer (fichiers config)
- **Sync interval** - Toutes les 60s
- **Sync type** - rclone sync (unidirectional, /data → S3)
- **Excludes** - DB, logs, cache
- **Recovery** - Automatic on pod restart (restore-config init container)

### Recovery Process
1. Pod démarre
2. `fix-permissions` init container - chown /data
3. `restore-config` init container - restore fichiers depuis S3
4. App container démarre - Litestream restore DB automatique
5. Litestream sidecar - continuous replication
6. Config-Syncer sidecar - sync fichiers toutes les 60s

## Composants automatiques

### Dev (`overlays/dev/`)
- `components/base` - Labels de base
- `components/probes/basic` - Probes standards
- `replicas: 0` - Désactivé par défaut
- `envSlug: dev` - Secrets Infisical dev

### Prod (`overlays/prod/`)
- `components/gold-maturity` - Probes + securityContext renforcés
- `components/base` - Labels de base
- `components/resources` - Requests/limits automatiques (via VPA)
- `components/poddisruptionbudget/0` - PDB maxUnavailable=0
- `components/priority/medium` - priorityClassName: vixens-medium
- `components/revision-history-limit` - revisionHistoryLimit: 3
- `envSlug: prod` - Secrets Infisical prod

## Validation

```bash
# Lint
yamllint -c yamllint-config.yml apps/CATEGORY/APP_NAME/**/*.yaml

# Build dev
kustomize build apps/CATEGORY/APP_NAME/overlays/dev

# Build prod
kustomize build apps/CATEGORY/APP_NAME/overlays/prod

# Vérifier les kinds
kustomize build apps/CATEGORY/APP_NAME/overlays/prod | grep '^kind:' | sort

# Test Litestream config
cat apps/CATEGORY/APP_NAME/base/litestream-config.yaml
```

## Checklist DoD

Avant de marquer ready-for-dev:
- [ ] Tous les placeholders remplacés
- [ ] Infisical secrets créés (dev + prod)
- [ ] S3 buckets créés (dev + prod)
- [ ] Probes adaptées à l'application
- [ ] Sizing labels corrects
- [ ] PVC size approprié
- [ ] Litestream config validé
- [ ] Config-Syncer excludes corrects
- [ ] yamllint passe
- [ ] kustomize build dev passe
- [ ] kustomize build prod passe
- [ ] Kinds diff OK

## Troubleshooting

### Litestream fails

**Symptômes:** Logs Litestream erreur S3
```bash
kubectl logs -n NAMESPACE pod/APP_NAME-xxx -c litestream
```

**Causes:**
- Secrets Infisical manquants/incorrects
- Bucket S3 n'existe pas
- Endpoint S3 inaccessible

**Fix:**
```bash
# Vérifier secrets
kubectl get secret -n NAMESPACE APP_NAME-secrets -o yaml

# Vérifier bucket
mc ls minio/vixens-ENV-APP_NAME

# Vérifier endpoint
curl http://192.168.111.69:9000
```

### Config-Syncer fails

**Symptômes:** Logs Config-Syncer erreur rclone
```bash
kubectl logs -n NAMESPACE pod/APP_NAME-xxx -c config-syncer
```

**Causes:** Mêmes que Litestream

### restore-config timeout

**Symptômes:** Init container bloqué
```bash
kubectl describe pod -n NAMESPACE APP_NAME-xxx
```

**Causes:**
- S3 endpoint inaccessible
- Bucket vide (première install) ← **Normal**

**Fix:** Si première install, config-syncer populera le bucket au premier sync.

### PVC pending

**Symptômes:** PVC bloqué "Pending"
```bash
kubectl get pvc -n NAMESPACE
```

**Causes:**
- StorageClass `synelia-iscsi-retain` n'existe pas
- Pas de volume disponible

**Fix:**
```bash
# Vérifier StorageClass
kubectl get storageclass

# Vérifier PV disponibles
kubectl get pv
```

### Permission denied /data

**Symptômes:** App crash "Permission denied"

**Causes:**
- `fix-permissions` init container failed
- `fsGroup` incorrect

**Fix:**
```bash
# Vérifier init container logs
kubectl logs -n NAMESPACE APP_NAME-xxx -c fix-permissions

# Vérifier pod securityContext
kubectl get pod -n NAMESPACE APP_NAME-xxx -o yaml | grep -A 10 securityContext
```

### OOMKilled

**Causes:** Sizing tier trop petit

**Fix:** Augmenter les sizing labels:
```yaml
vixens.io/sizing.APP_NAME: V-small  # V-nano → V-small
```

### Database corruption

**Symptômes:** App fail to start, DB errors

**Recovery:**
```bash
# 1. Scale down
kubectl scale deployment -n NAMESPACE APP_NAME --replicas=0

# 2. Delete PVC (DANGER: will restore from S3)
kubectl delete pvc -n NAMESPACE APP_NAME-data-pvc

# 3. Scale up (Litestream will restore)
kubectl scale deployment -n NAMESPACE APP_NAME --replicas=1

# 4. Watch restoration
kubectl logs -n NAMESPACE -f deployment/APP_NAME -c litestream
```

**ATTENTION:** PVC delete = restore from S3. Data loss = dernière sync Litestream.
