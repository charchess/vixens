# Changelog

All notable changes to the Vixens infrastructure project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to semantic versioning for infrastructure versions.

## [Sprint 6] - 2025-11-15 - COMPLETED ✅

### Added
- **cert-manager** (`apps/cert-manager/`)
  - Deployment of cert-manager v1.14.4 via ArgoCD
  - Externalized Helm values (common, dev, test, staging, prod)
  - Production-ready configurations (resources, HA, monitoring)
- **cert-manager-webhook-gandi** (`apps/cert-manager-webhook-gandi/`)
  - Deployment of cert-manager-webhook-gandi v0.5.2
  - Externalized Helm values
- **TLS Certificates**
  - Configuration of Let's Encrypt DNS-01 challenge with Gandi LiveDNS
  - Automatic TLS certificate provisioning for whoami, traefik, argocd
- **Documentation**
  - `CLAUDE.md` updated with Sprint 6 completion and DRY optimization details
  - `CONVENTIONS.md` updated to Version 2.0 (Post-Phase 2 DRY Optimization)
  - `docs/phase2-helm-values-externalization-report.md` detailing Phase 2 completion
  - `docs/argocd-multi-env-migration-report.md` detailing multi-environment migration

### Changed
- ArgoCD applications now use externalized Helm values, eliminating 354 lines of duplication.
- ArgoCD applications now use the multiple sources pattern for Helm charts and Git values.
- `CLAUDE.md` updated to reflect current project status and architecture.
- `CONVENTIONS.md` updated to reflect DRY principles and Helm values structure.

### Fixed
- Resolved issues with Cilium eBPF policies blocking webhooks by implementing explicit NetworkPolicies (see `docs/issues/cilium-network-policy-fix.md`).

### Validated
- ✅ Full GitOps automation for cert-manager and webhook-gandi deployment.
- ✅ Automatic provisioning of Let's Encrypt certificates.
- ✅ Zero downtime during Helm values externalization migration.
- ✅ Production-ready configurations for all environments.

### Technical Details
- **cert-manager**: v1.14.4
- **cert-manager-webhook-gandi**: v0.5.2
- **ArgoCD**: Multiple sources pattern for Helm values.
- **Code Reduction**: 354 lines of duplication eliminated.

### Archon Tasks
- ✅ Sprint 6: cert-manager + Let's Encrypt DNS-01 (Gandi) - COMPLETED

## [Sprint 5] - 2025-11-01 - COMPLETED ✅

### Added
- **Cilium L2 Announcements** (`apps/cilium-lb/`)
  - Replaced MetalLB with native Cilium L2 Announcements for LoadBalancer Services.
  - Deployment of `CiliumLoadBalancerIPPool` and `CiliumL2AnnouncementPolicy`.
- **Traefik Ingress Controller** (`apps/traefik/`)
  - Deployment of Traefik v2.10.5 (Helm v25.0.0) via ArgoCD.
  - Initial configuration for HTTP/HTTPS ingress.
- **Documentation**
  - `docs/adr/005-cilium-l2-announcements.md` detailing the decision to replace MetalLB.
  - `docs/validation/sprint-5-app-of-apps-validation.md` validating App-of-Apps pattern and MetalLB (before replacement).

### Changed
- MetalLB was replaced by Cilium L2 Announcements for LoadBalancer services.
- ArgoCD App-of-Apps pattern validated and implemented.

### Fixed
- Resolved incompatibility issues between MetalLB L2 mode and Cilium VXLAN tunnel mode.

### Validated
- ✅ Cilium L2 Announcements for LoadBalancer services.
- ✅ LoadBalancer IP assignment.
- ✅ Traefik Ingress Controller deployment.
- ✅ ArgoCD App-of-Apps pattern.

### Technical Details
- **Cilium**: v1.18.3 with L2 Announcements + LB IPAM.
- **Traefik**: v2.10.5 (Helm v25.0.0).

### Archon Tasks
- ✅ Sprint 5: Traefik Ingress + Cilium L2 Announcements - COMPLETED

## [Sprint 4] - 2025-11-01 - COMPLETED ✅

### Added
- **ArgoCD Bootstrap**
  - Deployment of ArgoCD v7.7.7 via Terraform Helm provider.
  - Creation of App-of-Apps structure (`argocd/base/`, `argocd/overlays/dev/`).
  - Automation of root-app bootstrap via `kubectl` provider.
- **Documentation**
  - `docs/argocd-deployment.md` detailing ArgoCD deployment.
  - `docs/adr/002-argocd-gitops.md` detailing the decision to use ArgoCD.

### Changed
- Implemented initial GitOps workflow with ArgoCD.

### Validated
- ✅ ArgoCD deployment and self-management.
- ✅ Full GitOps workflow with zero manual `kubectl` commands after initial bootstrap.

### Technical Details
- **ArgoCD**: v7.7.7.

### Archon Tasks
- ✅ Sprint 4: ArgoCD Bootstrap - COMPLETED

## [Sprint 3] - 2025-10-31 - COMPLETED ✅

### Added
- **High Availability Control Plane**
  - Scaled the `dev` cluster to 3 control plane nodes (obsy, onyx, opale).
- **Documentation**
  - `docs/validation-1cp-2w.md` validating 1 control plane + 2 workers configuration (before scaling to 3 CP).

### Changed
- Cluster topology changed from 1 control plane + 2 workers to 3 control planes for HA.

### Validated
- ✅ etcd 3-member quorum.
- ✅ HA failover scenarios.
- ✅ VIP switching between control planes.

### Technical Details
- **Nodes**: obsy, onyx, opale (all control planes).

### Archon Tasks
- ✅ Sprint 3: Scale to 3 control planes HA - COMPLETED

## [Sprint 2] - 2025-10-31 - COMPLETED ✅

### Added
- **Cilium CNI**
  - Deployment of Cilium v1.18.3 via Terraform Helm provider.
  - Enabled kube-proxy replacement.
  - Configured Hubble observability (relay + UI).
- **Documentation**
  - `docs/adr/004-cilium-cni.md` detailing the decision to use Cilium.

### Changed
- Integrated Cilium as the Container Network Interface.

### Validated
- ✅ Cilium operational and network connectivity.
- ✅ Hubble observability.

### Technical Details
- **Cilium**: v1.18.3.

### Archon Tasks
- ✅ Sprint 2: Cilium CNI - COMPLETED

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
  - Odd control plane validation (etcd quorum)
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

## [Upcoming] - Sprint 7 - Synology CSI (iSCSI storage)

### Planned
- Deploy Synology CSI driver for dynamic iSCSI storage.
- Configure StorageClass for iSCSI volumes.
- Validate persistent volume provisioning.

## [Upcoming] - Sprint 8 - Authentik (SSO/Auth)

### Planned
- Deploy Authentik for Single Sign-On (SSO) and authentication.
- Integrate Authentik with Traefik Ingress.

## [Upcoming] - Sprint 9 - Test cluster replication

### Planned
- Replicate the `dev` cluster configuration to the `test` environment.
- Validate Terraform and Kustomize on the `test` cluster.

## [Upcoming] - Sprints 10-11 - Phase 2 services

### Planned
- Deploy additional Phase 2 services as per roadmap.

## Project Status

**Current Phase**: Phase 2 - GitOps Infrastructure
**Current Sprint**: Sprint 6 ✅ → Sprint 7 ⏳
**Active Environment**: Dev (obsy, onyx, opale - 3 CP HA)
**Next Milestone**: Synology CSI deployment

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
