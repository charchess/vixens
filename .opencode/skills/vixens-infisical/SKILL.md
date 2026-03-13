---
name: vixens-infisical
description: >-
  Vixens Infisical secrets management expert. Use when: secrets not syncing,
  InfisicalSecret failing, need to update secrets in Infisical, secrets out of sync,
  403 errors from services using secrets, need to access Infisical API, secret rotation.
  Trigger on: "infisical", "secret", "credentials", "sync", "InfisicalSecret".
argument-hint: "[app-name or issue description]"
license: MIT
compatibility: opencode
metadata:
  domain: secrets-management
  audience: homelab-operators
---

# Vixens Infisical Secrets Management

Expert en gestion de secrets Infisical pour le cluster Vixens.

**Focus:** $ARGUMENTS

## 📍 Infisical Instance

- **URL**: http://192.168.111.69:8085
- **Type**: Self-hosted Infisical
- **Projet principal**: `vixens` (ID: `47aca60e-543b-4fd6-b646-8ebd5a7b3433`)
- **Environnements**: `dev`, `test`, `staging`, `prod`

## 🔐 Authentification

### Machine Identity (Lecture seule - pour operator K8s)

L'operator Infisical utilise une Universal Auth Machine Identity stockée dans `argocd/infisical-universal-auth`:

```bash
KUBECONFIG=.secrets/prod/kubeconfig-prod kubectl get secret -n argocd infisical-universal-auth -o jsonpath='{.data.clientId}' | base64 -d
# Output: c1a7962a-4766-4a4a-bbd0-8b9ae9d110fd

KUBECONFIG=.secrets/prod/kubeconfig-prod kubectl get secret -n argocd infisical-universal-auth -o jsonpath='{.data.clientSecret}' | base64 -d
# Output: 8d17602c939f77823d24d49a2df03d984f539ed94fc66a9a77d897c653ad6838
```

**Login:**
```bash
export INFISICAL_API_URL=http://192.168.111.69:8085
infisical login --method=universal-auth \
  --client-id=c1a7962a-4766-4a4a-bbd0-8b9ae9d110fd \
  --client-secret=8d17602c939f77823d24d49a2df03d984f539ed94fc66a9a77d897c653ad6838 \
  --silent
```

**⚠️ Limitation**: Cette identité a uniquement des **permissions READ-ONLY**. Elle ne peut pas modifier les secrets.

### Admin Access (Lecture + Écriture)

Pour **modifier** les secrets dans Infisical, utilise les credentials admin.

**Credentials admin** (à récupérer de manière sécurisée):
- Username: `admin@truxonline.com`
- Password: `[demander à l'utilisateur ou récupérer via pass/vault]`

**Login admin via CLI:**
```bash
export INFISICAL_API_URL=http://192.168.111.69:8085
infisical login --domain http://192.168.111.69:8085
# Suivre le prompt interactif ou fournir credentials
```

Le token de retour est un JSON base64-encodé:
```json
{
  "JTWToken": "eyJhbGci...",
  "email": "admin@truxonline.com",
  "privateKey": ""
}
```

Décode le token:
```bash
echo '<token-base64>' | base64 -d | python3 -m json.tool
```

**Utilisation du token admin pour l'API:**
```bash
ADMIN_TOKEN="<JWT-extrait-du-JTWToken>"

# Exemple: Mettre à jour un secret
curl -X PATCH "http://192.168.111.69:8085/api/v3/secrets/raw/SECRET_KEY_NAME" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workspaceId": "47aca60e-543b-4fd6-b646-8ebd5a7b3433",
    "environment": "prod",
    "type": "shared",
    "secretPath": "/apps/XX-category/app-name",
    "secretValue": "new-value"
  }'
```

## 🔄 Synchronisation Infisical → Kubernetes

### Architecture

```
Infisical (source of truth)
    ↓ (sync toutes les 60s)
InfisicalSecret CRD (spec)
    ↓ (controller)
Kubernetes Secret (data)
    ↓ (envFrom)
Pod containers
```

### InfisicalSecret CRD

Exemple pour homeassistant:
```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: homeassistant-secrets-sync
  namespace: homeassistant
spec:
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: argocd
      secretsScope:
        envSlug: prod
        projectSlug: vixens
        secretsPath: /apps/10-home/homeassistant
  hostAPI: http://192.168.111.69:8085
  managedSecretReference:
    creationPolicy: Owner
    secretName: homeassistant-secrets  # Secret K8s créé automatiquement
    secretNamespace: homeassistant
  resyncInterval: 60  # Sync toutes les 60 secondes
```

### Forcer une Synchronisation Immédiate

Après avoir modifié des secrets dans Infisical, tu peux forcer un resync:

```bash
KUBECONFIG=.secrets/prod/kubeconfig-prod kubectl annotate infisicalsecret \
  -n <namespace> <infisicalsecret-name> \
  reconcile.infisical.com/force=true-$(date +%s) \
  --overwrite
```

**Exemple:**
```bash
kubectl annotate infisicalsecret -n homeassistant homeassistant-secrets-sync \
  reconcile.infisical.com/force=true-$(date +%s) --overwrite
```

**Vérification:**
```bash
# Attendre 5-10 secondes, puis vérifier le secret K8s
kubectl get secret -n homeassistant homeassistant-secrets -o jsonpath='{.data.KEY_NAME}' | base64 -d
```

### Reload des Pods après Modification de Secret

**⚠️ Les pods ne rechargent PAS automatiquement les secrets.**

Après une modification de secret dans Infisical:

1. **Forcer resync** (voir ci-dessus)
2. **Redémarrer les pods:**
   ```bash
   # Option 1: Rolling restart (graceful)
   kubectl rollout restart deployment/<app-name> -n <namespace>
   
   # Option 2: Suppression directe (plus rapide mais downtime)
   kubectl delete pod -n <namespace> -l app=<app-name>
   ```

**Exemple complet:**
```bash
# 1. Modifier secret dans Infisical (via UI ou API)
# 2. Forcer resync
kubectl annotate infisicalsecret -n homeassistant homeassistant-secrets-sync \
  reconcile.infisical.com/force=true-$(date +%s) --overwrite

# 3. Vérifier que le secret K8s est à jour
kubectl get secret -n homeassistant homeassistant-secrets -o jsonpath='{.data.LITESTREAM_ACCESS_KEY_ID}' | base64 -d

# 4. Redémarrer le pod
kubectl rollout restart deployment/homeassistant -n homeassistant

# 5. Vérifier dans le nouveau pod
kubectl exec -n homeassistant <new-pod-name> -c <container-name> -- env | grep KEY_NAME
```

## 🛠️ Diagnostic

### Vérifier le Status d'une InfisicalSecret

```bash
kubectl get infisicalsecret -n <namespace>
kubectl describe infisicalsecret -n <namespace> <name>
```

**Status conditions à vérifier:**
- `secrets.infisical.com/LoadedInfisicalToken` → Token chargé OK
- `secrets.infisical.com/ReadyToSyncSecrets` → Synchro active
- `secrets.infisical.com/AutoRedeployReady` → Auto-redeploy (si configuré)

### Logs de l'Operator

```bash
kubectl logs -n infisical-operator-system deployment/infisical-opera-controller-manager
```

### Vérifier qu'un Secret K8s Contient les Bonnes Valeurs

```bash
# Lister toutes les clés
kubectl get secret -n <namespace> <secret-name> -o jsonpath='{.data}' | python3 -c "import sys,json; print('\n'.join(json.load(sys.stdin).keys()))"

# Décoder une valeur
kubectl get secret -n <namespace> <secret-name> -o jsonpath='{.data.KEY_NAME}' | base64 -d
```

## 📋 Checklist Troubleshooting

Quand un secret ne fonctionne pas:

- [ ] Le secret existe dans Infisical? (vérifier via UI http://192.168.111.69:8085)
- [ ] Le path dans `InfisicalSecret.spec.secretsScope.secretsPath` est correct?
- [ ] L'env slug (`prod`, `dev`, etc.) est correct?
- [ ] Le secret K8s existe? (`kubectl get secret -n <ns> <name>`)
- [ ] Le secret K8s contient la clé attendue? (voir "Vérifier qu'un Secret K8s...")
- [ ] L'InfisicalSecret a-t-elle des erreurs? (`kubectl describe infisicalsecret...`)
- [ ] Le pod a-t-il été redémarré après la modification? (voir "Reload des Pods")
- [ ] Le pod monte-t-il le secret correctement? (`kubectl describe pod` → `Volumes` et `Mounts`)
- [ ] Les credentials dans le pod sont-elles à jour? (`kubectl exec ... -- env | grep KEY`)

## 🔒 Bonnes Pratiques

### Rotation de Credentials

Quand tu remplaces des credentials (ex: clés S3, DB passwords):

1. **Créer les nouvelles credentials** (ex: user MinIO, password DB)
2. **Tester** que les nouvelles credentials fonctionnent (curl, mc, psql, etc.)
3. **Mettre à jour dans Infisical** (via UI ou API admin)
4. **Forcer resync** (`kubectl annotate...`)
5. **Redémarrer les pods** (`kubectl rollout restart...`)
6. **Vérifier dans les logs** que les nouvelles credentials sont utilisées
7. **Supprimer les anciennes credentials** (si applicable)

### Organisation des Secrets

Structure recommandée dans Infisical:
```
/apps/XX-category/app-name/
  ├── DB_HOST
  ├── DB_USER
  ├── DB_PASSWORD
  ├── S3_ACCESS_KEY
  ├── S3_SECRET_KEY
  └── ...
```

**Exemples:**
- `/apps/10-home/homeassistant/`
- `/apps/20-media/jellyfin/`
- `/apps/70-tools/netbox/`

### Secrets Communs vs Spécifiques

- **Spécifique à l'app**: Dans le path de l'app (`/apps/XX/app-name/`)
- **Partagé** (ex: credentials MinIO réutilisées): Dupliquer dans chaque app (évite les dépendances)

## 🚨 Sécurité

**⚠️ JAMAIS commiter de secrets dans Git**

- Les secrets vont **UNIQUEMENT** dans Infisical
- Les manifests K8s référencent les `InfisicalSecret`, pas les valeurs
- Utilise `.gitignore` pour exclure fichiers de secrets locaux

**Vérifier avant commit:**
```bash
git diff --cached | grep -iE "(password|secret|key|token|credential)" && echo "⚠️ Potentiel secret détecté"
```

## 📚 Références

- [Infisical Docs](https://infisical.com/docs)
- [Infisical Kubernetes Operator](https://infisical.com/docs/integrations/platforms/kubernetes)
- Vixens ADR: `docs/adr/011-infisical-secrets-management.md`

---

**Rappel:** Toute modification de secret nécessite **resync + pod restart**.
