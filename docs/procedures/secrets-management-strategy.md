# Stratégie de Gestion des Secrets - Analyse et Recommandations

**Date** : 2025-11-15
**Statut** : Proposition
**Priorité** : 🔴 CRITIQUE

---

## Contexte et Problématique

### Situation Actuelle (NON ACCEPTABLE en Production)

**Secrets actuellement gérés** :
1. **Gandi API Token** (cert-manager DNS-01 challenge)
   - Localisation : `.secrets/dev/gandi-credentials.yaml`
   - Format : Kubernetes Secret (stocké en clair)
   - Commité dans Git : ✅ OUI (risque de sécurité)

2. **Synology CSI Credentials** (storage backend)
   - Localisation : `.secrets/dev/client-info.yml`
   - Format : ConfigMap avec credentials
   - Commité dans Git : ✅ OUI (risque de sécurité)

**Problèmes identifiés** :
- ❌ Secrets en clair dans Git (risque de fuite)
- ❌ Application manuelle après chaque rebuild
- ❌ Pas de rotation des secrets
- ❌ Pas d'audit trail
- ❌ Impossible de partager le repo publiquement
- ❌ Non conforme RGPD/sécurité entreprise

---

## Besoins et Contraintes

### Besoins Fonctionnels

1. **Automatisation GitOps** : Les secrets doivent être déployés automatiquement par ArgoCD
2. **Multi-environnement** : Support dev/test/staging/prod avec secrets différents
3. **Destroy/Recreate** : Les secrets doivent être restaurés automatiquement
4. **Rotation** : Possibilité de changer les secrets sans modifier le code
5. **Audit** : Traçabilité des accès et modifications

### Besoins Techniques

1. **Compatibilité ArgoCD** : Intégration native ou via plugin
2. **Backup/Restore** : Les secrets doivent survivre à un disaster recovery
3. **Performance** : Pas d'impact sur le temps de déploiement
4. **Simplicité** : Facile à comprendre et maintenir pour un homelab

### Contraintes Infrastructure

1. **Homelab** : Pas de cloud provider (AWS Secrets Manager, etc.)
2. **Budget** : Solution gratuite ou très faible coût
3. **Talos Linux** : Système immutable, limitations sur les agents/daemonsets
4. **3 environnements actifs** : dev, test, staging (+ prod futur)
5. **Storage existant** : Synology NAS disponible (192.168.111.69)

---

## Solutions Étudiées

### Option 1 : Sealed Secrets (Bitnami)

**Principe** : Chiffrement asymétrique avec clé privée dans le cluster.

#### Architecture
```
Developer → SealedSecret (chiffré) → Git → ArgoCD → Cluster
                                                      ↓
                                    Controller déchiffre → Secret K8s
```

#### Avantages
- ✅ Simple à comprendre et utiliser
- ✅ Intégration native avec ArgoCD
- ✅ Secrets chiffrés dans Git (sûr)
- ✅ Pas de dépendance externe
- ✅ Open-source et mature (Bitnami)
- ✅ Fonctionne parfaitement avec Talos

#### Inconvénients
- ⚠️ Clé privée stockée dans le cluster (si cluster perdu = secrets perdus)
- ⚠️ Rotation manuelle des secrets (re-seal requis)
- ⚠️ Pas d'audit trail avancé
- ⚠️ Backup de la clé privée critique

#### Implémentation

**Déploiement** :
```yaml
# argocd/overlays/dev/apps/sealed-secrets.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: sealed-secrets
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://bitnami-labs.github.io/sealed-secrets
    chart: sealed-secrets
    targetRevision: 2.15.0
  destination:
    server: https://kubernetes.default.svc
    namespace: kube-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**Utilisation** :
```bash
# 1. Créer un secret normalement
kubectl create secret generic gandi-credentials \
  --from-literal=api-token=YOUR_TOKEN \
  --dry-run=client -o yaml > gandi-secret.yaml

# 2. Le chiffrer avec kubeseal
kubeseal -f gandi-secret.yaml -w gandi-sealed-secret.yaml

# 3. Commiter le SealedSecret dans Git
git add apps/cert-manager/secrets/gandi-sealed-secret.yaml
git commit -m "feat(secrets): Add encrypted Gandi credentials"

# 4. ArgoCD déploie automatiquement
```

**Backup de la clé** :
```bash
# Sauvegarder la clé privée (À FAIRE IMMÉDIATEMENT)
kubectl get secret -n kube-system sealed-secrets-key -o yaml > sealed-secrets-key-backup.yaml

# Stocker hors du cluster (Synology NAS chiffré, clé USB, etc.)
```

---

### Option 2 : SOPS (Mozilla) + age Encryption

**Principe** : Chiffrement de fichiers YAML avec clés age (successeur de GPG).

#### Architecture
```
Developer → Fichier YAML chiffré → Git → ArgoCD + SOPS Plugin
                                                    ↓
                                    Déchiffrement → Secret K8s
```

#### Avantages
- ✅ Chiffrement granulaire (par fichier ou par clé)
- ✅ Clés age simples (vs GPG complexe)
- ✅ Support multi-cloud (AWS KMS, Azure, GCP, age)
- ✅ Intégration ArgoCD via plugin
- ✅ Open-source (Mozilla + communauté)
- ✅ Audit trail via Git (qui a modifié quoi)

#### Inconvénients
- ⚠️ Configuration ArgoCD plus complexe (plugin requis)
- ⚠️ Clé age à gérer et sauvegarder
- ⚠️ Courbe d'apprentissage plus élevée
- ⚠️ Déchiffrement côté ArgoCD (pas dans le cluster)

#### Implémentation

**Installation SOPS** :
```bash
# Installer age
brew install age  # ou apt install age

# Installer SOPS
brew install sops  # ou wget binary

# Générer une clé age
age-keygen -o age-key.txt
# Public key: REDACTED_AGE_KEY
```

**Configuration SOPS** :
```yaml
# .sops.yaml (à la racine du repo)
creation_rules:
  - path_regex: .*/dev/.*\.yaml$
    age: REDACTED_AGE_KEY
  - path_regex: .*/test/.*\.yaml$
    age: age1abc...  # Clé différente pour test
  - path_regex: .*/prod/.*\.yaml$
    age: age1xyz...  # Clé différente pour prod
```

**Chiffrement d'un secret** :
```bash
# 1. Créer le secret normalement
cat > apps/cert-manager/secrets/dev/gandi-credentials.yaml <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: gandi-credentials
  namespace: cert-manager
stringData:
  api-token: "YOUR_GANDI_TOKEN"
EOF

# 2. Chiffrer avec SOPS
sops -e -i apps/cert-manager/secrets/dev/gandi-credentials.yaml

# 3. Le fichier est maintenant chiffré, safe pour Git
git add apps/cert-manager/secrets/dev/gandi-credentials.yaml
```

**ArgoCD avec SOPS Plugin** :
```yaml
# Ajouter au ArgoCD Helm values
repoServer:
  volumes:
    - name: sops-age
      secret:
        secretName: sops-age
  volumeMounts:
    - name: sops-age
      mountPath: /sops-keys
  env:
    - name: SOPS_AGE_KEY_FILE
      value: /sops-keys/age-key.txt
```

---

### Option 3 : External Secrets Operator + Minio

**Principe** : Secrets stockés dans un backend externe (Minio S3-compatible sur Synology).

#### Architecture
```
Secrets → Minio (S3) sur Synology NAS
              ↓
   External Secrets Operator (polling)
              ↓
         Secret K8s
```

#### Avantages
- ✅ Backend centralisé et sauvegardé (Synology NAS)
- ✅ Rotation facile (update Minio = auto-sync dans cluster)
- ✅ Audit trail (versioning S3)
- ✅ Multi-cluster (plusieurs clusters → même backend)
- ✅ UI Minio pour gestion

#### Inconvénients
- ⚠️ Dépendance à Minio (SPOF)
- ⚠️ Complexité accrue (operator + backend)
- ⚠️ Secrets Minio credentials à bootstrapper (chicken & egg)
- ⚠️ Performance (polling interval)
- ⚠️ Pas vraiment GitOps (secrets pas dans Git)

#### Implémentation

**Déployer Minio sur Synology** :
```bash
# Via Docker sur Synology DSM
# Ou via containerd sur un node dédié
```

**Déployer External Secrets Operator** :
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: external-secrets
  namespace: argocd
spec:
  source:
    repoURL: https://charts.external-secrets.io
    chart: external-secrets
    targetRevision: 0.9.0
```

**Créer un ExternalSecret** :
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: gandi-credentials
  namespace: cert-manager
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: minio-backend
    kind: SecretStore
  target:
    name: gandi-credentials
  data:
    - secretKey: api-token
      remoteRef:
        key: gandi/api-token
```

---

### Option 4 : HashiCorp Vault (NON RECOMMANDÉ)

**Pourquoi NON** :
- ❌ Trop complexe pour un homelab
- ❌ Ressources importantes (HA Vault cluster)
- ❌ Maintenance lourde
- ❌ Overkill pour 2-3 secrets

---

## Comparaison des Solutions

| Critère | Sealed Secrets | SOPS + age | External Secrets + Minio |
|---------|----------------|------------|---------------------------|
| **Simplicité** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **GitOps natif** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Rotation secrets** | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Backup/Restore** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Multi-cluster** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Audit trail** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Performance** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Maintenance** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Coût** | Gratuit | Gratuit | Gratuit (Minio OSS) |

---

## Recommandation

### 🏆 Solution Recommandée : **Sealed Secrets** (Court terme) + **SOPS** (Long terme)

#### Phase 1 : Sealed Secrets (Sprint 7 - Immédiat)

**Pourquoi maintenant** :
- ✅ Résout le problème critique immédiatement
- ✅ Simple à déployer (1 heure max)
- ✅ Fonctionne out-of-the-box avec ArgoCD
- ✅ Permet de rendre le repo public si besoin
- ✅ Valide le workflow GitOps

**Actions** :
1. Déployer Sealed Secrets via ArgoCD
2. Créer les SealedSecrets pour Gandi + Synology
3. Sauvegarder la clé privée sealed-secrets-key
4. Mettre à jour la procédure de rebuild
5. Supprimer `.secrets/` de Git

**Durée estimée** : 1-2 heures

---

#### Phase 2 : Migration vers SOPS (Sprint 8-9 - Après validation)

**Pourquoi plus tard** :
- ⭐ Meilleure séparation des secrets par environnement
- ⭐ Audit trail via Git (qui a modifié quel secret)
- ⭐ Rotation plus simple (re-encrypt vs re-seal)
- ⭐ Support multi-backend (age local → KMS cloud si migration future)

**Migration progressive** :
1. Installer SOPS CLI et age
2. Configurer ArgoCD SOPS plugin
3. Migrer 1 secret en SOPS (test)
4. Valider pendant 1-2 semaines
5. Migrer tous les secrets
6. Désinstaller Sealed Secrets

**Durée estimée** : 3-4 heures

---

## Plan d'Implémentation - Phase 1 (Sealed Secrets)

### Sprint 7 : Déploiement Sealed Secrets

#### Tâche 1 : Déployer Sealed Secrets (30 min)

```bash
# 1. Créer l'Application ArgoCD
cat > argocd/overlays/dev/apps/sealed-secrets.yaml <<'EOF'
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: sealed-secrets
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "-1"  # Déployer AVANT les autres apps
spec:
  project: default
  source:
    repoURL: https://bitnami-labs.github.io/sealed-secrets
    chart: sealed-secrets
    targetRevision: 2.15.0
    helm:
      values: |
        fullnameOverride: sealed-secrets-controller

        # Control plane tolerations
        tolerations:
          - key: node-role.kubernetes.io/control-plane
            operator: Exists
            effect: NoSchedule
  destination:
    server: https://kubernetes.default.svc
    namespace: kube-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=false
EOF

# 2. Ajouter à kustomization
echo "  - apps/sealed-secrets.yaml" >> argocd/overlays/dev/kustomization.yaml

# 3. Commit et push
git add argocd/overlays/dev/apps/sealed-secrets.yaml argocd/overlays/dev/kustomization.yaml
git commit -m "feat(secrets): Deploy Sealed Secrets controller"
git push origin dev

# 4. Attendre le déploiement (2-3 min)
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=sealed-secrets -n kube-system --timeout=180s
```

---

#### Tâche 2 : Installer kubeseal CLI (10 min)

```bash
# Sur la machine de gestion (grenat)
KUBESEAL_VERSION=0.26.0
wget "https://github.com/bitnami-labs/sealed-secrets/releases/download/v${KUBESEAL_VERSION}/kubeseal-${KUBESEAL_VERSION}-linux-amd64.tar.gz"
tar -xvzf kubeseal-${KUBESEAL_VERSION}-linux-amd64.tar.gz kubeseal
sudo install -m 755 kubeseal /usr/local/bin/kubeseal
rm kubeseal kubeseal-${KUBESEAL_VERSION}-linux-amd64.tar.gz

# Vérifier
kubeseal --version
```

---

#### Tâche 3 : Créer les SealedSecrets (30 min)

**Secret 1 : Gandi Credentials**

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# 1. Créer le secret (NE PAS APPLY)
kubectl create secret generic gandi-credentials \
  --namespace=cert-manager \
  --from-literal=api-token="$(cat /root/vixens/.secrets/dev/gandi-api-token.txt)" \
  --dry-run=client -o yaml > /tmp/gandi-secret.yaml

# 2. Le chiffrer
kubeseal -f /tmp/gandi-secret.yaml \
  -w apps/cert-manager/secrets/gandi-sealed-secret.yaml \
  --controller-namespace=kube-system \
  --controller-name=sealed-secrets-controller

# 3. Nettoyer le fichier temporaire
rm /tmp/gandi-secret.yaml
```

**Secret 2 : Synology CSI**

```bash
# 1. Créer le ConfigMap secret (chiffré)
kubectl create secret generic synology-client-info \
  --namespace=synology-csi \
  --from-file=client-info.yml=/root/vixens/.secrets/dev/client-info.yml \
  --dry-run=client -o yaml > /tmp/synology-secret.yaml

# 2. Le chiffrer
kubeseal -f /tmp/synology-secret.yaml \
  -w apps/synology-csi/secrets/client-info-sealed-secret.yaml \
  --controller-namespace=kube-system \
  --controller-name=sealed-secrets-controller

# 3. Nettoyer
rm /tmp/synology-secret.yaml
```

---

#### Tâche 4 : Intégrer les SealedSecrets dans ArgoCD (20 min)

**Créer l'application secrets** :

```yaml
# apps/cert-manager/secrets/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - gandi-sealed-secret.yaml
```

```yaml
# argocd/overlays/dev/apps/cert-manager-secrets.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cert-manager-secrets
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "-1"  # AVANT cert-manager
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/cert-manager/secrets
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

---

#### Tâche 5 : Backup de la Clé Privée (CRITIQUE - 10 min)

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# 1. Exporter la clé privée
kubectl get secret -n kube-system \
  -l sealedsecrets.bitnami.com/sealed-secrets-key=active \
  -o yaml > sealed-secrets-master-key-backup.yaml

# 2. Sauvegarder sur le NAS Synology (chiffré)
# Option A : Copie manuelle sécurisée
scp sealed-secrets-master-key-backup.yaml user@192.168.111.69:/volume1/backups/kubernetes/

# Option B : Via age encryption
age -r REDACTED_AGE_KEY \
  -o sealed-secrets-key.age \
  sealed-secrets-master-key-backup.yaml

# 3. Stocker age dans un endroit SÛR (KeePass, clé USB, etc.)

# 4. Nettoyer le fichier local
shred -u sealed-secrets-master-key-backup.yaml
```

**⚠️ IMPORTANT** : Cette clé est CRITIQUE. Sans elle, impossible de déchiffrer les secrets après un disaster recovery.

---

#### Tâche 6 : Supprimer `.secrets/` de Git (10 min)

```bash
# 1. Supprimer le répertoire (les secrets sont maintenant sealed)
git rm -r .secrets/

# 2. Ajouter au .gitignore
echo "" >> .gitignore
echo "# Secrets (use Sealed Secrets instead)" >> .gitignore
echo ".secrets/" >> .gitignore

# 3. Commit
git add .gitignore
git commit -m "feat(secrets): Remove plaintext secrets, use Sealed Secrets"

# 4. Push
git push origin dev
```

---

#### Tâche 7 : Mettre à Jour la Procédure de Rebuild (20 min)

Modifier `docs/procedures/cluster-rebuild-procedure.md` :

**Étape 5 devient** :
```markdown
### Étape 5 : Les Secrets Sont Déployés Automatiquement ✅

**ArgoCD déploie automatiquement** :
- `cert-manager-secrets` : Gandi credentials (SealedSecret)
- `synology-csi-secrets` : Client info (SealedSecret)

**Pas d'action manuelle requise !**

Vérification :
\`\`\`bash
# Les secrets doivent apparaître
kubectl get secrets -n cert-manager
kubectl get secrets -n synology-csi
\`\`\`
```

---

### Validation de la Solution

**Critères de succès** :
- [ ] Sealed Secrets controller déployé et Running
- [ ] SealedSecrets créés et committés dans Git
- [ ] Secrets Kubernetes automatiquement générés
- [ ] cert-manager-webhook-gandi démarre avec le secret
- [ ] synology-csi-controller démarre avec le secret
- [ ] Clé privée sauvegardée hors cluster
- [ ] `.secrets/` supprimé de Git
- [ ] Rebuild complet fonctionne sans intervention manuelle

---

## FAQ

### Q1 : Que se passe-t-il si je perds la clé Sealed Secrets ?

**R** : Les SealedSecrets dans Git ne pourront plus être déchiffrés. Vous devrez :
1. Restaurer la clé depuis le backup
2. Ou re-créer tous les SealedSecrets avec une nouvelle clé

**Mitigation** : TOUJOURS sauvegarder la clé dans 2+ endroits sûrs.

---

### Q2 : Comment faire un rotate d'un secret ?

**R** :
```bash
# 1. Créer le nouveau secret
kubectl create secret generic gandi-credentials \
  --namespace=cert-manager \
  --from-literal=api-token="NEW_TOKEN" \
  --dry-run=client -o yaml > /tmp/new-secret.yaml

# 2. Le chiffrer
kubeseal -f /tmp/new-secret.yaml \
  -w apps/cert-manager/secrets/gandi-sealed-secret.yaml \
  --controller-namespace=kube-system

# 3. Commit et push → ArgoCD sync automatique
git add apps/cert-manager/secrets/gandi-sealed-secret.yaml
git commit -m "feat(secrets): Rotate Gandi API token"
git push

# 4. Le secret K8s est mis à jour automatiquement
```

---

### Q3 : Comment gérer les secrets multi-environnements ?

**R** : Créer un SealedSecret par environnement :

```
apps/
├── cert-manager/
│   └── secrets/
│       ├── dev/
│       │   └── gandi-sealed-secret.yaml
│       ├── test/
│       │   └── gandi-sealed-secret.yaml
│       ├── staging/
│       │   └── gandi-sealed-secret.yaml
│       └── prod/
│           └── gandi-sealed-secret.yaml
```

ArgoCD déploie le bon environnement selon `targetRevision`.

---

### Q4 : Sealed Secrets ou SOPS, lequel choisir définitivement ?

**R** : **Les deux ont leur place** :

- **Sealed Secrets** : Parfait pour commencer, simple, GitOps natif
- **SOPS** : Mieux pour audit, rotation, multi-backend

**Recommandation finale** : Commencer avec Sealed Secrets (Sprint 7), migrer vers SOPS si besoin (Sprint 8-9) après validation.

---

## Prochaines Étapes

### Immédiat (Sprint 7)
1. Valider cette stratégie avec l'équipe
2. Déployer Sealed Secrets (suivre Tâche 1-7 ci-dessus)
3. Tester un rebuild complet pour valider l'automatisation
4. Documenter pour les autres environnements (test, staging, prod)

### Moyen Terme (Sprint 8-9)
1. Évaluer la migration vers SOPS
2. Tester SOPS sur 1 secret en dev
3. Décider : rester avec Sealed Secrets ou migrer

### Long Terme (Sprint 10+)
1. Ajouter rotation automatique des secrets (CronJob)
2. Audit trail des accès aux secrets
3. Intégration avec External Secrets si besoin backend centralisé

---

**Auteur** : Claude Code
**Version** : 1.0
**Dernière mise à jour** : 2025-11-15
