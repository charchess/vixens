# Vixens Application Patterns Skill

Skill OpenCode pour déploiement d'applications Kubernetes sur cluster Vixens homelab avec GitOps (ArgoCD + Kustomize).

## Quick Start

```bash
# 1. Lire SKILL.md (vue d'ensemble + decision matrix)
cat SKILL.md

# 2. Choisir le template approprié
# - Stateless native: apps/99-test/whoami/
# - Stateless Helm: apps/70-tools/it-tools/
# - Stateful: apps/60-services/vaultwarden/
# - Complex: apps/03-security/authentik/

# 3. Copier le template
cp -r templates/stateless-native apps/CATEGORY/APP_NAME

# 4. Remplacer placeholders
# APP_NAME, NAMESPACE_NAME, IMAGE_REGISTRY/IMAGE_NAME:IMAGE_TAG, etc.

# 5. Valider
yamllint -c yamllint-config.yml apps/CATEGORY/APP_NAME/**/*.yaml
kustomize build apps/CATEGORY/APP_NAME/overlays/dev
kustomize build apps/CATEGORY/APP_NAME/overlays/prod

# 6. Vérifier kinds diff (detect missing resources)
kustomize build apps/CATEGORY/APP_NAME/overlays/prod | grep '^kind:' | sort
```

## Structure

```
vixens-app-patterns/
├── SKILL.md                    # 📚 Main documentation (READ THIS FIRST)
├── README.md                   # 🚀 This file (Quick Start)
├── docs/
│   ├── decision-matrix.md      # 🎯 Choisir le bon template
│   └── checklist.md            # ✅ DoD validation checklist
├── templates/
│   ├── stateless-native/       # Template apps sans état (K8s natif)
│   ├── stateless-helm/         # Template apps sans état (Helm chart)
│   ├── stateful/               # Template apps avec SQLite + Litestream
│   └── complex/                # Template apps multi-composants
├── patterns/
│   ├── networkpolicy.yaml      # Pattern NetworkPolicy (Traefik ingress)
│   ├── servicemonitor.yaml     # Pattern ServiceMonitor (Prometheus)
│   └── vpa.yaml                # Pattern VerticalPodAutoscaler
└── examples/
    ├── whoami/                 # Exemple stateless native (simple)
    ├── vaultwarden/            # Exemple stateful (complet)
    └── authentik/              # Exemple complex (2 Deployments)
```

## Templates

### 1. Stateless Native (`templates/stateless-native/`)

**Pour:** Apps sans état, images Docker, pas de Helm

**Exemples:** whoami, stirling-pdf

**Contient:**
- deployment.yaml (probes, sizing, securityContext)
- service.yaml (ClusterIP)
- namespace.yaml
- kustomization.yaml (base)
- overlays/dev/ (replicas=0)
- overlays/prod/ (gold-maturity)
- README.md (guide complet)

### 2. Stateless Helm (`templates/stateless-helm/`)

**Pour:** Apps sans état via Helm chart + patches Kustomize

**Exemples:** it-tools, dashy

**Contient:**
- values.yaml (Helm values)
- kustomization.yaml (base)
- overlays/dev/ (replicas=0)
- overlays/prod/ (gold-maturity patches JSON)
- README.md (guide + ArgoCD multi-source config)

### 3. Stateful (`templates/stateful/`)

**Pour:** Apps avec SQLite + backup S3 (Litestream) + sync config

**Exemples:** vaultwarden, trilium, firefly-iii

**Contient:**
- deployment.yaml (3 containers: app + litestream + config-syncer)
- pvc.yaml (RWO volume)
- litestream-config.yaml (ConfigMap)
- infisical-secret.yaml (S3 credentials)
- service.yaml (http + metrics)
- kustomization.yaml (base)
- overlays/dev/ (replicas=0, envSlug=dev)
- overlays/prod/ (gold-maturity, envSlug=prod)
- README.md (guide complet backup/restore)

### 4. Complex (`templates/complex/`)

**Pour:** Apps multi-composants avec NetworkPolicy, ServiceMonitor, VPA

**Exemples:** authentik, homeassistant, postgresql-shared

**Contient:**
- README.md (patterns + exemples réels)
- Référence vers `patterns/` (NetworkPolicy, ServiceMonitor, VPA)
- Référence vers `examples/` (authentik = 2 Deployments)

## Patterns

Patterns réutilisables à copier dans vos apps:

### NetworkPolicy (`patterns/networkpolicy.yaml`)
- Ingress: Allow Traefik namespace
- Egress: Allow all (DNS, inter-app, external APIs)

### ServiceMonitor (`patterns/servicemonitor.yaml`)
- Prometheus scraping
- Port: metrics (9090)
- Interval: 60s

### VPA (`patterns/vpa.yaml`)
- Resource recommendations
- updateMode: Off (pas de auto-update)

## Examples

Exemples réels (copies) pour référence:

### whoami (`examples/whoami/`)
- Stateless native
- Simple (1 container)
- NetworkPolicy + PDB
- 4 overlays (dev/staging/test/prod)

### vaultwarden (`examples/vaultwarden/`)
- Stateful complet
- 3 containers (app + litestream + config-syncer)
- Infisical secrets + S3 backup
- ServiceMonitor + NetworkPolicy
- **RÉFÉRENCE COMPLÈTE** pour apps stateful

### authentik (`examples/authentik/`)
- Complex multi-composants
- 2 Deployments (server + worker)
- NetworkPolicy séparées
- ServiceMonitor
- Middleware Traefik (forward-auth)
- Infisical secrets (Redis + PostgreSQL)

## Decision Matrix

**Flowchart simplifié** (voir `docs/decision-matrix.md` pour détails):

```
1. État persistant?
   ├─ Non → 2
   └─ Oui → Stateful template

2. Helm chart disponible?
   ├─ Non → 3
   └─ Oui → Stateless Helm template

3. Multi-composants (>1 Deployment)?
   ├─ Non → Stateless Native template
   └─ Oui → Complex template
```

## DoD Checklist

Avant de déployer:
- [ ] Template copié et adapté
- [ ] Tous les placeholders remplacés
- [ ] Probes adaptées à l'application
- [ ] Sizing labels corrects
- [ ] yamllint passe
- [ ] kustomize build dev passe
- [ ] kustomize build prod passe
- [ ] Kinds diff OK (aucune ressource manquante)

**Stateful apps:**
- [ ] Infisical secrets créés (dev + prod)
- [ ] S3 buckets créés (dev + prod)
- [ ] Litestream config validé
- [ ] Config-Syncer excludes corrects

**Complex apps:**
- [ ] NetworkPolicy pour chaque Deployment
- [ ] ServiceMonitor si metrics
- [ ] VPA si recommendations
- [ ] Dependencies externes (Redis, PostgreSQL)

## Standards Vixens (appliqués automatiquement)

### Sizing v2 (Kyverno)
- Labels: `vixens.io/sizing.*` → auto-apply requests/limits
- Tiers: B-nano, V-nano/small/medium/large, G-nano/small/medium/large/xl

### Priority Classes
- vixens-critical, vixens-high, vixens-medium, vixens-low
- Component applique automatiquement

### Security Context
- runAsNonRoot, fsGroup, capabilities drop ALL
- Gold-maturity component enforce

### Probes
- Liveness, Readiness, Startup
- Probes component applique defaults

### Overlays
- Dev: replicas=0 (disabled)
- Prod: gold-maturity + resources + PDB + priority

## Troubleshooting

### yamllint errors
- Vérifier indentation (2 spaces)
- Vérifier trailing spaces
- Vérifier line length (<120 chars)

### kustomize build fails
- Vérifier paths relatifs (../../base)
- Vérifier components paths (../../../../_shared/components/*)
- Vérifier resources listées dans kustomization.yaml

### Kinds diff révèle resources manquantes
- Vérifier kustomization.yaml (resources:)
- Vérifier patches (target: correct?)
- Vérifier components (appliqués?)

### ArgoCD sync fails
- Multi-source config incorrect (Helm apps)
- Namespace pas créé (syncOptions: CreateNamespace=true)
- Secrets manquants (Infisical)

## Support

**Documentation:**
- `SKILL.md` - Main documentation (READ FIRST)
- `docs/decision-matrix.md` - Template selection guide
- `docs/checklist.md` - DoD validation checklist
- `templates/*/README.md` - Template-specific guides

**Exemples:**
- `examples/whoami/` - Stateless simple
- `examples/vaultwarden/` - Stateful complet (RÉFÉRENCE)
- `examples/authentik/` - Complex multi-composants

**Vixens docs:**
- `~/vixens/docs/reference/app-golden-standard.md` - Standards officiels
- `~/vixens/docs/guides/backup-restore-pattern.md` - Backup patterns
- `~/vixens/docs/guides/pattern-config-syncer.md` - Config-Syncer usage
- `~/vixens/docs/adr/014-litestream-backup-profiles-and-recovery-patterns.md` - Litestream ADR

## Contribute

Skill projet-local dans `~/vixens/.opencode/skills/vixens-app-patterns/`.

Pour améliorer:
1. Éditer les fichiers (SKILL.md, templates/*, docs/*)
2. Tester avec une vraie application
3. Commit dans repo vixens

**Skill version:** 1.0.0 (2026-03-10)
