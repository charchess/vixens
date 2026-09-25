# Application Template (Vixens)

Ce dossier est un **exemple de structure**, pas une spécification autonome. Les références canoniques restent :

- `AGENTS.md`
- `WORKFLOW.md`
- `docs/procedures/deployment-standard.md`
- `docs/guides/adding-new-application.md`
- `docs/guides/secret-management.md`

Toujours comparer ce template à une application récente du même type avant de le copier : stockage, exposition réseau, probes et sidecars varient selon le workload.

## Structure

### Base

- `deployment.yaml` — exemple de Deployment avec priorité, tolération control-plane, sizing labels, probes et sidecars optionnels ;
- `external-secret.yaml` — projection OpenBao via External Secrets Operator ;
- `litestream-config.yaml` — exemple de configuration Litestream pour SQLite ;
- `service.yaml` — Service Kubernetes ;
- `namespace.yaml` — exemple de namespace dédié.

### Overlays

Les overlays `dev` / `prod` portent les différences d'environnement nécessaires. Ne dupliquer dans un overlay que ce qui diffère réellement du `base`.

## Secrets

Le modèle actif est :

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

Ne pas créer de nouvel `InfisicalSecret` et ne jamais stocker de valeur secrète dans Git.

Le chemin OpenBao du template est un exemple. Lors de la création d'une application, utiliser la convention réellement en vigueur dans les manifests de la catégorie et distinguer dev/prod lorsque nécessaire.

## Ressources

Les sizing labels restent utiles pour les politiques/VPA du cluster, mais les workloads peuvent aussi conserver des `resources.requests/limits` explicites comme fallback de bootstrap lorsque le pattern actuel de la catégorie le fait.

Ne pas appliquer une règle historique du type « jamais de bloc `resources:` » sans vérifier les manifests et politiques actuels.

## Stockage et résilience

Les blocs Litestream / config-syncer de ce template sont **optionnels** :

- SQLite → Litestream peut être pertinent ;
- fichiers de configuration persistants → config-syncer peut être pertinent ;
- une application stateless n'a pas besoin de ces sidecars ;
- un PVC `ReadWriteOnce` peut nécessiter `strategy: Recreate` selon le comportement du CSI et du workload.

Réutiliser le pattern de stockage déjà validé pour une application comparable.

## Réseau, ingress et TLS

- Traefik est l'ingress controller ;
- cert-manager gère les certificats ;
- les flux réseau requis doivent être explicités avec les NetworkPolicies/CiliumNetworkPolicies appropriées ;
- ne pas ajouter d'egress large ou de middleware HTTP→HTTPS par habitude : vérifier le pattern actuel dans Git.

## Validation

Avant merge :

```bash
kustomize build apps/<category>/<app>/overlays/dev
```

Puis laisser la CI du repo exécuter les validations complètes.

Après merge, dev suit `main` via ArgoCD. La production est promue uniquement via `.github/workflows/promote-prod.yaml` après validation dev.

## Règle anti-template fossile

Si ce template contredit une application récente ou la documentation canonique, **ne pas reproduire la contradiction**. Corriger le template dans une PR séparée ou suivre le pattern actuel le mieux établi.
