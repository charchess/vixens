# Tasks - Bootstrap Infisical Machine Identity via Terraform

## Phase 1: Create Terraform Module

- [ ] Create module structure:
  - [ ] `mkdir -p terraform/modules/infisical-bootstrap`
  - [ ] Create `terraform/modules/infisical-bootstrap/main.tf`
  - [ ] Create `terraform/modules/infisical-bootstrap/variables.tf`
  - [ ] Create `terraform/modules/infisical-bootstrap/versions.tf`
  - [ ] Create `terraform/modules/infisical-bootstrap/outputs.tf`

- [ ] Implement module logic:
  - [ ] Add `kubectl` provider requirement (~> 2.0)
  - [ ] Add variable `environment` (string)
  - [ ] Add variable `argocd_bootstrap_complete` (dependency marker)
  - [ ] Read `.secrets/${environment}/infisical-machine-identity.yml`
  - [ ] Create `kubectl_manifest` resource for each namespace
  - [ ] Add `depends_on` to ensure namespace existence

- [ ] Validate module:
  - [ ] `cd terraform/modules/infisical-bootstrap`
  - [ ] `terraform init`
  - [ ] `terraform validate`
  - [ ] `terraform fmt -check`

## Phase 2: Prepare Secrets Files

- [ ] Create dev secrets file:
  - [ ] `mkdir -p .secrets/dev`
  - [ ] Copy current Machine Identity from `apps/cert-manager-webhook-gandi/base/infisical-auth-secret.yaml`
  - [ ] Create `.secrets/dev/infisical-machine-identity.yml` with clientId + clientSecret
  - [ ] Set file permissions: `chmod 600 .secrets/dev/infisical-machine-identity.yml`

- [ ] Verify .gitignore:
  - [ ] Check `.gitignore` includes `.secrets/`
  - [ ] Test: `git status` should NOT show `.secrets/` directory
  - [ ] If missing, add `.secrets/` to `.gitignore`

## Phase 3: Integrate Module in Dev Environment

- [ ] Update dev environment Terraform:
  - [ ] Add module call in `terraform/environments/dev/main.tf`:
    ```hcl
    module "infisical_bootstrap" {
      source = "../../modules/infisical-bootstrap"

      environment                = var.environment
      argocd_bootstrap_complete  = module.environment.argocd_bootstrap_complete
    }
    ```
  - [ ] Add output in `terraform/environments/dev/outputs.tf` for infisical secrets

- [ ] Test Terraform plan:
  - [ ] `cd terraform/environments/dev`
  - [ ] `terraform init -upgrade`
  - [ ] `terraform plan` - should show 2 new secrets (cert-manager, synology-csi)
  - [ ] Verify no destroy/recreate of existing resources

## Phase 4: Deploy Secrets via Terraform (Dev)

- [ ] Apply Terraform changes:
  - [ ] `terraform apply`
  - [ ] Wait for completion (~30 seconds)

- [ ] Validate secrets deployed:
  - [ ] Check cert-manager: `kubectl get secret -n cert-manager infisical-universal-auth -o yaml`
  - [ ] Verify keys exist: `clientId`, `clientSecret`
  - [ ] Check synology-csi: `kubectl get secret -n synology-csi infisical-universal-auth -o yaml`
  - [ ] Verify label `managed-by=terraform` present

- [ ] Validate InfisicalSecret CRDs still work:
  - [ ] Check cert-manager sync: `kubectl get infisicalsecret -n cert-manager gandi-credentials-sync -o yaml`
  - [ ] Verify status shows successful authentication
  - [ ] Check synology-csi sync: `kubectl get infisicalsecret -n synology-csi synology-csi-credentials-sync -o yaml`
  - [ ] Verify no authentication errors in events

## Phase 5: Remove Hardcoded Secrets from Git

- [ ] Delete hardcoded secret files:
  - [ ] `git rm apps/cert-manager-webhook-gandi/base/infisical-auth-secret.yaml`
  - [ ] `git rm apps/synology-csi/infisical/base/infisical-auth-secret.yaml`

- [ ] Update kustomization files:
  - [ ] Edit `apps/cert-manager-webhook-gandi/base/kustomization.yaml`
  - [ ] Remove `infisical-auth-secret.yaml` from resources list
  - [ ] Edit `apps/synology-csi/infisical/base/kustomization.yaml`
  - [ ] Remove `infisical-auth-secret.yaml` from resources list

- [ ] Validate kustomize builds:
  - [ ] `kustomize build apps/cert-manager-webhook-gandi/overlays/dev`
  - [ ] Should NOT contain `infisical-universal-auth` Secret
  - [ ] `kustomize build apps/synology-csi/infisical/overlays/dev`
  - [ ] Should NOT contain `infisical-universal-auth` Secret

## Phase 6: Commit and Validate (Dev)

- [ ] Git commit:
  - [ ] `git add terraform/modules/infisical-bootstrap/`
  - [ ] `git add terraform/environments/dev/main.tf`
  - [ ] `git add apps/*/base/kustomization.yaml`
  - [ ] `git commit -m "feat(secrets): Bootstrap Infisical Machine Identity via Terraform"`

- [ ] Push to dev branch:
  - [ ] `git push origin dev`

- [ ] Monitor ArgoCD sync:
  - [ ] Watch ArgoCD Applications: `kubectl get application -n argocd -w`
  - [ ] `cert-manager-secrets` should sync (removing hardcoded secret)
  - [ ] `synology-csi-secrets` should sync (removing hardcoded secret)
  - [ ] Both apps should remain Healthy

- [ ] Validate application functionality:
  - [ ] Check cert-manager certificates: `kubectl get certificate -A`
  - [ ] All certificates should be Ready
  - [ ] Check Synology CSI: `kubectl get pods -n synology-csi`
  - [ ] Controller and node pods should be Running
  - [ ] Check PVCs: `kubectl get pvc -A`
  - [ ] All PVCs should remain Bound

## Phase 7: Multi-Environment Rollout (Test)

- [ ] Create test Machine Identity in Infisical:
  - [ ] Open Infisical UI: http://192.168.111.69:8085
  - [ ] Create Machine Identity: `vixens-test-k8s-operator`
  - [ ] Grant access to project `vixens` environment `test`
  - [ ] Copy clientId and clientSecret

- [ ] Create test secrets file:
  - [ ] `mkdir -p .secrets/test`
  - [ ] Create `.secrets/test/infisical-machine-identity.yml`
  - [ ] Paste test Machine Identity credentials
  - [ ] `chmod 600 .secrets/test/infisical-machine-identity.yml`

- [ ] Deploy test environment:
  - [ ] `cd terraform/environments/test`
  - [ ] `terraform init -upgrade`
  - [ ] `terraform plan`
  - [ ] `terraform apply`

- [ ] Validate test environment:
  - [ ] Check secrets deployed
  - [ ] Monitor ArgoCD sync in test cluster
  - [ ] Validate app functionality (cert-manager, synology-csi)

## Phase 8: Multi-Environment Rollout (Staging)

- [ ] Create staging Machine Identity in Infisical
- [ ] Create `.secrets/staging/infisical-machine-identity.yml`
- [ ] Deploy staging environment with Terraform
- [ ] Validate staging cluster

## Phase 9: Multi-Environment Rollout (Prod)

- [ ] Create prod Machine Identity in Infisical
- [ ] Create `.secrets/prod/infisical-machine-identity.yml`
- [ ] Deploy prod environment with Terraform
- [ ] Validate prod cluster (critical - manual verification required)

## Phase 10: Documentation

- [ ] Update CLAUDE.md:
  - [ ] Add "Infisical Bootstrap" section
  - [ ] Document `.secrets/` directory structure
  - [ ] Document Terraform module usage
  - [ ] Add workflow example (create secrets file → terraform apply)

- [ ] Update terraform/modules/infisical-bootstrap/README.md:
  - [ ] Document module inputs/outputs
  - [ ] Provide usage example
  - [ ] Document secrets file format

- [ ] Update .secrets/README.md:
  - [ ] Create file explaining purpose
  - [ ] Provide template for `infisical-machine-identity.yml`
  - [ ] Warn about file permissions (600)

## Phase 11: Validation & Cleanup

- [ ] Security validation:
  - [ ] Run: `rg "ee279e5e-82b6-476b-9643" --files-with-matches`
  - [ ] Should return ONLY `.secrets/` files (gitignored)
  - [ ] Run: `git log --all --full-history --source -- '*infisical-auth-secret.yaml'`
  - [ ] Verify files were properly deleted (not just moved)

- [ ] Functional validation:
  - [ ] All 4 environments operational
  - [ ] All InfisicalSecret CRDs syncing successfully
  - [ ] No manual secret deployment needed
  - [ ] Terraform destroy/recreate works correctly

- [ ] Terraform state verification:
  - [ ] Check S3 backend encryption enabled
  - [ ] Verify state contains secrets (expected, acceptable for homelab)
  - [ ] Document state security in CLAUDE.md

---

## Notes

**Estimated Time:**
- Phase 1-6 (dev): 2 hours
- Phase 7-9 (multi-env): 1 hour per environment
- Phase 10-11 (docs): 30 minutes

**Dependencies:**
- Requires Infisical server operational (http://192.168.111.69:8085)
- Requires Machine Identities created in Infisical (manual UI step)
- Requires `.secrets/` directory gitignored

**Rollback:**
- Simple: `git revert <commit>` restores hardcoded secrets
- ArgoCD will automatically resync
- No data loss (PVCs unaffected)
