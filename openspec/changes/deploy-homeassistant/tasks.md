# Tasks - Deploy Home Assistant

## Phase 1: Preparation and Research

- [ ] Document current Home Assistant setup:
  - [ ] Identify current installation location (VM, bare metal, Docker)
  - [ ] Document configuration directory size
  - [ ] List active integrations (especially USB devices like Zigbee/Z-Wave)
  - [ ] Document network dependencies (mDNS, multicast, etc.)
  - [ ] Identify any add-ons that need Kubernetes equivalents

- [ ] Research Home Assistant on Kubernetes:
  - [ ] Review official Home Assistant container documentation
  - [ ] Research USB device passthrough strategies (if needed)
  - [ ] Document WebSocket requirements for frontend
  - [ ] Identify recommended resource limits

## Phase 2: Base Application Structure

- [ ] Create directory structure:
  - [ ] `mkdir -p apps/applications/homeassistant/base`
  - [ ] `mkdir -p apps/applications/homeassistant/overlays/{dev,test,staging,prod}`

- [ ] Create base manifests:
  - [ ] Create `base/namespace.yaml` - homeassistant namespace
  - [ ] Create `base/pvc.yaml` - Persistent volume for /config (10Gi, synology-iscsi-storage)
  - [ ] Create `base/deployment.yaml` - Home Assistant deployment
    - [ ] Container: ghcr.io/home-assistant/home-assistant:stable
    - [ ] Volume mount: /config from PVC
    - [ ] Port: 8123
    - [ ] Liveness/readiness probes
  - [ ] Create `base/service.yaml` - ClusterIP service on port 8123
  - [ ] Create `base/kustomization.yaml` - Include all base resources
  - [ ] Create `base/README.md` - Document deployment architecture

## Phase 3: Dev Environment Overlay

- [ ] Create dev-specific configuration:
  - [ ] Create `overlays/dev/kustomization.yaml`
    - [ ] Reference base
    - [ ] Include patches and ingress
  - [ ] Create `overlays/dev/patches.yaml`
    - [ ] Resource limits: 500m CPU, 1Gi memory
    - [ ] Image tag: stable (or specific version)
  - [ ] Create `overlays/dev/ingress.yaml`
    - [ ] Host: homeassistant.dev.truxonline.com
    - [ ] TLS: cert-manager ClusterIssuer letsencrypt-prod
    - [ ] Traefik annotations for WebSocket support

## Phase 4: ArgoCD Integration (Dev)

- [ ] Create ArgoCD Application:
  - [ ] Create `argocd/overlays/dev/apps/homeassistant.yaml`
    - [ ] Wave 4 (applications)
    - [ ] Path: apps/applications/homeassistant/overlays/dev
    - [ ] Auto-sync enabled
    - [ ] CreateNamespace: true
  - [ ] Update `argocd/overlays/dev/kustomization.yaml` to include homeassistant.yaml

## Phase 5: Deployment and Validation (Dev)

- [ ] Deploy to dev environment:
  - [ ] Git commit and push to dev branch
  - [ ] Monitor ArgoCD sync: `kubectl get application homeassistant -n argocd`
  - [ ] Wait for sync to complete

- [ ] Verify deployment:
  - [ ] Check pod status: `kubectl get pods -n homeassistant`
  - [ ] Check PVC bound: `kubectl get pvc -n homeassistant`
  - [ ] Check logs: `kubectl logs -n homeassistant -l app=homeassistant`
  - [ ] Verify service: `kubectl get svc -n homeassistant`
  - [ ] Verify ingress: `kubectl get ingress -n homeassistant`

- [ ] Test access:
  - [ ] Access https://homeassistant.dev.truxonline.com
  - [ ] Complete Home Assistant initial setup wizard
  - [ ] Verify WebSocket connection works
  - [ ] Test basic functionality (create automation, check logs)
  - [ ] Verify persistence (restart pod, check configuration persists)

## Phase 6: Test Environment

- [ ] Create test overlay:
  - [ ] Copy dev overlay structure to `overlays/test/`
  - [ ] Update ingress hostname to `homeassistant.test.truxonline.com`
  - [ ] Adjust resources if needed

- [ ] Create ArgoCD Application for test:
  - [ ] Create `argocd/overlays/test/apps/homeassistant.yaml`
  - [ ] Update test kustomization.yaml

- [ ] Deploy and validate in test environment

## Phase 7: Staging Environment

- [ ] Create staging overlay:
  - [ ] Copy test overlay structure to `overlays/staging/`
  - [ ] Update ingress hostname to `homeassistant.staging.truxonline.com`
  - [ ] Increase resources (750m CPU, 1.5Gi memory)

- [ ] Create ArgoCD Application for staging
- [ ] Deploy and validate in staging environment

## Phase 8: Production Environment

- [ ] Create production overlay:
  - [ ] Copy staging overlay structure to `overlays/prod/`
  - [ ] Update ingress hostname to `homeassistant.truxonline.com`
  - [ ] Production resources: 1 CPU, 2Gi memory
  - [ ] Add resource requests/limits
  - [ ] Consider PVC size increase (50Gi)
  - [ ] Pin image tag to specific version (not :stable)

- [ ] Create ArgoCD Application for prod:
  - [ ] Create `argocd/overlays/prod/apps/homeassistant.yaml`
  - [ ] Update prod kustomization.yaml

## Phase 9: Migration (If Applicable)

**Only if migrating from existing Home Assistant instance:**

- [ ] Backup existing configuration:
  - [ ] Tar entire Home Assistant config directory
  - [ ] Store backup securely (S3, NAS)

- [ ] Copy configuration to Kubernetes PVC:
  - [ ] Create temporary pod with PVC mounted
  - [ ] `kubectl cp` backup to temporary pod
  - [ ] Extract tar in /config
  - [ ] Verify ownership and permissions

- [ ] Test migration in dev:
  - [ ] Restart Home Assistant pod
  - [ ] Verify all integrations load
  - [ ] Check automations still work
  - [ ] Validate database integrity

- [ ] Blue/green deployment:
  - [ ] Keep old instance running
  - [ ] Update DNS to point to new ingress (or use /etc/hosts for testing)
  - [ ] Validate full functionality for 24-48 hours
  - [ ] Decommission old instance only after successful validation

## Phase 10: Documentation

- [ ] Update CLAUDE.md:
  - [ ] Add Home Assistant to applications list
  - [ ] Document ingress URLs for all environments

- [ ] Update README.md:
  - [ ] Add Home Assistant to deployed applications

- [ ] Create operational documentation:
  - [ ] Create `apps/applications/homeassistant/base/README.md`
    - [ ] Document deployment architecture
    - [ ] Explain persistent storage strategy
    - [ ] Document USB device passthrough (if applicable)
    - [ ] Add troubleshooting section

- [ ] Create runbook:
  - [ ] Document backup/restore procedures
  - [ ] Document configuration migration steps
  - [ ] Document pod restart procedures
  - [ ] Add common troubleshooting scenarios

## Phase 11: Advanced Configuration (Optional)

- [ ] USB device passthrough (if needed for Zigbee/Z-Wave):
  - [ ] Document USB device path on host
  - [ ] Add hostPath volume mount
  - [ ] Set privileged security context
  - [ ] Test device access from container

- [ ] Network discovery (if needed):
  - [ ] Test with hostNetwork: true
  - [ ] Validate mDNS discovery works
  - [ ] Document any limitations

- [ ] Backup automation:
  - [ ] Configure Velero backup for homeassistant namespace
  - [ ] Test restore from backup
  - [ ] Document backup schedule
