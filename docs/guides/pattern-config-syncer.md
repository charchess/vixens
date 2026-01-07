# Pattern: Sidecar Config-Syncer (File Backup)

Ce pattern assure la sauvegarde temps-réel et la restauration au démarrage des fichiers de configuration plats (`.ini`, `.xml`, `.json`, `.yaml`) qui ne sont pas gérés par une base de données.

**Objectif :** Résilience totale en cas de perte du Cluster ET du Stockage (PVC).

---

## Architecture

1.  **InitContainer (`restore-config`)** :
    *   Au démarrage, télécharge le contenu du bucket S3 vers le volume `/config`.
    *   Utilise `rclone copy`.
    *   Ne s'arrête pas si le bucket est vide (premier déploiement).

2.  **Sidecar Container (`config-syncer`)** :
    *   Surveille le dossier `/config` avec `inotifywait`.
    *   Déclenche `rclone sync` vers S3 lors d'un événement `CLOSE_WRITE` ou `MOVED_TO`.
    *   Exécute aussi une synchro périodique (toutes les 1h) par sécurité.

---

## Implémentation

### 1. Pré-requis (Secrets)

L'application doit avoir accès aux credentials S3 compatibles Rclone (souvent les mêmes que Litestream).
Secret recommandé : `shared-s3-credentials` ou via Infisical.

Variables d'environnement requises :
*   `RCLONE_CONFIG_S3_TYPE=s3`
*   `RCLONE_CONFIG_S3_PROVIDER=Other` (pour Minio/Scaleway/AWS)
*   `RCLONE_CONFIG_S3_ACCESS_KEY_ID=...`
*   `RCLONE_CONFIG_S3_SECRET_ACCESS_KEY=...`
*   `RCLONE_CONFIG_S3_ENDPOINT=...`
*   `SYNC_BUCKET=...` (ex: `vixens-backups`)
*   `SYNC_PATH=...` (ex: `sabnzbd/config`)

### 2. Définition du Deployment (Snippet)

```yaml
    spec:
      initContainers:
        - name: restore-config
          image: rclone/rclone:1.65
          command: ["sh", "-c"]
          args:
            - |
              echo "Restoring config from s3://$SYNC_BUCKET/$SYNC_PATH..."
              rclone copy s3:$SYNC_BUCKET/$SYNC_PATH /config --transfers 4
              echo "Restore complete."
          envFrom:
            - secretRef:
                name: app-s3-secret
          volumeMounts:
            - name: config
              mountPath: /config

      containers:
        # ... Application Container ...

        - name: config-syncer
          image: alpine:3.19
          command: ["sh", "-c"]
          args:
            - |
              apk add --no-cache rclone inotify-tools
              
              echo "Starting Config-Syncer for /config -> s3://$SYNC_BUCKET/$SYNC_PATH"
              
              # Fonction de synchro
              sync_s3() {
                echo "[$(date)] Syncing changes..."
                rclone sync /config s3:$SYNC_BUCKET/$SYNC_PATH --exclude "*.db*" --exclude "*.log"
              }
              
              # Boucle de surveillance
              while true; do
                inotifywait -r -e modify,create,delete,move /config
                sync_s3
                sleep 5 # Debounce
              done
          resources:
            requests:
              cpu: 10m
              memory: 32Mi
            limits:
              cpu: 100m
              memory: 128Mi
          envFrom:
             - secretRef:
                 name: app-s3-secret
          volumeMounts:
            - name: config
              mountPath: /config
```

### 3. Exclusions Importantes

Il est crucial d'exclure les fichiers gérés par **Litestream** (`*.db`, `*.db-wal`) et les logs pour éviter les conflits et le trafic inutile.
Utilisez le flag `--exclude` de rclone.

---

## Cas d'Usage Validés

*   **Sabnzbd** : Sauvegarde de `sabnzbd.ini`.
*   **Sonarr/Radarr** : Sauvegarde de `config.xml` (si non présent en DB).
*   **Domotique** : Sauvegarde des fichiers YAML de Home Assistant.
