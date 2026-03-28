# Configuration Management Strategy

**Status:** Active
**Last Updated:** 2026-03-28
**Supersedes:** [backup-restore-pattern.md](../guides/backup-restore-pattern.md) (old rclone/CronJob approach)
**Related:** [ADR-013](../adr/013-layered-configuration-disaster-recovery.md), [ADR-014](../adr/014-litestream-backup-profiles-and-recovery-patterns.md)

---

## Overview

La stratégie repose sur **4 couches** selon le type de donnée :

| Type | Stockage | Backup | Exemples |
|------|----------|--------|---------|
| **Secrets** | Infisical → K8s Secret | N/A (source of truth = Infisical) | API keys, passwords, tokens |
| **Config app** | local-path-retain PVC | DataAngel → MinIO | sonarr.db, /config, app settings |
| **Base de données PostgreSQL** | CloudNativePG | CNPG built-in | n8n, vikunja, authentik |
| **Fichiers partagés/média** | NFS direct (Synology) | N/A (NAS gère la redondance) | /movies, /music, /downloads |

---

## Couche 1 : Secrets → Infisical

Tous les secrets passent par **Infisical** (self-hosted sur le NAS à `192.168.111.69:8085`).

```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: myapp-secrets
spec:
  hostAPI: http://192.168.111.69:8085
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: argocd
      secretsScope:
        projectSlug: vixens
        envSlug: prod           # ou dev
        secretsPath: /apps/20-media/myapp
  managedSecretReference:
    secretName: myapp-secrets
    secretNamespace: media
    creationPolicy: Owner
```

**Règle :** zéro secret dans Git. Tout ce qui est sensible va dans Infisical, référencé via `secretKeyRef` dans le déploiement.

---

## Couche 2 : Config App → local-path-retain + DataAngel

### Stockage : local-path-retain

Les PVCs de config applicatif utilisent `local-path-retain` (stockage node-local). Plus rapide que iSCSI, pas de dépendance réseau.

```yaml
# overlay prod : patch PVC
- op: replace
  path: /spec/storageClassName
  value: local-path-retain
```

**Important :** `local-path-retain` = `WaitForFirstConsumer`. La PVC reste `Pending` jusqu'à ce qu'un pod soit schedulé. C'est normal.

### Backup/Restore : DataAngel

**DataAngel** est un init container (`restartPolicy: Always`) qui :
- Au démarrage : restaure depuis MinIO S3
- En fonctionnement : réplique en continu vers MinIO S3

Deux modes selon le contenu :

#### Mode SQLite (litestream)

Pour les apps avec une base SQLite :

```yaml
# overlay prod/dataangel.yaml
spec:
  template:
    metadata:
      annotations:
        dataangel.io/bucket: "vixens-prod-myapp"
        dataangel.io/sqlite-paths: "/config/myapp.db"
        dataangel.io/s3-endpoint: "http://192.168.111.69:9000"
        dataangel.io/deployment-name: "myapp"
        dataangel.io/rclone-interval: "60s"
        dataangel.io/metrics-enabled: "true"
        dataangel.io/lock-enabled: "false"
    spec:
      initContainers:
        - name: dataangel
          securityContext:
            runAsUser: 1000
            runAsGroup: 1000
          env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: myapp-secrets
                  key: LITESTREAM_ACCESS_KEY_ID
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: myapp-secrets
                  key: LITESTREAM_SECRET_ACCESS_KEY
          volumeMounts:
            - mountPath: /data
              $patch: delete
            - name: config
              mountPath: /config
```

#### Mode Filesystem (rclone)

Pour les apps sans SQLite (config files only) :

```yaml
annotations:
  dataangel.io/bucket: "vixens-prod-myapp"
  dataangel.io/fs-paths: "/config"          # pas de sqlite-paths
  dataangel.io/s3-endpoint: "http://192.168.111.69:9000"
  dataangel.io/rclone-interval: "60s"
```

#### Mode mixte (SQLite + FS)

```yaml
annotations:
  dataangel.io/sqlite-paths: "/config/myapp.db"
  dataangel.io/fs-paths: "/config"           # rclone les fichiers non-SQLite
```

### Prérequis : `initContainers: []` dans la base

Le patch JSON `op: add /spec/template/spec/initContainers/-` échoue si le champ n'existe pas. **Toujours** ajouter dans le déploiement base :

```yaml
spec:
  template:
    spec:
      initContainers: []   # ← obligatoire pour compat DataAngel
      containers:
        - name: myapp
```

### Composants Kustomize partagés

Le composant `_shared/components/dataangel` injecte automatiquement :
- L'init container DataAngel avec `restartPolicy: Always`
- Les env vars depuis les annotations
- Les ports metrics (9090)

L'overlay ne gère que les spécificités de l'app (credentials, chemins).

### Buckets MinIO

Convention de nommage : `vixens-{env}-{appname}`
Exemples : `vixens-prod-sonarr`, `vixens-prod-jellyfin`, `vixens-dev-mealie`

Les buckets sont créés automatiquement par DataAngel s'ils n'existent pas (option `auto-create`).

---

## Couche 3 : PostgreSQL → CloudNativePG

Les apps avec PostgreSQL n'utilisent **pas** DataAngel. CloudNativePG gère :
- Réplication HA (primary + replicas)
- Backup automatique (WAL archiving)
- Restore automatique

Apps concernées : n8n, vikunja, authentik, vaultwarden (optionnel).

**Règle :** Ne jamais utiliser DataAngel pour PostgreSQL.

---

## Couche 4 : Fichiers partagés → NFS

Les médias et fichiers volumineux montent directement sur le NAS Synology (`192.168.111.69`).

### Structure `/volume3/Content`

```
/volume3/Content/          ← root (perms 0000, inaccessible UID 1000)
├── movies/                → radarr, jellyfin
├── TV Show/               → sonarr (Animes/, Tv shows/)
├── music/                 → lidarr, music-assistant, jellyfin
├── ebooks/                → lazylibrarian
├── xxx/xxx/               → whisparr
└── pictures/hentai/       → hydrus-client
```

> ⚠️ **Le root `/volume3/Content` a des permissions `0000`** (Windows ACL Synology).
> Ne jamais monter la racine directement pour une app qui tourne en UID 1000.
> Utiliser soit `nfs.path` vers un sous-dossier, soit `subPath` dans le volumeMount.

**Pattern correct :**

```yaml
# Option A : path direct (recommandé si pas d'espace)
volumes:
  - name: movies
    nfs:
      server: 192.168.111.69
      path: /volume3/Content/movies

# Option B : subPath (si le chemin contient un espace, ex: "TV Show")
volumes:
  - name: content
    nfs:
      server: 192.168.111.69
      path: /volume3/Content   # kubelet (root) monte la racine
volumeMounts:
  - name: content
    mountPath: /media/tv
    subPath: TV Show           # bind-mount du sous-dossier
```

### `/volume3/Downloads`

Downloads partagés entre qbittorrent, sabnzbd, sonarr, radarr, etc. Montés directement en NFS.

---

## Décision Matrix

| Situation | Solution |
|-----------|----------|
| Secret / credential | Infisical → InfisicalSecret |
| Config app (fichiers) | local-path-retain PVC + DataAngel (fs-paths) |
| Base SQLite | local-path-retain PVC + DataAngel (sqlite-paths) |
| Base PostgreSQL | CloudNativePG |
| Médias / gros fichiers | NFS direct (Synology) |
| Cache / données éphémères | emptyDir |
| Config statique (no secret) | ConfigMap dans Git |

---

## Procédure : Ajouter DataAngel à une nouvelle app

1. Ajouter `initContainers: []` dans `base/deployment.yaml`
2. Créer `overlays/prod/dataangel.yaml` avec les annotations + credentials
3. Créer le bucket MinIO si besoin (DataAngel le crée auto sinon)
4. Ajouter les secrets MinIO dans Infisical (`LITESTREAM_ACCESS_KEY_ID`, `LITESTREAM_SECRET_ACCESS_KEY`)
5. Référencer le composant dans `overlays/prod/kustomization.yaml` :
   ```yaml
   components:
     - ../../../../_shared/components/dataangel
   ```
6. Ajouter le patch `dataangel.yaml` dans la section `patches:`

---

## Profils de backup (DataAngel / litestream)

Hérités de [ADR-014](../adr/014-litestream-backup-profiles-and-recovery-patterns.md) :

| Profil | Intervalle snapshot | Rétention | Usage |
|--------|-------------------|-----------|-------|
| `critical` | 1h | 14j | Haute activité, perte < 1h inacceptable |
| `standard` | 6h | 7j | Activité modérée |
| `relaxed` | 24h | 3j | Config, faible activité |
| `ephemeral` | — | — | Cache, ne pas backuper |

Le label `vixens.io/backup-profile: "relaxed"` sur le pod indique le profil.
