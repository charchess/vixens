# Stateless Native Template

Template pour applications stateless déployées avec des manifestes Kubernetes natifs (sans Helm).

## Quand utiliser ce template

- Application sans état (pas de données persistantes)
- Image Docker publique ou interne
- Configuration via variables d'environnement ou ConfigMap
- Probes HTTP simples
- Pas de dépendances complexes

## Exemples d'applications

- whoami (test HTTP)
- stirling-pdf (traitement PDF)
- it-tools (sans Helm)

## Structure

```
stateless-native/
├── base/
│   ├── deployment.yaml       # Deployment principal
│   ├── service.yaml          # Service ClusterIP
│   ├── namespace.yaml        # Namespace de l'app
│   └── kustomization.yaml    # Base Kustomize
└── overlays/
    ├── dev/
    │   └── kustomization.yaml  # Dev: replicas=0
    └── prod/
        └── kustomization.yaml  # Prod: composants gold-maturity
```

## Personnalisation

### 1. Remplacer les placeholders

Dans tous les fichiers:
- `APP_NAME` → nom de votre application
- `NAMESPACE_NAME` → namespace cible
- `IMAGE_REGISTRY/IMAGE_NAME:IMAGE_TAG` → image Docker

### 2. Ajuster les probes

Si votre application n'a pas `/health`:
- Modifier `path:` dans `livenessProbe` et `readinessProbe`
- Ou utiliser `tcpSocket:` pour probes TCP

Exemples:
```yaml
# HTTP probe (custom path)
livenessProbe:
  httpGet:
    path: /api/healthz
    port: http

# TCP probe
livenessProbe:
  tcpSocket:
    port: http
  initialDelaySeconds: 5
```

### 3. Ajuster le sizing

Modifier le label `vixens.io/sizing.APP_NAME`:
- **V-nano** (default) - 25m CPU / 32Mi RAM
- **V-small** - 50m CPU / 64Mi RAM
- **V-medium** - 100m CPU / 128Mi RAM
- **V-large** - 200m CPU / 256Mi RAM

### 4. Ajouter des variables d'environnement

Dans `base/deployment.yaml`, sous `containers[0]`:
```yaml
env:
  - name: VAR_NAME
    value: "value"
```

### 5. Ajouter un ConfigMap

Créer `base/configmap.yaml`:
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: APP_NAME-config
data:
  config.yaml: |
    key: value
```

Monter dans `base/deployment.yaml`:
```yaml
volumeMounts:
  - name: config
    mountPath: /config
volumes:
  - name: config
    configMap:
      name: APP_NAME-config
```

Ajouter dans `base/kustomization.yaml`:
```yaml
resources:
  - configmap.yaml
```

## Composants automatiques

### Dev (`overlays/dev/`)
- `components/base` - Labels de base
- `components/probes/basic` - Probes standards
- `replicas: 0` - Désactivé par défaut

### Prod (`overlays/prod/`)
- `components/gold-maturity` - Probes + securityContext renforcés
- `components/base` - Labels de base
- `components/resources` - Requests/limits automatiques (via VPA)
- `components/poddisruptionbudget/0` - PDB maxUnavailable=0
- `components/priority/low` - priorityClassName: vixens-low
- `components/revision-history-limit` - revisionHistoryLimit: 3

## Validation

```bash
# Lint
yamllint -c yamllint-config.yml apps/CATEGORY/APP_NAME/**/*.yaml

# Build dev
kustomize build apps/CATEGORY/APP_NAME/overlays/dev

# Build prod
kustomize build apps/CATEGORY/APP_NAME/overlays/prod

# Vérifier les kinds (detect missing resources)
kustomize build apps/CATEGORY/APP_NAME/overlays/prod | grep '^kind:' | sort
```

## Checklist DoD

Avant de marquer ready-for-dev:
- [ ] Tous les placeholders remplacés
- [ ] Probes adaptées à l'application
- [ ] Sizing label correct
- [ ] yamllint passe
- [ ] kustomize build dev passe
- [ ] kustomize build prod passe
- [ ] Kinds diff OK (aucune ressource manquante)

## Troubleshooting

**Probe failures:**
- Augmenter `initialDelaySeconds` si l'app démarre lentement
- Vérifier que le `path` existe (curl dans le pod)
- Utiliser `tcpSocket` si pas de HTTP health endpoint

**OOMKilled:**
- Augmenter le sizing tier (V-nano → V-small → V-medium)
- Vérifier les logs pour memory leaks

**CrashLoopBackOff:**
- Vérifier les logs: `kubectl logs -n NAMESPACE pod/APP_NAME-xxx`
- Vérifier les variables d'environnement manquantes
- Vérifier les permissions (runAsNonRoot, fsGroup)
