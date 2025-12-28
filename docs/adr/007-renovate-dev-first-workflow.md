# ADR-007: Renovate Dev-First Workflow

**Date:** 2025-12-28
**Status:** ✅ Accepted
**Deciders:** System Architecture
**Tags:** `renovate`, `gitops`, `workflow`, `automation`

## Context

Vixens utilise un workflow de promotion strict à 4 étapes:
```
dev → test → staging → main (prod)
```

Les merges directs vers `main` sont interdits. Toute modification doit passer par le processus de promotion complet via `promote.sh`.

**Problème:** Renovate Bot, par défaut, cible la branche par défaut du repository (`main`), ce qui contournerait complètement le workflow de promotion et déploierait les mises à jour directement en production sans tests préalables.

## Decision

**Configurer Renovate pour cibler uniquement la branche `dev` via `baseBranches: ["dev"]`.**

### Configuration appliquée:

```json
{
  "baseBranches": ["dev"],
  "repositories": ["charchess/vixens"],
  "platform": "github"
}
```

### Workflow complet:

1. **Renovate détecte une mise à jour**
2. **Renovate crée une PR vers `dev`**
3. **Review manuelle de la PR**
4. **Merge dans `dev`**
5. **ArgoCD déploie automatiquement dans cluster dev**
6. **Tests et validation dans dev**
7. **Exécution de `./promote.sh`** pour propager:
   - dev → test (PR + auto-merge)
   - test → staging (PR + auto-merge)
   - staging → main/prod (PR + auto-merge)

## Consequences

### Positives

✅ **Respect du workflow GitOps strict**
- Aucune mise à jour ne peut atteindre prod sans passer par dev/test/staging
- Garantit que toutes les updates sont testées avant production

✅ **Visibilité et contrôle**
- Chaque PR Renovate peut être reviewée
- Possibilité de refuser/reporter certaines updates
- Dashboard GitHub avec toutes les updates proposées

✅ **Alignement avec la stratégie de branches**
- Cohérent avec le modèle dev→test→staging→main
- Pas de conflits entre branches

✅ **Traçabilité complète**
- Chaque update passe par 4 PRs (dev, test, staging, main)
- Historique Git clair de la propagation

### Négatives

⚠️ **Intervention manuelle requise**
- Nécessite d'exécuter `promote.sh` après chaque merge Renovate
- Pas d'automatisation complète de bout en bout

⚠️ **Délai de déploiement**
- Les updates mettent plus de temps à atteindre prod
- Acceptable car sécurité > vitesse pour l'infrastructure

⚠️ **Charge de maintenance**
- Plus de PRs à gérer (1 Renovate + 3 promote)
- Mitigé par l'auto-merge de promote.sh

## Alternatives Considered

### Option A: `baseBranches: ["dev", "test", "staging", "main"]`
**Rejetée:** Créerait 4 PRs identiques par update, conflits constants

### Option B: Renovate vers `main` uniquement
**Rejetée:** Contourne le workflow, déploie directement en prod sans tests

### Option C: Dependency Dashboard only (pas de PRs auto)
**Rejetée:** Perd l'avantage de l'automatisation, trop manuel

### Option D: Renovate + GitHub Action auto-promote
**Reportée:** Trop complexe pour l'instant, pourra être implémentée plus tard

## Implementation

**Fichier:** `apps/70-tools/renovate/base/configmap.yaml`

```json
"baseBranches": ["dev"]
```

**Déployé dans:** dev, test, staging, prod

## Monitoring

- Surveiller les PRs Renovate dans GitHub (label: `renovate`, `dependencies`)
- Vérifier que les PRs ciblent bien la branche `dev`
- Suivre le nombre de PRs en attente

## Future Evolution

**Phase 2 (optionnelle):**
- GitHub Action pour déclencher `promote.sh` automatiquement après merge Renovate
- Auto-merge conditionnel pour certains types d'updates (patch, minor)
- Intégration avec ArgoCD Notifications pour alertes

## References

- [Renovate baseBranches Documentation](https://docs.renovatebot.com/configuration-options/)
- [Workflow GitOps Vixens](../WORKFLOW.md)
- Script de promotion: `promote.sh`

---

**Decision Owner:** System Architecture
**Implementation Date:** 2025-12-28
**Review Date:** 2026-03-28 (après 3 mois d'utilisation)
