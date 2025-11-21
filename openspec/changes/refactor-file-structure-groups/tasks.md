# Tasks - Refactor File Structure with Groups

## Phase 1: Preparation

- [ ] Create comprehensive inventory of all ArgoCD Application manifests
  - [ ] List all files in `argocd/overlays/*/apps/*.yaml`
  - [ ] Document current `spec.source.path` for each application
  - [ ] Identify which applications belong to which group

- [ ] Create backup branch before refactoring
  - [ ] `git checkout -b backup/pre-file-structure-refactor`
  - [ ] `git push origin backup/pre-file-structure-refactor`

- [ ] Document current directory structure
  - [ ] Run `tree apps/` > docs/file-structure-before.txt

## Phase 2: Directory Structure Creation

- [ ] Create new group directories:
  - [ ] `mkdir -p apps/infrastructure`
  - [ ] `mkdir -p apps/storage`
  - [ ] `mkdir -p apps/platform`
  - [ ] `mkdir -p apps/applications`

- [ ] Move applications to infrastructure group:
  - [ ] `git mv apps/cilium-lb apps/infrastructure/`
  - [ ] `git mv apps/cert-manager apps/infrastructure/`
  - [ ] `git mv apps/cert-manager-webhook-gandi apps/infrastructure/`
  - [ ] `git mv apps/traefik apps/infrastructure/`

- [ ] Move applications to storage group:
  - [ ] `git mv apps/synology-csi apps/storage/`

- [ ] Create group README files:
  - [ ] Create `apps/infrastructure/README.md` with purpose and applications list
  - [ ] Create `apps/storage/README.md` with purpose and applications list
  - [ ] Create `apps/platform/README.md` with purpose and future applications
  - [ ] Create `apps/applications/README.md` with purpose and future applications

## Phase 3: ArgoCD Application Updates (Dev Environment)

- [ ] Update infrastructure app paths in `argocd/overlays/dev/apps/`:
  - [ ] Update `cilium-lb.yaml`: path to `apps/infrastructure/cilium-lb/overlays/dev`
  - [ ] Update `cert-manager.yaml`: path to `apps/infrastructure/cert-manager/overlays/dev`
  - [ ] Update `cert-manager-secrets.yaml`: path to `apps/infrastructure/cert-manager-webhook-gandi/overlays/dev`
  - [ ] Update `cert-manager-webhook-gandi.yaml`: path to `apps/infrastructure/cert-manager-webhook-gandi/overlays/dev`
  - [ ] Update `cert-manager-config.yaml`: path to `apps/infrastructure/cert-manager-webhook-gandi/overlays/dev`
  - [ ] Update `traefik.yaml`: path to `apps/infrastructure/traefik/overlays/dev`

- [ ] Update storage app paths in `argocd/overlays/dev/apps/`:
  - [ ] Update `synology-csi-secrets.yaml`: path to `apps/storage/synology-csi/infisical/overlays/dev`
  - [ ] Update `synology-csi.yaml`: path to `apps/storage/synology-csi/overlays/dev`

## Phase 4: ArgoCD Application Updates (Test Environment)

- [ ] Replicate dev path updates for test environment:
  - [ ] Update all infrastructure app paths in `argocd/overlays/test/apps/`
  - [ ] Update all storage app paths in `argocd/overlays/test/apps/`

## Phase 5: ArgoCD Application Updates (Staging Environment)

- [ ] Replicate dev path updates for staging environment:
  - [ ] Update all infrastructure app paths in `argocd/overlays/staging/apps/`
  - [ ] Update all storage app paths in `argocd/overlays/staging/apps/`

## Phase 6: ArgoCD Application Updates (Prod Environment)

- [ ] Replicate dev path updates for prod environment:
  - [ ] Update all infrastructure app paths in `argocd/overlays/prod/apps/`
  - [ ] Update all storage app paths in `argocd/overlays/prod/apps/`

## Phase 7: Documentation Updates

- [ ] Update CLAUDE.md:
  - [ ] Replace repository structure diagram with grouped structure
  - [ ] Add explanation of group-based organization

- [ ] Update README.md:
  - [ ] Update quick start paths to reflect new structure
  - [ ] Add section on directory organization principles

- [ ] Create ADR:
  - [ ] Create `docs/adr/008-file-structure-groups.md`
  - [ ] Document rationale for grouping strategy
  - [ ] Reference Infrastructure → Platform → Applications pattern

- [ ] Document final structure:
  - [ ] Run `tree apps/` > docs/file-structure-after.txt

## Phase 8: Validation (Dev Environment First)

- [ ] Validate ArgoCD can parse updated manifests:
  - [ ] `kubectl apply --dry-run=client -k argocd/overlays/dev/`
  - [ ] Check for path resolution errors

- [ ] Deploy to dev environment:
  - [ ] Git commit all changes
  - [ ] Git push to dev branch
  - [ ] Monitor ArgoCD sync status

- [ ] Verify all applications sync successfully:
  - [ ] `kubectl get applications -n argocd`
  - [ ] Check all apps show "Healthy" and "Synced"
  - [ ] No path resolution errors in ArgoCD logs

- [ ] Test application functionality:
  - [ ] Verify Traefik ingress responds
  - [ ] Verify cert-manager certificates are valid
  - [ ] Verify Synology CSI volumes provision correctly

## Phase 9: Validation (Other Environments)

- [ ] Validate test environment (if deployed)
- [ ] Validate staging environment (if deployed)
- [ ] Validate prod environment (if deployed)

## Phase 10: Cleanup

- [ ] Search for any remaining references to old paths:
  - [ ] `grep -r "apps/cilium-lb" docs/`
  - [ ] `grep -r "apps/cert-manager" docs/`
  - [ ] `grep -r "apps/traefik" docs/`
  - [ ] `grep -r "apps/synology-csi" docs/`
  - [ ] Update any found references

- [ ] Verify Git history preserved:
  - [ ] `git log --follow apps/infrastructure/traefik/`
  - [ ] Confirm history shows pre-move commits

- [ ] Delete backup branch if validation successful:
  - [ ] `git branch -d backup/pre-file-structure-refactor`
  - [ ] `git push origin --delete backup/pre-file-structure-refactor`
