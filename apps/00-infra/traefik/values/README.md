# Traefik Helm Values

This directory contains Traefik Helm chart values organized for DRY (Don't Repeat Yourself) principles.

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
- **Providers**: kubernetesCRD, kubernetesIngress
- **API/Dashboard**: Enabled with insecure mode (internal network only)
- **IngressRoute**: Dashboard on web entrypoint
- **Tolerations**: Control-plane node tolerations
- **Ports**: web (80), websecure (443), traefik (9000)

## Environment-Specific Overrides

### Dev (dev.yaml)
- **VLAN**: 208 (192.168.208.0/24)
- **LoadBalancer IP**: 192.168.208.70
- **Log level**: DEBUG
- **Resources**: Low (100m CPU, 128Mi memory)
- **Replicas**: 1

### Test (test.yaml)
- **VLAN**: 209 (192.168.209.0/24)
- **LoadBalancer IP**: 192.168.209.70
- **Log level**: INFO
- **Resources**: Low (100m CPU, 128Mi memory)
- **Replicas**: 1

### Staging (staging.yaml)
- **VLAN**: 210 (192.168.210.0/24)
- **LoadBalancer IP**: 192.168.210.70
- **Log level**: INFO
- **Resources**: Medium (200m CPU, 256Mi memory)
- **Replicas**: 2

### Prod (prod.yaml)
- **VLAN**: 201 (192.168.201.0/24)
- **LoadBalancer IP**: 192.168.201.70
- **Log level**: WARN
- **Resources**: High (500m CPU, 512Mi memory)
- **Replicas**: 3
- **Metrics**: Prometheus enabled
- **PDB**: minAvailable: 1

## Usage with ArgoCD

The ArgoCD Application uses **multiple sources** pattern to combine these values:

```yaml
sources:
  # Helm chart from Traefik repository
  - repoURL: https://helm.traefik.io/traefik
    chart: traefik
    targetRevision: "v25.0.0"
    helm:
      valueFiles:
        - $values/apps/00-infra/traefik/values/common.yaml
        - $values/apps/00-infra/traefik/values/dev.yaml  # Environment-specific

  # Values from Git repository
  - repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    ref: values
```

## Local Testing

You can test these values locally with Helm:

```bash
# Test dev environment
helm template traefik traefik/traefik \
  -f apps/traefik/values/common.yaml \
  -f apps/traefik/values/dev.yaml \
  --version v25.0.0 \
  --namespace traefik

# Compare environments
diff \
  <(helm template traefik traefik/traefik -f common.yaml -f dev.yaml) \
  <(helm template traefik traefik/traefik -f common.yaml -f prod.yaml)
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

- [Traefik Helm Chart Documentation](https://github.com/traefik/traefik-helm-chart)
- [ArgoCD Multiple Sources](https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/)
- [Project CLAUDE.md](../../../CLAUDE.md)
