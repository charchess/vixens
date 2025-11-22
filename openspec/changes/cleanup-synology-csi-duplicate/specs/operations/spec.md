# Operations Specification Delta

## REMOVED Requirements

### Requirement: Maintain synology-csi-talos alternative implementation

**Reason**: Directory `apps/synology-csi-talos/` is not deployed, not referenced by ArgoCD, and creates maintenance overhead. The standard `apps/synology-csi` implementation (ghcr.io/zebernst/synology-csi:v1.2.0) works correctly on Talos Linux without requiring a separate "talos" variant.

**Migration**: No migration needed - directory was never deployed to any environment. Verification confirms `apps/synology-csi` is the active implementation with DaemonSet running on all 3 control plane nodes (obsy, onyx, opale) and StatefulSet controller operational.

**Cleanup Actions**:
- Remove `apps/synology-csi-talos/base/` directory
- Remove `apps/synology-csi-talos/overlays/dev/` directory
- Verify no ArgoCD Application references the removed path
- Document that `apps/synology-csi` is the canonical CSI driver implementation

## ADDED Requirements

### Requirement: Single Synology CSI implementation

The repository SHALL maintain only one Synology CSI driver implementation to reduce confusion and maintenance overhead.

#### Scenario: CSI driver uniquely identified
- **GIVEN** developer needs to modify CSI driver configuration
- **WHEN** developer searches for CSI implementation
- **THEN** only `apps/synology-csi/` SHALL exist
- **AND** no alternative "talos" variant SHALL be present

#### Scenario: Documentation clarifies CSI choice
- **GIVEN** developer reads CLAUDE.md or README
- **WHEN** CSI driver is mentioned
- **THEN** documentation SHALL specify `apps/synology-csi` uses ghcr.io/zebernst/synology-csi:v1.2.0
- **AND** documentation SHALL note this image is Talos-compatible without requiring separate variant
