# Plan d'Implémentation Sync Waves ArgoCD

**Objectif:** Passer de 2h30 à 30-45min de déploiement avec 0 CrashLoopBackOff

**Date création:** 2024-12-25
**Statut:** Ready to implement
**Priorité:** HIGH

---

## 📊 État Actuel

### Applications avec Sync Waves (14/61)

```
Wave -3: infisical-operator ✅
Wave -2: cilium-lb ✅
Wave -1: synology-csi-secrets ✅
Wave  0: cert-manager, argocd-image-updater, cert-manager-secrets ✅
Wave  1: cert-manager-webhook-gandi ✅
Wave  2: cert-manager-config, cloudnative-pg-crds ✅
Wave  3: cloudnative-pg ✅
Wave  4: postgresql-shared ✅
Wave  5: prometheus, alertmanager ✅
Wave  6: grafana, prometheus-ingress ✅
Wave  7: grafana-ingress, docspell-native ✅
```

**Bonne nouvelle:** L'infrastructure critique est déjà correctement configurée! 🎉

### Applications SANS Sync Waves (47/61)

La majorité des applications métier n'ont pas de wave configurée.

---

## 🎯 Stratégie d'Implémentation

### Principe

**Ne PAS modifier ce qui fonctionne déjà!**
- Infrastructure (waves -3 à 3): ✅ OK
- Monitoring (waves 5-7): ✅ OK

**Ajouter des waves uniquement aux apps qui en ont besoin:**
1. Apps avec dépendances (PostgreSQL, Redis, secrets)
2. Services partagés (nfs-storage, redis-shared)
3. Apps métier (wave par défaut)

### Nouvelle Stratégie Simplifiée

```
Wave -5: [Vide - réservé pour futurs CRDs]
Wave -4: [Vide - operators déjà en -3 et 0]
Wave -3: infisical-operator ✅ (déjà configuré)
Wave -2: cilium-lb ✅ (déjà configuré)
Wave -1: synology-csi-secrets, nfs-storage, redis-shared
Wave  0: cert-manager, synology-csi, traefik [infrastructure de base]
Wave  1: cert-manager-webhook-gandi ✅ (déjà configuré)
Wave  2: cert-manager-config, cloudnative-pg-crds ✅ (déjà configuré)
Wave  3: cloudnative-pg, argocd ✅ (déjà configuré)
Wave  4: postgresql-shared ✅ (déjà configuré)
Wave  5: Apps avec dépendances PostgreSQL (linkwarden, netbox, docspell)
Wave 10: Applications métier (par défaut - tout le reste)
```

---

## 📋 Plan d'Action

### Phase 1: Ajout Wave -1 (Services Partagés Storage/Cache)

**Objectif:** S'assurer que NFS et Redis sont prêts avant les apps

**Applications à modifier:**

1. **nfs-storage** → Wave -1
   - Fichier: `argocd/overlays/dev/apps/nfs-storage.yaml`
   - Raison: Stockage utilisé par certaines apps

2. **redis-shared** → Wave -1
   - Fichier: À CRÉER `argocd/overlays/dev/apps/redis-shared.yaml`
   - Raison: Cache utilisé par plusieurs apps

### Phase 2: Ajout Wave 0 (Infrastructure Réseau)

**Applications à modifier:**

3. **synology-csi** → Wave 0
   - Fichier: À CRÉER `argocd/overlays/dev/apps/synology-csi.yaml`
   - Raison: Storage provider critique

4. **traefik** → Wave 0
   - Fichier: `argocd/overlays/dev/apps/traefik.yaml`
   - Raison: Ingress controller (déjà existe, ajouter annotation)

5. **traefik-dashboard** → Wave 1
   - Fichier: `argocd/overlays/dev/apps/traefik-dashboard.yaml`
   - Raison: Dépend de Traefik

### Phase 3: Ajout Wave 5 (Apps avec PostgreSQL)

**Applications à modifier:**

6. **linkwarden** → Wave 5
   - Fichier: `argocd/overlays/dev/apps/linkwarden.yaml`
   - Raison: Dépend de postgresql-shared (wave 4)

7. **netbox** → Wave 5
   - Fichier: À CRÉER `argocd/overlays/dev/apps/netbox.yaml`
   - Raison: Dépend de postgresql-shared (wave 4)

8. **docspell** → Wave 5 (si existe séparément de docspell-native)
   - Fichier: `argocd/overlays/dev/apps/docspell.yaml`
   - Raison: Dépend de postgresql-shared

### Phase 4: Vérification Apps Métier (Wave 10 par défaut)

**Applications qui resteront à wave 10 (défaut):**

Toutes les apps sans dépendances critiques:
- homeassistant, mosquitto, birdnet-go
- media apps: jellyfin, sonarr, radarr, prowlarr, etc.
- tools: homepage, headlamp, changedetection
- auth: authentik, vaultwarden

**Raison:** Pas de dépendances inter-apps, peuvent démarrer en parallèle.

---

## 🛠️ Implémentation Technique

### Template Application avec Sync Wave

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: <app-name>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
  annotations:
    argocd.argoproj.io/sync-wave: "X"  # ← AJOUTER CETTE LIGNE
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/<category>/<app-name>/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: <namespace>
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Script d'Implémentation (Batch)

```bash
#!/bin/bash
# sync-waves-batch-update.sh

set -e

REPO_ROOT="/root/vixens"
cd "$REPO_ROOT"

# Phase 1: Wave -1 (Services Partagés)
echo "=== Phase 1: Wave -1 (Services Partagés) ==="

# nfs-storage (fichier existe déjà)
yq -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "-1"' \
  argocd/overlays/dev/apps/nfs-storage.yaml

# redis-shared (créer si n'existe pas)
if [ ! -f argocd/overlays/dev/apps/redis-shared.yaml ]; then
  cat > argocd/overlays/dev/apps/redis-shared.yaml <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: redis-shared
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
  annotations:
    argocd.argoproj.io/sync-wave: "-1"
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/04-databases/redis-shared/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: databases
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF
fi

# Phase 2: Wave 0 (Infrastructure Réseau)
echo "=== Phase 2: Wave 0 (Infrastructure Réseau) ==="

# synology-csi (créer si n'existe pas)
if [ ! -f argocd/overlays/dev/apps/synology-csi.yaml ]; then
  cat > argocd/overlays/dev/apps/synology-csi.yaml <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: synology-csi
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
  annotations:
    argocd.argoproj.io/sync-wave: "0"
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/01-storage/synology-csi/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: synology-csi
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF
fi

# traefik (fichier existe déjà)
yq -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "0"' \
  argocd/overlays/dev/apps/traefik.yaml

# traefik-dashboard (fichier existe déjà)
yq -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "1"' \
  argocd/overlays/dev/apps/traefik-dashboard.yaml

# Phase 3: Wave 5 (Apps PostgreSQL)
echo "=== Phase 3: Wave 5 (Apps avec PostgreSQL) ==="

# linkwarden (fichier existe déjà)
yq -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "5"' \
  argocd/overlays/dev/apps/linkwarden.yaml

# netbox (créer si n'existe pas)
if [ ! -f argocd/overlays/dev/apps/netbox.yaml ]; then
  cat > argocd/overlays/dev/apps/netbox.yaml <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: netbox
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
  annotations:
    argocd.argoproj.io/sync-wave: "5"
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/70-tools/netbox/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: tools
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF
fi

# docspell (si existe)
if [ -f argocd/overlays/dev/apps/docspell.yaml ]; then
  yq -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "5"' \
    argocd/overlays/dev/apps/docspell.yaml
fi

echo "✅ Sync waves ajoutées avec succès!"
echo ""
echo "Fichiers modifiés:"
git status --short argocd/overlays/dev/apps/
```

---

## 🧪 Tests et Validation

### Test 1: Vérification Syntax (Local)

```bash
# Vérifier que tous les fichiers sont du YAML valide
find argocd/overlays/dev/apps -name "*.yaml" -exec yamllint {} \;

# Vérifier avec yq
find argocd/overlays/dev/apps -name "*.yaml" -exec yq eval '.' {} \; > /dev/null

# Compter les apps par wave
grep -r "sync-wave" argocd/overlays/dev/apps/ | cut -d'"' -f2 | sort -n | uniq -c
```

### Test 2: Dry-Run ArgoCD

```bash
# Commit et push vers une branche test
git checkout -b feature/sync-waves
git add argocd/overlays/dev/apps/
git commit -m "feat(argocd): add sync waves to apps"
git push origin feature/sync-waves

# Dans ArgoCD, vérifier l'ordre prévu (sans sync)
argocd app get <app-name> --show-params
```

### Test 3: Déploiement Contrôlé (Dev)

```bash
# 1. Sauvegarder l'état actuel
kubectl get applications -n argocd -o yaml > /tmp/apps-backup.yaml

# 2. Merger la branche
git checkout dev
git merge feature/sync-waves
git push origin dev

# 3. Observer le comportement (ArgoCD auto-sync)
watch -n 5 'kubectl get applications -n argocd -o json | \
  jq -r ".items[] | \"\(.metadata.annotations.\"argocd.argoproj.io/sync-wave\" // \"none\") \(.metadata.name) \(.status.sync.status)/\(.status.health.status)\"" | \
  sort -n'

# 4. Surveiller les pods
watch -n 5 'kubectl get pods -A | grep -v Running | grep -v Completed'
```

### Test 4: Déploiement Complet (Destroy/Recreate)

**⚠️ UNIQUEMENT EN DEV!**

```bash
# 1. Destroy cluster
cd terraform/environments/dev
terraform destroy -auto-approve

# 2. Recreate cluster avec sync waves
terraform apply -auto-approve

# 3. Mesurer le temps de déploiement
START_TIME=$(date +%s)

# Attendre que tout soit Healthy
while true; do
  TOTAL=$(kubectl get applications -n argocd -o json | jq '.items | length')
  HEALTHY=$(kubectl get applications -n argocd -o json | \
    jq '[.items[] | select(.status.health.status == "Healthy")] | length')

  echo "$(date): $HEALTHY/$TOTAL apps Healthy"

  if [ "$HEALTHY" -eq "$TOTAL" ]; then
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    echo "✅ Déploiement complet en $DURATION secondes ($(($DURATION / 60)) minutes)"
    break
  fi

  sleep 30
done

# 4. Vérifier qu'il n'y a eu aucun CrashLoopBackOff
kubectl get events -A --sort-by='.lastTimestamp' | grep -i crash
```

---

## 📈 Métriques de Succès

### Avant (État Actuel)

- ⏱️ **Temps:** 1h30 - 2h30
- ❌ **CrashLoopBackOff:** 2-3 apps (linkwarden, netbox)
- 📊 **Ordre:** Chaotique (20-30 apps en parallèle)
- 🔄 **Redémarrages:** 5-10 pods inutiles

### Après (Objectif)

- ⏱️ **Temps:** 30-45 minutes
- ✅ **CrashLoopBackOff:** 0
- 📊 **Ordre:** Séquentiel par wave
- 🔄 **Redémarrages:** 0 (sauf échecs légitimes)

### KPIs à Mesurer

```bash
# 1. Temps total de déploiement
time ./deploy-cluster.sh

# 2. Nombre de restarts par app
kubectl get pods -A -o json | \
  jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name) \(.status.containerStatuses[0].restartCount)"' | \
  awk '$2 > 0' | wc -l

# 3. Apps qui ont eu des erreurs
kubectl get events -A | grep -i "error\|failed" | wc -l

# 4. Ordre de déploiement (vérifier waves respectées)
kubectl get events -n argocd --sort-by='.lastTimestamp' | grep "SyncSucceeded"
```

---

## 📅 Planning d'Exécution

### Sprint Actuel (Semaine 1)

**Jour 1-2: Préparation**
- [ ] Lire et comprendre ce plan
- [ ] Vérifier que `yq` est installé: `yq --version`
- [ ] Créer branche: `git checkout -b feature/sync-waves`

**Jour 3: Implémentation Phase 1-3**
- [ ] Exécuter script `sync-waves-batch-update.sh`
- [ ] Vérifier syntax YAML
- [ ] Commit + push vers branche feature

**Jour 4: Tests Local**
- [ ] Test 1: Vérification syntax ✅
- [ ] Test 2: Dry-run ArgoCD ✅
- [ ] Corriger erreurs éventuelles

**Jour 5: Déploiement Dev**
- [ ] Merger vers dev
- [ ] Test 3: Observer comportement ✅
- [ ] Mesurer temps de déploiement

### Semaine 2: Validation

**Jour 6-7: Test Complet**
- [ ] Backup cluster dev
- [ ] Test 4: Destroy/Recreate ✅
- [ ] Mesurer métriques de succès
- [ ] Documenter résultats

**Jour 8-9: Propagation**
- [ ] Appliquer aux overlays test/staging/prod
- [ ] Créer PR dev → test
- [ ] Documentation finale

**Jour 10: Finalisation**
- [ ] Mise à jour ADR (Architecture Decision Record)
- [ ] Mise à jour ARGOCD-SYNC-WAVES.md
- [ ] Close task dans Archon

---

## 🔄 Rollback Plan

Si quelque chose ne fonctionne pas:

### Rollback Rapide (sans redéploiement)

```bash
# 1. Revenir au commit précédent
git revert HEAD
git push origin dev

# 2. ArgoCD auto-sync retirera les annotations
# Les apps redémarreront dans l'ordre chaotique (comme avant)

# 3. Observer que tout revient à la normale
watch kubectl get applications -n argocd
```

### Rollback Complet (avec redéploiement)

```bash
# 1. Restaurer backup
kubectl apply -f /tmp/apps-backup.yaml

# 2. Ou revert Git + recreate cluster
git checkout dev
git revert <commit-hash>
git push origin dev
terraform destroy -auto-approve
terraform apply -auto-approve
```

---

## 🎯 Prochaines Étapes Après Succès

1. **Documentation:**
   - Créer ADR: `docs/adr/00X-argocd-sync-waves.md`
   - Mettre à jour `ARGOCD-SYNC-WAVES.md`

2. **Automation:**
   - Script de validation pre-commit
   - CI check pour vérifier les waves

3. **Monitoring:**
   - Dashboard Grafana: temps de déploiement par wave
   - Alertes si apps restent en Progressing > 10min

4. **Optimisation Future:**
   - Health checks améliorés (startup probes)
   - Resource requests/limits optimisés
   - Image pre-pulling

---

## 📚 Références

- ArgoCD Sync Waves: https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/
- ArgoCD Sync Options: https://argo-cd.readthedocs.io/en/stable/user-guide/sync-options/
- `docs/ARGOCD-SYNC-WAVES.md` - Stratégie globale
- `docs/troubleshooting/2024-12-25-cluster-redeploy-analysis.md` - Analyse problèmes actuels

---

**Auteur:** Claude Sonnet 4.5
**Date:** 2024-12-25
**Version:** 1.0
