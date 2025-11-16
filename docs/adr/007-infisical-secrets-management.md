# ADR 007: Infisical pour la Gestion des Secrets

**Date**: 2025-11-16
**Statut**: ✅ Accepté
**Auteur**: Claude Code

---

## Contexte

Le cluster Kubernetes vixens nécessite une solution de gestion des secrets pour:
- Credentials Gandi API (cert-manager DNS-01)
- Credentials Synology CSI (stockage iSCSI)
- Secrets futurs (bases de données, API keys, etc.)

### Problème actuel

Les secrets sont stockés en **clair** dans `.secrets/<env>/` et versionnés dans Git, ce qui pose des problèmes:
- ❌ Risque de sécurité (secrets exposés en clair)
- ❌ Pas de rotation automatique
- ❌ Pas d'audit trail
- ❌ Application manuelle après chaque rebuild

### Solutions évaluées

| Solution | Avantages | Inconvénients |
|----------|-----------|---------------|
| **Sealed Secrets** | Simple, natif K8s | Secrets chiffrés dans Git, rotation complexe |
| **SOPS** | Chiffrement fichiers | CLI requis, pas de rotation |
| **ESO + Infisical** | Multi-backend, flexible | Complexité additionnelle |
| **Infisical Operator** | Simple, UI intuitive, rotation native | Dépendance à Infisical |

---

## Décision

Utiliser **Infisical Kubernetes Operator** pour la gestion des secrets.

### Justification

**Architecture homelab adaptée:**
- ✅ Infisical déjà hébergé sur le NAS (192.168.111.69:8085)
- ✅ UI intuitive pour la gestion manuelle
- ✅ Pas de migration Vault prévue
- ✅ Rotation manuelle suffisante (suspicion de compromission uniquement)

**Simplicité opérationnelle:**
- ✅ Déploiement via ArgoCD (chart Helm)
- ✅ CRD `InfisicalSecret` déclaratif (GitOps)
- ✅ Auto-sync Infisical → Kubernetes
- ✅ Pas de CLI supplémentaire (kubeseal, sops, etc.)

**Sécurité:**
- ✅ Secrets jamais versionnés en clair dans Git
- ✅ Universal Auth (clientId/clientSecret par environnement)
- ✅ Audit trail dans Infisical UI
- ✅ Rotation via UI avec propagation automatique

---

## Architecture

### Structure Infisical

**Un seul projet "vixens" avec 4 environnements:**

```
Projet: vixens
├── Environment: dev
│   ├── gandi-api-token          # cert-manager DNS-01
│   ├── synology-csi-host        # Synology CSI
│   ├── synology-csi-username    # Synology CSI
│   └── synology-csi-password    # Synology CSI
├── Environment: test
│   └── (mêmes secrets, valeurs différentes)
├── Environment: staging
│   └── (mêmes secrets, valeurs différentes)
└── Environment: prod
    └── (mêmes secrets, valeurs différentes)
```

**Pourquoi un seul projet?**
- Alignement avec la structure Git (`argocd/overlays/{dev,test,staging,prod}`)
- Promotion facile des secrets entre environnements
- Gestion centralisée
- Isolation via Universal Auth credentials par environnement

### Flux GitOps

```
┌─────────────────┐
│  Infisical UI   │  1. Ajout/modification secret
│  (192.168...69) │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ Infisical API   │  2. Stockage chiffré
└────────┬────────┘
         │
         v
┌─────────────────┐
│ Infisical       │  3. Polling périodique (resync)
│ Operator (K8s)  │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ InfisicalSecret │  4. CRD déclaratif (Git)
│ (CRD)           │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ Kubernetes      │  5. Secret natif créé/mis à jour
│ Secret          │
└─────────────────┘
```

### Authentification

**Universal Auth** (Machine Identity):
- Chaque environnement a ses propres credentials
- `clientId` et `clientSecret` stockés dans K8s Secret
- Credential rotation via Infisical UI si nécessaire

---

## Implémentation

### Étape 1: Configuration Infisical (Prérequis)

**Dans Infisical UI (http://192.168.111.69:8085):**

1. Créer le projet "vixens"
2. Créer les 4 environnements: dev, test, staging, prod
3. Pour chaque environnement:
   - Aller dans Settings → Machine Identities → Create
   - Nom: `vixens-<env>-k8s-operator`
   - Générer Universal Auth credentials
   - Copier `clientId` et `clientSecret`

### Étape 2: Déploiement Infisical Operator

**Via ArgoCD:**

```yaml
# argocd/overlays/dev/apps/infisical-operator.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: infisical-operator
  namespace: argocd
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
  destination:
    server: https://kubernetes.default.svc
    namespace: infisical-operator-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Étape 3: Configuration Universal Auth

**Pour chaque environnement**, créer le Secret avec les credentials Infisical:

```bash
# Exemple pour dev
kubectl create secret generic infisical-universal-auth \
  --namespace=infisical-operator-system \
  --from-literal=clientId="<CLIENT_ID_DEV>" \
  --from-literal=clientSecret="<CLIENT_SECRET_DEV>"
```

⚠️ **Note**: Ce secret est créé manuellement (bootstrap), puis géré par Infisical lui-même (rotation si nécessaire).

### Étape 4: Création des InfisicalSecret CRDs

**Exemple: Gandi API token (cert-manager)**

```yaml
# apps/cert-manager-config/base/infisical-secret-gandi.yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: gandi-credentials
  namespace: cert-manager
spec:
  hostAPI: http://192.168.111.69:8085
  resyncInterval: 60  # Polling toutes les 60 secondes
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens
        envSlug: dev  # dev, test, staging, prod
        secretsPath: "/"
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: infisical-operator-system
  managedSecretReference:
    secretName: gandi-credentials
    secretNamespace: cert-manager
    secretType: Opaque
```

**L'operator créera automatiquement:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gandi-credentials
  namespace: cert-manager
type: Opaque
data:
  api-token: <base64(valeur depuis Infisical)>
```

**Exemple: Synology CSI credentials**

```yaml
# apps/synology-csi/base/infisical-secret-client-info.yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: synology-csi-credentials
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
    secretName: synology-client-info
    secretNamespace: synology-csi
    secretType: Opaque
```

**Synology CSI attend un ConfigMap**, donc on crée un Job pour convertir:

```yaml
# apps/synology-csi/base/configmap-from-secret-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: create-synology-configmap
  namespace: synology-csi
spec:
  template:
    spec:
      serviceAccountName: synology-csi-config-creator
      containers:
      - name: create-configmap
        image: bitnami/kubectl:latest
        command:
        - /bin/sh
        - -c
        - |
          kubectl create configmap synology-client-info \
            --from-literal=client-info.yml="$(cat <<EOF
          ---
          clients:
            - host: \"$(kubectl get secret synology-csi-credentials -o jsonpath='{.data.synology-csi-host}' | base64 -d)\"
              port: 5000
              https: false
              username: \"$(kubectl get secret synology-csi-credentials -o jsonpath='{.data.synology-csi-username}' | base64 -d)\"
              password: \"$(kubectl get secret synology-csi-credentials -o jsonpath='{.data.synology-csi-password}' | base64 -d)\"
          EOF
          )" \
            --dry-run=client -o yaml | kubectl apply -f -
      restartPolicy: OnFailure
```

⚠️ **Alternative plus simple**: Modifier Synology CSI pour lire un Secret au lieu d'un ConfigMap (PR upstream ou fork).

---

## Rotation des Secrets

### Cas d'usage: Suspicion de compromission

**Étapes:**

1. **Générer nouveau secret** dans Infisical UI:
   - Aller dans le projet vixens → environnement concerné
   - Modifier la valeur du secret (ex: `gandi-api-token`)
   - Sauvegarder

2. **Auto-propagation** (sous 60 secondes):
   - L'operator détecte le changement (polling)
   - Le Secret Kubernetes est mis à jour automatiquement
   - Les pods utilisant le secret doivent être redémarrés (manuellement ou via reloader)

3. **Vérification**:
   ```bash
   kubectl get secret gandi-credentials -n cert-manager -o yaml
   # La valeur data.api-token doit être mise à jour
   ```

4. **Redémarrage des pods consommateurs**:
   ```bash
   # Exemple: cert-manager
   kubectl rollout restart deployment cert-manager -n cert-manager
   ```

### Universal Auth Rotation

**Si `clientSecret` compromis:**

1. Générer nouveau Machine Identity dans Infisical UI
2. Mettre à jour le Secret `infisical-universal-auth`:
   ```bash
   kubectl delete secret infisical-universal-auth -n infisical-operator-system
   kubectl create secret generic infisical-universal-auth \
     --namespace=infisical-operator-system \
     --from-literal=clientId="<NEW_CLIENT_ID>" \
     --from-literal=clientSecret="<NEW_CLIENT_SECRET>"
   ```
3. Redémarrer l'operator:
   ```bash
   kubectl rollout restart deployment infisical-operator-controller-manager \
     -n infisical-operator-system
   ```

---

## Migration depuis .secrets/

### Secrets à migrer

**Dev environment:**
- `.secrets/dev/gandi-credentials.yaml` → Infisical `gandi-api-token`
- `.secrets/dev/client-info.yml` → Infisical `synology-csi-{host,username,password}`

**Étapes de migration:**

1. **Importer dans Infisical UI:**
   - Projet vixens → Environment dev
   - Créer secret `gandi-api-token` = `42a2a1c56e29bbff391345eafe811e2e03ba0586`
   - Créer secret `synology-csi-host` = `192.168.111.69`
   - Créer secret `synology-csi-username` = `REDACTED`
   - Créer secret `synology-csi-password` = `REDACTED`

2. **Déployer InfisicalSecret CRDs** (via ArgoCD)

3. **Valider la synchronisation:**
   ```bash
   kubectl get secret gandi-credentials -n cert-manager
   kubectl get secret synology-csi-credentials -n synology-csi
   ```

4. **Supprimer les secrets manuels:**
   ```bash
   kubectl delete secret gandi-credentials -n cert-manager
   # Le secret sera recréé par l'operator
   ```

5. **Nettoyer `.secrets/`:**
   ```bash
   git rm -r .secrets/dev/
   git commit -m "chore: Remove plaintext secrets after Infisical migration"
   ```

---

## Avantages

### Sécurité
- ✅ Secrets jamais en clair dans Git
- ✅ Chiffrement at-rest dans Infisical
- ✅ Audit trail complet
- ✅ Isolation par environnement (Universal Auth)

### Opérationnel
- ✅ UI intuitive pour rotation manuelle
- ✅ Auto-sync Infisical → K8s (60s)
- ✅ Déclaratif via CRD (GitOps)
- ✅ Pas de CLI supplémentaire

### Reproductibilité
- ✅ Destroy/recreate cluster: secrets automatiquement resynchronisés
- ✅ InfisicalSecret CRDs versionnés dans Git
- ✅ Bootstrap simplifié (juste Universal Auth credentials)

---

## Inconvénients et Mitigations

### Dépendance à Infisical

**Problème**: Si Infisical (NAS) est indisponible, les secrets ne peuvent pas être resynchronisés.

**Mitigation**:
- Secrets Kubernetes existants restent fonctionnels (cache)
- Backup régulier de la base Infisical (PostgreSQL)
- NAS haute disponibilité (future)

### Complexité initiale

**Problème**: Configuration Universal Auth + CRDs requiert compréhension de l'architecture.

**Mitigation**:
- Documentation complète (cette ADR)
- Templates CRD réutilisables
- Procédure de migration pas-à-pas

### Pas de rotation automatique

**Problème**: Rotation manuelle via UI (pas de scheduling).

**Mitigation**:
- Acceptable pour homelab (rotation rare)
- Monitoring des secrets anciens (future)
- Rotation scriptée si nécessaire (API Infisical)

---

## Conséquences

### À court terme (Sprint 7)

- Déploiement Infisical Operator via ArgoCD
- Migration des 2 secrets existants (Gandi, Synology CSI)
- Suppression de `.secrets/` de Git
- Mise à jour procédure rebuild cluster

### À moyen terme (Sprint 8-11)

- Ajout de nouveaux secrets (Authelia, PostgreSQL, etc.)
- Réutilisation des templates CRD
- Test de rotation sur environnement test

### À long terme (Phase 3)

- Réplication sur test/staging/prod
- Backup automatique Infisical
- Monitoring expiration secrets

---

## Alternatives rejetées

### Sealed Secrets

**Rejeté car:**
- Secrets chiffrés toujours versionnés dans Git (risque si master key compromise)
- Rotation complexe (re-seal tous les secrets)
- Pas d'UI (CLI kubeseal uniquement)

### SOPS (Mozilla)

**Rejeté car:**
- Chiffrement fichier par fichier (pas de centralization)
- Nécessite age/gpg keys management
- Pas de rotation native
- CLI requis pour chaque modification

### External Secrets Operator + Infisical

**Rejeté car:**
- Complexité supplémentaire (generic operator)
- Pas d'avantage pour homelab (pas de multi-backend prévu)
- Infisical Operator plus simple et natif

---

## Références

- [Infisical Kubernetes Operator Docs](https://infisical.com/docs/integrations/platforms/kubernetes)
- [Universal Auth Guide](https://infisical.com/docs/documentation/platform/identities/universal-auth)
- [InfisicalSecret CRD Spec](https://github.com/Infisical/infisical/tree/main/k8-operator)
- [ADR 002: ArgoCD GitOps](./002-argocd-gitops.md)

---

**Statut**: ✅ Accepté
**Version**: 1.0
**Dernière mise à jour**: 2025-11-16
