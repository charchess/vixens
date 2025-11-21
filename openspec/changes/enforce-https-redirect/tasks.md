# Tasks - Enforce HTTPS Redirect on All Ingress Resources

## Phase 1: Create Traefik Middleware

- [ ] Create middleware resource:
  - [ ] Create file `apps/traefik/base/middleware-redirect-https.yaml`
  - [ ] Define Middleware CRD with redirectScheme to https (permanent: true)
  - [ ] Set namespace to `traefik`

- [ ] Update Traefik kustomization:
  - [ ] Edit `apps/traefik/base/kustomization.yaml`
  - [ ] Add `middleware-redirect-https.yaml` to resources list

- [ ] Commit and deploy (dev):
  - [ ] `git add apps/traefik/base/`
  - [ ] `git commit -m "feat(traefik): Add HTTPS redirect middleware"`
  - [ ] `git push origin dev`
  - [ ] Wait for ArgoCD sync

- [ ] Validate middleware deployed:
  - [ ] `kubectl get middleware -n traefik redirect-https`
  - [ ] Verify status is Ready

## Phase 2: Test with homeassistant (Pilot)

- [ ] Add annotation to homeassistant ingress:
  - [ ] Edit `apps/homeassistant/base/ingress.yaml`
  - [ ] Add annotation: `traefik.ingress.kubernetes.io/router.middlewares: traefik-redirect-https@kubernetescrd`

- [ ] Commit and deploy:
  - [ ] `git add apps/homeassistant/base/ingress.yaml`
  - [ ] `git commit -m "feat(homeassistant): Enable HTTPS redirect"`
  - [ ] `git push origin dev`
  - [ ] Wait for ArgoCD sync

- [ ] Test HTTP redirect:
  - [ ] `curl -I http://homeassistant.dev.truxonline.com`
  - [ ] Verify response: `301 Moved Permanently`
  - [ ] Verify Location header: `https://homeassistant.dev.truxonline.com`

- [ ] Test HTTPS still works:
  - [ ] `curl -I https://homeassistant.dev.truxonline.com`
  - [ ] Verify response: `200 OK`
  - [ ] Test in browser: https://homeassistant.dev.truxonline.com
  - [ ] Verify application loads correctly

## Phase 3: Rollout to Remaining Applications

- [ ] Add annotation to mail-gateway:
  - [ ] Edit `apps/mail-gateway/base/ingress.yaml`
  - [ ] Add traefik.ingress.kubernetes.io/router.middlewares annotation

- [ ] Add annotation to traefik-dashboard:
  - [ ] Edit `apps/traefik-dashboard/base/ingress.yaml`
  - [ ] Add traefik.ingress.kubernetes.io/router.middlewares annotation

- [ ] Update argocd (consolidate):
  - [ ] Edit `apps/argocd/base/ingress.yaml`
  - [ ] Add traefik.ingress.kubernetes.io/router.middlewares annotation
  - [ ] Delete `apps/argocd/base/ingress-redirect.yaml` (no longer needed)
  - [ ] Update `apps/argocd/base/kustomization.yaml` to remove redirect ingress

- [ ] Update whoami (consolidate):
  - [ ] Edit `apps/whoami/base/ingress.yaml`
  - [ ] Add traefik.ingress.kubernetes.io/router.middlewares annotation
  - [ ] Delete `apps/whoami/base/ingress-redirect.yaml` (no longer needed)
  - [ ] Update `apps/whoami/base/kustomization.yaml` to remove redirect ingress

- [ ] Commit all changes:
  - [ ] `git add apps/*/base/`
  - [ ] `git commit -m "feat(ingress): Enforce HTTPS redirect on all applications"`
  - [ ] `git push origin dev`

## Phase 4: Validation (Dev Environment)

- [ ] Wait for ArgoCD sync:
  - [ ] Monitor: `kubectl get application -n argocd -w`
  - [ ] All apps should remain Healthy

- [ ] Test HTTP redirect for all apps:
  - [ ] `curl -I http://homeassistant.dev.truxonline.com` → 301
  - [ ] `curl -I http://mail.dev.truxonline.com` → 301
  - [ ] `curl -I http://traefik.dev.truxonline.com` → 301
  - [ ] `curl -I http://argocd.dev.truxonline.com` → 301
  - [ ] `curl -I http://whoami.dev.truxonline.com` → 301

- [ ] Test HTTPS access for all apps:
  - [ ] https://homeassistant.dev.truxonline.com → 200 OK
  - [ ] https://mail.dev.truxonline.com → 200 OK
  - [ ] https://traefik.dev.truxonline.com → 200 OK
  - [ ] https://argocd.dev.truxonline.com → 200 OK
  - [ ] https://whoami.dev.truxonline.com → 200 OK

- [ ] Browser validation:
  - [ ] Type http://homeassistant.dev.truxonline.com in browser
  - [ ] Verify automatic redirect to HTTPS (check URL bar)
  - [ ] Repeat for other applications

## Phase 5: Multi-Environment Rollout (Test)

- [ ] Merge dev → test:
  - [ ] Create PR from dev to test
  - [ ] Wait for CI validation
  - [ ] Merge PR

- [ ] Validate test environment:
  - [ ] Test HTTP redirects for all apps (*.test.truxonline.com)
  - [ ] Test HTTPS access for all apps
  - [ ] Verify ArgoCD applications Healthy

## Phase 6: Multi-Environment Rollout (Staging)

- [ ] Merge test → staging:
  - [ ] Create PR from test to staging
  - [ ] Merge PR

- [ ] Validate staging environment:
  - [ ] Test HTTP redirects (*.staging.truxonline.com)
  - [ ] Test HTTPS access
  - [ ] Verify applications Healthy

## Phase 7: Multi-Environment Rollout (Prod)

- [ ] Merge staging → main:
  - [ ] Create PR from staging to main
  - [ ] Request review (self-review acceptable)
  - [ ] Merge PR

- [ ] Validate prod environment:
  - [ ] Test HTTP redirects (*.truxonline.com)
  - [ ] Test HTTPS access
  - [ ] Verify applications Healthy
  - [ ] Monitor for 24h for any issues

## Phase 8: Documentation

- [ ] Update CLAUDE.md:
  - [ ] Add "HTTPS Redirect" section under Security
  - [ ] Document Traefik middleware approach
  - [ ] Document annotation pattern for new apps

- [ ] Update app README files:
  - [ ] Document HTTPS redirect behavior
  - [ ] Note that HTTP access automatically redirects

---

## Notes

**Estimated Time:**
- Phase 1-4 (dev): 1 hour
- Phase 5-7 (multi-env): 30 minutes per environment
- Phase 8 (docs): 20 minutes

**Testing Tools:**
```bash
# Quick test all apps
for app in homeassistant mail traefik argocd whoami; do
  echo "Testing $app..."
  curl -I http://$app.dev.truxonline.com 2>&1 | grep -E "HTTP|Location"
done
```

**Rollback:**
- Remove middleware annotation from ingress resources
- Git revert if needed
- Restore separate redirect ingress for argocd/whoami if critical
