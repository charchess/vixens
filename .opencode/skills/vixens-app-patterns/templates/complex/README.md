# Complex Template

Template pour applications complexes avec plusieurs Deployments, NetworkPolicy, ServiceMonitor, et autres ressources avancées.

## Quand utiliser ce template

- Application multi-composants (server + workers + jobs)
- NetworkPolicy spécifique (pas juste Traefik ingress)
- ServiceMonitor pour Prometheus scraping
- VPA pour recommendations
- Dependencies complexes (Redis, PostgreSQL externe)
- Configuration avancée (middleware Traefik, forward-auth)

## Exemples d'applications

- **authentik** - Server + Worker + NetworkPolicy + ServiceMonitor
- **homeassistant** - 5 containers + VPA + ServiceMonitor + ConfigMap
- **postgresql-shared** - CNPG cluster + pooler + backups

## Architecture

Applications complexes nécessitent:
- **Multiple Deployments/StatefulSets** - Plusieurs workloads coordonnés
- **NetworkPolicy** - Sécurité réseau fine-grained
- **ServiceMonitor** - Monitoring Prometheus
- **VPA** - Vertical Pod Autoscaler recommendations
- **Middleware** - Traefik middleware (forward-auth, headers)
- **External dependencies** - Redis, PostgreSQL, S3

## Structure recommandée

```
complex-app/
├── base/
│   ├── deployment-server.yaml      # Main deployment
│   ├── deployment-worker.yaml      # Worker deployment
│   ├── service.yaml               # Service(s)
│   ├── configmap.yaml             # Configuration
│   ├── infisical-secret.yaml      # Secrets
│   ├── networkpolicy.yaml         # Network security
│   ├── servicemonitor.yaml        # Prometheus scraping
│   ├── vpa.yaml                   # VPA recommendations
│   ├── middleware-*.yaml          # Traefik middleware
│   └── kustomization.yaml         # Base Kustomize
└── overlays/
    ├── dev/
    │   └── kustomization.yaml      # Dev patches
    └── prod/
        └── kustomization.yaml      # Prod patches
```

## Patterns disponibles

Copiez depuis `patterns/` directory:

### 1. NetworkPolicy (`patterns/networkpolicy.yaml`)
```yaml
# Traefik ingress + full egress
- Ingress: Allow from Traefik namespace
- Egress: Allow all (DNS, inter-app, external APIs)
```

**Personnalisation:**
- Changer `port:` si pas HTTP standard
- Ajouter ingress rules pour inter-pod communication
- Restreindre egress si besoin (production hardening)

### 2. ServiceMonitor (`patterns/servicemonitor.yaml`)
```yaml
# Prometheus scraping
- Port: metrics (9090)
- Path: /metrics
- Interval: 60s
```

**Personnalisation:**
- Changer `targetPort` selon votre app
- Changer `path` si pas `/metrics`
- Ajuster `interval` (10s pour metrics fréquents, 60s par défaut)

### 3. VPA (`patterns/vpa.yaml`)
```yaml
# Resource recommendations (updateMode: Off)
- Recommendations only
- No automatic updates
```

**Personnalisation:**
- Changer `kind:` si StatefulSet
- Utiliser `updateMode: Auto` si vous voulez auto-update (NOT recommended)

## Exemples réels

### Authentik (2 Deployments + NetworkPolicy + ServiceMonitor)

Voir `examples/authentik/`:
- `deployment-server.yaml` - API server
- `deployment-worker.yaml` - Background workers
- `networkpolicy.yaml` - Traefik ingress only
- `servicemonitor.yaml` - Prometheus metrics
- `infisical-secret.yaml` - Secrets (DB, Redis, SMTP)
- `middleware-forward-auth.yaml` - Traefik forward-auth

**Pattern clés:**
1. Worker sans ingress (NetworkPolicy séparée)
2. Secrets partagés (server + worker)
3. Redis externe (connection via secret)

### HomeAssistant (5 containers + VPA)

Voir `examples/homeassistant/` (si copié):
- Main container + 4 sidecars
- VPA pour recommendations
- ConfigMap volumineux
- PVC pour données

**Pattern clés:**
1. Multiple containers = multiple sizing labels
2. VPA analyse tous les containers
3. initContainer pour permissions

## Multi-Deployment Patterns

### Pattern 1: Server + Worker
```yaml
# deployment-server.yaml
metadata:
  name: APP_NAME-server
  labels:
    app: APP_NAME-server
    component: server

# deployment-worker.yaml
metadata:
  name: APP_NAME-worker
  labels:
    app: APP_NAME-worker
    component: worker
```

**Secrets partagés:**
```yaml
# infisical-secret.yaml (single source)
metadata:
  name: APP_NAME-secrets

# Both deployments reference
envFrom:
  - secretRef:
      name: APP_NAME-secrets
```

### Pattern 2: Multiple NetworkPolicies
```yaml
# networkpolicy-server.yaml
spec:
  podSelector:
    matchLabels:
      app: APP_NAME-server
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: traefik

# networkpolicy-worker.yaml
spec:
  podSelector:
    matchLabels:
      app: APP_NAME-worker
  ingress: []  # No ingress (internal only)
  egress:
    - {}  # Full egress
```

### Pattern 3: ServiceMonitor multi-targets
```yaml
# servicemonitor.yaml
spec:
  selector:
    matchLabels:
      monitoring: "true"  # Label commun
  endpoints:
    - port: metrics
      targetPort: 9090
```

Ajouter `monitoring: "true"` label sur tous les Services à monitorer.

## Composants automatiques

### Dev (`overlays/dev/`)
- `components/base` - Labels de base
- `components/probes/basic` - Probes standards
- `replicas: 0` - Tous les Deployments désactivés
- `envSlug: dev` - Secrets Infisical dev

### Prod (`overlays/prod/`)
- `components/gold-maturity` - Probes + securityContext renforcés
- `components/base` - Labels de base
- `components/resources` - Requests/limits automatiques
- `components/poddisruptionbudget/0` - PDB sur tous les Deployments
- `components/priority/high` - priorityClassName: vixens-high (apps critiques)
- `components/revision-history-limit` - revisionHistoryLimit: 3
- `envSlug: prod` - Secrets Infisical prod

## Validation

```bash
# Lint
yamllint -c yamllint-config.yml apps/CATEGORY/APP_NAME/**/*.yaml

# Build dev
kustomize build apps/CATEGORY/APP_NAME/overlays/dev

# Build prod
kustomize build apps/CATEGORY/APP_NAME/overlays/prod

# Vérifier les kinds (important: multiple Deployments)
kustomize build apps/CATEGORY/APP_NAME/overlays/prod | grep '^kind:' | sort

# Expected kinds (exemple authentik):
# - ConfigMap
# - Deployment (2x)
# - InfisicalSecret
# - Middleware
# - NetworkPolicy (2x)
# - PersistentVolumeClaim
# - Service
# - ServiceMonitor
```

## Checklist DoD

Avant de marquer ready-for-dev:
- [ ] Tous les Deployments ont sizing labels
- [ ] NetworkPolicy pour chaque Deployment
- [ ] ServiceMonitor si metrics exposés
- [ ] VPA si besoin de recommendations
- [ ] Infisical secrets créés (dev + prod)
- [ ] Dependencies externes configurées (Redis, PostgreSQL)
- [ ] yamllint passe
- [ ] kustomize build dev passe
- [ ] kustomize build prod passe
- [ ] Kinds diff OK (aucun Deployment/Service manquant)

## Troubleshooting

### Multiple Deployments pas tous déployés

**Symptômes:** `kubectl get deployment` montre seulement 1/2 deployments

**Cause:** Overlay `replicas: 0` patch cible un seul Deployment

**Fix:**
```yaml
# overlays/dev/kustomization.yaml
patches:
  # Patch TOUS les Deployments
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: any  # Wildcard
      spec:
        replicas: 0
    target:
      kind: Deployment  # Pas de name filter
```

### NetworkPolicy bloque traffic inter-pod

**Symptômes:** Worker ne peut pas joindre Server

**Fix:** Ajouter ingress rule sur Server NetworkPolicy:
```yaml
ingress:
  - from:
      - podSelector:
          matchLabels:
            app: APP_NAME-worker
    ports:
      - protocol: TCP
        port: 8080
```

### ServiceMonitor no targets

**Symptômes:** Prometheus ne scrape pas

**Causes:**
- Label selector mismatch
- Namespace selector mismatch
- Port name incorrect

**Debug:**
```bash
# Vérifier ServiceMonitor
kubectl get servicemonitor -n NAMESPACE APP_NAME -o yaml

# Vérifier Service labels
kubectl get service -n NAMESPACE APP_NAME -o yaml | grep -A 10 labels

# Vérifier port name
kubectl get service -n NAMESPACE APP_NAME -o yaml | grep -A 5 ports
```

### VPA no recommendations

**Symptômes:** VPA status empty

**Causes:**
- Pod pas assez de runtime (attendre 24h minimum)
- updateMode incorrect
- targetRef incorrect

**Debug:**
```bash
# Vérifier VPA status
kubectl describe vpa -n NAMESPACE APP_NAME

# Attendre metrics
kubectl get vpa -n NAMESPACE APP_NAME -o yaml | grep -A 10 recommendation
```

### Secrets synchronization between Deployments

**Pattern:** Single InfisicalSecret, multiple Deployments reference it.

**ATTENTION:** Secret mutation (Infisical sync) ne trigger pas Deployment rollout.

**Solution:** Utiliser Reloader:
```yaml
metadata:
  annotations:
    reloader.stakater.com/auto: "true"
```

## Migration depuis stateless/stateful templates

Si vous commencez simple (stateless) puis évoluez vers complex:

1. **Ajouter NetworkPolicy** - Copier `patterns/networkpolicy.yaml`
2. **Ajouter ServiceMonitor** - Si metrics exposés
3. **Split Deployments** - Server vs Worker pattern
4. **Ajouter VPA** - Si besoin de recommendations
5. **Tester overlay patches** - Vérifier tous les Deployments ciblés

**IMPORTANT:** Après chaque ajout, vérifier kinds diff:
```bash
kustomize build overlays/prod | grep '^kind:' | sort
```
