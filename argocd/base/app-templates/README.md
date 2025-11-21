# ArgoCD Application Templates

This directory contains reusable ArgoCD Application templates used across all environments.

## Overview

Instead of duplicating 50+ ArgoCD Application files with 96% identical content, we use Kustomize templates and patches to generate environment-specific applications from a single source of truth.

## Templates

### git-app-template.yaml

Generic template for Git-sourced Kustomize applications.

**Used for:**
- whoami
- argocd (ingress configuration)
- cilium-lb
- cert-manager-config
- traefik-dashboard
- nfs-storage
- mail-gateway
- homeassistant

**Placeholders:**
- `APP_NAME`: Application name (set via patch metadata.name)
- `TARGET_REVISION`: Git branch (dev/test/staging/main for prod)
- `APP_PATH`: Path to app in repo (apps/{name}/overlays/{env})
- `NAMESPACE`: Target namespace (usually same as app name)

**Common fields (shared by all apps):**
- `spec.project: default`
- `spec.source.repoURL: https://github.com/charchess/vixens.git`
- `spec.destination.server: https://kubernetes.default.svc`
- `spec.syncPolicy.automated: {prune: true, selfHeal: true}`
- `spec.syncPolicy.syncOptions: [CreateNamespace=true]`
- `finalizers: [resources-finalizer.argocd.argoproj.io]`

## Usage

Environment overlays reference these templates and apply patches to customize them.

Example patch in `argocd/overlays/dev/apps/whoami.yaml`:

```yaml
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: whoami
spec:
  source:
    targetRevision: dev
    path: apps/whoami/overlays/dev
  destination:
    namespace: whoami
```

The patch is applied via the environment's `kustomization.yaml`:

```yaml
resources:
  - ../../base/app-templates

patches:
  - path: apps/whoami.yaml
    target:
      kind: Application
      name: whoami
```

## Benefits

1. **Single source of truth**: Template changes apply to all apps
2. **DRY principle**: No duplication of common fields
3. **Consistency**: All apps follow the same structure
4. **Maintainability**: Easy to update common configurations
5. **Type safety**: Kustomize validates manifests

### helm-app-template.yaml

Generic template for Helm-sourced applications.

**Used for:**
- traefik
- cert-manager
- cert-manager-webhook-gandi
- synology-csi (future)

**Placeholders:**
- `APP_NAME`: Application name
- `HELM_REPO`: Helm repository URL (e.g., https://helm.traefik.io/traefik)
- `CHART_NAME`: Chart name (e.g., traefik)
- `CHART_VERSION`: Chart version (e.g., v25.0.0)
- `NAMESPACE`: Target namespace
- `HELM_VALUES`: Inline Helm values (will be externalized in Phase 2)

**Common fields (shared by all Helm apps):**
- `spec.project: default`
- `spec.destination.server: https://kubernetes.default.svc`
- `spec.syncPolicy.automated: {prune: true, selfHeal: true}`
- `spec.syncPolicy.syncOptions: [CreateNamespace=true, ServerSideApply=true]`
- `spec.syncPolicy.retry: {limit: 5, backoff: {...}}`
- `finalizers: [resources-finalizer.argocd.argoproj.io]`

**Example patch** in `argocd/overlays/dev/apps/traefik.yaml`:

```yaml
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: traefik
spec:
  source:
    repoURL: https://helm.traefik.io/traefik
    chart: traefik
    targetRevision: "v25.0.0"
    helm:
      values: |
        # Inline values here (to be externalized in Phase 2)
        service:
          type: LoadBalancer
```

## Future Enhancements

- **Phase 2**: Externalize Helm values to Git repository using ArgoCD multiple sources
- Use Kustomize replacements for dynamic field injection
- Add validation with OPA policies
- Add support for ApplicationSet for multi-environment generation
