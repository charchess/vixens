# Stateless Helm Template

Template pour applications stateless déployées via Helm chart avec Kustomize overlays pour patches spécifiques Vixens.

## Quand utiliser ce template

- Application fournie avec un Helm chart (officiel ou communauté)
- Configuration standard Helm pas assez flexible (besoin de patches Kustomize)
- Besoin de gérer replicas via Kustomize (dev vs prod)
- Besoin d'annotations spécifiques Vixens (gold-maturity, sync-waves)

## Exemples d'applications

- it-tools (chart communauté)
- dashy (chart communauté)
- Homepage (chart communauté)

## Structure

```
stateless-helm/
├── base/
│   ├── values.yaml           # Helm values (configuration partagée)
│   └── kustomization.yaml    # Base Kustomize (namespace uniquement)
└── overlays/
    ├── dev/
    │   └── kustomization.yaml  # Dev: replicas=0 + components
    └── prod/
        └── kustomization.yaml  # Prod: gold-maturity patches
```

## Personnalisation

### 1. Remplacer les placeholders

Dans tous les fichiers:
- `APP_NAME` → nom de votre application
- `NAMESPACE_NAME` → namespace cible
- `CHART_REPO/CHART_NAME` → Helm chart (ex: `jeffresc/it-tools`)
- `IMAGE_REGISTRY/IMAGE_NAME` → image Docker
- `IMAGE_TAG` → version spécifique (pas `latest`)

### 2. Adapter les values.yaml

**Sizing:**
```yaml
podLabels:
  vixens.io/sizing.APP_NAME: B-nano  # Change selon l'app
```

Tiers disponibles:
- **B-nano** (default) - 10m CPU / 16Mi RAM (ultra-léger)
- **V-nano** - 25m CPU / 32Mi RAM
- **V-small** - 50m CPU / 64Mi RAM
- **V-medium** - 100m CPU / 128Mi RAM

**Annotations:**
```yaml
podAnnotations:
  vixens.io/fast-start: "true"           # App démarre vite (<10s)
  vixens.io/service-binding: "false"     # Pas de service-binding
  vixens.io/nometrics: "true"            # Pas de metrics Prometheus
  vixens.io/no-long-connections: "true"  # Pas de connexions longues (WebSocket)
```

Retirez les annotations non applicables.

**Security context:**
```yaml
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  runAsNonRoot: true
  readOnlyRootFilesystem: false  # true si l'app supporte
  capabilities:
    drop:
      - ALL
```

Ajustez `runAsUser`/`runAsGroup` selon le chart Helm (souvent 1000, parfois 65534).

### 3. Configurer ArgoCD Application

Créer `argocd/APP_NAME.yaml`:
```yaml
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: APP_NAME-dev
  namespace: argocd
spec:
  project: default
  sources:
    # Source 1: Helm chart
    - repoURL: https://CHART_REPO_URL
      chart: CHART_NAME
      targetRevision: CHART_VERSION
      helm:
        valueFiles:
          - $values/apps/CATEGORY/APP_NAME/base/values.yaml
    # Source 2: Kustomize overlays
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: main
      ref: values
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: main
      path: apps/CATEGORY/APP_NAME/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: NAMESPACE_NAME
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**IMPORTANT:** Multi-source ArgoCD Application:
1. Source 1 = Helm chart + `valueFiles` référence relative
2. Source 2 = `ref: values` pour résoudre `$values/...`
3. Source 3 = Kustomize overlays pour patches

### 4. Patches spécifiques prod

Dans `overlays/prod/kustomization.yaml`, les patches JSON ciblent le Deployment généré par Helm:

```yaml
patches:
  - patch: |-
      - op: add
        path: /metadata/annotations/argocd.argoproj.io~1sync-wave
        value: "10"
    target:
      kind: Deployment
      name: APP_NAME  # DOIT correspondre au nom du Deployment Helm
```

**Trouver le nom du Deployment:**
```bash
helm template APP_NAME CHART_REPO/CHART_NAME -f base/values.yaml | grep 'kind: Deployment' -A 5
```

### 5. Limitations Helm à contourner

**Helm ne supporte pas:**
- `deploymentAnnotations` (uniquement `podAnnotations`)
- `revisionHistoryLimit` customisable
- Patches sur pod template

**Solution:**
- Utiliser patches JSON Kustomize pour Deployment metadata
- Utiliser `components/revision-history-limit` component
- Utiliser patches strategicMerge pour pod template

## Composants automatiques

### Dev (`overlays/dev/`)
- `components/revision-history-limit` - revisionHistoryLimit: 3
- `replicas: 0` - Désactivé par défaut

### Prod (`overlays/prod/`)
- `components/revision-history-limit` - revisionHistoryLimit: 3
- Gold-maturity annotations (via JSON patches):
  - `argocd.argoproj.io/sync-wave: "10"`
  - `goldilocks.fairwinds.com/enabled: "true"`
  - `vpa.kubernetes.io/updateMode: "Off"`

## Validation

```bash
# Lint values.yaml
yamllint -c yamllint-config.yml apps/CATEGORY/APP_NAME/base/values.yaml

# Test Helm render
helm template APP_NAME CHART_REPO/CHART_NAME -f apps/CATEGORY/APP_NAME/base/values.yaml

# Build dev overlay
kustomize build apps/CATEGORY/APP_NAME/overlays/dev

# Build prod overlay
kustomize build apps/CATEGORY/APP_NAME/overlays/prod

# Vérifier les kinds (detect missing resources)
kustomize build apps/CATEGORY/APP_NAME/overlays/prod | grep '^kind:' | sort
```

## Checklist DoD

Avant de marquer ready-for-dev:
- [ ] Tous les placeholders remplacés
- [ ] Helm chart existe et version spécifiée
- [ ] values.yaml adapté (sizing, annotations, securityContext)
- [ ] ArgoCD Application créée (multi-source config)
- [ ] Patches prod ciblent le bon Deployment name
- [ ] yamllint passe sur values.yaml
- [ ] helm template passe sans erreur
- [ ] kustomize build dev passe
- [ ] kustomize build prod passe
- [ ] Kinds diff OK (aucune ressource manquante)

## Troubleshooting

**Helm template fails:**
- Vérifier le chart existe: `helm search repo CHART_NAME`
- Vérifier les values.yaml correspondent au chart (consulter chart README)
- Utiliser `helm show values CHART_REPO/CHART_NAME` pour voir les defaults

**ArgoCD sync fails (multi-source):**
- Vérifier `ref: values` présent dans source 2
- Vérifier `$values/...` path est correct (relatif au repo)
- Vérifier les 3 sources sont dans le bon ordre

**Patches ne s'appliquent pas:**
- Vérifier `target.name` correspond exactement au nom du Deployment
- Utiliser `kubectl get deployment -n NAMESPACE` pour confirmer le nom
- Vérifier le path JSON est correct (tildes `~1` pour slashes `/`)

**revisionHistoryLimit ignoré:**
- Utiliser component `revision-history-limit` au lieu de patch direct
- Le component gère tous les kinds (Deployment, StatefulSet, DaemonSet)

**OOMKilled:**
- Augmenter le sizing tier dans `podLabels`
- Vérifier les requests/limits via `kubectl top pod`
- VPA suggérera des valeurs après quelques jours
