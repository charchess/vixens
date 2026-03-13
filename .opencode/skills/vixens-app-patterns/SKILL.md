# Vixens Application Patterns - OpenCode Skill

**Version:** 1.0.0  
**Created:** 2026-03-10  
**Scope:** Project-local (~/vixens/.opencode/skills/)  

Skill complet pour déploiement d'applications Kubernetes sur cluster Vixens homelab avec GitOps (ArgoCD + Kustomize).

## 📋 Résumé

Ce skill documente **tous les patterns de déploiement** validés sur le cluster Vixens, basés sur l'analyse de 42 applications réelles (39 Deployments + 3 StatefulSets). Il fournit des templates prêts à l'emploi, une matrice de décision, et des exemples réels.

**Validation:** 292 occurrences de sizing labels, 100% adoption base/overlays, conformité DoD complète.

## 🎯 Quand utiliser ce skill

**Pour AI agents (Claude, etc.):**
- Déployer une nouvelle application sur cluster Vixens
- Migrer une application existante vers les standards Vixens
- Comprendre les patterns de déploiement Kubernetes + GitOps
- Valider la conformité DoD d'une application
- Troubleshooter un déploiement problématique

**Triggers:**
- "déployer [app] sur vixens"
- "créer une nouvelle app [type]"
- "migrer [app] vers stateful pattern"
- "valider [app] contre DoD"
- "pourquoi [app] crashloop?"

## 📚 Structure du skill

```
vixens-app-patterns/
├── SKILL.md (ce fichier)          # Vue d'ensemble + guide agent
├── README.md                       # Quick start utilisateur
├── docs/
│   ├── decision-matrix.md          # Matrice décision (flowchart)
│   └── checklist.md                # DoD validation checklist
├── templates/
│   ├── stateless-native/           # K8s manifests (whoami)
│   ├── stateless-helm/             # Helm + Kustomize (it-tools)
│   ├── stateful/                   # SQLite + Litestream (vaultwarden)
│   └── complex/                    # Multi-composants (authentik)
├── patterns/
│   ├── networkpolicy.yaml          # Traefik ingress pattern
│   ├── servicemonitor.yaml         # Prometheus scraping
│   └── vpa.yaml                    # VPA recommendations
└── examples/
    ├── whoami/                     # Exemple stateless simple
    ├── vaultwarden/                # Exemple stateful complet
    └── authentik/                  # Exemple complex (2 Deployments)
```

## 🚀 Workflow Agent (Déploiement nouvelle app)

### Phase 1: Discovery

**Questions à poser à l'utilisateur:**

1. **Application info:**
   - Nom de l'application?
   - Image Docker (registry/name:tag)?
   - Port HTTP principal?
   - Health check endpoint?

2. **État persistant:**
   - Besoin de stocker des données? (Oui → stateful, Non → stateless)
   - Type de base de données? (SQLite → stateful template, PostgreSQL → external DB)

3. **Helm chart:**
   - Chart Helm disponible? (Oui → stateless-helm, Non → stateless-native)
   - Repo Helm? Version?

4. **Complexité:**
   - Plusieurs composants? (Oui → complex)
   - Workers séparés? (Oui → complex)
   - NetworkPolicy custom? (Oui → complex)

### Phase 2: Template Selection

**Consulter:** `docs/decision-matrix.md`

**Flowchart simplifié:**

```
1. État persistant (SQLite)?
   ├─ Oui → STATEFUL template
   └─ Non → 2

2. Helm chart disponible?
   ├─ Oui → STATELESS-HELM template
   └─ Non → 3

3. Multi-composants (>1 Deployment)?
   ├─ Oui → COMPLEX template
   └─ Non → STATELESS-NATIVE template
```

### Phase 3: Implementation

**Actions:**

1. **Copier template**
   ```bash
   cp -r templates/[TEMPLATE_TYPE] apps/[CATEGORY]/[APP_NAME]
   ```

2. **Remplacer placeholders**
   - `APP_NAME` → nom application
   - `NAMESPACE_NAME` → namespace cible
   - `IMAGE_REGISTRY/IMAGE_NAME:IMAGE_TAG` → image Docker
   - `CATEGORY` → catégorie app (ex: 60-services)
   - `ENV_SLUG` → dev ou prod

3. **Adapter configuration**
   - Probes (liveness/readiness/startup paths)
   - Sizing labels (tier approprié)
   - Annotations spécifiques (metrics, service-binding, etc.)
   - Variables d'environnement

4. **Stateful apps only:**
   - Créer buckets S3 (Minio)
   - Créer secrets Infisical (dev + prod)
   - Configurer Litestream excludes
   - Ajuster PVC size

5. **Complex apps only:**
   - Copier patterns (NetworkPolicy, ServiceMonitor, VPA)
   - Configurer dependencies externes (Redis, PostgreSQL)
   - Créer Middleware Traefik si nécessaire

### Phase 4: Validation

**Checklist DoD:** `docs/checklist.md`

**Commandes obligatoires:**

```bash
# 1. Lint YAML
yamllint -c yamllint-config.yml apps/[CATEGORY]/[APP_NAME]/**/*.yaml

# 2. Build dev overlay
kustomize build apps/[CATEGORY]/[APP_NAME]/overlays/dev

# 3. Build prod overlay
kustomize build apps/[CATEGORY]/[APP_NAME]/overlays/prod

# 4. Vérifier kinds diff (CRITIQUE)
kustomize build apps/[CATEGORY]/[APP_NAME]/overlays/prod | grep '^kind:' | sort
```

**Validation kinds diff:**
- Comparer kinds before/after modifications kustomization.yaml
- Missing kind = ressource silencieusement droppée (régression)
- Exemple: it-tools ingress removal après changement kustomization.yaml

**Validation spécifique stateful:**
- Buckets S3 créés (dev + prod)
- Secrets Infisical créés et validés
- Litestream config correcte (path DB, excludes)

**Validation spécifique complex:**
- Tous les Deployments présents dans kinds diff
- NetworkPolicy pour chaque Deployment
- ServiceMonitor targetPort correct

### Phase 5: ArgoCD Application

**Stateless apps:**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: [APP_NAME]-[ENV]
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: main
    path: apps/[CATEGORY]/[APP_NAME]/overlays/[ENV]
  destination:
    server: https://kubernetes.default.svc
    namespace: [NAMESPACE_NAME]
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**Stateless-Helm apps (multi-source):**

```yaml
spec:
  sources:
    # Source 1: Helm chart
    - repoURL: https://[CHART_REPO_URL]
      chart: [CHART_NAME]
      targetRevision: [CHART_VERSION]
      helm:
        valueFiles:
          - $values/apps/[CATEGORY]/[APP_NAME]/base/values.yaml
    # Source 2: Values reference
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: main
      ref: values
    # Source 3: Kustomize overlays
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: main
      path: apps/[CATEGORY]/[APP_NAME]/overlays/[ENV]
```

## 🎨 Templates Détaillés

### Template 1: Stateless Native

**Quand utiliser:**
- Application sans état
- Image Docker publique/interne
- Pas de Helm chart
- Configuration simple

**Structure:**
```
stateless-native/
├── base/
│   ├── deployment.yaml      # 1 container, probes HTTP
│   ├── service.yaml         # ClusterIP port 80
│   ├── namespace.yaml       # PSS baseline
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml   # replicas=0, components/base
    └── prod/
        └── kustomization.yaml   # gold-maturity, resources, PDB
```

**Features:**
- Probes: liveness, readiness, startup (HTTP)
- Security: runAsNonRoot, fsGroup, drop ALL
- Sizing: V-nano default (25m/32Mi)
- Priority: vixens-medium
- Tolerations: control-plane

**Guide complet:** `templates/stateless-native/README.md`

**Exemples:** whoami, stirling-pdf

### Template 2: Stateless Helm

**Quand utiliser:**
- Helm chart disponible (officiel/communauté)
- Besoin de patches Kustomize spécifiques Vixens
- Annotations non supportées par chart (deploymentAnnotations)

**Structure:**
```
stateless-helm/
├── base/
│   ├── values.yaml           # Helm values (sizing, annotations)
│   └── kustomization.yaml    # Namespace only
└── overlays/
    ├── dev/
    │   └── kustomization.yaml   # replicas=0, components
    └── prod/
        └── kustomization.yaml   # JSON patches (gold-maturity)
```

**Features:**
- Values.yaml: podLabels, podAnnotations, securityContext
- Patches JSON: Deployment metadata annotations
- Sizing: B-nano default (10m/16Mi) - ultra-léger
- ArgoCD: multi-source config (3 sources)

**Guide complet:** `templates/stateless-helm/README.md`

**Exemples:** it-tools, dashy

### Template 3: Stateful

**Quand utiliser:**
- Données persistantes (SQLite)
- Backup temps-réel requis (Litestream → S3)
- Sync fichiers config (Config-Syncer)
- Résilience critique (diamond tier)

**Structure:**
```
stateful/
├── base/
│   ├── deployment.yaml           # 3 containers + 2 init
│   ├── pvc.yaml                  # RWO 5Gi retain
│   ├── litestream-config.yaml    # ConfigMap
│   ├── infisical-secret.yaml     # S3 credentials
│   ├── service.yaml              # http + metrics
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml    # replicas=0, envSlug=dev
    └── prod/
        └── kustomization.yaml    # gold-maturity, envSlug=prod
```

**Architecture:**

**Init containers:**
1. `fix-permissions` - chown 1000:1000 /data
2. `restore-config` - rclone copy S3 → /data (config files)

**Main containers:**
1. `app` - Application principale (SQLite in /data)
2. `litestream` - Backup continu SQLite → S3 (WAL + snapshots)
3. `config-syncer` - Sync /data → S3 every 60s (excludes DB)

**Backup strategy:**
- Litestream: WAL segments 1s, snapshots 1h, retention 24h
- Config-Syncer: sync 60s, excludes: `*.sqlite3*`, `*.log`, cache
- Recovery: automatic on pod restart (init containers)

**Guide complet:** `templates/stateful/README.md` (372 lignes)

**Exemples:** vaultwarden (référence complète), trilium, firefly-iii

### Template 4: Complex

**Quand utiliser:**
- Multi-composants (server + workers + jobs)
- NetworkPolicy custom (pas juste Traefik)
- ServiceMonitor Prometheus
- VPA recommendations
- Dependencies externes (Redis, PostgreSQL)

**Structure:**
```
complex/
├── base/
│   ├── deployment-server.yaml    # Main workload
│   ├── deployment-worker.yaml    # Background workers
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── infisical-secret.yaml
│   ├── networkpolicy-*.yaml      # Per-deployment policies
│   ├── servicemonitor.yaml       # Prometheus scraping
│   ├── vpa.yaml                  # VPA recommendations
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml    # replicas=0 ALL deployments
    └── prod/
        └── kustomization.yaml    # gold-maturity components
```

**Patterns clés:**

**1. Multi-Deployment:**
- Label separation: `component: server` vs `component: worker`
- Shared secrets: single InfisicalSecret, multiple refs
- Overlay patches: target `kind: Deployment` sans name filter

**2. NetworkPolicy per Deployment:**
- Server: Traefik ingress + full egress
- Worker: no ingress, full egress
- podSelector: matchLabels spécifiques

**3. ServiceMonitor:**
- Label selector commun (`monitoring: "true"`)
- Multiple endpoints possibles
- Interval: 60s default

**4. VPA:**
- updateMode: Off (recommendations only)
- Analyse tous les containers
- Multi-Deployment support

**Guide complet:** `templates/complex/README.md` (353 lignes)

**Exemples:** authentik (server + worker), homeassistant (5 containers), postgresql-shared (CNPG)

## 📐 Patterns Réutilisables

### Pattern: NetworkPolicy

**Fichier:** `patterns/networkpolicy.yaml`

**Usage:** Apps exposées via Traefik avec egress illimité

**Configuration:**
```yaml
ingress:
  - from:
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: traefik
    ports:
      - protocol: TCP
        port: 80  # Ajuster selon app
egress:
  - {}  # Allow all (DNS, inter-app, external APIs)
```

**Personnalisation:**
- Changer `port:` si pas HTTP standard (ex: 9000)
- Ajouter ingress pour inter-pod (worker → server)
- Restreindre egress si production hardening

**Exemples:** authentik, vaultwarden, homeassistant

### Pattern: ServiceMonitor

**Fichier:** `patterns/servicemonitor.yaml`

**Usage:** Apps exposant metrics Prometheus

**Configuration:**
```yaml
endpoints:
  - port: metrics
    targetPort: 9090  # Ajuster
    path: /metrics    # Ajuster
    interval: 60s     # 10s pour metrics fréquents
```

**Personnalisation:**
- `targetPort`: port metrics réel (9090 Litestream, 9300 authentik)
- `path`: endpoint metrics (`/metrics`, `/api/metrics`)
- `interval`: 10s (fréquent), 60s (standard), 300s (relaxed)

**Validation:**
```bash
# Vérifier Prometheus scrape
kubectl get servicemonitor -n [NAMESPACE] [APP] -o yaml
kubectl logs -n monitoring prometheus-xxx | grep [APP]
```

**Exemples:** vaultwarden (Litestream), authentik, homeassistant

### Pattern: VPA

**Fichier:** `patterns/vpa.yaml`

**Usage:** Recommendations ressources (NOT auto-apply)

**Configuration:**
```yaml
spec:
  targetRef:
    kind: Deployment  # Or StatefulSet
    name: [APP_NAME]
  updatePolicy:
    updateMode: "Off"  # Recommendations only
```

**Personnalisation:**
- `kind:` StatefulSet si applicable
- `updateMode: Auto` si auto-update voulu (NOT recommended)

**Validation:**
```bash
# Attendre 24h minimum pour recommendations
kubectl get vpa -n [NAMESPACE] [APP] -o yaml | grep -A 10 recommendation
```

**Exemples:** homeassistant, authentik

## ✅ DoD (Definition of Done)

**Checklist complète:** `docs/checklist.md`

### Checklist Minimal (Toutes apps)

- [ ] Template copié et adapté
- [ ] Tous placeholders remplacés
- [ ] Probes adaptées (`/health`, `/healthz`, `/api/health`)
- [ ] Sizing labels corrects (tier approprié)
- [ ] yamllint passe (no errors)
- [ ] kustomize build dev passe
- [ ] kustomize build prod passe
- [ ] Kinds diff OK (compare before/after)

### Checklist Stateful (En plus)

- [ ] Buckets S3 créés (dev + prod)
- [ ] Secrets Infisical créés et testés
- [ ] Litestream config validé (path DB correct)
- [ ] Config-Syncer excludes corrects (no DB sync)
- [ ] PVC size approprié (5Gi default)
- [ ] Recovery test (delete pod, restore works)

### Checklist Complex (En plus)

- [ ] Tous Deployments dans kinds diff
- [ ] NetworkPolicy pour chaque Deployment
- [ ] ServiceMonitor si metrics (port correct)
- [ ] VPA si recommendations needed
- [ ] Dependencies externes configurées (Redis, PostgreSQL)
- [ ] Middleware Traefik si forward-auth

## 📊 Standards Vixens (Auto-appliqués)

### Sizing v2 (Kyverno)

**Labels:** `vixens.io/sizing.[container-name]: [TIER]`

**Tiers disponibles:**

| Tier | CPU | Memory | Usage |
|------|-----|--------|-------|
| B-nano | 10m | 16Mi | Ultra-léger (Helm apps) |
| V-nano | 25m | 32Mi | Standard stateless |
| V-small | 50m | 64Mi | Apps moyennes |
| V-medium | 100m | 128Mi | Apps lourdes |
| V-large | 200m | 256Mi | Apps très lourdes |
| G-nano | 50m | 64Mi | Init containers |
| G-small | 100m | 128Mi | Init containers lourds |
| SB-* | Custom | Custom | Snowflake apps |

**Auto-apply:** Kyverno policy converts labels → requests/limits

### Priority Classes

**Classes disponibles:**
- `vixens-critical` - Apps critiques (auth, ingress)
- `vixens-high` - Apps importantes (monitoring, backup)
- `vixens-medium` - Apps standard (most apps)
- `vixens-low` - Apps non-critiques (test, tools)

**Auto-apply:** Components `priority/[CLASS]` patch Deployment

### Security Context

**Mandatory:**
- `runAsNonRoot: true`
- `fsGroup: 1000` (or app-specific)
- `capabilities: drop: [ALL]`
- `seccompProfile: RuntimeDefault`
- `allowPrivilegeEscalation: false`

**Auto-apply:** Components `gold-maturity` enforce

### Probes

**Mandatory:**
- `livenessProbe` - Detect crashed containers
- `readinessProbe` - Detect not-ready containers
- `startupProbe` - Slow-starting apps (failureThreshold: 15)

**Defaults:** Components `probes/basic` provide

### Overlays

**Dev:**
- `replicas: 0` - Disabled by default
- `components/base` - Labels standards
- `components/probes/basic` - Probes defaults

**Prod:**
- `components/gold-maturity` - Probes + security
- `components/resources` - Auto-sizing
- `components/poddisruptionbudget/0` - PDB maxUnavailable=0
- `components/priority/[CLASS]` - Priority class
- `components/revision-history-limit` - revisionHistoryLimit: 3

## 🔧 Troubleshooting Common Issues

### Issue: yamllint errors

**Symptômes:** Erreurs indentation, trailing spaces, line length

**Fix:**
```bash
# Vérifier règles
cat yamllint-config.yml

# Fixer indentation (2 spaces)
# Fixer trailing spaces (trim)
# Wrapper lignes longues (<120 chars)
```

### Issue: kustomize build fails

**Symptômes:** Error loading resources, component not found

**Causes:**
- Path relatifs incorrects (`../../base` vs `../base`)
- Components path incorrect (count `../`)
- Resources non listées dans kustomization.yaml

**Debug:**
```bash
# Vérifier paths
cat kustomization.yaml | grep -E 'resources|components'

# Vérifier composants existent
ls ../../../../_shared/components/gold-maturity/
```

### Issue: Kinds diff reveals missing resources

**Symptômes:** Ingress présent avant, absent après modification kustomization.yaml

**Cause:** Resource silencieusement droppée (typo, wrong path)

**Prevention:**
```bash
# Before change
kustomize build overlays/prod | grep '^kind:' | sort > before.txt

# After change
kustomize build overlays/prod | grep '^kind:' | sort > after.txt

# Compare
diff before.txt after.txt
```

**Fix:** Ajouter resource manquante dans kustomization.yaml

### Issue: ArgoCD sync fails (multi-source Helm)

**Symptômes:** Error: valueFiles not found

**Causes:**
- Missing `ref: values` dans source 2
- Wrong `$values/...` path (relatif au repo)
- Sources dans mauvais ordre

**Fix:**
```yaml
sources:
  # 1. Helm chart + valueFiles reference
  - repoURL: https://helm-repo
    chart: chart-name
    helm:
      valueFiles:
        - $values/apps/[CATEGORY]/[APP]/base/values.yaml
  # 2. Values reference (MUST be second)
  - repoURL: https://github.com/charchess/vixens.git
    targetRevision: main
    ref: values  # CRITIQUE
  # 3. Kustomize overlays
  - repoURL: https://github.com/charchess/vixens.git
    path: apps/[CATEGORY]/[APP]/overlays/[ENV]
```

### Issue: Litestream backup fails

**Symptômes:** Logs Litestream erreur S3, no WAL segments

**Causes:**
- Secrets Infisical manquants/incorrects
- Bucket S3 n'existe pas
- Endpoint S3 inaccessible
- DB path incorrect in litestream.yml

**Debug:**
```bash
# Vérifier secrets
kubectl get secret -n [NAMESPACE] [APP]-secrets -o yaml

# Vérifier bucket
mc ls minio/vixens-[ENV]-[APP]

# Vérifier endpoint
curl http://192.168.111.69:9000

# Vérifier DB path
kubectl exec -n [NAMESPACE] [POD] -c litestream -- ls -la /data/
```

**Fix:**
- Créer secrets Infisical
- Créer bucket: `mc mb minio/vixens-[ENV]-[APP]`
- Vérifier DB path dans litestream-config.yaml

### Issue: NetworkPolicy blocks traffic

**Symptômes:** Worker cannot reach Server, timeout

**Cause:** Ingress rule manquante sur Server NetworkPolicy

**Fix:**
```yaml
# networkpolicy-server.yaml
ingress:
  - from:
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: traefik
  - from:
      - podSelector:
          matchLabels:
            app: [APP]-worker  # Allow worker
    ports:
      - protocol: TCP
        port: 8080
```

### Issue: ServiceMonitor no targets

**Symptômes:** Prometheus dashboard shows 0/0 targets

**Causes:**
- Label selector mismatch (Service vs ServiceMonitor)
- Namespace selector mismatch
- Port name incorrect

**Debug:**
```bash
# Vérifier ServiceMonitor
kubectl get servicemonitor -n [NAMESPACE] [APP] -o yaml

# Vérifier Service labels
kubectl get service -n [NAMESPACE] [APP] -o yaml | grep -A 5 labels

# Vérifier port name matches
kubectl get service -n [NAMESPACE] [APP] -o yaml | grep -A 5 ports
```

**Fix:** Align labels + port names

### Issue: VPA no recommendations

**Symptômes:** `kubectl get vpa` shows empty status

**Causes:**
- Pod runtime < 24h (metrics insuffisants)
- updateMode incorrect
- targetRef incorrect (wrong Deployment name)

**Fix:** Attendre 24h minimum, vérifier targetRef

### Issue: Multiple Deployments only 1 deployed

**Symptômes:** `kubectl get deployment` shows 1/2

**Cause:** Overlay patch targets single Deployment

**Fix:**
```yaml
# overlays/[ENV]/kustomization.yaml
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: any  # Wildcard - match ALL
      spec:
        replicas: 0
    target:
      kind: Deployment  # No name filter
```

## 🎓 Agent Best Practices

### 1. Always Read Template README

Chaque template a un README.md détaillé:
- `templates/stateless-native/README.md` (170 lignes)
- `templates/stateless-helm/README.md` (228 lignes)
- `templates/stateful/README.md` (372 lignes)
- `templates/complex/README.md` (353 lignes)

**Lire avant de commencer l'implémentation.**

### 2. Validate Early, Validate Often

```bash
# Après chaque modification majeure
yamllint -c yamllint-config.yml apps/[CATEGORY]/[APP]/**/*.yaml
kustomize build apps/[CATEGORY]/[APP]/overlays/dev
kustomize build apps/[CATEGORY]/[APP]/overlays/prod
```

### 3. Use Examples as Reference

Exemples réels = vérité terrain:
- `examples/whoami/` - Stateless simple et propre
- `examples/vaultwarden/` - Stateful complet (RÉFÉRENCE)
- `examples/authentik/` - Complex multi-composants

**Copy patterns directement des exemples.**

### 4. Document Your Decisions

Si deviation from template:
- Commenter dans YAML (`# Why: ...`)
- Noter dans `docs/applications/[CATEGORY]/[APP].md`
- Expliquer dans commit message

### 5. Test Recovery (Stateful Apps)

Avant de marquer done:
```bash
# Delete pod, watch restore
kubectl delete pod -n [NAMESPACE] [POD]
kubectl logs -n [NAMESPACE] [NEW_POD] -c litestream --follow
# Should see: "restoring snapshot" + "replaying WAL"
```

## 📖 Documentation Hierarchy

**Pour AI agents, ordre de lecture:**

1. **SKILL.md** (ce fichier) - Vue d'ensemble + workflow agent
2. **docs/decision-matrix.md** - Choisir template
3. **templates/[TYPE]/README.md** - Guide template spécifique
4. **docs/checklist.md** - Validation DoD
5. **examples/[APP]/** - Référence réelle

**Pour utilisateurs humains:**

1. **README.md** - Quick start + structure
2. **docs/decision-matrix.md** - Décision template
3. **templates/[TYPE]/README.md** - Implémentation
4. **patterns/*.yaml** - Patterns réutilisables
5. **examples/[APP]/** - Exemples réels

## 🔗 Liens Utiles

**Vixens docs (external):**
- `~/vixens/docs/reference/app-golden-standard.md` - Standards officiels
- `~/vixens/docs/guides/backup-restore-pattern.md` - Backup patterns
- `~/vixens/docs/guides/pattern-config-syncer.md` - Config-Syncer usage
- `~/vixens/docs/adr/014-litestream-backup-profiles-and-recovery-patterns.md` - Litestream ADR

**Infra Vixens:**
- Minio: http://192.168.111.69:9000 (S3 buckets)
- Infisical: http://192.168.111.69:8085 (secrets management)
- ArgoCD: https://argocd.truxonline.com (GitOps)
- Traefik: https://traefik.truxonline.com (ingress)

**Sizing tiers (Kyverno):**
- Policy: `~/vixens/apps/_shared/kyverno-policies/sizing-v2.yaml`
- Tiers definition: voir policy ConfigMap

## 🤖 Agent Integration

**Claude Code (MCP):**
```python
# Use skill directly
skill(name="vixens-app-patterns", user_message="Deploy new app X")

# Load skill for task
task(
    category="quick",
    load_skills=["vixens-app-patterns"],
    description="Deploy app X to Vixens cluster",
    prompt="..."
)
```

**Triggers automatiques:**
- Détection pattern "deploy * vixens"
- Détection fichier path `apps/[CATEGORY]/[APP]/`
- Détection keywords: kustomize, argocd, litestream, sizing

## 📊 Metrics & Success

**Skill basé sur:**
- 42 applications analysées (39 Deployments + 3 StatefulSets)
- 292 sizing label occurrences (widespread adoption)
- 100% base/overlays adoption
- 4 templates validés
- 3 patterns réutilisables
- 3 exemples réels

**Validation:**
- Tous templates = apps réelles fonctionnelles en production
- DoD checklist = exhaustive (20 points)
- Troubleshooting = issues réels rencontrés + fixes

## 🎯 Next Steps (Agent Workflow)

Après avoir lu ce SKILL.md:

1. **Ask user questions** (Phase 1: Discovery)
2. **Consult decision matrix** (`docs/decision-matrix.md`)
3. **Read template README** (`templates/[TYPE]/README.md`)
4. **Copy template & customize**
5. **Validate against checklist** (`docs/checklist.md`)
6. **Create ArgoCD Application**
7. **Monitor deployment & troubleshoot**

**Rappel:** NEVER skip validation steps (yamllint, kustomize build, kinds diff).

---

**Skill ready to use.** 🚀
