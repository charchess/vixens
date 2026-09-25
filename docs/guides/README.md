# Guides

Guides pratiques du dépôt Vixens. `WORKFLOW.md` reste la référence du cycle GitOps ; les guides détaillent des tâches spécifiques sans redéfinir le workflow.

## Workflow et contribution

- **[Adding a New Application](adding-new-application.md)** — structure et conventions pour ajouter une application.
- **[Task Management](task-management.md)** — GitHub Issues comme backlog canonique.
- **[GitOps Workflow](gitops-workflow.md)** — contexte GitOps complémentaire ; `WORKFLOW.md` prévaut en cas de divergence.
- **[Promotion Workflow](promotion-workflow.md)** — promotion dev → prod via GitHub Actions.
- **[Workflow Concurrency](workflow-concurrency.md)** — gestion des changements concurrents.
- **[Merge Queue Configuration](merge-queue-configuration.md)** — configuration/usage de la merge queue lorsqu'elle est activée.

## Configuration et secrets

- **[Secret Management](secret-management.md)** — OpenBao + External Secrets Operator.
- **[Config Syncer Pattern](pattern-config-syncer.md)** — persistance/synchronisation de configurations lorsqu'approprié.

Le backend secret canonique est OpenBao via `ClusterSecretStore/openbao` et `ExternalSecret`. Les documents historiques Infisical ne sont pas des guides d'implémentation.

## Résilience et exploitation

- **[Backup / Restore Pattern](backup-restore-pattern.md)** — patterns généraux de sauvegarde/restauration.
- **[Production Hibernation](prod-hibernation.md)** — comportement de mise en veille prod lorsque ce mécanisme est utilisé.
- **[Security & Observability](security-observability.md)** — repères sécurité/observabilité.

## Sizing et qualité

- **[Sizing Migration](sizing-migration.md)** — contexte et conventions de sizing.
- **[Quality Reports](quality-reports.md)** — exploitation des rapports qualité.
- **[Bronzification Action Plan](bronzification-action-plan.md)** — plan historique/opérationnel de montée en maturité ; vérifier les standards actuels avant de reprendre une action ancienne.

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

**Last Updated:** 2026-09-25
