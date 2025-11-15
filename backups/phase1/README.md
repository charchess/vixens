# Phase 1 Backups

Backups créés pendant la Phase 1 de l'optimisation DRY ArgoCD (Novembre 2025).

## Structure

### Backups Before Phase 2 (⭐ Important - À garder)
Ces fichiers contiennent les configurations AVANT l'externalisation des values Helm :

- `traefik.yaml.before-phase2` - Traefik avec values inline (66 lignes)
- `cert-manager.yaml.before-phase2` - cert-manager avec values inline (61 lignes)
- `cert-manager-webhook-gandi.yaml.before-phase2` - webhook avec values inline (41 lignes)

**Utilité** : Rollback ou comparaison avec les nouvelles configurations

### Backups Phase 1 (Archives)
Snapshots complets pris lors de la Phase 1 :

- `argocd-apps-20251114.yaml` - Tous les apps ArgoCD avant migration
- `argocd-apps-20251115.yaml` - Tous les apps ArgoCD après migration
- `argocd-{env}-20251114.yaml` - Snapshots par environnement

**Utilité** : Historique et audit de la migration

### Pilot Apps Backups
Backups des 3 applications pilotes testées en Phase 1.5 :

- `pilot-apps/argocd.yaml.backup`
- `pilot-apps/cilium-lb.yaml.backup`
- `pilot-apps/whoami.yaml.backup`

**Utilité** : Référence pour la décision "pas de migration" (Phase 1.5)

## Stratégie de Backup

### Garder
- ✅ `*.before-phase2` - Essentiels pour rollback Phase 2
- ✅ `argocd-apps-20251115.yaml` - Snapshot final Phase 1
- ✅ `pilot-apps/*` - Documentation décision Phase 1.5

### Peut Supprimer (si besoin d'espace)
- `argocd-apps-20251114.yaml` - Redondant avec 20251115
- `argocd-{env}-20251114.yaml` - Redondants avec argocd-apps

## Taille Totale
~288 KB

## Voir Aussi
- [docs/phase1-progress-report.md](../../docs/phase1-progress-report.md)
- [docs/phase1-pilot-migration-decision.md](../../docs/phase1-pilot-migration-decision.md)
- [docs/argocd-dry-optimization-plan.md](../../docs/argocd-dry-optimization-plan.md)
