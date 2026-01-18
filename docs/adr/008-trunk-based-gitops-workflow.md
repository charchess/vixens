# ADR-008: Migration vers Trunk-Based GitOps Workflow

**Date:** 2025-12-28
**Status:** Superseded by [ADR-017](017-pure-trunk-based-single-branch.md)
**Deciders:** Architecture, DevOps
**Tags:** gitops, workflow, migration

---

## Context

### Problème Actuel

Workflow multi-branches (dev/test/staging/main):
- ❌ 4 branches longue durée qui divergent constamment
- ❌ Merge conflicts fréquents
- ❌ Complexité avec Renovate (baseBranches configuration)
- ❌ Historique Git fragmenté
- ❌ Promotion manuelle complexe (promote.sh)
- ❌ Pas aligné avec les best practices 2025
- ❌ Branches test/staging inutiles pour les apps (seulement utiles pour tests Terraform)

### Best Practices Industrie 2025

**Trunk-Based Development:**
- ✅ Une seule branche principale (main)
- ✅ Feature branches courtes (<2 jours)
- ✅ Promotion via Git tags
- ✅ ArgoCD targetRevision pointant vers des tags

**Sources:**
- [GitOps Best Practices (Akuity)](https://akuity.io/blog/gitops-best-practices-whitepaper)
- [ArgoCD Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
- [Environment Promotion with Tags](https://mattias.engineer/blog/2023/gitops-environment-promotion/)

## Decision

**Migrer vers un workflow trunk-based simplifié avec 2 branches (dev/main) et promotion par tags Git.**

### Nouveau Modèle

```
Feature Branch → dev (auto-deploy cluster dev) → main (auto-deploy cluster prod)
                  ↓                                ↓
              dev-latest                      prod-stable
              dev-v1.2.3                      prod-v1.2.3
```

### Structure des Tags

**Dev (auto-tagging):**
- **dev-latest** - Auto-updated after merge to dev
- **dev-v1.2.3** - Specific dev version (auto-created)

**Prod (manual promotion):**
- **prod-stable** - Current prod version (manually promoted)
- **prod-v1.2.3** - Specific prod version (manually promoted)

**Supprimé:** test-*, staging-* (branches test/staging archivées)

### Workflow

1. **Feature Development:**
   ```bash
   git checkout -b feature/xyz
   # develop
   git push
   gh pr create -B dev  # PR vers dev (pas main directement)
   ```

2. **Auto-deploy Dev (après merge):**
   - GitHub Action `auto-tag-dev.yaml` crée automatiquement:
     - `dev-vX.Y.Z` (version incrémentée)
     - Update `dev-latest`
   - ArgoCD dev (targetRevision: dev) sync automatiquement

3. **Promotion Prod (manuel, après validation dev):**
   ```bash
   gh workflow run promote-prod.yaml -f version=v1.2.3
   ```
   - Crée tag `prod-v1.2.3` pointant sur dev-v1.2.3
   - Update tag `prod-stable`
   - ArgoCD prod (targetRevision: prod-stable) sync automatiquement

## Consequences

### Positives

✅ **Simplicité:**
- 2 branches au lieu de 4 (dev/main seulement)
- Pas de merge conflicts entre environnements
- Historique Git linéaire et clair
- 1 PR au lieu de 4 pour chaque feature

✅ **Renovate Native:**
- Configuration simple: `"baseBranches": ["dev"]`
- PRs vers dev, test en dev, puis promote vers prod
- Workflow naturel pour Renovate

✅ **Rollback Instantané:**
- Changer le tag ArgoCD suffit
- Pas besoin de revert dans 4 branches

✅ **Audit Trail:**
- Tags = versions déployées
- Facile de voir quelle version est où

✅ **Industry Standard:**
- Aligné avec Google, GitLab, CNCF
- Documentation et support abondants
- Outils compatibles (Renovate, Dependabot, etc.)

✅ **Feature Flags:**
- Possibilité d'ajouter des toggles pour contrôle fin
- Déploiement != activation

### Négatives

⚠️ **Migration Complexe:**
- Nécessite de refondre les ArgoCD Applications
- Migration des branches existantes
- Formation de l'équipe

⚠️ **GitHub Actions Requises:**
- Dépendance à GitHub Actions
- Coût potentiel (mais probablement gratuit pour ce repo)

⚠️ **Changement Culturel:**
- Trunk-based dev = petites features
- Merge fréquent (plusieurs fois par jour)
- Nécessite discipline d'équipe

## Implementation Plan

### Phase 1: Préparation (2-3 jours)
1. ✅ Créer ADR-008 (ce document)
2. ✅ Valider avec l'équipe
3. 📝 Créer GitHub Actions workflows
4. 📝 Documenter le nouveau workflow

### Phase 2: Migration Test (1 semaine)
1. 🔄 Créer nouvelle structure de tags
2. 🔄 Migrer une application test (ex: whoami)
3. 🔄 Valider le workflow complet
4. 🔄 Ajuster si nécessaire

### Phase 3: Migration Progressive (2-3 semaines)
1. 🔄 Migrer applications par catégorie:
   - Infra (Traefik, cert-manager)
   - Monitoring (Prometheus, Grafana)
   - Apps (Home Assistant, etc.)
2. 🔄 Mettre à jour toutes les ArgoCD Applications
3. 🔄 Supprimer promote.sh

### Phase 4: Cleanup (1 semaine)
1. 🔄 Archiver les branches test/staging (dev et main conservées)
2. 🔄 Mettre à jour la documentation
3. 🔄 Former l'équipe au nouveau workflow

## Rollback Strategy

Si la migration échoue:
1. Conserver les branches dev/test/staging en backup
2. Restaurer les ArgoCD Applications originales
3. Revenir à promote.sh
4. Post-mortem pour identifier les problèmes

## Alternatives Considered

### Alternative A: Garder workflow actuel + améliorer promote.sh
**Rejetée:** Ne résout pas les problèmes fondamentaux (merge conflicts, complexité)

### Alternative B: GitFlow (develop branch)
**Rejetée:** Trop complexe pour une équipe small/solo, pas adapté au CD continu

### Alternative C: Flux Image Automation
**Reportée:** Trop spécialisé, nécessite migration vers Flux

## Success Metrics

**Après 3 mois, succès si:**
- ✅ 0 merge conflicts entre environnements
- ✅ Temps de promotion < 5 minutes (vs 15+ actuellement)
- ✅ 100% des PRs Renovate mergées automatiquement
- ✅ Rollbacks en < 2 minutes
- ✅ Historique Git lisible par un humain

## References

- [Trunk-Based Development](https://trunkbaseddevelopment.com/)
- [GitOps Best Practices (Akuity 2025)](https://akuity.io/blog/gitops-best-practices-whitepaper)
- [ArgoCD Environment Promotion](https://github.com/argoproj/argo-cd/discussions/5667)
- [How to Promote Releases Between GitOps Environments](https://mattias.engineer/blog/2023/gitops-environment-promotion/)
- [Continuous Promotion on Kubernetes with GitOps](https://piotrminkowski.com/2025/01/14/continuous-promotion-on-kubernetes-with-gitops/)

---

**Next Steps:**
1. ✅ Discussion avec l'équipe
2. ✅ Approbation de l'ADR
3. 🚀 Phase 1 en cours (Préparation)

**Decision Owner:** Architecture Team
**Target Implementation Date:** 2025-01-15
**Review Date:** 2025-04-15 (après 3 mois d'utilisation)
