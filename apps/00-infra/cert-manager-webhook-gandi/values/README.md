# cert-manager-webhook-gandi Helm Values

This directory contains cert-manager-webhook-gandi Helm chart values organized for DRY principles.

## Overview

The Gandi LiveDNS webhook enables cert-manager to perform DNS-01 challenges using Gandi's DNS service. This is required for wildcard certificates and certificates for domains behind private networks.

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
- **groupName**: `acme.truxonline.com` (ACME DNS solver group)
- **Tolerations**: Control-plane node tolerations

## Environment-Specific Overrides

### Dev (dev.yaml)
- Currently uses all common values

### Test (test.yaml)
- Currently uses all common values

### Staging (staging.yaml)
- Currently uses all common values

### Prod (prod.yaml)
- **Resources**: Higher limits
  - Requests: 50m CPU, 64Mi memory
  - Limits: 200m CPU, 256Mi memory
- **Replicas**: 2 (HA)

## Usage with ArgoCD

The ArgoCD Application uses **multiple sources** pattern:

```yaml
sources:
  - repoURL: https://sintef.github.io/cert-manager-webhook-gandi
    chart: cert-manager-webhook-gandi
    targetRevision: "v0.5.2"
    helm:
      valueFiles:
        - $values/apps/00-infra/cert-manager-webhook-gandi/values/common.yaml
        - $values/apps/00-infra/cert-manager-webhook-gandi/values/dev.yamls

  - repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    ref: values
```

## Required Secrets

The webhook requires a Gandi API token stored as a Kubernetes Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gandi-credentials
  namespace: cert-manager
type: Opaque
stringData:
  api-token: <your-gandi-api-token>
```

## ClusterIssuer Configuration

The webhook is referenced in ClusterIssuers for DNS-01 challenges:

```yaml
solvers:
  - dns01:
      webhook:
        groupName: acme.truxonline.com
        solverName: gandi
        config:
          apiKeySecretRef:
            name: gandi-credentials
            key: api-token
```

## See Also

- [cert-manager-webhook-gandi GitHub](https://github.com/SINTEF/cert-manager-webhook-gandi)
- [cert-manager DNS-01 Documentation](https://cert-manager.io/docs/configuration/acme/dns01/)
- [Project CLAUDE.md](../../../CLAUDE.md)
