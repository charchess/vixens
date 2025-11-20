# Tasks - Propagate Infisical to Multi-Environment

## Phase 1: Infisical UI Configuration

- [ ] Create environment folders in Infisical project "vixens":
  - [ ] test environment: `/cert-manager`, `/synology-csi`
  - [ ] staging environment: `/cert-manager`, `/synology-csi`
  - [ ] prod environment: `/cert-manager`, `/synology-csi`

- [ ] Populate secrets in each environment:
  - [ ] test: Gandi API token in `/cert-manager/api-token`
  - [ ] test: Synology CSI credentials in `/synology-csi/client-info.yaml`
  - [ ] staging: Gandi API token in `/cert-manager/api-token`
  - [ ] staging: Synology CSI credentials in `/synology-csi/client-info.yaml`
  - [ ] prod: Gandi API token in `/cert-manager/api-token`
  - [ ] prod: Synology CSI credentials in `/synology-csi/client-info.yaml`

- [ ] Create Machine Identities in Infisical UI:
  - [ ] `vixens-test-k8s-operator` with test environment access
  - [ ] `vixens-staging-k8s-operator` with staging environment access
  - [ ] `vixens-prod-k8s-operator` with prod environment access
  - [ ] Document clientId and clientSecret for each identity

## Phase 2: GitOps Configuration (Test Environment)

- [ ] cert-manager-webhook-gandi test overlay:
  - [ ] Create `apps/cert-manager-webhook-gandi/overlays/test/kustomization.yaml`
  - [ ] Create `apps/cert-manager-webhook-gandi/overlays/test/infisical-auth-secret.yaml`
  - [ ] Create `apps/cert-manager-webhook-gandi/overlays/test/gandi-infisical-secret-patch.yaml`

- [ ] synology-csi test overlay:
  - [ ] Create `apps/synology-csi/infisical/overlays/test/kustomization.yaml`
  - [ ] Create `apps/synology-csi/infisical/overlays/test/infisical-auth-secret.yaml`
  - [ ] Create `apps/synology-csi/infisical/overlays/test/synology-infisical-secret-patch.yaml`

- [ ] ArgoCD test applications:
  - [ ] Create `argocd/overlays/test/apps/cert-manager-secrets.yaml`
  - [ ] Create `argocd/overlays/test/apps/synology-csi-secrets.yaml`
  - [ ] Update `argocd/overlays/test/kustomization.yaml` to include new apps

- [ ] Validate test environment:
  - [ ] Deploy to test cluster
  - [ ] Verify InfisicalSecret CRDs reconcile successfully
  - [ ] Verify Kubernetes secrets created in namespaces
  - [ ] Test cert-manager certificate issuance
  - [ ] Test Synology CSI volume provisioning

## Phase 3: GitOps Configuration (Staging Environment)

- [ ] Replicate test overlay structure for staging:
  - [ ] cert-manager-webhook-gandi staging overlay
  - [ ] synology-csi staging overlay
  - [ ] ArgoCD staging applications
  - [ ] Update staging kustomization.yaml

- [ ] Validate staging environment (same checks as test)

## Phase 4: GitOps Configuration (Prod Environment)

- [ ] Replicate test overlay structure for prod:
  - [ ] cert-manager-webhook-gandi prod overlay
  - [ ] synology-csi prod overlay
  - [ ] ArgoCD prod applications
  - [ ] Update prod kustomization.yaml

- [ ] Validate prod environment (same checks as test)

## Phase 5: Cleanup and Documentation

- [ ] Delete `.secrets/test/`, `.secrets/staging/`, `.secrets/prod/` directories
- [ ] Update `.gitignore` to ensure `.secrets/` is fully excluded
- [ ] Update ADR 007 with multi-environment status
- [ ] Update CLAUDE.md with multi-environment Infisical configuration
- [ ] Document secret rotation procedures for all environments
- [ ] Create runbook for Infisical backup/restore

## Phase 6: Validation

- [ ] Run `kubectl get infisicalsecret -A` in all environments
- [ ] Verify no plaintext secrets remain in Git
- [ ] Test secret rotation in non-prod environment
- [ ] Validate sync intervals are working (60s resync)
- [ ] Check Infisical operator logs for errors in all environments
