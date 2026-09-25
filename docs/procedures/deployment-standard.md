# Standard de déploiement Vixens

Ce document décrit le pattern de déploiement actuel des applications Vixens avec GitOps, Kustomize, Traefik, cert-manager et External Secrets Operator.

## Structure

```text
apps/<category>/<app-name>/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── external-secret.yaml      # si nécessaire
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

La structure exacte peut varier selon l'application ; privilégier les patterns déjà présents dans le repo avant d'en créer de nouveaux.

## 1. Secrets

Les valeurs secrètes vivent dans **OpenBao**. Les manifests Git contiennent uniquement des `ExternalSecret` qui utilisent le store canonique :

```yaml
secretStoreRef:
  name: openbao
  kind: ClusterSecretStore
```

Convention de chemins :

```text
vixens/dev/apps/<category>/<app>
vixens/prod/apps/<category>/<app>
```

Ne jamais utiliser `InfisicalSecret` pour un nouveau déploiement.

Voir `docs/guides/secret-management.md`.

## 2. Ingress et TLS

- Ingress controller : Traefik.
- Dev : `<app>.dev.truxonline.com`.
- Prod : `<app>.truxonline.com`.
- Certificats : cert-manager.
- Issuer dev : `letsencrypt-staging` lorsque l'application suit ce pattern.
- Issuer prod : `letsencrypt-prod`.

Réutiliser le pattern Traefik/cert-manager déjà utilisé par les applications similaires ; ne pas ajouter une nouvelle stratégie de redirection HTTP→HTTPS sans vérifier le comportement global de Traefik.

## 3. Réseau

Le cluster applique des politiques Cilium. Une nouvelle application doit déclarer explicitement les flux nécessaires lorsque le namespace ou le workload est soumis au default-deny.

Ne pas élargir une CiliumNetworkPolicy à `world` ou à tout le cluster par commodité sans identifier le besoin réel.

## 4. Stockage

Pour un volume `ReadWriteOnce`, vérifier la stratégie de rollout adaptée ; les applications qui ne peuvent pas monter deux fois le même volume utilisent généralement `strategy: Recreate`.

La couche CSI/TrueNAS est un contrat d'infrastructure séparé : réutiliser les `StorageClass` existantes plutôt que coder un endpoint NAS directement dans une application.

## 5. Ressources et scheduling

- Définir `requests` et `limits` selon les standards du repo.
- Ajouter une tolération control-plane uniquement lorsqu'elle est réellement requise par le pattern de l'application.
- Réutiliser les labels de sizing/maturité existants lorsqu'ils s'appliquent.

## 6. Observabilité

Quand l'application expose des métriques utiles :

- déclarer un `ServiceMonitor`/ressource équivalente selon le pattern monitoring existant ;
- éviter les labels à forte cardinalité ;
- ajouter dashboard/alerting seulement lorsqu'ils apportent un signal opérationnel utile.

## 7. ArgoCD

Dev suit `main`. Prod suit `prod-stable` via le workflow de promotion documenté dans `WORKFLOW.md`.

Tout changement persistant passe par Git → PR → CI → ArgoCD. Les commandes `kubectl apply/edit/delete` ne sont pas un mécanisme de déploiement normal.

## 8. Validation minimale

Avant PR :

```bash
kustomize build apps/<category>/<app>/overlays/dev >/dev/null
```

Après merge :

```bash
kubectl -n argocd get application <app>
```

Attendre `Synced` + `Healthy`, puis faire le test fonctionnel ciblé.

## Référence

- `WORKFLOW.md` — workflow canonique.
- `AGENTS.md` — règles agents.
- `docs/guides/secret-management.md` — secrets.
- `apps/template-app/` — exemple uniquement si son contenu est encore conforme aux standards ci-dessus.
