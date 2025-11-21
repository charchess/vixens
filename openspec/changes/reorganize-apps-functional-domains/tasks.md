# Tasks - Reorganize Apps into Functional Domains

## Phase 1: Preparation and Validation

- [ ] Document current state:
  - [ ] List all apps in `apps/` directory
  - [ ] Document which apps are deployed (ArgoCD Applications)
  - [ ] Identify obsolete code (`synology-csi-talos`)
  - [ ] Map current ArgoCD Application paths

- [ ] Validate migration safety:
  - [ ] Check for hardcoded paths in scripts
  - [ ] Check for hardcoded paths in documentation (CLAUDE.md, README.md, docs/)
  - [ ] Verify ArgoCD can handle moved directories (test with one app)
  - [ ] Confirm Kustomize paths are relative (will work after move)

- [ ] Create migration plan:
  - [ ] List all directory moves (old → new)
  - [ ] List all ArgoCD Application updates needed
  - [ ] List all documentation updates needed

## Phase 2: Create New Directory Structure

- [ ] Create domain directories:
  - [ ] `mkdir -p apps/00-infra`
  - [ ] `mkdir -p apps/01-storage`
  - [ ] `mkdir -p apps/02-monitoring`
  - [ ] `mkdir -p apps/03-security`
  - [ ] `mkdir -p apps/10-home`
  - [ ] `mkdir -p apps/20-media`
  - [ ] `mkdir -p apps/30-productivity`
  - [ ] `mkdir -p apps/40-network`
  - [ ] `mkdir -p apps/50-backup`
  - [ ] `mkdir -p apps/99-test`

## Phase 3: Move Applications (00-infra)

- [ ] Move infrastructure apps:
  - [ ] `git mv apps/argocd apps/00-infra/argocd`
  - [ ] `git mv apps/cert-manager apps/00-infra/cert-manager`
  - [ ] `git mv apps/cert-manager-webhook-gandi apps/00-infra/cert-manager-webhook-gandi`
  - [ ] `git mv apps/cilium-lb apps/00-infra/cilium-lb`
  - [ ] `git mv apps/traefik apps/00-infra/traefik`
  - [ ] `git mv apps/traefik-dashboard apps/00-infra/traefik-dashboard`

- [ ] Update ArgoCD Applications for 00-infra:
  - [ ] Update `argocd/overlays/dev/apps/argocd.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/cert-manager.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/cert-manager-webhook-gandi.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/cilium-lb.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/traefik.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/traefik-dashboard.yaml` path

## Phase 4: Move Applications (01-storage)

- [ ] Move storage apps:
  - [ ] `git mv apps/synology-csi apps/01-storage/synology-csi`
  - [ ] `git mv apps/nfs-storage apps/01-storage/nfs-storage`

- [ ] Update ArgoCD Applications for 01-storage:
  - [ ] Update `argocd/overlays/dev/apps/synology-csi.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/synology-csi-secrets.yaml` path (if exists)
  - [ ] Update `argocd/overlays/dev/apps/nfs-storage.yaml` path

## Phase 5: Move Applications (10-home, 40-network, 99-test)

- [ ] Move home domain apps:
  - [ ] `git mv apps/homeassistant apps/10-home/homeassistant`

- [ ] Move network domain apps:
  - [ ] `git mv apps/mail-gateway apps/40-network/mail-gateway`

- [ ] Move test utilities:
  - [ ] `git mv apps/whoami apps/99-test/whoami`

- [ ] Update ArgoCD Applications:
  - [ ] Update `argocd/overlays/dev/apps/homeassistant.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/mail-gateway.yaml` path
  - [ ] Update `argocd/overlays/dev/apps/whoami.yaml` path

## Phase 6: Remove Obsolete Code

- [ ] Delete obsolete synology-csi-talos:
  - [ ] Verify not deployed: `kubectl get application synology-csi-talos -n argocd` (should not exist)
  - [ ] Verify no PVCs using it: `kubectl get pvc -A` (check provisioner)
  - [ ] Verify no references in docs: `rg synology-csi-talos docs/ CLAUDE.md`
  - [ ] Delete directory: `git rm -rf apps/synology-csi-talos`

- [ ] Clean up any ArgoCD artifacts:
  - [ ] Check for old Application manifests referencing deleted apps
  - [ ] Remove if found

## Phase 7: Update Documentation

- [ ] Update CLAUDE.md:
  - [ ] Add "Repository Structure" section explaining new organization
  - [ ] Update app paths in examples (if any)
  - [ ] Add numeric prefix legend (00=infra, 01=storage, etc.)

- [ ] Update README.md:
  - [ ] Update directory tree showing new structure
  - [ ] Add explanation of numeric prefixes
  - [ ] Update any app-specific links

- [ ] Update docs/:
  - [ ] Search for hardcoded app paths: `rg "apps/[a-z]" docs/`
  - [ ] Update paths to new structure
  - [ ] Verify architecture diagrams don't reference old paths

## Phase 8: Validation (Dev Environment)

- [ ] Git commit and push:
  - [ ] Create commit with all moves and updates
  - [ ] Push to dev branch

- [ ] Monitor ArgoCD sync:
  - [ ] Watch ArgoCD Applications: `kubectl get application -n argocd -w`
  - [ ] Wait for all apps to sync
  - [ ] Check for any sync errors or degraded health

- [ ] Verify app health:
  - [ ] Check all apps are Healthy: `kubectl get application -n argocd`
  - [ ] Check all pods running: `kubectl get pods -A | grep -E "(cert-manager|traefik|homeassistant|synology)"`
  - [ ] Test critical ingresses:
    - [ ] https://homeassistant.dev.truxonline.com
    - [ ] https://traefik.dev.truxonline.com
    - [ ] https://argocd.dev.truxonline.com
    - [ ] https://whoami.dev.truxonline.com

- [ ] Verify ArgoCD paths:
  - [ ] Check Application source paths: `kubectl get application -n argocd -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.source.path}{"\n"}{end}'`
  - [ ] Confirm all paths use new structure (00-infra, 01-storage, etc.)

## Phase 9: Multi-Environment Rollout (Test)

- [ ] Apply to test environment:
  - [ ] Update test ArgoCD Applications
  - [ ] Push to test branch (if separate)
  - [ ] Monitor sync and health
  - [ ] Validate critical apps

## Phase 10: Multi-Environment Rollout (Staging)

- [ ] Apply to staging environment:
  - [ ] Update staging ArgoCD Applications
  - [ ] Push to staging branch (if separate)
  - [ ] Monitor sync and health
  - [ ] Validate critical apps

## Phase 11: Multi-Environment Rollout (Prod)

- [ ] Apply to prod environment:
  - [ ] Update prod ArgoCD Applications
  - [ ] Push to main branch
  - [ ] Monitor sync and health carefully
  - [ ] Validate all production apps
  - [ ] Test production ingresses

## Phase 12: Cleanup and Documentation

- [ ] Verify no broken references:
  - [ ] Search for old paths: `rg "apps/(argocd|cert-manager|homeassistant|traefik|synology-csi)" .`
  - [ ] Fix any remaining references

- [ ] Update ROADMAP.md:
  - [ ] Document completion of repository reorganization
  - [ ] Note obsolete code removed (synology-csi-talos)

- [ ] Create migration runbook:
  - [ ] Document migration procedure for future reference
  - [ ] Add to `docs/procedures/app-directory-migration.md`
  - [ ] Include rollback steps

## Phase 13: Archive Old OpenSpec

- [ ] Archive refactor-file-structure-groups:
  - [ ] This proposal replaces the previous refactor proposal
  - [ ] Archive old OpenSpec: `openspec archive refactor-file-structure-groups --skip-specs --yes`
  - [ ] Reference this proposal in archive notes

---

## 📝 Migration Notes

**Critical Safety Checks:**
- Git tracks file moves (`git mv`), so history is preserved
- ArgoCD syncs based on paths in Application manifests
- Kustomize uses relative paths, works after directory moves
- No namespace changes, so no pod recreation

**Rollback is Simple:**
- `git revert <commit-with-moves>`
- ArgoCD will detect reverted paths and resync
- All apps return to previous state

**Estimated Time:**
- Preparation: 30 minutes
- Migration (dev): 2 hours
- Validation: 1 hour
- Multi-env: 1 hour per environment
