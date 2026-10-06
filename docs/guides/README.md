# Guides

Guides pratiques du dépôt Vixens. `WORKFLOW.md` reste la référence du cycle GitOps ; les guides détaillent des tâches spécifiques sans redéfinir le workflow.

## Workflow et contribution

- **[Adding a New Application](adding-new-application.md)** — structure et conventions pour ajouter une application.
- **[Task Management](task-management.md)** — GitHub Issues comme backlog canonique.
- **[GitOps Workflow](gitops-workflow.md)** — contexte GitOps complémentaire ; `WORKFLOW.md` prévaut en cas de divergence.
- **[Promotion Workflow](promotion-workflow.md)** — promotion dev → prod via GitHub Actions.
- **[Workflow Concurrency](workflow-concurrency.md)** — gestion des changements concurrents.
- **[Merge Queue Status](merge-queue-configuration.md)** — état réel (non actif actuellement) et prérequis d'une future activation.

## Configuration et secrets

- **[Secret Management](secret-management.md)** — OpenBao + External Secrets Operator.
- **[Config Syncer Pattern](pattern-config-syncer.md)** — persistance/synchronisation de configurations lorsqu'approprié.

Le backend secret canonique est OpenBao via `ClusterSecretStore/openbao` et `ExternalSecret`. Les documents historiques Infisical ne sont pas des guides d'implémentation.

## Résilience et exploitation

- **[Backup / Restore Pattern](backup-restore-pattern.md)** — patterns généraux de sauvegarde/restauration.
- **[Production Hibernation](prod-hibernation.md)** — comportement de mise en veille prod lorsque ce mécanisme est utilisé.
- **[Security & Observability](security-observability.md)** — repères sécurité/observabilité.

## Sizing et qualité

- **[Sizing Migration](sizing-migration.md)** — **stub historique** de l'ancienne migration v1 ; utiliser RESOURCE_STANDARDS/ADR-029 pour le sizing courant.
- **[Quality Reports](quality-reports.md)** — exploitation des rapports qualité.
- **[Bronzification Action Plan](bronzification-action-plan.md)** — **stub historique** ; ne pas l'utiliser comme runbook courant.

Références canoniques associées :

- `docs/reference/quality-standards.md`
- `docs/reference/RESOURCE_STANDARDS.md`
- `docs/reference/app-golden-standard.md`
- `docs/procedures/deployment-standard.md`

## Règles de maintenance

- Un guide actif doit décrire l'architecture **courante**.
- Les décisions historiques restent dans les ADR/audits/post-mortems.
- Ne pas recopier une procédure depuis un rapport ancien sans la confronter à `main`.
- Ajouter ici tout nouveau guide durable.

**Last Updated:** 2026-10-06
