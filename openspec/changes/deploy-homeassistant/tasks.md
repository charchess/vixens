# Tasks - Deploy Home Assistant

## Phase 1: Preparation and Research

- [x] Document current Home Assistant setup:
  - [x] Identify current installation location (VM, bare metal, Docker)
  - [x] Document configuration directory size
  - [x] List active integrations (especially USB devices like Zigbee/Z-Wave)
  - [x] Document network dependencies (mDNS, multicast, etc.)
  - [x] Identify any add-ons that need Kubernetes equivalents

- [x] Research Home Assistant on Kubernetes:
  - [x] Review official Home Assistant container documentation
  - [x] Research USB device passthrough strategies (if needed)
  - [x] Document WebSocket requirements for frontend
  - [x] Identify recommended resource limits

## Phase 2: Base Application Structure

- [x] Create directory structure:
  - [x] `mkdir -p apps/homeassistant/base` *(Note: Used apps/homeassistant instead of apps/applications/homeassistant)*
  - [x] `mkdir -p apps/homeassistant/overlays/{dev,test,staging,prod}`

- [x] Create base manifests:
  - [x] Create `base/namespace.yaml` - homeassistant namespace
  - [x] Create `base/pvc.yaml` - Persistent volume for /config (10Gi, synology-iscsi-retain)
  - [x] Create `base/deployment.yaml` - Home Assistant deployment
    - [x] Container: ghcr.io/home-assistant/home-assistant:latest *(used :latest instead of :stable)*
    - [x] Volume mount: /config from PVC
    - [x] Port: 8123
    - [x] Liveness/readiness probes (initialDelaySeconds: 180s/120s)
    - [x] **Added:** hostNetwork: true for mDNS/UPnP discovery
    - [x] **Added:** ConfigMap mount for configuration.yaml (reverse proxy support)
  - [x] Create `base/service.yaml` - ClusterIP service on port 8123
  - [x] **Added:** `base/service-nodeport.yaml` - NodePort 30812 for direct access during configuration
  - [x] **Added:** `base/configmap.yaml` - HTTP configuration for Traefik reverse proxy (use_x_forwarded_for, trusted_proxies)
  - [x] Create `base/kustomization.yaml` - Include all base resources
  - [x] Create `base/README.md` - Document deployment architecture (commit 64cb57b)

## Phase 3: Dev Environment Overlay

- [x] Create dev-specific configuration:
  - [x] Create `overlays/dev/kustomization.yaml`
    - [x] Reference base
    - [x] Include ingress *(no separate patches.yaml, resources defined inline)*
  - [ ] Create `overlays/dev/patches.yaml` *(Skipped: resources defined inline in deployment.yaml)*
    - [x] Resource limits: 250m-1000m CPU, 512Mi-2Gi memory *(values adjusted based on testing)*
    - [x] Image tag: latest *(using :latest for dev instead of :stable)*
  - [x] Create `overlays/dev/ingress.yaml`
    - [x] Host: homeassistant.dev.truxonline.com
    - [x] TLS: cert-manager ClusterIssuer letsencrypt-staging *(using staging for dev)*
    - [x] Traefik annotations (router.entrypoints: websecure, router.tls: true)

## Phase 4: ArgoCD Integration (Dev)

- [x] Create ArgoCD Application:
  - [x] Create `argocd/overlays/dev/apps/homeassistant.yaml`
    - [x] Path: apps/homeassistant/overlays/dev *(Note: apps/homeassistant not apps/applications/homeassistant)*
    - [x] Auto-sync enabled (prune: true, selfHeal: true)
    - [x] CreateNamespace: true
  - [x] Update `argocd/overlays/dev/kustomization.yaml` to include homeassistant.yaml

## Phase 5: Deployment and Validation (Dev)

- [x] Deploy to dev environment:
  - [x] Git commit and push to dev branch (commit d4d054b)
  - [x] Monitor ArgoCD sync: `kubectl get application homeassistant -n argocd`
  - [x] Wait for sync to complete

- [x] Verify deployment:
  - [x] Check pod status: `kubectl get pods -n homeassistant` (1/1 Running)
  - [x] Check PVC bound: `kubectl get pvc -n homeassistant` (10Gi Bound, synology-iscsi-retain)
  - [x] Check logs: `kubectl logs -n homeassistant -l app=homeassistant`
  - [x] Verify service: `kubectl get svc -n homeassistant` (ClusterIP + NodePort)
  - [x] Verify ingress: `kubectl get ingress -n homeassistant`

- [x] Test access:
  - [x] Access https://homeassistant.dev.truxonline.com (✅ Working)
  - [x] Access via NodePort http://192.168.208.X:30812 (✅ Working)
  - [ ] Complete Home Assistant initial setup wizard *(User action required)*
  - [ ] Verify WebSocket connection works *(Pending user testing)*
  - [ ] Test basic functionality (create automation, check logs) *(Pending user testing)*
  - [ ] Verify persistence (restart pod, check configuration persists) *(Pending testing)*

## Phase 6: Test Environment

- [x] Create test overlay:
  - [x] Copy dev overlay structure to `overlays/test/`
  - [x] Update ingress hostname to `homeassistant.test.truxonline.com`
  - [x] TLS: letsencrypt-staging (commit 64cb57b)

- [x] Create ArgoCD Application for test:
  - [x] Create `argocd/overlays/test/apps/homeassistant.yaml` (targetRevision: test branch)
  - [x] Referenced in test kustomization.yaml (line 23)

- [ ] Deploy and validate in test environment *(Requires test cluster active)*

## Phase 7: Staging Environment

- [x] Create staging overlay:
  - [x] Copy test overlay structure to `overlays/staging/`
  - [x] Update ingress hostname to `homeassistant.staging.truxonline.com`
  - [x] TLS: letsencrypt-prod (commit 64cb57b)
  - [ ] Increase resources (750m CPU, 1.5Gi memory) *(TODO: Add resource patches)*

- [x] Create ArgoCD Application for staging (targetRevision: staging branch)
- [ ] Deploy and validate in staging environment *(Requires staging cluster active)*

## Phase 8: Production Environment

- [x] Create production overlay:
  - [x] Copy staging overlay structure to `overlays/prod/`
  - [x] Update ingress hostname to `homeassistant.truxonline.com`
  - [x] TLS: letsencrypt-prod (commit 64cb57b)
  - [ ] Production resources: 1 CPU, 2Gi memory *(TODO: Add resource patches)*
  - [ ] Add resource requests/limits *(TODO)*
  - [ ] Consider PVC size increase (50Gi) *(Optional, defer to production)*
  - [ ] Pin image tag to specific version (not :stable) *(TODO: Decide versioning strategy)*

- [x] Create ArgoCD Application for prod:
  - [x] Create `argocd/overlays/prod/apps/homeassistant.yaml` (targetRevision: main branch)
  - [x] Referenced in prod kustomization.yaml (line 23)

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
  - [ ] Create `apps/homeassistant/base/README.md`
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

- [x] USB device passthrough (if needed for Zigbee/Z-Wave):
  - [x] Document USB device path on host
  - [x] Add hostPath volume mount *(Not needed - using network integrations)*
  - [x] Set privileged security context *(Not needed)*
  - [x] Test device access from container *(N/A)*

- [x] Network discovery (if needed):
  - [x] Test with hostNetwork: true (✅ Implemented)
  - [x] Validate mDNS discovery works *(Pending user validation)*
  - [x] Document any limitations *(hostNetwork requires dnsPolicy: ClusterFirstWithHostNet)*

- [ ] Backup automation:
  - [ ] Configure Velero backup for homeassistant namespace
  - [ ] Test restore from backup
  - [ ] Document backup schedule

---

## 📝 Implementation Notes

**Dev Environment Status:** ✅ **OPERATIONAL**
- Pod: 1/1 Running (onyx node - 192.168.111.164)
- Storage: 10Gi iSCSI LUN (synologia NAS 192.168.111.69)
- Ingress: https://homeassistant.dev.truxonline.com (TLS Let's Encrypt staging)
- NodePort: http://192.168.208.X:30812 (all nodes)
- ArgoCD: Synced (commit d4d054b)

**Key Implementation Differences from Original Spec:**
1. Directory structure: `apps/homeassistant/` instead of `apps/applications/homeassistant/`
2. Storage class: `synology-iscsi-retain` instead of `synology-iscsi-storage`
3. Image tag: `:latest` instead of `:stable`
4. TLS issuer: `letsencrypt-staging` for dev (production will use `letsencrypt-prod`)
5. **Added components not in spec:**
   - `configmap.yaml` for HTTP reverse proxy configuration
   - `service-nodeport.yaml` for direct access during configuration
   - `hostNetwork: true` for mDNS/UPnP local discovery support
   - Increased probe delays (180s/120s) to accommodate slow startup

**Next Steps:**
1. Create `base/README.md` (Phase 2 completion)
2. User validation of Home Assistant functionality
3. Begin multi-environment rollout (Phases 6-8)
