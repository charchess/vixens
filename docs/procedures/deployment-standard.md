# Standard Deployment Patcher

This document defines the standard pattern for deploying applications in the Vixens cluster using GitOps, Kustomize, and Infisical.

## Directory Structure

Applications should follow a standard Kustomize structure:

```
apps/<category>/<app-name>/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── infisical-secret.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   ├── ingress.yaml
    │   └── http-redirect.yaml
    └── prod/
        ├── kustomization.yaml
        ├── ingress.yaml
        └── http-redirect.yaml
```

## Core Standards

### 1. Infisical (Secrets Management)
- Use `InfisicalSecret` to sync secrets from the centralized Infisical vault.
- **Host API**: `http://192.168.111.69:8085`
- **Environment Slug**: Default to `dev` in `base`, patch to `prod` in `prod` overlay.
- **Secrets Path**: `/apps/<category>/<app-name>`

### 2. Ingress & TLS
- **Annotations**:
    - `cert-manager.io/cluster-issuer`: `letsencrypt-staging` (dev) / `letsencrypt-prod` (prod).
    - `traefik.ingress.kubernetes.io/router.entrypoints`: `web, websecure`.
    - `traefik.ingress.kubernetes.io/router.middlewares`: `monitoring-redirect-to-https@kubernetescrd`.
- **Hostname**: `<app>.<env>.truxonline.com` (dev) / `<app>.truxonline.com` (prod).

### 3. Stability & Resources
- **Tolerations**: Always include control-plane tolerations to allow scheduling on all nodes.
- **Strategy**: Use `type: Recreate` for deployments using RWO volumes (iSCSI).
- **Resources**: Always define `requests` and `limits`.

### 4. Monitoring
- **Namespace Label**: `goldilocks.fairwinds.com/enabled: "true"` for resource recommendations.
- **Homepage Integration** (Optional): Use `gethomepage.dev/` annotations for dashboard integration.

### 5. DNS (ExternalDNS)
- **Internal DNS**: Automatically managed for all Ingresses. Use `external-dns.alpha.kubernetes.io/target` to force a CNAME hostname if needed.
- **Public DNS**: Add `external-dns.alpha.kubernetes.io/public: "true"` to expose the Ingress on Gandi (Prod only).

## Example: The Template App
A reference implementation is available in `apps/template-app/`.
