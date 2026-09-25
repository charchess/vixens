# Application Standard — Vixens Reference

Ce document donne les conventions applicatives courantes. Il ne remplace pas les sources spécialisées :

- `AGENTS.md` / `WORKFLOW.md` — cycle de changement GitOps ;
- [ADR-023](../adr/023-7-tier-goldification-system-v2.md) + [ADR-029](../adr/029-align-maturity-with-current-platform.md) — maturité ;
- [RESOURCE_STANDARDS.md](RESOURCE_STANDARDS.md) — ressources/priorités ;
- [Secret Management](../guides/secret-management.md) — OpenBao / External Secrets ;
- [Deployment Standard](../procedures/deployment-standard.md) — structure de déploiement.

`apps/template-app/` est un exemple vivant. Une application récente du même type reste souvent la meilleure référence pour les détails spécifiques.

## Principes

Une application Vixens doit :

1. être déclarative et reproductible depuis Git ;
2. être validée par CI avant merge ;
3. être déployée par ArgoCD, pas par une mutation persistante manuelle du cluster ;
4. avoir des ressources explicites et un sizing adapté ;
5. déclarer seulement les flux réseau nécessaires ;
6. garder les valeurs secrètes hors Git ;
7. choisir stockage, backup et observabilité selon le workload réel.

## Structure courante

```text
apps/<category>/<app>/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml | statefulset.yaml | manifests Helm
│   ├── service.yaml
│   ├── external-secret.yaml       # si nécessaire
│   ├── networkpolicy.yaml         # si utilisé
│   ├── cilium-networkpolicy.yaml  # si utilisé
│   └── ...
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

Toutes les applications n'ont pas besoin de tous ces fichiers. Ne pas ajouter un composant uniquement pour satisfaire une forme de template.

## Ressources

Les `resources.requests` et `resources.limits` explicites sont le socle Kubernetes et servent également de fallback lorsque les mécanismes d'admission/tuning ne sont pas disponibles pendant un bootstrap ou une récupération.

```yaml
containers:
  - name: app
    resources:
      requests:
        cpu: 100m
        memory: 256Mi
      limits:
        cpu: 1000m
        memory: 1Gi
```

Les labels de sizing peuvent compléter ce bloc :

```yaml
metadata:
  labels:
    vixens.io/sizing.app: V-medium
```

Ne pas recopier une ancienne règle « jamais de `resources:` dans les manifests ». Utiliser `RESOURCE_STANDARDS.md`, les policies actuelles et les recommandations Goldilocks/VPA.

## PriorityClass

Choisir la classe selon la criticité réelle, pas selon la taille de l'application :

- `vixens-critical` — infrastructure indispensable ;
- `vixens-high` — services vitaux ;
- `vixens-medium` — applications interactives standards ;
- `vixens-low` — tâches de fond/sacrifiables.

Les valeurs exactes restent définies dans `RESOURCE_STANDARDS.md` et les manifests de PriorityClass courants.

## Probes

Quand l'application les supporte :

- **startupProbe** : laisse le temps de démarrer ;
- **readinessProbe** : décide si le pod peut recevoir du trafic ;
- **livenessProbe** : détecte un processus réellement bloqué.

Éviter les probes basées sur un processus intermittent. Préférer les endpoints health natifs lorsqu'ils existent.

## Secrets

Architecture canonique :

```text
OpenBao
  ↓
ClusterSecretStore/openbao
  ↓
ExternalSecret
  ↓
Kubernetes Secret
  ↓
workload
```

Exemple :

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-secrets
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: app-secrets
    creationPolicy: Owner
  dataFrom:
    - extract:
        key: vixens/prod/apps/category/app
```

Les chemins sont des exemples : utiliser la convention réellement présente dans les overlays/manifests actuels.

**Ne pas créer de nouvel `InfisicalSecret`.** Les anciennes mentions Infisical sont historiques sauf preuve contraire dans les manifests courants.

## Réseau

Le cluster utilise Cilium et un modèle default-deny sur les zones concernées.

Une policy doit exprimer le besoin réel :

- ingress depuis Traefik ou un producteur identifié ;
- egress DNS si nécessaire ;
- egress vers des services/namespaces/ports déterminés ;
- accès `world` uniquement lorsqu'il est nécessaire.

En cas de blocage, utiliser Hubble/Grafana/Loki pour identifier source, destination, port et raison avant d'élargir une policy.

## Ingress et TLS

- Traefik assure l'entrée HTTP/HTTPS ;
- cert-manager gère les certificats ;
- réutiliser les patterns courants `Ingress` / `IngressRoute` ;
- ne pas ajouter automatiquement un middleware de redirection HTTP→HTTPS sans vérifier la stratégie globale Traefik.

## Stockage

Choisir le pattern selon les caractéristiques du workload :

- stateless → pas de PVC inutile ;
- PVC `ReadWriteOnce` → vérifier la stratégie de rollout ;
- SQLite → Litestream seulement si ce mécanisme répond réellement au besoin ;
- fichiers de config persistants → Config-Syncer seulement si pertinent ;
- stockage CSI → utiliser les `StorageClass` existantes plutôt que coder les détails NAS dans l'application.

`strategy: Recreate` peut être pertinent lorsque deux réplicas ne peuvent pas monter simultanément un volume RWO.

## Résilience et backup

La maturité ne signifie pas « ajouter tous les sidecars ».

Déterminer :

1. quelles données sont importantes ;
2. où elles vivent ;
3. leur RPO/RTO utile ;
4. quel mécanisme les protège ;
5. comment la restauration est testée.

Une sauvegarde non testée n'est pas une restauration validée.

## Observabilité

Ajouter uniquement ce qui est exploitable :

- métriques applicatives ;
- `ServiceMonitor` lorsqu'approprié ;
- alertes avec action opérateur claire ;
- dashboard lorsque le signal mérite une visualisation dédiée ;
- logs structurés lorsque possible.

Éviter les labels à forte cardinalité dans les backends de métriques/logs.

## Kustomize components

Un component doit exprimer **un seul concern**. Préférer :

```text
revision-history-limit
sync-wave/wave-6
goldilocks/enabled
poddisruptionbudget/1
```

à un composant monolithique qui mélange sizing, priorité, observabilité et disponibilité.

## Validation

Avant merge :

```bash
kustomize build apps/<category>/<app>/overlays/dev
```

La CI du repo reste l'autorité pour la validation complète.

Après merge :

1. laisser ArgoCD converger en dev ;
2. tester le comportement ciblé ;
3. vérifier réseau/logs/métriques lorsque pertinent ;
4. promouvoir avec `.github/workflows/promote-prod.yaml` uniquement après validation.

## Maturité

La progression reste :

```text
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum
```

Le système mesure la **complétude de configuration**, pas une certification absolue de fiabilité.

Voir `quality-standards.md`, ADR-023 et ADR-029 pour les critères.

## Règle anti-fossile

Si ce document, un template ou un skill contredit :

- le `main` actuel ;
- une décision plus récente ;
- les manifests qui tournent réellement ;

ne pas recopier aveuglément l'ancien pattern. Identifier la contradiction et corriger la documentation dans une PR séparée.

**Last Updated:** 2026-09-25
