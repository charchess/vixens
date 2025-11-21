# Cleanup Synology CSI Duplicate Implementation

## Why

The repository contains two Synology CSI driver implementations (`apps/synology-csi` and `apps/synology-csi-talos`), creating confusion and maintenance overhead. Only one implementation is actively deployed and functional in dev environment.

Investigation shows that **`apps/synology-csi`** (version v1.1.1) is the active implementation deployed via ArgoCD, with 3 DaemonSet pods running successfully on all nodes and 1 StatefulSet controller pod operational.

The `apps/synology-csi-talos` directory contains an alternative implementation that is not deployed and appears to be an abandoned experiment or backup approach.

## What Changes

- **REMOVED**: `apps/synology-csi-talos/` directory and all its contents
- **UPDATED**: Documentation to clarify which CSI driver is used
- **VALIDATED**: Verify no ArgoCD Applications reference the removed directory

## Impact

- **Affected specs**: operations (cleanup)
- **Affected code**:
  - Remove `apps/synology-csi-talos/` (4 files: base/, overlays/dev/)
  - Verify `argocd/overlays/*/apps/` have no references to synology-csi-talos
- **Benefits**: Clearer repository structure, reduced confusion, easier maintenance
- **Risk**: None - directory is not deployed or referenced

## Validation

Before removal:
- ✅ Confirm `apps/synology-csi` is deployed (DaemonSet + StatefulSet running)
- ✅ Confirm `csi.san.synology.com` CSI driver is registered
- ✅ Confirm no ArgoCD Applications reference synology-csi-talos

After removal:
- Verify existing CSI volumes remain operational (homeassistant PVCs)
- Verify new PVC provisioning still works
