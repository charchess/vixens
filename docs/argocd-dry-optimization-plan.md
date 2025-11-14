# ArgoCD & Apps DRY Optimization Plan

**Date**: 2025-11-14
**Status**: Planning
**Impact**: ~40% codebase reduction (~1,220 lines)
**Maintainability**: Significant improvement

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Current State Analysis](#current-state-analysis)
3. [Phase 1: ArgoCD Application Templates](#phase-1-argocd-application-templates)
4. [Phase 2: Helm Values Externalization](#phase-2-helm-values-externalization)
5. [Phase 3: Hostname Standardization](#phase-3-hostname-standardization)
6. [Phase 4: Cilium LB IP Pool Standardization](#phase-4-cilium-lb-ip-pool-standardization)
7. [Phase 5: Cleanup & Best Practices](#phase-5-cleanup--best-practices)
8. [Implementation Roadmap](#implementation-roadmap)
9. [Validation & Testing](#validation--testing)
10. [Rollback Strategy](#rollback-strategy)

---

## Executive Summary

### Current Issues

- **50 ArgoCD Application files** for only **14 unique applications** (4 environments × ~13 apps)
- **96% duplication** in ArgoCD app definitions (only `targetRevision` and `path` differ)
- **240+ lines** of inline Helm values duplicated across environments (Traefik)
- **20+ ingress files** with only hostname differences
- **No base templates** for ArgoCD applications
- **Unused files** still present in repository

### Optimization Goals

- Reduce codebase by **~1,220 lines** (~40%)
- Implement **DRY principles** for ArgoCD applications
- Standardize **environment-specific configurations**
- Improve **maintainability** and reduce human error
- Align with **GitOps best practices**

### Priority Matrix

| Phase | Impact | Effort | Priority | Lines Saved |
|-------|--------|--------|----------|-------------|
| 1. ArgoCD App Templates | Very High | Medium | **P0** | ~600 |
| 2. Helm Values Externalization | High | Low | **P1** | ~240 |
| 5. Cleanup Unused Files | Low | Very Low | **P1** | ~100 |
| 3. Hostname Standardization | Medium | Medium | P2 | ~200 |
| 4. Cilium LB Standardization | Medium | Low | P2 | ~80 |

---

## Current State Analysis

### ArgoCD Applications Duplication

```bash
# Current structure
argocd/overlays/
├── dev/
│   ├── whoami-app.yaml           # 23 lines
│   ├── traefik-app.yaml          # 66 lines
│   ├── cert-manager-app.yaml     # 27 lines
│   └── ... (11 more apps)
├── test/
│   ├── whoami-app.yaml           # 23 lines (96% identical to dev)
│   ├── traefik-app.yaml          # 66 lines (96% identical to dev)
│   └── ...
├── staging/ (same pattern)
└── prod/ (same pattern)

# Statistics
Total app files: 50
Unique apps: 14
Duplication factor: 3.57x
Only 2 fields differ: targetRevision, path
```

### Example: whoami-app.yaml Duplication

```yaml
# argocd/overlays/dev/whoami-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: whoami
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev        # ONLY THIS DIFFERS
    path: apps/whoami/overlays/dev  # AND THIS
  destination:
    server: https://kubernetes.default.svc
    namespace: whoami
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**Comparison**:
```bash
$ diff argocd/overlays/dev/whoami-app.yaml argocd/overlays/test/whoami-app.yaml
13c13
<     targetRevision: dev
---
>     targetRevision: test
14c14
<     path: apps/whoami/overlays/dev
---
>     path: apps/whoami/overlays/test
```

---

## Phase 1: ArgoCD Application Templates

**Goal**: Eliminate 72% of ArgoCD application files using Kustomize templates and replacements.

### Problem Statement

- 50 app files with 96% identical content
- Adding a new app requires creating 4 files (dev, test, staging, prod)
- Changes to app structure require updates in 4 places
- High risk of inconsistencies

### Solution Architecture

Use **Kustomize bases** + **replacements** to generate environment-specific apps from templates.

#### Step 1.1: Create Base Templates

```bash
# New structure
argocd/
├── base/
│   ├── app-templates/
│   │   ├── git-app-template.yaml      # Template for Git-sourced apps
│   │   ├── helm-app-template.yaml     # Template for Helm-sourced apps
│   │   └── kustomization.yaml
│   ├── namespace.yaml
│   ├── argocd-install.yaml
│   └── root-app.yaml.tpl
└── overlays/
    ├── dev/
    │   ├── env-config.yaml            # Environment variables
    │   ├── apps/                      # App-specific patches
    │   │   ├── whoami.yaml
    │   │   ├── traefik.yaml
    │   │   └── ...
    │   └── kustomization.yaml
    └── ...
```

#### Step 1.2: Git App Template

**File**: `argocd/base/app-templates/git-app-template.yaml`

```yaml
---
# Generic template for Git-sourced Kustomize apps
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: PLACEHOLDER_APP_NAME
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: PLACEHOLDER_ENV
    path: apps/PLACEHOLDER_APP_NAME/overlays/PLACEHOLDER_ENV
  destination:
    server: https://kubernetes.default.svc
    namespace: PLACEHOLDER_NAMESPACE
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

#### Step 1.3: Helm App Template

**File**: `argocd/base/app-templates/helm-app-template.yaml`

```yaml
---
# Generic template for Helm-sourced apps
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: PLACEHOLDER_APP_NAME
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: PLACEHOLDER_HELM_REPO
    chart: PLACEHOLDER_CHART_NAME
    targetRevision: PLACEHOLDER_CHART_VERSION
    helm:
      valueFiles:
        - $values/apps/PLACEHOLDER_APP_NAME/overlays/PLACEHOLDER_ENV/values.yaml
  sources:
    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: PLACEHOLDER_ENV
      ref: values
  destination:
    server: https://kubernetes.default.svc
    namespace: PLACEHOLDER_NAMESPACE
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

#### Step 1.4: Environment Config

**File**: `argocd/overlays/dev/env-config.yaml`

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: env-config
  namespace: argocd
data:
  environment: dev
  git_branch: dev
  domain_suffix: dev.truxonline.com
  vlan_id: "208"
```

#### Step 1.5: App-Specific Patches

**File**: `argocd/overlays/dev/apps/whoami.yaml`

```yaml
---
# Minimal patch for whoami app
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: whoami
spec:
  destination:
    namespace: whoami
```

**File**: `argocd/overlays/dev/apps/traefik.yaml`

```yaml
---
# Patch for Traefik Helm app
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: traefik
spec:
  source:
    repoURL: https://helm.traefik.io/traefik
    chart: traefik
    targetRevision: "v25.0.0"
  destination:
    namespace: traefik
```

#### Step 1.6: Kustomization with Replacements

**File**: `argocd/overlays/dev/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: argocd

# Include base templates
resources:
  - ../../base/app-templates/git-app-template.yaml
  - env-config.yaml

# Apply app-specific patches
patches:
  - path: apps/whoami.yaml
    target:
      kind: Application
      name: PLACEHOLDER_APP_NAME
  - path: apps/traefik.yaml
    target:
      kind: Application
      name: PLACEHOLDER_APP_NAME

# Replace placeholders with environment-specific values
replacements:
  # Replace PLACEHOLDER_ENV with actual environment
  - source:
      kind: ConfigMap
      name: env-config
      fieldPath: data.environment
    targets:
      - select:
          kind: Application
        fieldPaths:
          - spec.source.targetRevision
        options:
          create: true

  # Replace PLACEHOLDER_ENV in path
  - source:
      kind: ConfigMap
      name: env-config
      fieldPath: data.environment
    targets:
      - select:
          kind: Application
        fieldPaths:
          - spec.source.path
        options:
          delimiter: "/"
          index: 3  # apps/APP_NAME/overlays/[ENV]

  # Replace PLACEHOLDER_APP_NAME with actual app name from metadata
  - source:
      kind: Application
      fieldPath: metadata.name
    targets:
      - select:
          kind: Application
        fieldPaths:
          - spec.source.path
        options:
          delimiter: "/"
          index: 1  # apps/[APP_NAME]/overlays/ENV
```

### Alternative: ConfigMapGenerator Pattern

**Simpler approach** for teams less familiar with Kustomize replacements:

**File**: `argocd/overlays/dev/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: argocd

resources:
  - apps/whoami-app.yaml
  - apps/traefik-app.yaml
  - apps/cert-manager-app.yaml
  - apps/argocd-app.yaml
  - apps/cilium-lb-app.yaml

# Generate environment-specific values
configMapGenerator:
  - name: env-vars
    literals:
      - ENVIRONMENT=dev
      - GIT_BRANCH=dev
      - DOMAIN=dev.truxonline.com

# Use vars to inject into resources
vars:
  - name: ENVIRONMENT
    objref:
      kind: ConfigMap
      name: env-vars
      apiVersion: v1
    fieldref:
      fieldpath: data.ENVIRONMENT
  - name: GIT_BRANCH
    objref:
      kind: ConfigMap
      name: env-vars
      apiVersion: v1
    fieldref:
      fieldpath: data.GIT_BRANCH
```

**File**: `argocd/overlays/dev/apps/whoami-app.yaml`

```yaml
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: whoami
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: $(ENVIRONMENT)
    path: apps/whoami/overlays/$(ENVIRONMENT)
  destination:
    server: https://kubernetes.default.svc
    namespace: whoami
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Implementation Steps

1. **Create base templates**:
   ```bash
   mkdir -p argocd/base/app-templates
   # Create git-app-template.yaml and helm-app-template.yaml
   ```

2. **Create env-config for dev**:
   ```bash
   # Create argocd/overlays/dev/env-config.yaml
   ```

3. **Create app patches**:
   ```bash
   mkdir -p argocd/overlays/dev/apps
   # Create minimal patches for each app
   ```

4. **Update kustomization.yaml**:
   ```bash
   # Add replacements configuration
   ```

5. **Test with kustomize build**:
   ```bash
   kustomize build argocd/overlays/dev --enable-helm
   ```

6. **Validate generated manifests**:
   ```bash
   kustomize build argocd/overlays/dev | kubectl apply --dry-run=client -f -
   ```

7. **Repeat for test, staging, prod**:
   ```bash
   # Copy pattern to other environments
   cp -r argocd/overlays/dev argocd/overlays/test
   # Update env-config.yaml with test values
   ```

8. **Remove old files**:
   ```bash
   # After validation, remove individual *-app.yaml files
   git rm argocd/overlays/dev/whoami-app.yaml
   git rm argocd/overlays/dev/traefik-app.yaml
   # ... etc
   ```

### Benefits

- **50 files → ~20 files** (60% reduction)
- **Single source of truth** for app structure
- **Environment config in one place** (env-config.yaml)
- **Easier to add new apps** (just create a small patch)
- **Consistent across all environments**
- **Type-safe with Kustomize validation**

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Kustomize replacements complexity | Medium | Use simpler ConfigMapGenerator pattern as alternative |
| ArgoCD doesn't support Kustomize replacements | High | Test with `kustomize build` first, ArgoCD uses Kustomize internally |
| Breaking existing apps | High | Implement in parallel, validate before switching |
| Team unfamiliarity | Medium | Document patterns, provide training |

### Testing Strategy

```bash
# 1. Build and compare output
kustomize build argocd/overlays/dev --enable-helm > /tmp/new-dev.yaml
kubectl get applications -n argocd -o yaml > /tmp/old-dev.yaml
diff /tmp/old-dev.yaml /tmp/new-dev.yaml

# 2. Dry-run apply
kustomize build argocd/overlays/dev | kubectl apply --dry-run=server -f -

# 3. Apply to test environment first
cd argocd/overlays/test
kustomize build . | kubectl apply -f -

# 4. Monitor ArgoCD sync status
kubectl get applications -n argocd -w

# 5. Rollback if needed
kubectl apply -f argocd/overlays/test/ (old files)
```

---

## Phase 2: Helm Values Externalization

**Goal**: Extract inline Helm values from ArgoCD apps to dedicated values files.

### Problem Statement

Current Traefik app definition:

```yaml
# argocd/overlays/dev/traefik-app.yaml (66 lines)
spec:
  source:
    repoURL: https://helm.traefik.io/traefik
    chart: traefik
    targetRevision: "v25.0.0"
    helm:
      values: |                        # 50+ lines of inline YAML
        providers:
          kubernetesCRD:
            enabled: true
        api:
          dashboard: true
        tolerations:
          - key: "node-role.kubernetes.io/control-plane"
            operator: "Exists"
            effect: "NoSchedule"
        service:
          type: LoadBalancer
          annotations:
            io.cilium/lb-ipam-ips: "192.168.208.70"
        # ... 40 more lines
```

**Issues**:
- 66 lines × 4 environments = 264 lines total
- Inline YAML is hard to read and compare
- No reusability between environments
- Difficult to use tools like `helm diff`

### Solution Architecture

Extract values to dedicated files with inheritance:

```
apps/traefik/
├── values/
│   ├── common.yaml                  # Shared across all environments
│   ├── dev.yaml                     # Dev-specific (imports common)
│   ├── test.yaml                    # Test-specific
│   ├── staging.yaml
│   └── prod.yaml
└── overlays/
    └── dev/
        └── kustomization.yaml
```

### Step 2.1: Create Common Values

**File**: `apps/traefik/values/common.yaml`

```yaml
---
# Common Traefik Helm values for all environments
# Sourced from terraform/modules/shared/locals.tf

providers:
  kubernetesCRD:
    enabled: true
  kubernetesIngress:
    enabled: true

# Enable API and Dashboard
api:
  dashboard: true
  insecure: true

# Enable dashboard IngressRoute on web entrypoint
ingressRoute:
  dashboard:
    enabled: true
    entryPoints: ["web"]

# Control plane tolerations (from shared module)
tolerations:
  - key: "node-role.kubernetes.io/control-plane"
    operator: "Exists"
    effect: "NoSchedule"

# Service ports configuration
ports:
  web:
    port: 80
    expose: true
  websecure:
    port: 443
    expose: true
    tls:
      enabled: true
  traefik:
    port: 9000
    expose: false
    exposedPort: 9000

# Security contexts
securityContext:
  capabilities:
    drop: [ALL]
    add: [NET_BIND_SERVICE]
  readOnlyRootFilesystem: true
  runAsGroup: 65532
  runAsNonRoot: true
  runAsUser: 65532

podSecurityContext:
  fsGroup: 65532
  fsGroupChangePolicy: "OnRootMismatch"
```

### Step 2.2: Environment-Specific Values

**File**: `apps/traefik/values/dev.yaml`

```yaml
---
# Dev environment overrides for Traefik
# LoadBalancer IP from Cilium LB IPAM pool (VLAN 208)

service:
  type: LoadBalancer
  annotations:
    io.cilium/lb-ipam-ips: "192.168.208.70"

# Additional dev-specific settings
log:
  level: DEBUG

# Dev-specific resource limits (lower for VM environment)
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

**File**: `apps/traefik/values/test.yaml`

```yaml
---
# Test environment overrides
service:
  type: LoadBalancer
  annotations:
    io.cilium/lb-ipam-ips: "192.168.209.70"

log:
  level: INFO

resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

**File**: `apps/traefik/values/prod.yaml`

```yaml
---
# Production environment overrides
service:
  type: LoadBalancer
  annotations:
    io.cilium/lb-ipam-ips: "192.168.201.70"

log:
  level: WARN

# Production-grade resources
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "2000m"
    memory: "2Gi"

# Enable metrics for production monitoring
metrics:
  prometheus:
    enabled: true
    addEntryPointsLabels: true
    addServicesLabels: true
```

### Step 2.3: Update ArgoCD Application

**File**: `argocd/overlays/dev/apps/traefik.yaml` (new simplified version)

```yaml
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: traefik
  namespace: argocd
spec:
  project: default
  destination:
    server: https://kubernetes.default.svc
    namespace: traefik

  # Multiple sources pattern for Helm + values from Git
  sources:
    - repoURL: https://helm.traefik.io/traefik
      chart: traefik
      targetRevision: "v25.0.0"
      helm:
        valueFiles:
          - $values/apps/traefik/values/common.yaml
          - $values/apps/traefik/values/dev.yaml

    - repoURL: https://github.com/charchess/vixens.git
      targetRevision: dev
      ref: values

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

### Step 2.4: Alternative - Single Values File

If you prefer simpler structure:

**File**: `apps/traefik/overlays/dev/values.yaml`

```yaml
---
# Import common values (if tool supports it)
# Or just duplicate common values here

# From common.yaml
providers:
  kubernetesCRD:
    enabled: true
  # ... etc

# Environment-specific overrides
service:
  type: LoadBalancer
  annotations:
    io.cilium/lb-ipam-ips: "192.168.208.70"
```

**ArgoCD App**:

```yaml
sources:
  - repoURL: https://helm.traefik.io/traefik
    chart: traefik
    targetRevision: "v25.0.0"
    helm:
      valueFiles:
        - $values/apps/traefik/overlays/dev/values.yaml
  - repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    ref: values
```

### Implementation Steps

1. **Create values directory structure**:
   ```bash
   mkdir -p apps/traefik/values
   ```

2. **Extract common values**:
   ```bash
   # Copy from existing traefik-app.yaml
   # Remove environment-specific settings
   ```

3. **Create environment-specific files**:
   ```bash
   # Extract LoadBalancer IPs, resource limits, log levels
   ```

4. **Update ArgoCD application**:
   ```bash
   # Replace inline values with valueFiles reference
   ```

5. **Test with helm template**:
   ```bash
   helm template traefik traefik/traefik \
     -f apps/traefik/values/common.yaml \
     -f apps/traefik/values/dev.yaml \
     --version v25.0.0
   ```

6. **Apply and sync**:
   ```bash
   kubectl apply -f argocd/overlays/dev/apps/traefik.yaml
   argocd app sync traefik
   ```

7. **Validate Traefik is running**:
   ```bash
   kubectl get pods -n traefik
   kubectl get svc -n traefik
   ```

### Benefits

- **66 lines → 15 lines** per ArgoCD app (77% reduction)
- **Reusable common values** across environments
- **Easy comparison** between environments (diff values files)
- **Helm-native** approach (can use `helm diff`)
- **Better separation** of concerns

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Multiple sources not supported in old ArgoCD | High | Requires ArgoCD v2.6+ (we have v2.7.7) ✅ |
| Values file not found | High | Use ArgoCD UI to validate source paths |
| Merge conflicts in common.yaml | Medium | Use YAML merge keys or separate files |

---

## Phase 3: Hostname Standardization

**Goal**: Centralize hostname generation logic using environment variables.

### Problem Statement

Current state:
- `whoami.dev.truxonline.com` (dev)
- `whoami.test.truxonline.com` (test)
- `whoami.stg.truxonline.com` (staging)
- `whoami.truxonline.com` (prod)

**Issues**:
- Hostnames hardcoded in 20+ ingress files
- Pattern is `<app>.<env_suffix>.truxonline.com`
- No validation of naming convention
- Difficult to change domain or pattern

### Solution Architecture

Use Kustomize replacements to inject domain suffix:

```yaml
# apps/whoami/base/ingress.yaml
spec:
  rules:
    - host: whoami.ENVIRONMENT.truxonline.com  # Placeholder

# apps/whoami/overlays/dev/kustomization.yaml
replacements:
  - source:
      kind: ConfigMap
      name: env-config
      fieldPath: data.domain_suffix
    targets:
      - select:
          kind: Ingress
        fieldPaths:
          - spec.rules[*].host
```

### Step 3.1: Create Base Ingress Templates

**File**: `apps/whoami/base/ingress.yaml`

```yaml
---
# HTTP -> HTTPS redirect ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: whoami-redirect
  namespace: whoami
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
    traefik.ingress.kubernetes.io/router.middlewares: traefik-redirect-https@kubernetescrd
spec:
  ingressClassName: traefik
  rules:
    - host: whoami.ENV_SUFFIX.truxonline.com  # Will be replaced
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: whoami
                port:
                  number: 80
---
# HTTPS ingress with TLS
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: whoami
  namespace: whoami
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: traefik
  tls:
    - hosts:
        - whoami.ENV_SUFFIX.truxonline.com  # Will be replaced
      secretName: whoami-tls
  rules:
    - host: whoami.ENV_SUFFIX.truxonline.com  # Will be replaced
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: whoami
                port:
                  number: 80
```

### Step 3.2: Environment Config with Domain

**File**: `apps/whoami/overlays/dev/env-config.yaml`

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: env-config
  namespace: whoami
data:
  environment: dev
  domain_suffix: dev         # For "whoami.dev.truxonline.com"
  base_domain: truxonline.com
```

**File**: `apps/whoami/overlays/prod/env-config.yaml`

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: env-config
  namespace: whoami
data:
  environment: prod
  domain_suffix: ""          # Empty for prod = "whoami.truxonline.com"
  base_domain: truxonline.com
```

### Step 3.3: Kustomization with Replacements

**File**: `apps/whoami/overlays/dev/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: whoami

resources:
  - ../../base
  - env-config.yaml

# Replace ENV_SUFFIX with actual environment suffix
replacements:
  - source:
      kind: ConfigMap
      name: env-config
      fieldPath: data.domain_suffix
    targets:
      - select:
          kind: Ingress
        fieldPaths:
          - spec.rules[*].host
          - spec.tls[*].hosts[*]
        options:
          delimiter: "."
          index: 1  # Replace second segment: whoami.[ENV_SUFFIX].truxonline.com

# Alternative: Use sed-like pattern replacement
# replacements:
#   - source:
#       kind: ConfigMap
#       name: env-config
#       fieldPath: data.domain_suffix
#     targets:
#       - select:
#           kind: Ingress
#         fieldPaths:
#           - spec.rules[*].host
#         options:
#           pattern: ENV_SUFFIX
#           replacement: "$(domain_suffix)"
```

### Step 3.4: Advanced - Template Function

For more complex logic (like removing `.` for prod):

**File**: `apps/whoami/overlays/dev/hostname-transformer.yaml`

```yaml
---
apiVersion: builtin
kind: ReplacementTransformer
metadata:
  name: hostname-transformer
replacements:
  - source:
      kind: ConfigMap
      name: env-config
    targets:
      - select:
          kind: Ingress
        fieldPaths:
          - spec.rules[0].host
        template: |
          {{- if eq .domain_suffix "" -}}
          whoami.truxonline.com
          {{- else -}}
          whoami.{{ .domain_suffix }}.truxonline.com
          {{- end -}}
```

### Alternative: External-DNS Annotations

**Simpler approach** if using external-dns:

**File**: `apps/whoami/base/ingress.yaml`

```yaml
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: whoami
  namespace: whoami
  annotations:
    # external-dns will generate hostname automatically
    external-dns.alpha.kubernetes.io/hostname: "whoami.{{ .Values.environment }}.{{ .Values.baseDomain }}"
    external-dns.alpha.kubernetes.io/target: "traefik.{{ .Values.environment }}.{{ .Values.baseDomain }}"
spec:
  ingressClassName: traefik
  rules:
    - host: whoami  # Will be replaced by external-dns
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: whoami
                port:
                  number: 80
```

### Implementation Steps

1. **Update base ingress files**:
   ```bash
   # Replace hardcoded hostnames with ENV_SUFFIX placeholder
   sed -i 's/whoami\.dev\.truxonline\.com/whoami.ENV_SUFFIX.truxonline.com/g' apps/whoami/base/ingress.yaml
   ```

2. **Create env-config.yaml**:
   ```bash
   # For each overlay (dev, test, staging, prod)
   ```

3. **Add replacements to kustomization.yaml**:
   ```bash
   # Update all overlays with replacement configuration
   ```

4. **Test with kustomize build**:
   ```bash
   kustomize build apps/whoami/overlays/dev
   # Verify hostnames are correct
   ```

5. **Apply and validate**:
   ```bash
   kubectl apply -k apps/whoami/overlays/dev
   curl -I https://whoami.dev.truxonline.com
   ```

### Benefits

- **20+ ingress files** → base + small env configs
- **Centralized hostname logic**
- **Consistent naming** enforced by templates
- **Easy to change domain** (update base files only)
- **Validation** via Kustomize

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Replacement syntax complexity | Medium | Use simpler pattern-based replacements |
| Prod hostname different pattern | Low | Handle with conditional logic or separate base |
| DNS changes | High | Test in dev/test first, validate DNS before prod |

---

## Phase 4: Cilium LB IP Pool Standardization

**Goal**: Use base templates for IP pools with only ranges as patches.

### Problem Statement

Current structure (4 nearly identical files):

```yaml
# apps/cilium-lb/overlays/dev/ippool.yaml
apiVersion: cilium.io/v2alpha1
kind: CiliumLoadBalancerIPPool
metadata:
  name: vixens-dev-pool
spec:
  blocks:
    - start: "192.168.208.70"  # Only these
      stop: "192.168.208.89"   # lines differ
  serviceSelector:
    matchLabels: {}

# apps/cilium-lb/overlays/test/ippool.yaml
# ... same structure, only IPs differ (192.168.209.x)
```

### Solution Architecture

Create base template with overlay patches for IP ranges:

```
apps/cilium-lb/
├── base/
│   ├── ippool.yaml              # Template with placeholder
│   ├── l2policy.yaml            # Common L2 announcement policy
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   ├── ippool-patch.yaml    # Only IP ranges
    │   └── kustomization.yaml
    └── ...
```

### Step 4.1: Create Base Template

**File**: `apps/cilium-lb/base/ippool.yaml`

```yaml
---
apiVersion: cilium.io/v2alpha1
kind: CiliumLoadBalancerIPPool
metadata:
  name: vixens-pool  # Will be patched with environment prefix
spec:
  blocks:
    - start: "PLACEHOLDER_START_IP"
      stop: "PLACEHOLDER_STOP_IP"

  serviceSelector:
    matchLabels: {}
```

**File**: `apps/cilium-lb/base/l2policy.yaml`

```yaml
---
apiVersion: cilium.io/v2alpha1
kind: CiliumL2AnnouncementPolicy
metadata:
  name: l2-policy
spec:
  serviceSelector:
    matchLabels: {}

  nodeSelector:
    matchExpressions:
      - key: node-role.kubernetes.io/control-plane
        operator: Exists

  interfaces:
    - ^enx.*  # Match all ethernet interfaces

  externalIPs: true
  loadBalancerIPs: true
```

**File**: `apps/cilium-lb/base/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ippool.yaml
  - l2policy.yaml
```

### Step 4.2: Environment Patches

**File**: `apps/cilium-lb/overlays/dev/ippool-patch.yaml`

```yaml
---
apiVersion: cilium.io/v2alpha1
kind: CiliumLoadBalancerIPPool
metadata:
  name: vixens-dev-pool
spec:
  blocks:
    # Assigned pool for static IPs
    - start: "192.168.208.70"
      stop: "192.168.208.79"
    # Auto pool for dynamic allocation
    - start: "192.168.208.80"
      stop: "192.168.208.89"
```

**File**: `apps/cilium-lb/overlays/dev/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: kube-system

resources:
  - ../../base

patches:
  - path: ippool-patch.yaml
    target:
      kind: CiliumLoadBalancerIPPool
```

### Step 4.3: Use Replacements (Alternative)

**File**: `apps/cilium-lb/overlays/dev/vlan-config.yaml`

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: vlan-config
  namespace: kube-system
data:
  vlan_id: "208"
  ip_prefix: "192.168.208"
  pool_start: "70"
  pool_end: "89"
```

**File**: `apps/cilium-lb/overlays/dev/kustomization.yaml`

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: kube-system

resources:
  - ../../base
  - vlan-config.yaml

replacements:
  # Replace pool name with environment
  - source:
      kind: ConfigMap
      name: vlan-config
      fieldPath: metadata.namespace
    targets:
      - select:
          kind: CiliumLoadBalancerIPPool
        fieldPaths:
          - metadata.name
        options:
          pattern: "vixens-pool"
          replacement: "vixens-dev-pool"

  # Replace IP addresses
  - source:
      kind: ConfigMap
      name: vlan-config
      fieldPath: data.ip_prefix
    targets:
      - select:
          kind: CiliumLoadBalancerIPPool
        fieldPaths:
          - spec.blocks[*].start
          - spec.blocks[*].stop
        options:
          delimiter: "."
          index: 2  # Replace third octet
```

### Implementation Steps

1. **Move common config to base**:
   ```bash
   mv apps/cilium-lb/overlays/dev/l2policy.yaml apps/cilium-lb/base/
   ```

2. **Create ippool template**:
   ```bash
   # Extract common structure from existing files
   ```

3. **Create environment patches**:
   ```bash
   # Extract only IP ranges to patches
   ```

4. **Update kustomization files**:
   ```bash
   # Reference base and apply patches
   ```

5. **Test and apply**:
   ```bash
   kustomize build apps/cilium-lb/overlays/dev
   kubectl apply -k apps/cilium-lb/overlays/dev
   ```

6. **Validate IP allocation**:
   ```bash
   kubectl get ciliumpools
   kubectl get svc -A | grep LoadBalancer
   ```

### Benefits

- **Reduced duplication** (4 full files → 1 base + 4 small patches)
- **Clearer IP range management**
- **Easier to add new VLANs**
- **Consistent L2 policy** across environments

### Implementation Steps

1. **Create base structure**
2. **Extract common resources**
3. **Create environment-specific patches**
4. **Validate with kustomize build**
5. **Apply and test**

---

## Phase 5: Cleanup & Best Practices

**Goal**: Remove unused files, add validation, improve documentation.

### Step 5.1: Remove Unused Files

```bash
# Files to remove (8 files × 4 environments = 32 files)
find argocd/overlays -name "metallb-*.yaml.unused" -delete

# Or move to archive
mkdir -p archive/metallb
git mv argocd/overlays/*/metallb-*.yaml.unused archive/metallb/
```

### Step 5.2: Add GitHub Actions Validation

**File**: `.github/workflows/validate-argocd.yaml`

```yaml
---
name: Validate ArgoCD Applications

on:
  pull_request:
    paths:
      - 'argocd/**'
      - 'apps/**'
  push:
    branches: [dev, test, staging, main]

jobs:
  validate-kustomize:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [dev, test, staging, prod]

    steps:
      - uses: actions/checkout@v4

      - name: Setup Kustomize
        uses: imranismail/setup-kustomize@v2
        with:
          kustomize-version: "5.0.0"

      - name: Validate ArgoCD manifests
        run: |
          echo "Validating ${{ matrix.environment }} environment"
          kustomize build argocd/overlays/${{ matrix.environment }} --enable-helm

      - name: Validate Apps manifests
        run: |
          for app in apps/*/overlays/${{ matrix.environment }}; do
            if [ -d "$app" ]; then
              echo "Validating $app"
              kustomize build "$app"
            fi
          done

      - name: Dry-run with kubectl
        run: |
          kustomize build argocd/overlays/${{ matrix.environment }} \
            | kubectl apply --dry-run=client -f -

  validate-yaml:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install yamllint
        run: pip install yamllint

      - name: Lint YAML files
        run: |
          yamllint -c .yamllint.yaml argocd/ apps/
```

**File**: `.yamllint.yaml`

```yaml
---
extends: default

rules:
  line-length:
    max: 120
    level: warning
  indentation:
    spaces: 2
  document-start:
    present: true
  comments:
    min-spaces-from-content: 1
```

### Step 5.3: Add Pre-commit Hooks

**File**: `.pre-commit-config.yaml`

```yaml
---
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
      - id: check-merge-conflict

  - repo: https://github.com/adrienverge/yamllint
    rev: v1.33.0
    hooks:
      - id: yamllint
        args: [-c=.yamllint.yaml]

  - repo: local
    hooks:
      - id: validate-kustomize
        name: Validate Kustomize builds
        entry: bash -c 'for env in dev test staging prod; do kustomize build argocd/overlays/$env --enable-helm > /dev/null; done'
        language: system
        pass_filenames: false
```

### Step 5.4: Add Documentation

**File**: `docs/argocd-adding-new-app.md`

```markdown
# Adding a New Application to ArgoCD

## Quick Start

1. Create app structure in `apps/`:
   ```bash
   mkdir -p apps/myapp/{base,overlays/{dev,test,staging,prod}}
   ```

2. Create base resources:
   ```bash
   # apps/myapp/base/kustomization.yaml
   # apps/myapp/base/deployment.yaml
   # apps/myapp/base/service.yaml
   ```

3. Create ArgoCD application patch:
   ```bash
   # argocd/overlays/dev/apps/myapp.yaml
   ```

4. Add to kustomization:
   ```bash
   # Add to argocd/overlays/dev/kustomization.yaml
   ```

5. Test and apply:
   ```bash
   kustomize build argocd/overlays/dev | kubectl apply -f -
   ```

## See Also

- [ArgoCD App Templates](argocd-app-templates.md)
- [Hostname Conventions](hostname-conventions.md)
```

### Step 5.5: Add Helm Chart Verification

**Script**: `scripts/verify-helm-versions.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "=== Verifying Helm Chart Versions ==="
echo ""

# Check Traefik version
echo "Traefik:"
current=$(grep "targetRevision:" argocd/overlays/dev/apps/traefik.yaml | awk '{print $2}' | tr -d '"')
latest=$(helm search repo traefik/traefik --versions | head -2 | tail -1 | awk '{print $2}')
echo "  Current: $current"
echo "  Latest:  $latest"
if [ "$current" != "$latest" ]; then
  echo "  ⚠️  Update available"
fi
echo ""

# Check cert-manager
echo "cert-manager:"
current=$(grep "targetRevision:" argocd/overlays/dev/apps/cert-manager.yaml | awk '{print $2}' | tr -d '"')
latest=$(helm search repo jetstack/cert-manager --versions | head -2 | tail -1 | awk '{print $2}')
echo "  Current: $current"
echo "  Latest:  $latest"
if [ "$current" != "$latest" ]; then
  echo "  ⚠️  Update available"
fi
```

### Step 5.6: Add Consistency Verification

**Script**: `scripts/verify-env-consistency.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "=== Verifying Environment Consistency ==="
echo ""

# Check all environments have same apps
dev_apps=$(cd argocd/overlays/dev/apps && ls *.yaml | sort)
test_apps=$(cd argocd/overlays/test/apps && ls *.yaml | sort)
staging_apps=$(cd argocd/overlays/staging/apps && ls *.yaml | sort)
prod_apps=$(cd argocd/overlays/prod/apps && ls *.yaml | sort)

if [ "$dev_apps" != "$test_apps" ] || [ "$dev_apps" != "$staging_apps" ] || [ "$dev_apps" != "$prod_apps" ]; then
  echo "❌ Environments have different apps!"
  echo ""
  echo "Dev:     $(echo $dev_apps | wc -w) apps"
  echo "Test:    $(echo $test_apps | wc -w) apps"
  echo "Staging: $(echo $staging_apps | wc -w) apps"
  echo "Prod:    $(echo $prod_apps | wc -w) apps"
  exit 1
else
  echo "✅ All environments have same apps ($(echo $dev_apps | wc -w) apps)"
fi

echo ""
echo "=== Verifying IP Pool Ranges ==="
# Extract VLAN IDs from IP pools
dev_vlan=$(grep "start:" apps/cilium-lb/overlays/dev/ippool.yaml | head -1 | cut -d. -f3)
test_vlan=$(grep "start:" apps/cilium-lb/overlays/test/ippool.yaml | head -1 | cut -d. -f3)

echo "Dev VLAN:  208 (expected: 208) $([ "$dev_vlan" = "208" ] && echo "✅" || echo "❌")"
echo "Test VLAN: 209 (expected: 209) $([ "$test_vlan" = "209" ] && echo "✅" || echo "❌")"
```

### Implementation Steps

1. **Delete unused files**:
   ```bash
   git rm argocd/overlays/*/metallb-*.yaml.unused
   ```

2. **Add GitHub workflows**:
   ```bash
   cp .github/workflows/validate-kustomize.yaml .github/workflows/validate-argocd.yaml
   # Update paths
   ```

3. **Setup pre-commit**:
   ```bash
   pip install pre-commit
   pre-commit install
   ```

4. **Add documentation**:
   ```bash
   # Create docs/argocd-*.md files
   ```

5. **Create verification scripts**:
   ```bash
   chmod +x scripts/verify-*.sh
   ./scripts/verify-env-consistency.sh
   ```

---

## Implementation Roadmap

### Sprint 1: Foundation (P0 - Week 1)

**Goals**: Implement Phase 1 (ArgoCD app templates) for dev environment

**Tasks**:
1. Create `argocd/base/app-templates/` structure
2. Create Git app template
3. Create Helm app template
4. Create `argocd/overlays/dev/env-config.yaml`
5. Create app patches for 3 apps (whoami, argocd, cilium-lb)
6. Update `argocd/overlays/dev/kustomization.yaml`
7. Test with `kustomize build`
8. Apply to dev cluster
9. Validate apps sync correctly
10. Document pattern in `docs/argocd-app-templates.md`

**Success Criteria**:
- [ ] 3 apps migrated to template pattern
- [ ] Dev environment functional
- [ ] ArgoCD sync successful
- [ ] Documentation complete

**Rollback Plan**: Keep old `-app.yaml` files until validation complete

### Sprint 2: Expansion (P0 - Week 2)

**Goals**: Migrate all apps to template pattern in dev, replicate to test

**Tasks**:
1. Migrate remaining 11 apps to template pattern (dev)
2. Remove old `-app.yaml` files from dev
3. Replicate structure to test environment
4. Create `argocd/overlays/test/env-config.yaml`
5. Apply to test cluster
6. Validate both environments

**Success Criteria**:
- [ ] All 14 apps using templates (dev, test)
- [ ] Old files removed
- [ ] Both environments syncing

### Sprint 3: Helm Values (P1 - Week 3)

**Goals**: Implement Phase 2 (Helm values externalization)

**Tasks**:
1. Create `apps/traefik/values/` structure
2. Extract common values
3. Create environment-specific values
4. Update ArgoCD app to use multiple sources
5. Test and apply
6. Repeat for cert-manager if applicable

**Success Criteria**:
- [ ] Traefik using external values files
- [ ] Inline values removed from ArgoCD apps
- [ ] All environments functional

### Sprint 4: Cleanup (P1 - Week 3-4)

**Goals**: Implement Phase 5 (cleanup and validation)

**Tasks**:
1. Remove unused metallb files
2. Add GitHub Actions validation
3. Add pre-commit hooks
4. Add verification scripts
5. Create documentation

**Success Criteria**:
- [ ] No unused files in repo
- [ ] CI/CD validating all changes
- [ ] Documentation complete

### Sprint 5: Standardization (P2 - Week 5)

**Goals**: Implement Phases 3-4 (hostname and IP pool standardization)

**Tasks**:
1. Implement hostname standardization
2. Implement Cilium LB standardization
3. Replicate to staging/prod
4. Final validation

**Success Criteria**:
- [ ] All 4 environments using optimized structure
- [ ] Full test suite passing
- [ ] Team trained on new patterns

### Timeline Summary

| Week | Phase | Status | Deliverable |
|------|-------|--------|-------------|
| 1 | Foundation | Planned | ArgoCD templates (dev) |
| 2 | Expansion | Planned | All apps templated (dev, test) |
| 3 | Helm + Cleanup | Planned | External values + CI/CD |
| 4 | Cleanup cont. | Planned | Documentation |
| 5 | Standardization | Planned | All phases complete |

---

## Validation & Testing

### Pre-Implementation Checklist

- [ ] Backup current configuration
  ```bash
  kubectl get applications -n argocd -o yaml > backup-argocd-apps.yaml
  ```

- [ ] Create feature branch
  ```bash
  git checkout -b feature/argocd-dry-optimization
  ```

- [ ] Set up test environment
  ```bash
  # Ensure test cluster is available
  kubectl config use-context vixens-test
  ```

### During Implementation

#### Phase 1 Validation

```bash
# 1. Build and inspect
kustomize build argocd/overlays/dev --enable-helm > /tmp/new-apps.yaml
cat /tmp/new-apps.yaml | grep -A5 "kind: Application"

# 2. Compare with current state
kubectl get applications -n argocd -o yaml > /tmp/current-apps.yaml
diff /tmp/current-apps.yaml /tmp/new-apps.yaml

# 3. Dry-run apply
kubectl apply --dry-run=server -f /tmp/new-apps.yaml

# 4. Apply to test first
kubectl config use-context vixens-test
kustomize build argocd/overlays/test | kubectl apply -f -

# 5. Monitor sync status
watch kubectl get applications -n argocd

# 6. Check app health
argocd app list
argocd app get whoami
```

#### Phase 2 Validation

```bash
# 1. Template Helm chart locally
helm template traefik traefik/traefik \
  -f apps/traefik/values/common.yaml \
  -f apps/traefik/values/dev.yaml \
  --version v25.0.0 \
  > /tmp/traefik-templated.yaml

# 2. Compare with running config
kubectl get deployment traefik -n traefik -o yaml > /tmp/traefik-current.yaml
diff /tmp/traefik-current.yaml /tmp/traefik-templated.yaml

# 3. Apply and watch
kubectl apply -f argocd/overlays/dev/apps/traefik.yaml
kubectl rollout status deployment/traefik -n traefik -w

# 4. Validate service
kubectl get svc -n traefik
curl -I http://192.168.208.70
```

#### Phase 3 Validation

```bash
# 1. Build and check hostnames
kustomize build apps/whoami/overlays/dev | grep "host:"

# 2. Validate DNS
nslookup whoami.dev.truxonline.com

# 3. Test HTTPS
curl -I https://whoami.dev.truxonline.com
openssl s_client -connect whoami.dev.truxonline.com:443 -servername whoami.dev.truxonline.com
```

#### Phase 4 Validation

```bash
# 1. Check IP pools
kubectl get ciliumpools -A

# 2. Verify L2 announcements
kubectl get ciliuml2announcementpolicies -A

# 3. Test LoadBalancer allocation
kubectl get svc -A | grep LoadBalancer

# 4. Validate IP assignment
kubectl get svc traefik -n traefik -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

### Post-Implementation Verification

#### Automated Test Suite

**File**: `scripts/validate-all-environments.sh`

```bash
#!/bin/bash
set -euo pipefail

ENVIRONMENTS=(dev test staging prod)
FAILED=0

echo "=================================="
echo "VIXENS ARGOCD VALIDATION SUITE"
echo "=================================="
echo ""

for env in "${ENVIRONMENTS[@]}"; do
  echo "=== Validating $env environment ==="

  # 1. Kustomize build
  echo "  [1/5] Building manifests..."
  if ! kustomize build argocd/overlays/$env --enable-helm > /tmp/validate-$env.yaml 2>&1; then
    echo "  ❌ FAILED: Kustomize build"
    FAILED=1
    continue
  fi
  echo "  ✅ Manifests built successfully"

  # 2. YAML syntax
  echo "  [2/5] Validating YAML syntax..."
  if ! yamllint -c .yamllint.yaml argocd/overlays/$env apps/*/overlays/$env; then
    echo "  ❌ FAILED: YAML syntax errors"
    FAILED=1
    continue
  fi
  echo "  ✅ YAML syntax valid"

  # 3. Kubectl dry-run
  echo "  [3/5] Dry-run with kubectl..."
  if ! kubectl apply --dry-run=client -f /tmp/validate-$env.yaml > /dev/null 2>&1; then
    echo "  ❌ FAILED: Kubectl validation"
    FAILED=1
    continue
  fi
  echo "  ✅ Kubernetes validation passed"

  # 4. Check required apps
  echo "  [4/5] Checking required applications..."
  required_apps=(whoami traefik cert-manager argocd cilium-lb)
  for app in "${required_apps[@]}"; do
    if ! grep -q "name: $app" /tmp/validate-$env.yaml; then
      echo "  ❌ FAILED: Missing app $app"
      FAILED=1
    fi
  done
  echo "  ✅ All required apps present"

  # 5. Validate environment-specific values
  echo "  [5/5] Checking environment config..."
  if ! grep -q "targetRevision: $env" /tmp/validate-$env.yaml; then
    echo "  ❌ FAILED: Wrong targetRevision"
    FAILED=1
    continue
  fi
  echo "  ✅ Environment config correct"

  echo "  ✅ $env validation complete"
  echo ""
done

if [ $FAILED -eq 1 ]; then
  echo "❌ VALIDATION FAILED"
  exit 1
else
  echo "✅ ALL VALIDATIONS PASSED"
  exit 0
fi
```

**Usage**:
```bash
chmod +x scripts/validate-all-environments.sh
./scripts/validate-all-environments.sh
```

#### Live Cluster Validation

```bash
# 1. Check all ArgoCD apps are healthy
argocd app list | grep -v "Healthy.*Synced" && echo "❌ Unhealthy apps found" || echo "✅ All apps healthy"

# 2. Check all pods running
kubectl get pods -A | grep -v "Running\|Completed" && echo "❌ Non-running pods" || echo "✅ All pods running"

# 3. Validate Traefik ingress
for env in dev test; do
  echo "Testing $env..."
  curl -sSL -o /dev/null -w "%{http_code}" https://whoami.$env.truxonline.com | grep 200
done

# 4. Check Cilium LoadBalancer IPs
kubectl get svc -A -o jsonpath='{range .items[?(@.spec.type=="LoadBalancer")]}{.metadata.namespace}/{.metadata.name}: {.status.loadBalancer.ingress[0].ip}{"\n"}{end}'
```

---

## Rollback Strategy

### Preparation

Before starting any phase:

```bash
# 1. Tag current state
git tag -a pre-dry-optimization-$(date +%Y%m%d) -m "Before DRY optimization"
git push origin --tags

# 2. Backup ArgoCD applications
kubectl get applications -n argocd -o yaml > backups/argocd-apps-$(date +%Y%m%d).yaml

# 3. Backup app manifests
for env in dev test staging prod; do
  kustomize build argocd/overlays/$env > backups/argocd-$env-$(date +%Y%m%d).yaml
done

# 4. Export cluster state
kubectl get all,ingress,certificates -A -o yaml > backups/cluster-state-$(date +%Y%m%d).yaml
```

### Phase-by-Phase Rollback

#### Phase 1 Rollback

If ArgoCD app templates fail:

```bash
# 1. Checkout previous state
git checkout HEAD~1 argocd/overlays/dev/

# 2. Reapply old configuration
kubectl apply -f argocd/overlays/dev/whoami-app.yaml
kubectl apply -f argocd/overlays/dev/traefik-app.yaml
# ... etc

# 3. Force ArgoCD sync
argocd app sync whoami --force
argocd app sync traefik --force

# 4. Validate apps recover
argocd app list
```

#### Phase 2 Rollback

If Helm values externalization fails:

```bash
# 1. Revert ArgoCD app
git checkout HEAD~1 argocd/overlays/dev/apps/traefik.yaml
kubectl apply -f argocd/overlays/dev/apps/traefik.yaml

# 2. Sync Traefik
argocd app sync traefik --force

# 3. Check Traefik pods
kubectl rollout status deployment/traefik -n traefik
```

#### Complete Rollback

If major issues occur:

```bash
# 1. Revert to tagged version
git reset --hard pre-dry-optimization-YYYYMMDD
git push origin HEAD --force

# 2. Reapply all ArgoCD apps
kubectl apply -f backups/argocd-dev-YYYYMMDD.yaml

# 3. Force sync all apps
argocd app sync --all --force

# 4. Monitor recovery
watch kubectl get applications -n argocd
```

### Rollback Decision Matrix

| Issue | Severity | Action | Rollback Scope |
|-------|----------|--------|----------------|
| Single app not syncing | Low | Fix app patch | App-specific |
| Multiple apps unhealthy | Medium | Revert phase | Environment-specific |
| Cluster unstable | High | Full rollback | All environments |
| Data loss risk | Critical | Emergency rollback + restore | Full cluster + backup |

### Recovery Validation

After rollback:

```bash
# 1. Verify all apps synced
argocd app list | grep -v "Healthy.*Synced"

# 2. Check pods
kubectl get pods -A | grep -v "Running\|Completed"

# 3. Test ingress
curl -sSL https://whoami.dev.truxonline.com

# 4. Verify LoadBalancer IPs
kubectl get svc -A | grep LoadBalancer
```

---

## Conclusion

This optimization plan will:

- **Reduce codebase by ~40%** (~1,220 lines)
- **Eliminate 96% duplication** in ArgoCD apps
- **Centralize environment configuration**
- **Improve maintainability** and reduce errors
- **Align with GitOps best practices**
- **Enable faster development** (adding new apps easier)

### Key Benefits

1. **DRY Principle**: Single source of truth for app templates
2. **Scalability**: Easy to add new environments
3. **Consistency**: Enforced patterns across all apps
4. **Maintainability**: Fewer files to manage
5. **Best Practices**: Follows Kustomize and ArgoCD patterns

### Recommended Next Steps

1. **Review this plan** with the team
2. **Start with Sprint 1** (Phase 1 - dev environment)
3. **Validate approach** with 3 apps before full migration
4. **Document lessons learned** during implementation
5. **Train team** on new patterns

---

**Questions or feedback?** Open an issue or PR in the vixens repository.
