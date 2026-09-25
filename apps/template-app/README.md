# Application Template (Vixens Standard)

Ce dossier sert de base pour toute nouvelle application intégrée au cluster.
Il implémente le **Golden Standard** défini dans `docs/reference/app-golden-standard.md`.

> **Référence canonique** : [`docs/reference/app-golden-standard.md`](../../docs/reference/app-golden-standard.md)

---

## Standards obligatoires (DoD)

Pour qu'une application soit « Production Ready », elle doit :

1. **`priorityClassName`** — défini selon la criticité (voir [Priority Classes](../../docs/reference/app-golden-standard.md#priority-classes))
2. **CP Toleration** — `node-role.kubernetes.io/control-plane` présente lorsque le pattern applicatif l'exige
3. **Kyverno sizing labels** — utiliser les labels `vixens.io/sizing.<container>: <tier>` selon les conventions actuelles du repo
4. **`revisionHistoryLimit: 3`** — sauf justification spécifique
5. **Résilience data** selon le workload : Litestream, Config-Syncer, CSI ou autre pattern déjà présent dans le repo
6. **Sécurité réseau** — flux Cilium explicitement déclarés avec le minimum nécessaire
7. **Secrets** — OpenBao + External Secrets Operator ; aucun secret en clair dans Git
8. **Exposition HTTP/TLS** — réutiliser les patterns Traefik/cert-manager actuels plutôt que copier d'anciens middlewares par défaut

---

## Structure des fichiers

### Base
- `deployment.yaml` — patron du workload
- `external-secret.yaml` — uniquement si l'application a besoin de secrets
- `litestream-config.yaml` — si Litestream est utilisé
- `service.yaml` — Service Kubernetes
- `namespace.yaml` — seulement pour les namespaces dédiés
- `cilium-networkpolicy.yaml` / `networkpolicy.yaml` — selon les patterns de sécurité applicables

### Overlays
- `dev/` — différences explicites de développement
- `prod/` — différences explicites de production

Pour les secrets, les overlays doivent modifier le **chemin OpenBao** ou les paramètres ESO nécessaires, jamais un ancien `envSlug` Infisical.

Pattern canonique :

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secrets
spec:
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: my-app-secrets
    creationPolicy: Owner
    deletionPolicy: Retain
  dataFrom:
    - extract:
        key: vixens/dev/apps/<category>/<app>
```

En production, utiliser le chemin canonique `vixens/prod/apps/<category>/<app>`.

---

## Sizing — comment ça marche

Suivre les conventions réellement présentes sur `main` et les références actuelles du repo. Ne pas recopier des tiers ou labels historiques sans vérifier `docs/reference/RESOURCE_STANDARDS.md` et les manifests applicatifs récents.

Exemple de labels :

```yaml
spec:
  template:
    metadata:
      labels:
        app: my-app
        vixens.io/sizing.my-app: small
        vixens.io/sizing.litestream: micro
```

---

## Workflow

Avant d'utiliser ce template :

```bash
git fetch origin
git switch main
git pull --ff-only
gh pr list --state open
```

Créer ou rattacher le changement à une GitHub Issue, travailler sur une branche courte, ouvrir une PR, laisser la CI valider puis laisser ArgoCD réconcilier le cluster. Ne pas utiliser ce template pour contourner le workflow GitOps.

---

## Guides techniques

- [Golden Standard complet](../../docs/reference/app-golden-standard.md)
- [Ajouter une application](../../docs/guides/adding-new-application.md)
- [Gestion des secrets](../../docs/guides/secret-management.md)
- [Pattern Config-Syncer](../../docs/guides/pattern-config-syncer.md)
- [Standards de ressources](../../docs/reference/RESOURCE_STANDARDS.md)
- [Niveaux de qualité](../../docs/reference/quality-standards.md)
- [`WORKFLOW.md`](../../WORKFLOW.md) — workflow canonique
