# DataAngel — Brief technique

## Objectif

Remplacer le pattern copy-collé de backup par app par un container unique et standardisé.

### Aujourd'hui (par app, chaque app copie ce schéma)

```
initContainers:
  - rclone         → restore fichiers config depuis S3
  - litestream     → restore DB SQLite depuis S3
containers:
  - litestream     → réplication WAL continue (SQLite)
  - rclone         → sync config périodique (filesystem)
  - ConfigMap      → litestream-config.yaml par app
```

### Avec DataAngel (cible)

```
initContainers:
  - dataangel      → restore SQLite + filesystem depuis S3
containers:
  - dataangel      → backup continu SQLite + filesystem
```

Configuration via **env vars uniquement**, injectées depuis un secret Infisical.

---

## Contrat technique

### Init container (`./init`)

Comportement attendu, deux cas indépendants :

**SQLite** → utiliser `litestream restore`
```
litestream restore -if-db-not-exists -if-replica-exists <local_path>
```
- Exit 0 si la DB locale existe déjà (skip)
- Exit 0 si pas de replica en S3 (skip, premier démarrage)
- Exit 1 si restore échoue

**Filesystem** → utiliser `rclone copy`
```
rclone copy s3:<bucket>/<remote_path> <local_path> --exclude "*.db*"
```
- Exit 0 si rien à copier
- Exit 1 si erreur

**`DATA_GUARD_CHECKSUM` doit être optionnel.** Si absent → skip la vérification. Un checksum statique en annotation est incompatible avec un backup continu (le contenu change à chaque écriture).

### Sidecar (`./sidecar`)

**SQLite** → Litestream en mode streaming WAL (réplication continue, pas upload périodique)
```yaml
dbs:
  - path: <DATA_GUARD_SQLITE_PATHS>
    replicas:
      - url: s3://<bucket>/<app>.db
        endpoint: <DATA_GUARD_S3_ENDPOINT>
```

**Filesystem** → Rclone sync périodique (toutes les `DATA_GUARD_RCLONE_INTERVAL`, défaut 60s)

Les deux modes sont **indépendants et cumulables** : une app peut avoir SQLite seul, FS seul, ou les deux.

### Variables d'environnement

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| `DATA_GUARD_BUCKET` | Oui | Nom du bucket S3 |
| `DATA_GUARD_S3_ENDPOINT` | Non | Endpoint custom (MinIO) |
| `DATA_GUARD_SQLITE_PATHS` | Non* | Chemins DB SQLite, virgule-séparés |
| `DATA_GUARD_FS_PATHS` | Non* | Chemins répertoires, virgule-séparés |
| `DATA_GUARD_RCLONE_INTERVAL` | Non | Intervalle sync FS (défaut: `60s`) |
| `DATA_GUARD_METRICS_PORT` | Non | Port Prometheus (défaut: `9090`) |
| `AWS_ACCESS_KEY_ID` | Oui | Clé S3 (convention SDK standard) |
| `AWS_SECRET_ACCESS_KEY` | Oui | Secret S3 (convention SDK standard) |

*Au moins un de `SQLITE_PATHS` ou `FS_PATHS` requis.

`DATA_GUARD_CHECKSUM` : supprimé ou optionnel.

### Credentials

Les credentials S3 arrivent via **secret Kubernetes** géré par l'opérateur Infisical — exactement comme les setups Litestream actuels. L'app n'a pas à se soucier d'où vient le secret.

Convention : les vars `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` sont lues automatiquement par le SDK AWS Go, Litestream, et Rclone — aucun mapping nécessaire.

### Kustomize component

Le component doit injecter **init container ET sidecar** en une seule opération.

Le `mountPath` ne doit **pas** être hardcodé — il doit être paramétrable par app (chaque app monte ses volumes où elle veut).

---

## Compatibilité cluster vixens

- S3 backend : **MinIO** (endpoint interne `http://minio.minio.svc.cluster.local:9000`)
- Format Litestream : le sidecar doit écrire dans le même format WAL que Litestream standalone (pour migration sans perte de données existantes)
- `securityContext`: `runAsUser: 1000` (les fichiers appartiennent à uid=1000 dans nos apps)
- Métriques Prometheus sur port `9090` avec label `app=dataangel`

---

## Non-objectifs (hors scope)

- Webhook de mutation automatique (annotation → injection automatique) — V2
- Helm chart — V2
- Backup scheduling / rétention — géré par MinIO lifecycle policies
- Support de backends autres que S3/MinIO
