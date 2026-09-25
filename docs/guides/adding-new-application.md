# Ajouter une nouvelle application

Ce guide décrit le chemin canonique pour ajouter une application à Vixens. `WORKFLOW.md` reste la référence pour le cycle Issue → PR → dev → promotion prod.

## 1. Commencer par le repo actuel

Avant d'écrire :

```bash
git fetch origin
git switch main
git pull --ff-only
gh pr list --state open
```

Créer ou lire l'issue GitHub concernée puis ouvrir une branche courte depuis `main`.

## 2. Choisir le pattern le plus proche

Chercher d'abord une application comparable dans `apps/` :

- même type de stockage ;
- même exposition réseau ;
- même namespace ou catégorie ;
- même besoin de secrets ;
- même mode de déploiement (Deployment/StatefulSet/Helm).

Réutiliser les conventions existantes plutôt que créer un nouveau pattern.

## 3. Structure Kustomize

Pattern courant :

```text
apps/<category>/<app>/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── external-secret.yaml   # si nécessaire
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

Une application Helm peut suivre une structure différente ; utiliser alors les applications Helm existantes comme référence.

## 4. Secrets

Les valeurs secrètes vivent dans **OpenBao** et sont projetées via **External Secrets Operator**.

Pattern minimal :

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secrets
  namespace: my-namespace
spec:
  refreshInterval: 60s
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

L'overlay prod pointe vers :

```text
vixens/prod/apps/<category>/<app>
```

Ne jamais créer de nouvel `InfisicalSecret`. Voir `docs/guides/secret-management.md`.

## 5. Ingress et TLS

- Traefik est l'ingress controller.
- Dev utilise généralement `<app>.dev.truxonline.com`.
- Prod utilise généralement `<app>.truxonline.com`.
- Les certificats sont gérés par cert-manager.
- Réutiliser les patterns d'Ingress/IngressRoute et de `Certificate` déjà présents dans la catégorie concernée.

Ne pas ajouter un middleware HTTP→HTTPS par habitude sans vérifier le comportement global de Traefik : la stratégie est en cours de normalisation.

## 6. Réseau Cilium

Avec le default-deny, déclarer explicitement les flux nécessaires :

- ingress depuis Traefik ou les workloads autorisés ;
- egress DNS/API/backend nécessaires ;
- accès `world` uniquement si l'application doit réellement joindre Internet.

Éviter les règles larges « pour que ça marche ».

## 7. Stockage

Réutiliser les `StorageClass` et patterns CSI existants.

Pour les PVC `ReadWriteOnce`, vérifier si le workload doit utiliser `strategy: Recreate` afin d'éviter un double montage pendant un rollout.

Ne pas coder d'adresse NAS ou d'identifiant TrueNAS directement dans le manifest applicatif si une abstraction CSI existe déjà.

## 8. Ressources et observabilité

- définir les ressources selon les conventions Vixens ;
- ajouter probes lorsque l'application les supporte ;
- exposer les métriques seulement si elles sont opérationnellement utiles ;
- utiliser `ServiceMonitor` et dashboards existants comme patterns ;
- éviter les labels de forte cardinalité.

## 9. ArgoCD

Ajouter l'application dans l'overlay ArgoCD approprié :

```text
argocd/overlays/dev/apps/
argocd/overlays/prod/apps/
```

Dev cible `main`. Prod cible le mécanisme de release défini par `WORKFLOW.md` (`prod-stable`).

## 10. Validation locale

Au minimum :

```bash
kustomize build apps/<category>/<app>/overlays/dev >/dev/null
kustomize build apps/<category>/<app>/overlays/prod >/dev/null
```

Puis ouvrir une PR et laisser les checks CI faire autorité.

## 11. Validation dev

Après merge :

```bash
kubectl -n argocd get application <app>
```

Attendre `Synced` et `Healthy`, puis tester le comportement réellement ajouté : HTTP, métriques, secret sync, stockage, réseau, etc.

## 12. Promotion prod

Après validation dev, utiliser le tag dev généré automatiquement :

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Ne jamais déplacer `prod-stable` manuellement.

## Checklist

- [ ] Issue GitHub définie.
- [ ] Branche créée depuis le `main` actuel.
- [ ] Pattern existant identifié et réutilisé.
- [ ] Aucun secret en clair dans Git.
- [ ] `ExternalSecret` utilise `ClusterSecretStore/openbao` si nécessaire.
- [ ] Policies Cilium minimales et explicites.
- [ ] Stockage conforme au CSI existant.
- [ ] Kustomize dev/prod rend sans erreur.
- [ ] CI verte.
- [ ] ArgoCD dev `Synced/Healthy`.
- [ ] Test fonctionnel dev réussi.
- [ ] Promotion prod via GitHub Actions uniquement.
