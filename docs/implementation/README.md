# Quick Start: Sync Waves Implementation

Ce guide rapide vous permet de démarrer l'implémentation des sync waves en 5 minutes.

## 📋 Pré-requis

```bash
# Installer yq si nécessaire
snap install yq
# ou
brew install yq

# Vérifier
yq --version
```

## 🚀 Implémentation Rapide (30 min)

### 1. Dry-Run (5 min)

```bash
# Voir ce qui sera modifié SANS appliquer
./scripts/sync-waves-batch-update.sh --dry-run
```

### 2. Appliquer (5 min)

```bash
# Appliquer les modifications
./scripts/sync-waves-batch-update.sh
```

### 3. Valider (5 min)

```bash
# Vérifier syntax et dépendances
./scripts/validate-sync-waves.sh
```

### 4. Commit (5 min)

```bash
# Vérifier les changements
git diff argocd/overlays/dev/apps/

# Commit
git add argocd/overlays/dev/apps/ scripts/ docs/implementation/
git commit -m "feat(argocd): implement sync waves for optimized deployment

- Add sync waves to 50+ applications
- Infrastructure: waves -1 to 3
- Apps with dependencies: wave 5
- Standard apps: wave 10

Expected improvement: 2h30 → 30-45min deployment time

Ref: docs/implementation/sync-waves-implementation-plan.md"

# Push
git push origin dev
```

### 5. Observer (10 min)

```bash
# Surveiller le déploiement
watch -n 5 'kubectl get applications -n argocd -o json | \
  jq -r ".items[] | \"\(.metadata.annotations.\"argocd.argoproj.io/sync-wave\" // \"0\") \(.metadata.name) \(.status.health.status)\"" | \
  sort -n'
```

## 🧪 Test Complet (Optionnel - 2h)

⚠️ **UNIQUEMENT EN DEV!**

```bash
# Destroy/Recreate cluster pour tester
cd terraform/environments/dev
terraform destroy -auto-approve
terraform apply -auto-approve

# Mesurer le temps
cd ../../..
./scripts/test-deployment-time.sh
```

## 📚 Documentation Complète

- **Plan détaillé:** [sync-waves-implementation-plan.md](sync-waves-implementation-plan.md)
- **Stratégie:** [../ARGOCD-SYNC-WAVES.md](../ARGOCD-SYNC-WAVES.md)

## 🆘 En Cas de Problème

### Rollback

```bash
# Annuler les changements
git revert HEAD
git push origin dev
```

### Debug

```bash
# Voir les apps par wave
grep -r "sync-wave" argocd/overlays/dev/apps/ | cut -d'"' -f2 | sort -n | uniq -c

# Vérifier une app spécifique
kubectl get application -n argocd <app-name> -o yaml | grep sync-wave
```

## 📞 Support

Pour toute question, voir:
- [WORKFLOW.md](../../WORKFLOW.md) - Processus de travail
- [CLAUDE.md](../../CLAUDE.md) - Instructions générales
