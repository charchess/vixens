# GitOps Workflow Vixens

## Principes GitOps

### Git as Source of Truth

- **Déclaratif** : L'état désiré du cluster est décrit en YAML dans Git
- **Versionné** : Tout changement est tracé (commits, PRs)
- **Immutable** : Pas de `kubectl apply` manuel (sauf bootstrap)
- **Auditable** : Git log = historique complet des modifications

### Continuous Reconciliation

ArgoCD surveille le repo Git et :
- **Détecte les drifts** (état cluster ≠ état Git)
- **Auto-synchronise** (syncPolicy.automated)
- **Auto-heal** (correction drift automatique)

---

## Architecture Git

### Structure Repository

```
vixens/
├── terraform/                    # Phase 1 : Infrastructure as Code
│   ├── modules/
│   │   └── talos/               # Module réutilisable Talos cluster
│   │       ├── main.tf
│   │       ├── variables.tf
│   │       ├── outputs.tf
│   │       ├── providers.tf
│   │       └── versions.tf
│   └── environments/
│       ├── dev/
│       │   ├── main.tf          # Appel module talos
│       │   ├── terraform.tfvars # Variables spécifiques dev
│       │   ├── providers.tf
│       │   └── versions.tf
│       ├── test/
│       ├── staging/
│       └── prod/
│
├── argocd/                      # Phase 2 : ArgoCD self-management
│   ├── base/
│   │   ├── kustomization.yaml
│   │   ├── namespace.yaml
│   │   ├── argocd-install.yaml  # Application Helm ArgoCD
│   │   └── root-app.yaml        # App-of-Apps racine
│   └── overlays/
│       ├── dev/
│       │   ├── kustomization.yaml
│       │   └── patches/
│       │       └── root-app-patch.yaml
│       ├── test/
│       ├── staging/
│       └── prod/
│
├── apps/                        # Applications infrastructure
│   ├── metallb/
│   │   ├── base/
│   │   │   ├── kustomization.yaml
│   │   │   ├── helm-release.yaml      # ArgoCD Application (Helm)
│   │   │   └── ipaddresspool.yaml     # Manifestes natifs
│   │   └── overlays/
│   │       ├── dev/
│   │       │   ├── kustomization.yaml
│   │       │   └── ipaddresspool-dev.yaml  # Patch VLAN 208
│   │       ├── test/
│   │       │   └── ipaddresspool-test.yaml # Patch VLAN 209
│   │       └── prod/
│   │
│   ├── traefik/
│   │   ├── base/
│   │   │   ├── kustomization.yaml
│   │   │   ├── helm-release.yaml
│   │   │   └── values.yaml
│   │   └── overlays/
│   │       ├── dev/
│   │       ├── test/
│   │       └── prod/
│   │
│   ├── cert-manager/
│   ├── synology-csi/
│   ├── authelia/
│   └── monitoring/
│
├── docs/                        # Documentation
│   ├── architecture/
│   ├── runbooks/
│   ├── operations/
│   └── adr/
│
└── .github/
    └── workflows/
        ├── validate-terraform.yaml
        ├── validate-kustomize.yaml
        └── promote.yaml
```

---

## Stratégie Branches

### Mapping Branches ↔ Clusters

| Branch    | Cluster | Auto-Sync | Protection | Description                    |
|-----------|---------|-----------|------------|--------------------------------|
| `dev`     | Dev     | ✅ Oui    | ❌ Non     | Développement, tests rapides   |
| `test`    | Test    | ✅ Oui    | ⚠️ Oui     | Validation pré-staging         |
| `staging` | Staging | ✅ Oui    | ✅ Oui     | Pré-production                 |
| `main`    | Prod    | ⚠️ Manuel | ✅ Oui     | Production (sync manuel)       |

### Protection Branches

**`test` / `staging` / `main`** :
- Require PR pour merge
- Require 1 approval (soi-même OK pour homelab)
- Require CI checks pass (validation Terraform + Kustomize)
- No force push

---

## Workflow de Promotion

### 1. Développement (Branch `dev`)

```bash
# Modifier une app
vim apps/traefik/overlays/dev/values.yaml

# Commit + push
git add apps/traefik/
git commit -m "feat(traefik): enable access logs"
git push origin dev
```

**ArgoCD Dev** sync automatiquement en < 3 minutes.

**Validation** :
```bash
argocd app get traefik --kubeconfig kubeconfig-dev
kubectl logs -n traefik -l app=traefik --tail=50
```

---

### 2. Promotion Dev → Test

**Créer Pull Request** :
```bash
gh pr create --base test --head dev \
  --title "chore: promote dev to test" \
  --body "Changes:
- Traefik access logs enabled
- MetalLB pool extended

Tests done:
- Ingress routing validated
- LoadBalancer IPs assigned
"
```

**CI Checks** (GitHub Actions) :
- ✅ Terraform validate (tous environments)
- ✅ Kustomize build (tous overlays)
- ✅ YAML lint
- ✅ ArgoCD dry-run (validation apps)

**Review + Merge** :
```bash
gh pr merge 42 --merge
```

**ArgoCD Test** sync automatiquement la branch `test`.

---

### 3. Promotion Test → Staging

**Similaire Dev → Test** :
```bash
gh pr create --base staging --head test \
  --title "chore: promote test to staging"
```

**Validation plus stricte** :
- Tests manuels complets
- Smoke tests automatisés (si disponibles)
- Documentation mise à jour

---

### 4. Promotion Staging → Prod

**PR Staging → Main** :
```bash
gh pr create --base main --head staging \
  --title "release: deploy to production" \
  --body "Release Notes:
## Changes
- Feature X deployed
- Bug Y fixed

## Validation
- Staging smoke tests passed
- Performance tests OK
- Security scan clean
"
```

**Sync MANUEL** (pas d'auto-sync prod) :
```bash
argocd app sync traefik --kubeconfig kubeconfig-prod
argocd app wait traefik --health --kubeconfig kubeconfig-prod
```

**Rollback si problème** :
```bash
git revert <commit-sha>
git push origin main
argocd app sync traefik --kubeconfig kubeconfig-prod
```

---

## Kustomize Overlays

### Pattern Base + Overlay

**Base** : Configuration commune à tous les environnements
**Overlay** : Patches spécifiques par environnement

### Exemple : MetalLB

**`apps/metallb/base/kustomization.yaml`** :
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - helm-release.yaml
  - ipaddresspool.yaml
```

**`apps/metallb/base/ipaddresspool.yaml`** :
```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: assigned-pool
  namespace: metallb-system
spec:
  addresses:
    - 192.168.XXX.70-192.168.XXX.79  # Template, patched par overlay
```

**`apps/metallb/overlays/dev/kustomization.yaml`** :
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - ../../base

patchesStrategicMerge:
  - ipaddresspool-dev.yaml
```

**`apps/metallb/overlays/dev/ipaddresspool-dev.yaml`** :
```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: assigned-pool
  namespace: metallb-system
spec:
  addresses:
    - 192.168.208.70-192.168.208.79  # VLAN 208 (dev)
```

**`apps/metallb/overlays/test/ipaddresspool-test.yaml`** :
```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: assigned-pool
  namespace: metallb-system
spec:
  addresses:
    - 192.168.209.70-192.168.209.79  # VLAN 209 (test)
```

---

## ArgoCD Applications

### Structure Application

**`apps/metallb/base/helm-release.yaml`** :
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: metallb
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: vixens
  source:
    repoURL: https://github.com/charchess/vixens
    targetRevision: dev  # Patched par overlay (test, staging, main)
    path: apps/metallb/overlays/dev  # Patched par overlay
  destination:
    server: https://kubernetes.default.svc
    namespace: metallb-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    syncWave: 0  # MetalLB avant Traefik (wave 1)
```

### Sync Waves

Ordre de déploiement via annotations :

| App             | Sync Wave | Raison                              |
|-----------------|-----------|-------------------------------------|
| cert-manager    | 0         | Aucune dépendance                   |
| metallb         | 0         | Aucune dépendance                   |
| traefik         | 1         | Dépend de MetalLB (IP pool)         |
| synology-csi    | 1         | Aucune dépendance apps              |
| authelia        | 2         | Dépend de Traefik (middleware)      |
| monitoring      | 2         | Dépend de services (ServiceMonitor) |

**Annotation** :
```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
```

---

## App-of-Apps Pattern

### Root Application

**`argocd/base/root-app.yaml`** :
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: vixens-root
  namespace: argocd
spec:
  project: vixens
  source:
    repoURL: https://github.com/charchess/vixens
    targetRevision: dev  # Patched par overlay
    path: apps
    directory:
      recurse: true
      include: '*/overlays/dev/*'  # Only dev overlays
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**Overlay Dev** :
```yaml
# argocd/overlays/dev/patches/root-app-patch.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: vixens-root
spec:
  source:
    targetRevision: dev
    directory:
      include: '*/overlays/dev/*'
```

---

## CI/CD Validation

### GitHub Actions Workflow

**`.github/workflows/validate.yaml`** :
```yaml
name: Validate Infrastructure

on:
  pull_request:
    branches: [test, staging, main]

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: hashicorp/setup-terraform@v2

      - name: Terraform Format Check
        run: terraform fmt -check -recursive

      - name: Terraform Validate
        run: |
          cd terraform/environments/dev
          terraform init -backend=false
          terraform validate

  kustomize:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Kustomize
        run: |
          curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
          sudo mv kustomize /usr/local/bin/

      - name: Build Overlays
        run: |
          for env in dev test staging prod; do
            echo "Building $env overlays..."
            for app in apps/*/overlays/$env; do
              if [ -d "$app" ]; then
                kustomize build $app > /dev/null
              fi
            done
          done

  yaml-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: ibiqlik/action-yamllint@v3
```

---

## Troubleshooting GitOps

### App Out of Sync

```bash
# Voir différences
argocd app diff metallb

# Forcer sync
argocd app sync metallb

# Voir logs sync
argocd app logs metallb --follow
```

### Sync Loop (selfHeal en boucle)

**Cause** : Ressource modifiée par controller externe (ex: MetalLB modifie IPAddressPool)

**Solution** : Ignorer champs dans ArgoCD
```yaml
spec:
  ignoreDifferences:
    - group: metallb.io
      kind: IPAddressPool
      jsonPointers:
        - /status
```

### Kustomize Build Error

```bash
# Valider localement
kustomize build apps/metallb/overlays/dev

# Debug
kustomize build apps/metallb/overlays/dev --enable-alpha-plugins
```

---

## Best Practices

### 1. Commits Conventionnels

```
feat(traefik): add prometheus metrics
fix(metallb): correct IP pool range
chore(docs): update runbook
refactor(kustomize): simplify overlays structure
```

### 2. PRs Descriptives

- **Titre clair** : Qu'est-ce qui change ?
- **Body détaillé** : Pourquoi ? Tests effectués ?
- **Checklist** : Validation pre-merge

### 3. Git Tags pour Releases

```bash
git tag -a v1.0.0 -m "Release: Production ready"
git push origin v1.0.0
```

### 4. Secrets Management (Future)

**Phase 1-2** : Secrets en clair dans Git (OK pour homelab privé)

**Phase 3** : SOPS + Age encryption
```bash
sops --encrypt --age <age-public-key> secret.yaml > secret.enc.yaml
```

ArgoCD déchiffre automatiquement via plugin SOPS.

---

## Changelog

| Date       | Version | Changement                          |
|------------|---------|-------------------------------------|
| 2025-10-30 | 1.0     | Workflow GitOps initial             |
