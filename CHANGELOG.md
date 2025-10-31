# Changelog

All notable changes to the Vixens infrastructure project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to semantic versioning for infrastructure versions.

## [Sprint 1] - 2025-10-31 - COMPLETED ✅

### Added
- **Terraform Module Talos** (`terraform/modules/talos/`)
  - Per-node configuration structure for control planes and workers
  - Dual-VLAN automatic configuration (internal 111 + services 20X)
  - VIP automatic extraction and configuration on VLAN 111
  - Hostname automatic configuration for all nodes
  - certSANs with VIP in API server certificates
  - Worker node support (separate from control planes)
  - Automatic node reset provisioner on destroy
  - Odd control plane validation for etcd quorum
  - Custom Talos image support (factory/schematic)
  - Local kubeconfig and talosconfig generation

- **Dev Environment** (`terraform/environments/dev/`)
  - Control plane: obsy (192.168.111.162)
  - Worker: onyx (192.168.111.164)
  - VIP: 192.168.111.160
  - Complete infrastructure as code

- **Documentation**
  - CLAUDE.md comprehensive guide
  - Sprint 1 completion report (`docs/sprints/sprint-1-completed.md`)
  - CHANGELOG.md (this file)

### Changed
- Updated module to support both control plane and worker nodes
- Enhanced network configuration with automatic VIP placement
- Improved destroy/recreate workflow with automatic node reset

### Validated
- ✅ Terraform apply/destroy/recreate cycle
- ✅ VIP High Availability configuration
- ✅ Hostname persistence across recreates
- ✅ Infrastructure idempotence (terraform plan = no changes)
- ✅ Kubernetes cluster functionality (v1.34.0)
- ✅ Talos Linux operation (v1.11.0)

### Technical Details
- **VIP**: 192.168.111.160/32 on VLAN 111
- **Kubernetes**: v1.34.0 with 9 system pods
- **Talos**: v1.11.0 immutable OS
- **Terraform**: Module-based architecture
- **Performance**: ~2.5 minutes full destroy/recreate cycle

### Archon Tasks
- ✅ Task 1.1: Module structure
- ✅ Task 1.2: Configure dev environment
- ✅ Task 1.3: Apply Terraform and provision
- ✅ Task 1.4: Validate cluster access
- ✅ Bonus: VIP configuration
- ✅ Bonus: Hostname configuration
- ✅ Bonus: Worker node support
- ✅ Bonus: Destroy/recreate validation

## [Upcoming] - Sprint 2 - Cilium CNI

### Planned
- Configure Helm provider in Terraform
- Deploy Cilium v1.16.5 via Terraform Helm
- Enable kube-proxy replacement
- Configure Hubble observability (relay + UI)
- Validate network connectivity with Cilium tests

## [Upcoming] - Sprint 3 - Scale to 3 Control Planes HA

### Planned
- Add opale control plane node
- Validate etcd 3-member quorum
- Test HA failover scenarios
- Validate VIP switching between control planes

## [Upcoming] - Sprint 4 - ArgoCD Bootstrap

### Planned
- Deploy ArgoCD via Terraform Helm
- Create App-of-Apps structure
- Establish GitOps workflow
- Configure auto-sync policies

---

## Version History

- **Sprint 1** (2025-10-31): Terraform Module Talos + Dev 2 nodes ✅
- **Sprint 2** (Planned): Cilium CNI
- **Sprint 3** (Planned): 3 Control Planes HA
- **Sprint 4** (Planned): ArgoCD Bootstrap
- **Sprints 5-11** (Planned): Phase 2 Services

## Project Status

**Current Phase**: Phase 1 - Infrastructure as Code
**Current Sprint**: Sprint 1 ✅ → Sprint 2 ⏳
**Active Environment**: Dev (obsy + onyx)
**Next Milestone**: Cilium CNI deployment
