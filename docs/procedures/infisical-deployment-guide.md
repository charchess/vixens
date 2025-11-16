# Guide de Déploiement Infisical Operator

**Date de création**: 2025-11-16
**Version**: 1.0
**Environnement**: Multi-cluster (dev, test, staging, prod)

---

## Vue d'Ensemble

Ce guide documente le déploiement complet de la gestion des secrets via **Infisical Kubernetes Operator** pour le projet vixens.

**Architecture:**
- **Infisical instance**: NAS Synology (192.168.111.69:8085)
- **Projet Infisical**: `vixens`
- **Environnements**: dev, test, staging, prod
- **Operator**: Infisical Kubernetes Operator v0.3.0
- **Authentification**: Universal Auth (Machine Identity par environnement)

---

## Prérequis

### 1. Infisical Instance Opérationnelle

Vérifier que l'instance Infisical est accessible:

```bash
curl -I http://192.168.111.69:8085
# Attendu: HTTP/1.1 200 OK
```

### 2. Cluster Kubernetes Fonctionnel

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
kubectl get nodes
# Les 3 control planes doivent être Ready
```

### 3. ArgoCD Déployé

```bash
kubectl get pods -n argocd
# argocd-server, argocd-repo-server, etc. doivent être Running
```

---

## Étape 1: Configuration Infisical UI

### 1.1 Créer le Projet

1. Accéder à http://192.168.111.69:8085
2. Se connecter avec votre compte
3. Créer nouveau projet:
   - **Name**: `vixens`
   - **Description**: "Homelab Kubernetes multi-cluster"

### 1.2 Créer les Environnements

Les environnements sont créés automatiquement lors de la création du projet:
- Development (dev) ✅
- Staging (staging) ✅
- Production (prod) ✅

**Ajouter l'environnement test:**
1. Settings → Environments → Add Environment
2. **Name**: `test`
3. **Slug**: `test`
4. Save

**Résultat attendu:**
```
vixens/
├── dev
├── test
├── staging
└── prod
```

### 1.3 Créer les Machine Identities (Universal Auth)

**Pour CHAQUE environnement** (dev, test, staging, prod):

1. Settings → Machine Identities → Create Identity
2. Remplir:
   - **Name**: `vixens-dev-k8s-operator` (remplacer `dev` par l'environnement)
   - **Description**: "Kubernetes Operator for dev cluster"
   - **Organization Role**: `No Access` (permissions au niveau projet)
3. Créer
4. Cliquer sur l'identité créée → Universal Auth → Enable
5. **IMPORTANT**: Copier et sauvegarder:
   - `Client ID`: (ex: `8f9d2e4a-1b3c-4d5e-6f7g-8h9i0j1k2l3m`)
   - `Client Secret`: (ex: `abc123def456...`) **⚠️ Visible une seule fois!**
6. Configurer les permissions:
   - Project → `vixens`
   - Environments → Sélectionner uniquement l'environnement correspondant (ex: `dev`)
   - Role → `Admin` (ou `Developer` si read-only secrets souhaité)
7. Save

**Répéter pour les 4 environnements.**

**Credentials à sauvegarder** (temporairement, dans un fichier sécurisé):

```
# vixens-dev-k8s-operator
CLIENT_ID_DEV="8f9d2e4a-1b3c-4d5e-6f7g-8h9i0j1k2l3m"
CLIENT_SECRET_DEV="abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"

# vixens-test-k8s-operator
CLIENT_ID_TEST="..."
CLIENT_SECRET_TEST="..."

# vixens-staging-k8s-operator
CLIENT_ID_STAGING="..."
CLIENT_SECRET_STAGING="..."

# vixens-prod-k8s-operator
CLIENT_ID_PROD="..."
CLIENT_SECRET_PROD="..."
```

### 1.4 Importer les Secrets Existants

**Pour l'environnement dev** (exemple):

1. Aller dans le projet `vixens` → Environment `dev`
2. Cliquer sur **Add Secret**

**Secret 1: Gandi API Token**
- **Key**: `gandi-api-token`
- **Value**: `42a2a1c56e29bbff391345eafe811e2e03ba0586`
- Save

**Secret 2: Synology CSI Host**
- **Key**: `synology-csi-host`
- **Value**: `192.168.111.69`
- Save

**Secret 3: Synology CSI Username**
- **Key**: `synology-csi-username`
- **Value**: `talos-csi`
- Save

**Secret 4: Synology CSI Password**
- **Key**: `synology-csi-password`
- **Value**: `w61CpDQs2RB7bfE8r2eSY3zB7dXJ6pBCAudfuK8fDDN43Oi3Jw`
- Save

**Résultat dans Infisical:**

```
vixens/dev:
├── gandi-api-token
├── synology-csi-host
├── synology-csi-username
└── synology-csi-password
```

**Répéter pour test/staging/prod** avec les valeurs spécifiques à chaque environnement.

---

## Étape 2: Déploiement Infisical Operator via ArgoCD

### 2.1 Créer l'Application ArgoCD

**Fichier**: `argocd/overlays/dev/apps/infisical-operator.yaml`

```yaml
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: infisical-operator
  namespace: argocd
  labels:
    app.kubernetes.io/part-of: vixens
    app.kubernetes.io/component: secrets-management
spec:
  project: default
  source:
    repoURL: https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/
    chart: infisical-secrets-operator
    targetRevision: 0.3.0
    helm:
      values: |
        controllerManager:
          manager:
            image:
              repository: infisical/kubernetes-operator
              tag: v0.3.0
            resources:
              limits:
                cpu: 500m
                memory: 256Mi
              requests:
                cpu: 100m
                memory: 128Mi
          kubeRbacProxy:
            image:
              repository: gcr.io/kubebuilder/kube-rbac-proxy
              tag: v0.13.1
  destination:
    server: https://kubernetes.default.svc
    namespace: infisical-operator-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

### 2.2 Référencer dans Kustomization

Éditer `argocd/overlays/dev/kustomization.yaml`:

```yaml
resources:
  # ... autres apps
  - apps/infisical-operator.yaml  # Ajouter cette ligne (wave -1, avant les apps utilisant des secrets)
  - apps/sealed-secrets.yaml
  # ... reste
```

**Wave ordering recommandé:**

```yaml
# argocd/overlays/dev/apps/infisical-operator.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "-3"  # Déployé avant sealed-secrets (-1) et cilium-lb (-2)
```

### 2.3 Commiter et Pousser

```bash
cd /root/vixens
git add argocd/overlays/dev/apps/infisical-operator.yaml
git add argocd/overlays/dev/kustomization.yaml
git commit -m "feat(secrets): Add Infisical Operator deployment"
git push origin feature/infisical-secrets
```

### 2.4 Vérifier le Déploiement

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Attendre que l'application soit Synced
kubectl get application infisical-operator -n argocd -w

# Vérifier les pods (peut prendre 2-3 minutes)
kubectl get pods -n infisical-operator-system

# Attendu:
# NAME                                                   READY   STATUS    RESTARTS   AGE
# infisical-operator-controller-manager-xxx-xxx         2/2     Running   0          2m
```

### 2.5 Vérifier les CRDs

```bash
kubectl get crd | grep infisical

# Attendu:
# infisicalsecrets.secrets.infisical.com   2025-11-16T...
```

---

## Étape 3: Configuration Universal Auth (Bootstrap)

### 3.1 Créer le Secret Universal Auth (Dev)

**⚠️ IMPORTANT**: Ce secret contient les credentials Infisical. Il doit être créé **une seule fois** manuellement.

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Remplacer par les vraies valeurs copiées depuis Infisical UI
kubectl create secret generic infisical-universal-auth \
  --namespace=infisical-operator-system \
  --from-literal=clientId="8f9d2e4a-1b3c-4d5e-6f7g-8h9i0j1k2l3m" \
  --from-literal=clientSecret="abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"
```

**Vérification:**

```bash
kubectl get secret infisical-universal-auth -n infisical-operator-system

# NAME                         TYPE     DATA   AGE
# infisical-universal-auth     Opaque   2      5s
```

**⚠️ Sécurité**: Ce secret est critique. En cas de compromission, regénérer la Machine Identity dans Infisical UI.

### 3.2 Répéter pour Test/Staging/Prod

Lors du déploiement sur les autres environnements, créer le même secret avec les credentials correspondants:

```bash
# Test cluster
export KUBECONFIG=/root/vixens/terraform/environments/test/kubeconfig-test
kubectl create secret generic infisical-universal-auth \
  --namespace=infisical-operator-system \
  --from-literal=clientId="$CLIENT_ID_TEST" \
  --from-literal=clientSecret="$CLIENT_SECRET_TEST"

# Staging cluster
export KUBECONFIG=/root/vixens/terraform/environments/staging/kubeconfig-staging
kubectl create secret generic infisical-universal-auth \
  --namespace=infisical-operator-system \
  --from-literal=clientId="$CLIENT_ID_STAGING" \
  --from-literal=clientSecret="$CLIENT_SECRET_STAGING"

# Prod cluster
export KUBECONFIG=/root/vixens/terraform/environments/prod/kubeconfig-prod
kubectl create secret generic infisical-universal-auth \
  --namespace=infisical-operator-system \
  --from-literal=clientId="$CLIENT_ID_PROD" \
  --from-literal=clientSecret="$CLIENT_SECRET_PROD"
```

---

## Étape 4: Migration des Secrets Existants

### 4.1 Secret Gandi (cert-manager)

**Créer le fichier InfisicalSecret:**

`apps/cert-manager-config/base/infisical-secret-gandi.yaml`

```yaml
---
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: gandi-credentials-infisical
  namespace: cert-manager
spec:
  hostAPI: http://192.168.111.69:8085
  resyncInterval: 60  # Polling toutes les 60 secondes

  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens
        envSlug: dev  # Changera selon l'overlay (dev, test, staging, prod)
        secretsPath: "/"
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: infisical-operator-system

  managedSecretReference:
    secretName: gandi-credentials
    secretNamespace: cert-manager
    secretType: Opaque
    creationPolicy: "Owner"  # L'operator gère complètement le secret
```

**Ajouter au kustomization.yaml:**

`apps/cert-manager-config/base/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: cert-manager

resources:
  - infisical-secret-gandi.yaml  # Nouvelle ligne
  - clusterissuer-staging.yaml
  - clusterissuer-prod.yaml
```

**Créer l'overlay dev pour envSlug:**

`apps/cert-manager-config/overlays/dev/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: cert-manager

resources:
  - ../../base

patches:
  - target:
      kind: InfisicalSecret
      name: gandi-credentials-infisical
    patch: |-
      - op: replace
        path: /spec/authentication/universalAuth/secretsScope/envSlug
        value: dev
```

**Mettre à jour l'Application ArgoCD:**

`argocd/overlays/dev/apps/cert-manager-config.yaml`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cert-manager-config
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "2"  # Après cert-manager (wave 0) et webhook (wave 1)
spec:
  project: default
  source:
    repoURL: https://github.com/thetruefuss/vixens.git  # Remplacer par votre repo
    targetRevision: dev
    path: apps/cert-manager-config/overlays/dev  # Utilise l'overlay
  destination:
    server: https://kubernetes.default.svc
    namespace: cert-manager
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**Commiter et pousser:**

```bash
cd /root/vixens
git add apps/cert-manager-config/
git add argocd/overlays/dev/apps/cert-manager-config.yaml
git commit -m "feat(secrets): Migrate Gandi credentials to Infisical"
git push origin feature/infisical-secrets
```

**Vérifier la synchronisation:**

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Attendre sync ArgoCD (auto ou manuel)
kubectl get application cert-manager-config -n argocd -w

# Vérifier que l'InfisicalSecret est créé
kubectl get infisicalsecrets -n cert-manager

# NAME                            AGE
# gandi-credentials-infisical     30s

# Vérifier que le Secret K8s est créé/mis à jour par l'operator
kubectl get secret gandi-credentials -n cert-manager -o yaml

# Le secret doit contenir:
# data:
#   gandi-api-token: <base64(42a2a1c56e29bbff391345eafe811e2e03ba0586)>
```

**⚠️ Important**: Si le secret `gandi-credentials` existait déjà (créé manuellement), il sera remplacé par l'operator.

### 4.2 Secret Synology CSI

**Problème**: Synology CSI attend un **ConfigMap** (`synology-client-info`) avec la clé `client-info.yml`, pas un Secret.

**Solutions possibles:**

#### Option A: Job de conversion Secret → ConfigMap (Recommandé pour migration rapide)

**Créer le fichier InfisicalSecret:**

`apps/synology-csi/base/infisical-secret-synology.yaml`

```yaml
---
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: synology-csi-credentials-infisical
  namespace: synology-csi
spec:
  hostAPI: http://192.168.111.69:8085
  resyncInterval: 60

  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens
        envSlug: dev
        secretsPath: "/"
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: infisical-operator-system

  managedSecretReference:
    secretName: synology-csi-credentials
    secretNamespace: synology-csi
    secretType: Opaque
    creationPolicy: "Owner"
```

**Créer un Job pour convertir:**

`apps/synology-csi/base/configmap-sync-job.yaml`

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: synology-configmap-syncer
  namespace: synology-csi
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: synology-configmap-syncer
  namespace: synology-csi
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["synology-csi-credentials"]
    verbs: ["get", "watch"]
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: ["synology-client-info"]
    verbs: ["get", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: synology-configmap-syncer
  namespace: synology-csi
subjects:
  - kind: ServiceAccount
    name: synology-configmap-syncer
roleRef:
  kind: Role
  name: synology-configmap-syncer
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: synology-configmap-sync
  namespace: synology-csi
spec:
  schedule: "*/2 * * * *"  # Toutes les 2 minutes
  concurrencyPolicy: Replace
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: synology-configmap-syncer
          restartPolicy: OnFailure
          containers:
          - name: sync
            image: bitnami/kubectl:1.30
            command:
            - /bin/bash
            - -c
            - |
              set -e

              # Récupérer les valeurs du Secret
              HOST=$(kubectl get secret synology-csi-credentials -n synology-csi -o jsonpath='{.data.synology-csi-host}' | base64 -d)
              USERNAME=$(kubectl get secret synology-csi-credentials -n synology-csi -o jsonpath='{.data.synology-csi-username}' | base64 -d)
              PASSWORD=$(kubectl get secret synology-csi-credentials -n synology-csi -o jsonpath='{.data.synology-csi-password}' | base64 -d)

              # Créer le ConfigMap
              kubectl create configmap synology-client-info \
                --namespace=synology-csi \
                --from-literal=client-info.yml="$(cat <<EOF
              ---
              clients:
                - host: \"${HOST}\"
                  port: 5000
                  https: false
                  username: \"${USERNAME}\"
                  password: \"${PASSWORD}\"
              EOF
              )" \
                --dry-run=client -o yaml | kubectl apply -f -

              echo "ConfigMap synology-client-info synchronized successfully"
```

**⚠️ Limitation**: Le ConfigMap est mis à jour toutes les 2 minutes maximum (intervalle CronJob). Ce n'est pas du temps réel.

#### Option B: Patcher Synology CSI pour lire un Secret (Recommandé long terme)

**Étapes futures (Sprint 8+):**
1. Fork du repo Synology CSI
2. Modifier le code pour lire `client-info.yml` depuis un Secret au lieu d'un ConfigMap
3. Rebuild de l'image Docker
4. Push vers registry privé ou PR upstream

**Avantage**: Synchronisation temps réel avec Infisical (60s max).

#### Option C: Utiliser le ConfigMap existant (Pas de migration)

**Simple mais non recommandé**: Continuer à créer le ConfigMap manuellement après chaque rebuild.

**Pour cette migration, utiliser Option A (CronJob).**

---

## Étape 5: Validation de la Migration

### 5.1 Vérifier les InfisicalSecrets

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

kubectl get infisicalsecrets -A

# NAMESPACE        NAME                              AGE
# cert-manager     gandi-credentials-infisical       5m
# synology-csi     synology-csi-credentials-infisical 5m
```

### 5.2 Vérifier les Secrets K8s Générés

```bash
# Gandi
kubectl get secret gandi-credentials -n cert-manager -o jsonpath='{.data.api-token}' | base64 -d
# Attendu: 42a2a1c56e29bbff391345eafe811e2e03ba0586

# Synology
kubectl get secret synology-csi-credentials -n synology-csi -o yaml
# Doit contenir les clés: synology-csi-host, synology-csi-username, synology-csi-password
```

### 5.3 Vérifier le ConfigMap Synology (si Option A)

```bash
kubectl get configmap synology-client-info -n synology-csi -o yaml

# Doit contenir client-info.yml avec les bonnes valeurs
```

### 5.4 Tester cert-manager

```bash
# Forcer un renouvellement de certificat
kubectl delete certificate whoami-tls -n whoami
kubectl get certificate -n whoami -w

# Le certificat doit être recréé avec succès (challenge DNS-01 Gandi)
```

### 5.5 Tester Synology CSI (si déployé)

```bash
# Créer un PVC test
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: synology-iscsi
EOF

# Vérifier le provisioning
kubectl get pvc test-pvc -w

# Nettoyer
kubectl delete pvc test-pvc
```

---

## Étape 6: Nettoyage des Secrets Manuels

### 6.1 Supprimer les Fichiers Plaintext

Une fois la migration validée:

```bash
cd /root/vixens

# Supprimer .secrets/dev/
git rm -r .secrets/dev/

# Mettre à jour .gitignore (si nécessaire)
echo ".secrets/" >> .gitignore

git add .gitignore
git commit -m "chore(secrets): Remove plaintext secrets after Infisical migration"
git push origin feature/infisical-secrets
```

### 6.2 Supprimer le Script Bootstrap (optionnel)

Si `scripts/bootstrap-secrets.sh` n'est plus nécessaire:

```bash
git rm scripts/bootstrap-secrets.sh
git commit -m "chore(secrets): Remove manual secret bootstrap script"
```

### 6.3 Mettre à Jour la Documentation

Éditer `docs/procedures/cluster-rebuild-procedure.md`:

**Remplacer la section "Étape 5: Appliquer les Secrets Manuels"** par:

```markdown
### Étape 5: Vérifier la Synchronisation Infisical (1 minute)

**⚠️ PRÉREQUIS**: Universal Auth credentials doivent être configurés dans le cluster.

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Vérifier que l'Infisical Operator est Running
kubectl get pods -n infisical-operator-system

# Vérifier que les secrets sont synchronisés
kubectl get infisicalsecrets -A
kubectl get secret gandi-credentials -n cert-manager
kubectl get secret synology-csi-credentials -n synology-csi
```

**Si les secrets n'existent pas:**
- Vérifier que le Secret `infisical-universal-auth` existe dans `infisical-operator-system`
- Vérifier les logs de l'operator: `kubectl logs -n infisical-operator-system -l app.kubernetes.io/name=infisical-operator`
```

Commiter:

```bash
git add docs/procedures/cluster-rebuild-procedure.md
git commit -m "docs(procedures): Update rebuild procedure with Infisical sync"
git push origin feature/infisical-secrets
```

---

## Étape 7: Rotation de Secrets (Procédure)

### 7.1 Rotation d'un Secret Application (Ex: Gandi API Token)

**Cas d'usage**: Suspicion de compromission du token Gandi.

**Étapes:**

1. **Générer nouveau token** dans Gandi UI:
   - https://account.gandi.net/
   - Aller dans Security → API Keys → Generate new API key
   - Copier le nouveau token: `new-token-123abc456def789`

2. **Mettre à jour dans Infisical UI:**
   - http://192.168.111.69:8085
   - Projet `vixens` → Environment `dev`
   - Modifier le secret `gandi-api-token`
   - **Value**: `new-token-123abc456def789`
   - Save

3. **Attendre la synchronisation automatique** (max 60 secondes):
   ```bash
   export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

   # Vérifier que le Secret K8s est mis à jour
   kubectl get secret gandi-credentials -n cert-manager -o jsonpath='{.data.api-token}' | base64 -d
   # Doit afficher: new-token-123abc456def789
   ```

4. **Redémarrer les pods consommateurs** (si nécessaire):
   ```bash
   # cert-manager lit le secret au démarrage et lors des challenges DNS
   # Redémarrage optionnel pour forcer rechargement:
   kubectl rollout restart deployment cert-manager -n cert-manager
   kubectl rollout restart deployment cert-manager-webhook-gandi -n cert-manager
   ```

5. **Valider** en testant un challenge DNS-01:
   ```bash
   # Forcer renouvellement certificat
   kubectl delete certificate whoami-tls -n whoami
   kubectl get certificate -n whoami -w
   # STATUS doit passer à Ready
   ```

6. **Révoquer l'ancien token** dans Gandi UI (si compromis confirmé)

### 7.2 Rotation Universal Auth Credentials

**Cas d'usage**: `clientSecret` compromis.

**Étapes:**

1. **Créer nouvelle Machine Identity** dans Infisical UI:
   - Settings → Machine Identities → Create Identity
   - **Name**: `vixens-dev-k8s-operator-new`
   - Générer Universal Auth
   - Copier `clientId` et `clientSecret`
   - Configurer permissions (projet `vixens`, env `dev`, role `Admin`)

2. **Mettre à jour le Secret K8s:**
   ```bash
   export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

   kubectl delete secret infisical-universal-auth -n infisical-operator-system
   kubectl create secret generic infisical-universal-auth \
     --namespace=infisical-operator-system \
     --from-literal=clientId="<NEW_CLIENT_ID>" \
     --from-literal=clientSecret="<NEW_CLIENT_SECRET>"
   ```

3. **Redémarrer l'operator:**
   ```bash
   kubectl rollout restart deployment infisical-operator-controller-manager \
     -n infisical-operator-system
   ```

4. **Vérifier la reconnexion:**
   ```bash
   kubectl logs -n infisical-operator-system -l app.kubernetes.io/name=infisical-operator --tail=50
   # Doit afficher des logs de synchronisation réussie
   ```

5. **Supprimer l'ancienne Machine Identity** dans Infisical UI

---

## Troubleshooting

### Problème 1: InfisicalSecret Créé mais Secret K8s Non Créé

**Symptômes:**
```bash
kubectl get infisicalsecrets -n cert-manager
# NAME                          AGE
# gandi-credentials-infisical   5m

kubectl get secret gandi-credentials -n cert-manager
# Error from server (NotFound): secrets "gandi-credentials" not found
```

**Diagnostic:**

```bash
# Vérifier les logs de l'operator
kubectl logs -n infisical-operator-system -l app.kubernetes.io/name=infisical-operator --tail=100

# Vérifier le status de l'InfisicalSecret
kubectl describe infisicalsecret gandi-credentials-infisical -n cert-manager
```

**Causes possibles:**

1. **Credentials Universal Auth invalides:**
   - Vérifier que `infisical-universal-auth` existe et contient les bonnes valeurs
   - Solution: Recréer le secret avec les bons credentials

2. **Machine Identity sans permissions:**
   - Vérifier dans Infisical UI que l'identité a bien accès au projet/env
   - Solution: Configurer les permissions dans Infisical UI

3. **Infisical API inaccessible:**
   ```bash
   # Tester depuis un pod dans le cluster
   kubectl run test-infisical --image=curlimages/curl --rm -it --restart=Never -- \
     curl -v http://192.168.111.69:8085/api/status
   ```
   - Solution: Vérifier réseau, firewall, état du NAS

4. **Secret path invalide:**
   - Vérifier que `secretsPath: "/"` correspond au path dans Infisical
   - Solution: Ajuster `secretsPath` dans l'InfisicalSecret

### Problème 2: Secret Synchronisé mais Valeur Incorrecte

**Symptômes:**
```bash
kubectl get secret gandi-credentials -n cert-manager -o jsonpath='{.data.api-token}' | base64 -d
# Affiche une valeur différente de celle dans Infisical
```

**Diagnostic:**

```bash
# Vérifier le mapping des clés dans Infisical
# Key dans Infisical: gandi-api-token
# Key dans Secret K8s: api-token
```

**Cause**: Mapping clé Infisical → Secret K8s incorrect.

**Solution**: Par défaut, l'operator utilise les clés telles quelles. Pour mapper:

```yaml
# Dans InfisicalSecret
spec:
  managedSecretReference:
    secretName: gandi-credentials
    secretNamespace: cert-manager
    secretType: Opaque
    # AJOUT: mapping des clés
    keyMapping:
      - infisicalKey: gandi-api-token
        kubernetesKey: api-token
```

**Alternative**: Renommer la clé dans Infisical pour correspondre à l'attendu par l'application.

### Problème 3: Synchronisation Lente (> 60 secondes)

**Symptômes:**
- Modification dans Infisical UI
- Secret K8s mis à jour après 3-5 minutes

**Cause**: `resyncInterval: 60` = polling toutes les 60 secondes, pas temps réel.

**Solutions:**

1. **Réduire l'intervalle** (compromis charge réseau):
   ```yaml
   spec:
     resyncInterval: 30  # 30 secondes
   ```

2. **Forcer resync manuel**:
   ```bash
   # Redémarrer l'operator
   kubectl rollout restart deployment infisical-operator-controller-manager \
     -n infisical-operator-system
   ```

3. **Utiliser webhooks** (future fonctionnalité Infisical, pas encore supporté)

### Problème 4: ConfigMap Synology Non Mis à Jour (Option A)

**Symptômes:**
- Secret `synology-csi-credentials` mis à jour
- ConfigMap `synology-client-info` conserve anciennes valeurs

**Cause**: CronJob `synology-configmap-sync` s'exécute toutes les 2 minutes.

**Solutions:**

1. **Forcer exécution immédiate**:
   ```bash
   kubectl create job --from=cronjob/synology-configmap-sync \
     synology-configmap-sync-manual -n synology-csi

   # Vérifier les logs
   kubectl logs job/synology-configmap-sync-manual -n synology-csi
   ```

2. **Réduire l'intervalle CronJob** (compromis charge):
   ```yaml
   spec:
     schedule: "* * * * *"  # Toutes les minutes
   ```

3. **Migrer vers Option B** (patcher Synology CSI pour lire Secret directement)

---

## Checklist de Validation Post-Déploiement

Avant de considérer la migration comme réussie:

- [ ] Infisical Operator déployé et Running (1 pod 2/2)
- [ ] CRD `infisicalsecrets.secrets.infisical.com` créé
- [ ] Secret `infisical-universal-auth` créé dans `infisical-operator-system`
- [ ] InfisicalSecret `gandi-credentials-infisical` créé dans `cert-manager`
- [ ] Secret K8s `gandi-credentials` créé et contient la bonne valeur
- [ ] InfisicalSecret `synology-csi-credentials-infisical` créé (si applicable)
- [ ] Secret K8s `synology-csi-credentials` créé (si applicable)
- [ ] ConfigMap `synology-client-info` créé via CronJob (si Option A)
- [ ] Certificat TLS whoami renouvelé avec succès (challenge DNS-01 Gandi)
- [ ] PVC Synology CSI créé avec succès (si applicable)
- [ ] Fichiers `.secrets/dev/` supprimés de Git
- [ ] Documentation `cluster-rebuild-procedure.md` mise à jour
- [ ] Test de rotation d'un secret (ex: Gandi token) réussi

---

## Commandes de Référence Rapide

```bash
# Vérifier état Infisical Operator
kubectl get pods -n infisical-operator-system
kubectl logs -n infisical-operator-system -l app.kubernetes.io/name=infisical-operator

# Lister tous les InfisicalSecrets
kubectl get infisicalsecrets -A

# Décrire un InfisicalSecret (voir status, erreurs)
kubectl describe infisicalsecret <name> -n <namespace>

# Vérifier un Secret K8s géré par Infisical
kubectl get secret <name> -n <namespace> -o yaml

# Forcer resync immédiat (redémarrer operator)
kubectl rollout restart deployment infisical-operator-controller-manager -n infisical-operator-system

# Tester connectivité Infisical API depuis cluster
kubectl run test-infisical --image=curlimages/curl --rm -it --restart=Never -- \
  curl -v http://192.168.111.69:8085/api/status

# Exécuter manuellement le CronJob Synology (Option A)
kubectl create job --from=cronjob/synology-configmap-sync \
  synology-configmap-sync-manual -n synology-csi
```

---

## Prochaines Étapes

### Sprint 7 (Actuel)
- [x] Documentation complète Infisical
- [ ] Déploiement Infisical Operator sur dev
- [ ] Migration secrets Gandi et Synology CSI
- [ ] Test rotation Gandi token
- [ ] Validation complète

### Sprint 8+
- [ ] Répliquer sur test/staging/prod
- [ ] Migrer Synology CSI vers lecture Secret (Option B)
- [ ] Ajouter secrets pour nouvelles applications (Authelia, PostgreSQL, etc.)
- [ ] Automatiser backup base Infisical (PostgreSQL)
- [ ] Monitoring expiration secrets

---

**Auteur**: Claude Code
**Version**: 1.0
**Dernière mise à jour**: 2025-11-16
