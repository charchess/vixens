# Application Template (Vixens Standard)

Ce dossier sert de base pour toute nouvelle application intégrée au cluster. Il respecte les standards de résilience et de QoS définis dans `docs/reference/`.

## Standards Obligatoires (DoD)

Pour qu'une application soit "Production Ready", elle doit :
1.  **Respecter le Scoring Model** : Atteindre au moins 85/100 (`docs/reference/APPLICATION_SCORING_MODEL.md`).
2.  **Définir sa QoS** : Utiliser un `priorityClassName` et des `resources` conformes (`docs/reference/RESOURCE_STANDARDS.md`).
3.  **Assurer la Résilience Data** :
    *   **SQLite** : Sidecar Litestream + Init Restore.
    *   **Config Plats** : Sidecar Config-Syncer + Init Restore.
4.  **Sécurité** : HTTPS obligatoire avec redirection (Middleware Traefik).

## Structure des Fichiers

### Base
- `deployment.yaml` : Contient la logique de résilience (InitContainers & Sidecars).
- `infisical-secret.yaml` : Récupération des secrets métiers.
- `litestream-secret.yaml` : Récupération des secrets S3 (Backup).
- `litestream.yml` : Configuration de la réplication SQLite.

### Overlays
- `prod/` : Patcher le `envSlug` vers `prod` et ajuster les ressources.

## Aide à l'implémentation

Consultez les guides techniques :
- [Pattern Config-Syncer](../../docs/guides/pattern-config-syncer.md)
- [Backup SQLite (Litestream)](../../docs/guides/adding-new-application.md#sqlite-backup-strategy-litestream)
