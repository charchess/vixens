# Implementation Tasks

## 1. Pre-Validation
- [ ] 1.1 Verify synology-csi pods are running in dev
- [ ] 1.2 Verify CSI driver `csi.san.synology.com` is registered
- [ ] 1.3 Verify existing homeassistant PVCs are bound and operational
- [ ] 1.4 Search all ArgoCD overlays for references to synology-csi-talos

## 2. Cleanup Execution
- [ ] 2.1 Remove `apps/synology-csi-talos/` directory
- [ ] 2.2 Verify no git references remain (grep search)
- [ ] 2.3 Commit removal with clear message

## 3. Post-Validation
- [ ] 3.1 Verify homeassistant PVCs still bound and accessible
- [ ] 3.2 Test new PVC creation (create test PVC, verify LUN appears on Synology)
- [ ] 3.3 Delete test PVC and verify cleanup

## 4. Documentation
- [ ] 4.1 Update CLAUDE.md to remove confusion about two implementations
- [ ] 4.2 Document which CSI driver version is used (v1.1.1 ghcr.io/zebernst/synology-csi:v1.2.0)
