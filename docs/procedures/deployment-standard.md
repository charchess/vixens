# Application deployment standard

Ce document résume les conventions de déploiement Vixens. Il ne constitue pas une autorité concurrente : `WORKFLOW.md` reste canonique pour le cycle de changement, puis les workflows GitHub, manifests et policies courants définissent le comportement exécutable. `docs/reference/app-golden-standard.md` décrit les conventions applicatives générales.

## Structure

Pattern courant :

```text
apps/<category>/<app>/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml / statefulset.yaml / ressources natives
│   ├── service.yaml
│   ├── external-secret.yaml      # seulement si nécessaire
│   └── ...
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

Le nom des fichiers est descriptif ; il ne doit pas refléter une technologie retirée.

## Secrets

Backend canonique : **OpenBao + External Secrets Operator**.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-secrets
spec:
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: app-secrets
    creationPolicy: Owner
```

Les valeurs vivent dans OpenBao, jamais dans Git. Voir `docs/guides/secret-management.md`.

## Ingress / TLS

- Traefik est l'ingress controller.
- TLS est géré via cert-manager.
- dev et prod utilisent leurs issuers/hostnames prévus par les manifests actuels.
- authentification/middlewares seulement lorsque le besoin l'exige.
- ne pas créer de redirection HTTP→HTTPS applicative si le comportement est déjà globalement fourni.

Le chantier de normalisation Traefik/cert-manager peut encore faire évoluer ce pattern : copier une application récemment maintenue plutôt qu'un vieux document.

## Ressources et disponibilité

- définir `requests` et `limits` ;
- probes adaptées au comportement réel ;
- utiliser `strategy: Recreate` lorsque le stockage RWO l'impose ;
- PriorityClass/PDB seulement selon le niveau de maturité et la criticité ;
- ne pas forcer toutes les applications sur les control planes sans raison.

## Sécurité réseau

Cilium/default-deny implique des policies explicites.

Autoriser le minimum nécessaire :

- Traefik → backend ;
- consommateurs internes connus ;
- DNS ;
- APIs externes réellement utilisées.

Les drops Hubble servent à comprendre les flux avant de les autoriser.

## Observabilité

- `ServiceMonitor` lorsque des métriques utiles existent ;
- dashboards/alertes basés sur un besoin opératoire ;
- logs via les pipelines centraux ;
- VPA/Goldilocks en recommandation lorsque le pattern de maturité le demande.

## GitOps

- pas de mutation persistante via `kubectl apply/edit/delete` ;
- PR obligatoire vers `main` ;
- ArgoCD dev suit `main` ;
- production via `promote-prod.yaml` uniquement ;
- validation après sync.

Les commandes `kubectl` de lecture/diagnostic restent appropriées (`get`, `describe`, `logs`, tests ciblés). Une expérimentation qui modifie le cluster doit être éphémère et ne remplace jamais l'état désiré dans Git.

## Référence applicative

`apps/template-app/` peut aider à naviguer, mais les applications actives et récemment modifiées sont une meilleure source pour les détails. Toujours chercher au moins un exemple réel de même type avant de créer un nouveau pattern.
