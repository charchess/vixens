# Application Template (Vixens Standard)

Ce dossier sert de base pour toute nouvelle application intégrée au cluster.
Il implémente le **Golden Standard** défini dans `docs/reference/app-golden-standard.md`.

> **Reference canonique** : [`docs/reference/app-golden-standard.md`](../../docs/reference/app-golden-standard.md)

---

## Standards Obligatoires (DoD)

Pour qu'une application soit "Production Ready", elle doit :

1. **`priorityClassName`** — défini selon la criticité (voir [Priority Classes](../../docs/reference/app-golden-standard.md#priority-classes))
2. **CP Toleration** — `node-role.kubernetes.io/control-plane` présente dans le manifest
3. **Kyverno Sizing Labels** — labels `vixens.io/sizing.<container>: <tier>` sur le pod template (jamais de blocs `resources:` explicites)
4. **`revisionHistoryLimit: 3`** — réduit la croissance etcd
5. **Résilience Data** :
   - **SQLite** : sidecar Litestream + init restore-db
   - **Config plats** : sidecar Config-Syncer + init restore-config
6. **Sécurité** : HTTPS obligatoire avec redirection (Middleware Traefik)

---

## Structure des Fichiers

### Base
- `deployment.yaml` — Patron complet (priorité, toleration, sizing labels, resilience inits & sidecars)
- `infisical-secret.yaml` — Récupération des secrets via Infisical
- `litestream-config.yaml` — Configuration de la réplication SQLite (Litestream)
- `service.yaml` — Service Kubernetes
- `namespace.yaml` — Namespace de l'application

### Overlays
- `prod/` — Patch `envSlug` vers `prod`, ajustements spécifiques prod

---

## Sizing — Comment Ça Marche

Les ressources (`requests`/`limits`) ne sont **jamais** définies dans le YAML.
Elles sont injectées à l'admission par la policy Kyverno `sizing-mutate` via des labels :

```yaml
spec:
  template:
    metadata:
      labels:
        app: my-app
        vixens.io/sizing: small                # fallback générique
        vixens.io/sizing.my-app: small         # container principal
        vixens.io/sizing.litestream: micro     # sidecar litestream
        vixens.io/sizing.config-syncer: micro  # sidecar config-syncer
        vixens.io/sizing.restore-config: micro # init restore-config
        vixens.io/sizing.restore-db: micro     # init restore-db
```

Tiers disponibles : `micro`, `small`, `medium`, `large`, `xlarge`, `G-small`, `G-medium`, `G-large`, `G-xl`

---

## Guides Techniques

- [Golden Standard complet](../../docs/reference/app-golden-standard.md)
- [Pattern Config-Syncer](../../docs/guides/pattern-config-syncer.md)
- [Backup SQLite (Litestream)](../../docs/guides/adding-new-application.md#sqlite-backup-strategy-litestream)
- [Standards de ressources](../../docs/reference/RESOURCE_STANDARDS.md)
- [Niveaux de qualité (tiers)](../../docs/reference/quality-standards.md)
