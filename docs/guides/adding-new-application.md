# Adding a new application

Ce guide décrit le chemin **canonique** pour ajouter une application à Vixens. Il complète `WORKFLOW.md` ; en cas de divergence, `WORKFLOW.md` fait autorité.

## 1. Commencer par l'état réel

Avant de créer des fichiers :

```bash
gh issue view <issue>
gh pr list --state open
git fetch origin
git switch main
git pull --ff-only
```

Rechercher ensuite une application **réelle et récente** de complexité similaire dans `apps/`. Les patterns du dépôt courant priment sur les vieux templates ou rapports.

## 2. Structure générale

Le pattern courant est généralement :

```text
apps/<category>/<app>/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml        # ou StatefulSet / ressources natives
│   ├── service.yaml
│   └── ...
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

Toutes les applications n'ont pas besoin des mêmes ressources. Ne pas créer un objet uniquement parce qu'un ancien template le contenait.

Les applications packagées upstream peuvent être déployées via Helm/multi-source ArgoCD si cela réduit la maintenance. Le code propre à Vixens reste en manifests/Kustomize.

## 3. Base Kubernetes

Au minimum, définir explicitement :

- labels `app.kubernetes.io/*` cohérents ;
- ressources CPU/mémoire ;
- probes pertinentes ;
- security context adapté ;
- Service si nécessaire ;
- stratégie de déploiement compatible avec le stockage ;
- tolérations/affinité seulement lorsqu'elles sont réellement nécessaires.

Pour un PVC `ReadWriteOnce` qui ne peut pas être monté simultanément par ancien et nouveau pod, vérifier si `strategy: Recreate` est requise.

## 4. Secrets : OpenBao + External Secrets

**Jamais de valeur secrète dans Git.**

Si l'application a besoin de credentials, utiliser le store canonique `openbao` :

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: <app>-secrets
  namespace: <namespace>
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: <app>-secrets
    creationPolicy: Owner
  dataFrom:
    - extract:
        key: vixens/dev/apps/<category>/<app>
```

L'overlay prod peut patcher uniquement le chemin distant :

```yaml
- op: replace
  path: /spec/dataFrom/0/extract/key
  value: vixens/prod/apps/<category>/<app>
```

Puis le workload consomme le Secret Kubernetes généré via `secretKeyRef`, `envFrom` ou volume Secret.

Voir `docs/guides/secret-management.md`.

## 5. Ingress / TLS

Avant de copier un Ingress/IngressRoute, regarder les applications récemment maintenues et les conventions Traefik/cert-manager courantes.

Principes :

- dev et prod ont des hostnames distincts ;
- TLS est explicite ;
- dev utilise l'issuer de staging lorsque le pattern courant le prévoit ;
- prod utilise `letsencrypt-prod` ;
- ne pas dupliquer une redirection HTTP→HTTPS si Traefik la fournit déjà globalement ;
- l'authentification/ForwardAuth est ajoutée lorsque l'application ne doit pas être publique.

Le chantier Traefik/cert-manager peut faire évoluer ces conventions : toujours lire le code courant.

## 6. NetworkPolicy

Vixens utilise Cilium et un modèle default-deny. Une application doit donc déclarer les flux dont elle a réellement besoin :

- ingress depuis Traefik si exposée ;
- ingress depuis des applications clientes spécifiques ;
- DNS ;
- egress vers les dépendances internes ;
- egress externe uniquement lorsque nécessaire.

Éviter `world`/`cluster` génériques par réflexe. Utiliser les observations Hubble pour comprendre un flux avant de l'autoriser.

## 7. Observabilité

Selon l'application :

- métriques natives → `ServiceMonitor` ;
- logs → pipeline central existant ;
- dashboard/alerting uniquement s'ils répondent à un besoin opérationnel ;
- annotations/labels de maturité selon les policies du dépôt.

## 8. ArgoCD

Ajouter l'application à l'overlay ArgoCD de dev en suivant une application du même type.

Dev suit `main`. Prod suit `prod-stable` et n'est promu qu'après validation dev via le workflow GitHub prévu.

Ne jamais créer une branche `dev` ou déplacer `prod-stable` manuellement.

## 9. Validation avant PR

Au minimum :

```bash
kustomize build apps/<category>/<app>/overlays/dev >/dev/null
kustomize build apps/<category>/<app>/overlays/prod >/dev/null
```

Puis vérifier :

- aucun secret en clair ;
- aucun fichier local/tooling accidentel ;
- les kinds attendus sont toujours rendus ;
- les références Kustomize existent ;
- les selectors/labels correspondent ;
- les ports Service/Deployment correspondent ;
- dev et prod diffèrent uniquement pour de bonnes raisons.

Ouvrir ensuite une PR et laisser CI appliquer les contrôles partagés.

## 10. Après merge

1. attendre ArgoCD dev `Synced` + `Healthy` ;
2. réaliser le test fonctionnel ciblé ;
3. vérifier NetworkPolicy/Hubble si pertinent ;
4. documenter les particularités durables de l'application ;
5. promouvoir la version dev via `promote-prod.yaml` lorsque prête ;
6. valider la production.

## Règle anti-template fossilisé

Les dossiers `.opencode`, les ADR historiques et les vieux rapports ne sont pas des bibliothèques de manifests à copier aveuglément. Pour créer une application, chercher d'abord **les manifests actifs dans `apps/`** et comprendre pourquoi leur pattern s'applique.
