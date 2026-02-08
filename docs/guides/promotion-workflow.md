# Production Promotion Workflow

**Guide pour promouvoir des changements vers la production en toute sécurité.**

---

## Vue d'ensemble

Le workflow de promotion suit le modèle trunk-based (ADR-017):
- **Dev**: ArgoCD suit `main` branch (HEAD)
- **Prod**: ArgoCD suit tag `prod-stable`

**Promotion = Déplacer le tag `prod-stable` vers un commit validé en dev**

---

## Méthode 1: Automatisée (Recommandée)

### Commande: `just SendToProd <version>`

**Workflow complet automatisé:**

```bash
# Exemple: promouvoir la version 1.2.3
just SendToProd v1.2.3
```

**Ce que ça fait:**

1. ✅ Vérifie que vous êtes sur `main`
2. ✅ Vérifie que le working tree est propre
3. ✅ Pull les derniers changements
4. ✅ Crée le tag `dev-v1.2.3` sur HEAD
5. ✅ Push le tag dev
6. ✅ Déclenche le workflow GitHub `promote-prod.yaml`
7. ✅ Attend que le workflow se termine (timeout: 10 min)
8. ✅ Vérifie que les tags prod sont créés

**Gestion d'erreur:**
- ❌ Si le tag existe déjà → demande confirmation pour recréer
- ❌ Si le workflow échoue → affiche les logs et rollback
- ❌ Si timeout (> 10 min) → arrête et affiche instructions manuelles

**Prérequis:**
- `gh` CLI installé et authentifié (`gh auth login`)
- Droits d'écriture sur le repo GitHub

---

## Méthode 2: Manuelle

### Étape 1: Créer le tag dev

```bash
VERSION="1.2.3"

# Créer tag dev sur commit actuel
git tag -a "dev-v${VERSION}" -m "Dev release v${VERSION}"
git push origin "dev-v${VERSION}"
```

### Étape 2: Déclencher le workflow GitHub

```bash
gh workflow run promote-prod.yaml -f version="v${VERSION}"
```

**Ou via l'interface GitHub:**
1. Aller sur `Actions` → `Promote to Production`
2. Cliquer `Run workflow`
3. Entrer la version (ex: `v1.2.3`)

### Étape 3: Vérifier le déploiement

```bash
# Attendre que le workflow se termine (~2-5 min)
gh run list --workflow=promote-prod.yaml --limit=1

# Vérifier les tags créés
git fetch --tags
git tag -l "prod-v*" | tail -5
git tag -l "prod-stable"

# Vérifier ArgoCD sur prod
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod
kubectl -n argocd get applications
```

---

## Workflow GitHub (promote-prod.yaml)

**Ce que fait le workflow:**

1. **Validation**
   - Vérifie que le tag `dev-v${VERSION}` existe
   - Affiche les tags dev disponibles si erreur

2. **Création tag prod**
   - Crée `prod-v${VERSION}` pointant vers le même commit que `dev-v${VERSION}`
   - Tag annoté avec metadata (version, promoted by, timestamp)

3. **Mise à jour prod-stable**
   - Supprime l'ancien tag `prod-stable`
   - Crée nouveau `prod-stable` pointant vers `prod-v${VERSION}`
   - Force push le tag

4. **Record de promotion**
   - Crée `.promotions/prod-v${VERSION}.json` avec metadata
   - Commit et push automatique

**Résultat:** ArgoCD prod détecte le changement de `prod-stable` et sync automatiquement.

---

## Sauvegarder la config fonctionnelle

Après validation que prod fonctionne:

```bash
# Créer tag de backup
git tag prod-working prod-stable
git push origin prod-working
```

**Usage:**
- `prod-stable`: Version actuellement déployée (bouge à chaque promotion)
- `prod-working`: Dernière version confirmée stable (backup manuel)

**Rollback rapide:**
```bash
# En cas de problème, revenir à la dernière version stable
git tag -f prod-stable prod-working
git push origin prod-stable --force
```

---

## Checklist de promotion

### Avant promotion

- [ ] ✅ Changements validés sur dev cluster
- [ ] ✅ Tâche Beads fermée (validation réussie)
- [ ] ✅ Tests fonctionnels passés
- [ ] ✅ Documentation à jour
- [ ] ✅ No breaking changes (ou plan de migration prêt)

### Pendant promotion

- [ ] ✅ Tag dev créé
- [ ] ✅ Workflow GitHub déclenché
- [ ] ✅ Workflow terminé avec succès
- [ ] ✅ Tags prod créés (`prod-v${VERSION}` + `prod-stable`)

### Après promotion

- [ ] ✅ ArgoCD prod synced (vérifier dans 2-5 min)
- [ ] ✅ Applications healthy sur prod
- [ ] ✅ Tests smoke prod passés
- [ ] ✅ Monitoring vérifié (Grafana)
- [ ] ✅ Tag `prod-working` mis à jour (backup)

---

## Troubleshooting

### Le workflow échoue: "Tag dev-vX.Y.Z does not exist"

**Cause:** Tag dev pas créé ou pas pushé

**Solution:**
```bash
VERSION="1.2.3"
git tag -a "dev-v${VERSION}" -m "Dev release v${VERSION}"
git push origin "dev-v${VERSION}"
```

### Le workflow timeout

**Cause:** Problème réseau ou GitHub Actions en panne

**Solution manuelle:**
```bash
VERSION="1.2.3"
COMMIT_SHA=$(git rev-parse "dev-v${VERSION}")

# Créer prod tag manuellement
git tag -a "prod-v${VERSION}" "$COMMIT_SHA" -m "Prod release v${VERSION}"
git push origin "prod-v${VERSION}"

# Mettre à jour prod-stable
git tag -f prod-stable "$COMMIT_SHA"
git push origin prod-stable --force
```

### ArgoCD prod ne sync pas

**Vérifier:**
```bash
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod

# Vérifier ArgoCD
kubectl -n argocd get application argocd -o yaml | grep -A 5 "targetRevision"

# Forcer sync
kubectl -n argocd patch application argocd \
  -p '{"operation": {"initiatedBy": {"username": "admin"}, "sync": {}}}' \
  --type merge
```

### Rollback urgent

```bash
# Option 1: Revenir à prod-working
git tag -f prod-stable prod-working
git push origin prod-stable --force

# Option 2: Revenir à version spécifique
git tag -f prod-stable prod-v1.2.2
git push origin prod-stable --force

# Vérifier sync ArgoCD (1-2 min)
kubectl -n argocd get applications
```

---

## Versioning

**Convention:** Semantic Versioning (SemVer)

```
vMAJOR.MINOR.PATCH

Exemples:
- v1.0.0   - Release initiale
- v1.1.0   - Nouvelle feature (non-breaking)
- v1.1.1   - Bugfix
- v2.0.0   - Breaking change
```

**Quand incrémenter:**
- **MAJOR**: Breaking change (modif incompatible)
- **MINOR**: Nouvelle feature (compatible)
- **PATCH**: Bugfix uniquement

---

## Monitoring post-promotion

### Vérifier ArgoCD

```bash
export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod

# Status applications
kubectl -n argocd get applications

# Détails app spécifique
kubectl -n argocd get application <app-name> -o yaml
```

### Vérifier pods

```bash
# Pods en erreur
kubectl get pods -A | grep -v Running | grep -v Completed

# Events récents
kubectl get events -A --sort-by='.lastTimestamp' | tail -20
```

### Grafana

- URL: https://grafana.truxonline.com
- Dashboards: `Kubernetes / Cluster` et `Kubernetes / Applications`

---

## Références

- **[ADR-017](../adr/017-pure-trunk-based-single-branch.md)**: Trunk-based workflow
- **[GitOps Workflow](gitops-workflow.md)**: Processus GitOps complet
- **[WORKFLOW.md](../../WORKFLOW.md)**: Workflow Beads + Just

---

**Last Updated:** 2026-02-08
