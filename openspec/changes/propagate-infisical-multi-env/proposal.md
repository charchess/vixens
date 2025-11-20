# Propagate Infisical Integration to Test/Staging/Prod Environments

## Why

Infisical Operator is currently deployed and operational only in the **dev** environment. To maintain consistency across all environments and enable proper secrets management in test, staging, and production clusters, Infisical integration must be propagated.

**Current State:**
- ✅ Dev environment: Infisical fully operational with path isolation
- ❌ Test environment: No Infisical integration
- ❌ Staging environment: No Infisical integration
- ❌ Prod environment: No Infisical integration

**Benefits of Multi-Environment Infisical:**
- Centralized secrets management across all clusters
- Environment-specific secret isolation (dev/test/staging/prod)
- Automated secret rotation capabilities
- Audit trail for secret access
- GitOps-compliant secret delivery (no plaintext in Git)

## What Changes

### 1. Infisical Project Structure
Create environment-specific folders in Infisical UI:
- `/cert-manager` (per environment)
- `/synology-csi` (per environment)
- Future applications as needed

### 2. Machine Identities
Create Universal Auth Machine Identities for each cluster:
- `vixens-test-k8s-operator` (envSlug: test)
- `vixens-staging-k8s-operator` (envSlug: staging)
- `vixens-prod-k8s-operator` (envSlug: prod)

### 3. GitOps Configuration
Replicate Infisical integration structure from dev:

**For cert-manager-webhook-gandi:**
- `apps/cert-manager-webhook-gandi/overlays/{test,staging,prod}/`
  - `kustomization.yaml` - Include Infisical base resources
  - `infisical-auth-secret.yaml` - Environment-specific clientId/clientSecret
  - `gandi-infisical-secret.yaml` - Patch envSlug to match environment

**For synology-csi:**
- `apps/synology-csi/infisical/overlays/{test,staging,prod}/`
  - `kustomization.yaml` - Include Infisical base resources
  - `infisical-auth-secret.yaml` - Environment-specific clientId/clientSecret
  - `synology-infisical-secret-patch.yaml` - Patch envSlug to match environment

**For ArgoCD:**
- `argocd/overlays/{test,staging,prod}/apps/`
  - `cert-manager-secrets.yaml` - Deploy Infisical integration
  - `synology-csi-secrets.yaml` - Deploy Infisical integration

### 4. Infisical Operator Deployment
Ensure Infisical Operator is deployed in each environment:
- Already deployed via Terraform in `modules/shared/`
- Verify operator pods are running in `infisical-system` namespace

## Impact

**Security:**
- ✅ Eliminates plaintext secrets in `.secrets/` directories
- ✅ Centralized secret management with access control
- ✅ Audit trail for secret modifications

**Operations:**
- ✅ Consistent secret delivery across all environments
- ✅ Environment-specific secret isolation (test secrets ≠ prod secrets)
- ✅ Automated synchronization (60s resync interval)

**Maintenance:**
- ✅ Single source of truth for secrets (Infisical UI)
- ✅ Easy secret rotation without Git commits
- ✅ No accidental secret leaks in version control

**Risk:**
- ⚠️ Dependency on Infisical availability (self-hosted at 192.168.111.69:8085)
- ⚠️ Requires proper backup of Infisical data
- Mitigation: Document Infisical backup/restore procedures

## Non-Goals

- Not migrating additional applications beyond cert-manager and synology-csi
- Not changing Infisical instance location (keeping self-hosted)
- Not implementing Infisical high availability (single instance acceptable for homelab)
- Not creating automated secret rotation policies (manual rotation via UI)
