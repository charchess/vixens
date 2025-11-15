# Vixens Project Conventions

Documentation des conventions et standards établis pour le projet vixens homelab.

**Dernière mise à jour**: 2025-11-15
**Version**: 2.0 (Post-Phase 2 DRY Optimization)

---

## Table des Matières

1. [ArgoCD Applications](#argocd-applications)
2. [Helm Values](#helm-values)
3. [Kustomize Overlays](#kustomize-overlays)
4. [Git Workflow](#git-workflow)
5. [Naming Conventions](#naming-conventions)
6. [Documentation](#documentation)

---

## ArgoCD Applications

### Structure de Fichiers

```
argocd/
├── base/
│   ├── argocd-install.yaml      # ArgoCD self-management
│   ├── root-app.yaml            # App-of-Apps root
│   └── app-templates/           # Templates de référence (non déployés)
│       ├── git-app-template.yaml
│       ├── helm-app-template.yaml
│       └── README.md
└── overlays/
    ├── dev/
    │   ├── apps/                # Applications par environnement
    │   │   ├── traefik.yaml
    │   │   ├── cert-manager.yaml
    │   │   └── ...
    │   ├── env-config.yaml      # Config centralisée (build-time only)
    │   └── kustomization.yaml
    ├── test/
    ├── staging/
    └── prod/
```

### Conventions ArgoCD Applications

**✅ Bonnes Pratiques :**
- Un fichier par application : `{app-name}.yaml`
- Localisation : `argocd/overlays/{env}/apps/`
- Finalizer requis : `resources-finalizer.argocd.argoproj.io`
- Auto-sync : `automated: {prune: true, selfHeal: true}`
- Namespace creation : `syncOptions: [CreateNamespace=true]`

**❌ À Éviter :**
- ~~Inline Helm values~~ (utiliser external values)
- ~~ServerSideApply pour Helm~~ (cause des sync stuck)
- ~~Duplication des specs ArgoCD~~ (référer templates si besoin)

### Helm Applications - Multiple Sources Pattern

Pour les applications Helm, utiliser le pattern **multiple sources** :

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  sources:
    # Source 1: Helm chart
    - repoURL: https://helm.example.com/
      chart: app-name
      targetRevision: "v1.0.0"
      helm:
        valueFiles:
          - $values/apps/app-name/values/common.yaml
          - $values/apps/app-name/values/dev.yaml

    # Source 2: Values from Git
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: dev
      ref: values
```

**Avantages** :
- DRY : values partagés entre environnements
- Versionné dans Git
- Facile à tester localement (`helm template -f ...`)

---

## Helm Values

### Structure DRY (Don't Repeat Yourself)

```
apps/
└── {app-name}/
    └── values/
        ├── common.yaml       # Partagé entre TOUS les environnements
        ├── dev.yaml          # Surcharges dev uniquement
        ├── test.yaml         # Surcharges test
        ├── staging.yaml      # Surcharges staging
        ├── prod.yaml         # Surcharges production
        └── README.md         # Documentation des values
```

### Conventions Values Files

**common.yaml** - Configuration partagée :
```yaml
---
# Common {app-name} configuration for ALL environments

# Section comments avec ===
# ============================================================================
# Section Name
# ============================================================================
config:
  key: value
```

**{env}.yaml** - Overrides par environnement :
```yaml
---
# {Env} environment overrides for {app-name}
# VLAN {vlan-id} (192.168.{vlan}.0/24)

# ============================================================================
# {Env}-Specific Settings
# ============================================================================
resources:
  requests: ...
```

### Exemples de Séparation

**Common** (tous les envs) :
- Providers, API config, ports
- Tolerations control-plane
- Configuration fonctionnelle de base

**Dev/Test** :
- Resources faibles (VM)
- Log level DEBUG/INFO
- Replicas: 1

**Staging** :
- Resources moyens
- Replicas: 2
- Similar to prod

**Prod** :
- Resources élevés
- Replicas: 3+
- Monitoring (Prometheus)
- Pod Disruption Budgets

---

## Kustomize Overlays

### Structure Standard

```
apps/
└── {app-name}/
    ├── base/                 # Configuration de base
    │   ├── deployment.yaml
    │   ├── service.yaml
    │   └── kustomization.yaml
    └── overlays/
        ├── dev/              # Patches dev
        ├── test/
        ├── staging/
        └── prod/
```

### Conventions Kustomize

**Base** :
- Configuration minimale fonctionnelle
- Pas de valeurs spécifiques à l'environnement

**Overlays** :
- Patches pour différences environnement
- Images tags, replicas, resources
- ConfigMaps/Secrets refs

---

## Git Workflow

### Branches

| Branch | Usage | Protection |
|--------|-------|-----------|
| `main` | Production | ⚠️ Protected |
| `dev` | Development | Auto-deploy to dev |
| `test` | Testing | Auto-deploy to test |
| `staging` | Pre-prod | Auto-deploy to staging |
| `feature/*` | Features | PR to dev |
| `fix/*` | Bugfixes | PR to dev/main |

### Commit Messages

Format : `type(scope): description`

**Types** :
- `feat`: Nouvelle fonctionnalité
- `fix`: Correction de bug
- `refactor`: Refactoring sans changement fonctionnel
- `docs`: Documentation
- `chore`: Tâches maintenance

**Exemples** :
```
feat(argocd): Add Traefik ingress controller
fix(cilium): Correct L2 announcements configuration
refactor(terraform): Implement DRY module structure
docs(readme): Update installation instructions
```

### Pull Requests

**Titre** : Même format que commits
**Corps** :
```markdown
## Summary
Brief description

## Changes
- List of changes

## Testing
- Test plan

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

---

## Naming Conventions

### Kubernetes Resources

**Format général** : `{app-name}-{component}-{env}` (si multi-env dans même cluster)

**Exemples** :
- Deployment: `traefik`, `cert-manager`
- Service: `traefik`, `whoami`
- Namespace: `traefik`, `cert-manager`, `whoami`
- ConfigMap: `{app}-config`
- Secret: `{app}-credentials`

### Hostnames/DNS

**Format** : `{service}.{env}.truxonline.com`

**Exemples** :
- Dev: `whoami.dev.truxonline.com`
- Test: `whoami.test.truxonline.com`
- Staging: `whoami.stg.truxonline.com`
- Prod: `whoami.truxonline.com`

### LoadBalancer IPs (Cilium)

**Pools par environnement** :

| Env | VLAN | Pool Range | Usage |
|-----|------|------------|-------|
| Dev | 208 | .70-.89 | .70-.79: assigned, .80-.89: auto |
| Test | 209 | .70-.89 | Same pattern |
| Staging | 210 | .70-.89 | Same pattern |
| Prod | 201 | .70-.89 | Same pattern |

**Assignation** :
- `.70` : Traefik LoadBalancer
- `.71` : ArgoCD LoadBalancer
- `.72-79` : Services assignés manuellement
- `.80-89` : Auto-assignés par Cilium IPAM

---

## Documentation

### READMEs Requis

Chaque composant majeur doit avoir un README.md :

**apps/{app-name}/values/README.md** :
- Structure des values
- Différences par environnement
- Exemples de test local
- Liens documentation upstream

**argocd/base/app-templates/README.md** :
- Usage des templates
- Quand les utiliser
- Exemples

### Documentation Projet

**CLAUDE.md** : Instructions pour Claude Code
**README.md** : Quick start pour utilisateurs
**CONVENTIONS.md** : Ce fichier
**docs/** : Documentation détaillée

### ADRs (Architecture Decision Records)

Localisation : `docs/adr/`

**Format** :
```markdown
# ADR-{number}: {title}

## Status
Accepted / Proposed / Deprecated

## Context
Why this decision?

## Decision
What was decided?

## Consequences
Positive and negative impacts
```

---

## Maintenance

### Reviews

Ce document doit être revu :
- Après chaque phase majeure de refactoring
- Quand une nouvelle convention est établie
- Au moins trimestriellement

### Contributions

Suggestions de conventions :
1. Créer issue GitHub
2. Proposer dans PR
3. Discuter avant adoption

---

## Voir Aussi

- [CLAUDE.md](./CLAUDE.md) - Instructions pour Claude Code
- [README.md](./README.md) - Quick start guide
- [docs/ROADMAP.md](./docs/ROADMAP.md) - Sprint roadmap
- [docs/adr/](./docs/adr/) - Architecture decisions
