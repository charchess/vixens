# Tasks - Promote Configurations to Test/Staging/Prod

## Phase 1: Test Environment Bootstrap

- [ ] Provision test VMs (if not exists):
  - [ ] Verify nodes carny, celesty, citrine exist in Hyper-V
  - [ ] Verify VLAN 111 + 209 configuration
  - [ ] Verify Talos boot ISO configured

- [ ] Apply Terraform for test:
  - [ ] Create `.secrets/test/infisical-machine-identity.yml` (test Machine Identity)
  - [ ] `cd terraform/environments/test`
  - [ ] `terraform init`
  - [ ] `terraform plan` (review carefully)
  - [ ] `terraform apply` (wait ~30 min)

- [ ] Validate test cluster operational:
  - [ ] `export KUBECONFIG=$(pwd)/kubeconfig-test`
  - [ ] `kubectl get nodes` (3 nodes Ready)
  - [ ] `kubectl get pods -n kube-system` (Cilium running)
  - [ ] `kubectl get application -n argocd` (ArgoCD auto-bootstrapped)

## Phase 2: Infisical Bootstrap (Test)

- [ ] Create test Machine Identity in Infisical:
  - [ ] Open http://192.168.111.69:8085
  - [ ] Create Machine Identity `vixens-test-k8s-operator`
  - [ ] Grant access to project `vixens` environment `test`
  - [ ] Copy clientId + clientSecret to `.secrets/test/infisical-machine-identity.yml`

- [ ] Deploy Infisical secrets via Terraform:
  - [ ] `terraform apply` (infisical-bootstrap module)
  - [ ] Verify secrets: `kubectl get secret -n cert-manager infisical-universal-auth`
  - [ ] Verify secrets: `kubectl get secret -n synology-csi infisical-universal-auth`

## Phase 3: Application Rollout (Test)

- [ ] Create PR dev → test:
  - [ ] `git checkout test`
  - [ ] `git merge dev` (or rebase if conflicts)
  - [ ] `git push origin test`
  - [ ] Create PR on GitHub

- [ ] Wait for CI validation:
  - [ ] yamllint passes
  - [ ] terraform-validate passes
  - [ ] kustomize-build passes
  - [ ] openspec-validate passes

- [ ] Merge PR:
  - [ ] Review changes carefully
  - [ ] Click "Merge" button

- [ ] Monitor ArgoCD sync (test cluster):
  - [ ] `watch kubectl get application -n argocd`
  - [ ] Wait for all apps to show "Healthy"

- [ ] Validate applications (test):
  - [ ] Test ingresses: `curl -I https://homeassistant.test.truxonline.com`
  - [ ] Test traefik: `curl -I https://traefik.test.truxonline.com`
  - [ ] Test argocd: `curl -I https://argocd.test.truxonline.com`
  - [ ] Login to Home Assistant web UI
  - [ ] Verify PVCs bound: `kubectl get pvc -A`

## Phase 4: Staging Environment Bootstrap

- [ ] Provision staging VMs (if not exists):
  - [ ] Create 3 VMs in Hyper-V (VLAN 111 + 210)
  - [ ] Configure Talos boot ISO

- [ ] Apply Terraform for staging:
  - [ ] Create `.secrets/staging/infisical-machine-identity.yml`
  - [ ] `cd terraform/environments/staging`
  - [ ] `terraform init && terraform apply`

- [ ] Validate staging cluster operational (same checks as test)

## Phase 5: Infisical Bootstrap (Staging)

- [ ] Create staging Machine Identity in Infisical
- [ ] Update `.secrets/staging/infisical-machine-identity.yml`
- [ ] Deploy via Terraform
- [ ] Verify secrets deployed

## Phase 6: Application Rollout (Staging)

- [ ] Create PR test → staging
- [ ] Wait for CI validation
- [ ] Merge PR
- [ ] Monitor ArgoCD sync (staging cluster)
- [ ] Validate applications (staging)

## Phase 7: Prod Environment Bootstrap (Bare Metal)

- [ ] Prepare physical nodes:
  - [ ] Verify nodes powered on and network configured
  - [ ] Verify VLAN 111 + 201 access
  - [ ] Install Talos via ISO or PXE

- [ ] Apply Terraform for prod:
  - [ ] Create `.secrets/prod/infisical-machine-identity.yml` (PROD Machine Identity)
  - [ ] `cd terraform/environments/prod`
  - [ ] `terraform init && terraform plan` (REVIEW VERY CAREFULLY)
  - [ ] `terraform apply` (PRODUCTION - take your time)

- [ ] Validate prod cluster operational:
  - [ ] All checks pass
  - [ ] etcd quorum healthy
  - [ ] Cilium CNI operational
  - [ ] ArgoCD bootstrapped

## Phase 8: Infisical Bootstrap (Prod)

- [ ] Create prod Machine Identity in Infisical (separate, prod-only access)
- [ ] Update `.secrets/prod/infisical-machine-identity.yml`
- [ ] Deploy via Terraform
- [ ] Verify secrets deployed (PRODUCTION - double check)

## Phase 9: Application Rollout (Prod)

- [ ] Create PR staging → main (prod):
  - [ ] Review changes extensively
  - [ ] Verify no unexpected modifications
  - [ ] Request manual approval (even self-approval acceptable)

- [ ] Merge PR (PRODUCTION):
  - [ ] Merge carefully
  - [ ] Monitor ArgoCD sync in prod cluster
  - [ ] DO NOT walk away - watch for issues

- [ ] Validate applications (prod):
  - [ ] Test all ingresses (*.truxonline.com - production domain)
  - [ ] Login to all web UIs
  - [ ] Verify certificates are Let's Encrypt PRODUCTION (not staging)
  - [ ] Check PVCs bound
  - [ ] Monitor for 1 hour after deployment

## Phase 10: Documentation & Validation

- [ ] Update CLAUDE.md:
  - [ ] Document all 4 environments operational
  - [ ] Update "Current Infrastructure Status" section
  - [ ] Document Git workflow (dev → test → staging → main)

- [ ] Create runbook:
  - [ ] Document promotion workflow
  - [ ] Document validation steps per environment
  - [ ] Document rollback procedures

- [ ] Test rollback in test environment:
  - [ ] Create intentional failure
  - [ ] Practice rollback (git revert + ArgoCD sync)
  - [ ] Verify recovery works

---

## Notes

**Estimated Time:**
- Phase 1-3 (test): 3-4 hours
- Phase 4-6 (staging): 2-3 hours
- Phase 7-9 (prod): 4-5 hours (careful validation)
- Phase 10 (docs): 1 hour

**Critical Path:**
Test → Staging → Prod (sequential, not parallel)

**Checkpoints:**
- Validate EVERYTHING in test before staging
- Validate EVERYTHING in staging before prod
- Never skip validation steps for prod
