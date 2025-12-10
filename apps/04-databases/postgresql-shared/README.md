# PostgreSQL Shared Cluster

Cluster PostgreSQL mutualisé pour toutes les applications nécessitant une base de données.

## Architecture

- **Operator**: CloudNativePG v1.28.0
- **PostgreSQL**: 17.6
- **Namespace**: `databases`
- **Instances**: 1 (dev) → 3 (staging/prod pour HA)
- **Storage**: 20Gi (Synology CSI)
- **Resources**: 100m/256Mi → 500m/512Mi

## Services

- `postgresql-shared-rw:5432` - Read-Write (pour les applications)
- `postgresql-shared-ro:5432` - Read-Only (pour analytics/backups)
- `postgresql-shared-r:5432` - Replication

## Infisical Secrets Structure

```
/postgresql/
  ├── admin/           # Super-user (postgres)
  │   ├── username     # postgres
  │   └── password     # <strong-password>
  │
  └── {app}/           # Per-app credentials
      ├── username     # {app}
      ├── password     # <strong-password>
      └── database     # {app} (optional, defaults to username)
```

## Ajouter une nouvelle application

### 1. Créer les secrets Infisical

Dans Infisical (project: vixens, env: dev):
- Path: `/postgresql/{app}/`
- Secrets:
  - `username` = `{app}`
  - `password` = `<generate-strong-password>`

### 2. Créer le user PostgreSQL

**⚠️ MANUEL pour l'instant (automatisation planifiée)**

```bash
# Get password from Infisical secret
PASS=$(kubectl get secret -n databases {app}-postgresql-credentials -o jsonpath='{.data.password}' | base64 -d)

# Create user
kubectl exec -n databases postgresql-shared-1 -- psql -U postgres -c "CREATE USER {app} WITH PASSWORD '$PASS';"
```

### 3. Créer la Database CRD

Fichier: `apps/04-databases/postgresql-shared/base/databases/{app}.yaml`

```yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Database
metadata:
  name: {app}
  namespace: databases
spec:
  cluster:
    name: postgresql-shared
  name: {app}
  owner: {app}
  ensure: present
```

### 4. Créer l'InfisicalSecret CRD

Fichier: `apps/04-databases/postgresql-shared/base/credentials/{app}-secret.yaml`

```yaml
---
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: {app}-postgresql-credentials
  namespace: databases
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
        envSlug: dev
        secretsPath: "/postgresql/{app}"
  managedSecretReference:
    secretName: {app}-postgresql-credentials
    secretNamespace: databases
    creationPolicy: "Owner"
```

### 5. Mettre à jour kustomization.yaml

```yaml
resources:
  - databases/{app}.yaml
  - credentials/{app}-secret.yaml
```

### 6. Connexion depuis l'application

Connection string:
```
postgresql://{app}:<password>@postgresql-shared-rw:5432/{app}
```

Ou variables d'environnement:
```yaml
env:
  - name: DB_HOST
    value: "postgresql-shared-rw"
  - name: DB_PORT
    value: "5432"
  - name: DB_NAME
    value: "{app}"
  - name: DB_USER
    valueFrom:
      secretKeyRef:
        name: {app}-postgresql-credentials
        key: username
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: {app}-postgresql-credentials
        key: password
```

## Applications déployées

- **Docspell**: Document management system (database: `docspell`, user: `docspell`)

## Tâches à venir

- [ ] Automatiser la création de users PostgreSQL via Job/Script
- [ ] Implémenter backup strategy (S3/MinIO)
- [ ] Configurer HA (3 replicas) pour staging/prod
- [ ] Intégrer monitoring Prometheus

## Troubleshooting

### Database CRD "role does not exist"

Le user PostgreSQL doit exister **avant** que la Database CRD ne crée la base.

```bash
# Vérifier si le user existe
kubectl exec -n databases postgresql-shared-1 -- psql -U postgres -c "\du" | grep {app}

# Créer le user si absent (voir étape 2)
```

### Secret non synchronisé

```bash
# Vérifier le statut de l'InfisicalSecret
kubectl get infisicalsecret -n databases {app}-postgresql-credentials -o yaml

# Forcer la reconciliation
kubectl annotate infisicalsecret {app}-postgresql-credentials \
  -n databases \
  --overwrite \
  reconcile="$(date +%s)"
```

### Connexion refusée

```bash
# Tester la connexion depuis un pod
kubectl run psql-test --rm -it --image=postgres:17 -- \
  psql -h postgresql-shared-rw -U {app} -d {app}
```
