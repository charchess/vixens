# cert-manager Helm Values

This directory contains cert-manager Helm chart values organized for DRY (Don't Repeat Yourself) principles.

## Structure

```
values/
├── common.yaml      # Shared configuration for ALL environments
├── dev.yaml         # Dev-specific overrides
├── test.yaml        # Test-specific overrides
├── staging.yaml     # Staging-specific overrides
├── prod.yaml        # Production-specific overrides
└── README.md        # This file
```

## Common Values (common.yaml)

Shared across all environments:
- **installCRDs**: true (install CRDs with Helm)
- **Global config**: Leader election in cert-manager namespace
- **Tolerations**: Control-plane node tolerations for all components
  - cert-manager controller
  - webhook
  - cainjector
  - startupapicheck

## Environment-Specific Overrides

### Dev (dev.yaml)
- Currently uses all common values
- Placeholder for future dev-specific config

### Test (test.yaml)
- Currently uses all common values
- Placeholder for future test-specific config

### Staging (staging.yaml)
- Currently uses all common values
- Placeholder for production-like resource limits

### Prod (prod.yaml)
- **Resources**: Higher limits for all components
  - Controller: 100m/500m CPU, 128Mi/512Mi memory
  - Webhook: 50m/200m CPU, 64Mi/256Mi memory
  - CA Injector: 50m/200m CPU, 64Mi/256Mi memory
- **Replicas**: 2 (HA)
- **Monitoring**: Prometheus metrics + ServiceMonitor enabled

## Usage with ArgoCD

The ArgoCD Application uses **multiple sources** pattern to combine these values:

```yaml
sources:
  # Helm chart from Jetstack repository
  - repoURL: https://charts.jetstack.io
    chart: cert-manager
    targetRevision: "v1.14.4"
    helm:
      valueFiles:
        - $values/apps/00-infra/cert-manager/values/common.yaml
        - $values/apps/00-infra/cert-manager/values/dev.yaml  # Environment-specific

  # Values from Git repository
  - repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    ref: values
```

## Local Testing

You can test these values locally with Helm:

```bash
# Add cert-manager Helm repository
helm repo add jetstack https://charts.jetstack.io
helm repo update

# Test dev environment
helm template cert-manager jetstack/cert-manager \
  -f apps/cert-manager/values/common.yaml \
  -f apps/cert-manager/values/dev.yaml \
  --version v1.14.4 \
  --namespace cert-manager

# Compare environments
diff \
  <(helm template cert-manager jetstack/cert-manager -f common.yaml -f dev.yaml) \
  <(helm template cert-manager jetstack/cert-manager -f common.yaml -f prod.yaml)
```

## Modifying Values

### To change common values (affects all environments):
1. Edit `common.yaml`
2. Test locally with helm template
3. Commit and push
4. ArgoCD will auto-sync all environments

### To change environment-specific values:
1. Edit the appropriate `{env}.yaml` file
2. Test locally
3. Commit and push
4. ArgoCD will auto-sync that environment only

## See Also

- [cert-manager Helm Chart Documentation](https://cert-manager.io/docs/installation/helm/)
- [ArgoCD Multiple Sources](https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/)
- [Project CLAUDE.md](../../../CLAUDE.md)
