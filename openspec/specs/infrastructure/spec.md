# Infrastructure Specification - Vixens

## Purpose
Gestion complète de l'infrastructure via Terraform pour provisionner et maintenir les clusters Kubernetes Talos Linux dans les environnements dev, test, staging et prod. Projet homelab personnel en phase 2/4 avec matériel NiPoGi et NAS Synology DS1821+.

## Requirements

### Requirement: IP Allocation SHALL be Consistent
The Terraform code SHALL use the IPs defined in the "Reference Data" section below.

#### Scenario: Dev VIP allocation
- **WHEN** provisioning cluster dev
- **THEN** VIP SHALL be 192.168.111.160

#### Scenario: Test VIP allocation
- **WHEN** provisioning cluster test
- **THEN** VIP SHALL be 192.168.111.170

#### Scenario: Staging VIP allocation
- **WHEN** provisioning cluster staging
- **THEN** VIP SHALL be 192.168.111.180

#### Scenario: Prod VIP allocation
- **WHEN** provisioning cluster prod
- **THEN** VIP SHALL be 192.168.111.170

### Requirement: VLAN Segmentation SHALL be Respected
The network SHALL be segmented into distinct VLANs for management et services selon l'environnement.

#### Scenario: Dev VLAN configuration
- **WHEN** configuring network for dev
- **THEN** VLAN 111 SHALL be used for management
- **AND** VLAN 208 SHALL be used for services

#### Scenario: Test VLAN configuration
- **WHEN** configuring network for test
- **THEN** VLAN 111 SHALL be used for management
- **AND** VLAN 209 SHALL be used for services

#### Scenario: Staging VLAN configuration
- **WHEN** configuring network for staging
- **THEN** VLAN 111 SHALL be used for management
- **AND** VLAN 210 SHALL be used for services

#### Scenario: Prod VLAN configuration
- **WHEN** configuring network for prod
- **THEN** VLAN 200 SHALL be used for management
- **AND** VLAN 200 SHALL be used for services

### Requirement: LoadBalancer IP Allocation SHALL follow Strict Rules
LoadBalancer IPs SHALL be allocated according to predefined pool ranges without overlap.

#### Scenario: Traefik LB IP allocation dev
- **WHEN** Traefik is deployed in dev
- **THEN** it SHALL take IP 192.168.208.70

#### Scenario: ArgoCD LB IP allocation dev
- **WHEN** ArgoCD is deployed in dev
- **THEN** it SHALL take IP 192.168.208.71

#### Scenario: Traefik LB IP allocation test
- **WHEN** Traefik is deployed in test
- **THEN** it SHALL take IP 192.168.209.70

#### Scenario: ArgoCD LB IP allocation test
- **WHEN** ArgoCD is deployed in test
- **THEN** it SHALL take IP 192.168.209.71

#### Scenario: Traefik LB IP allocation staging
- **WHEN** Traefik is deployed in staging
- **THEN** it SHALL take IP 192.168.210.70

#### Scenario: ArgoCD LB IP allocation staging
- **WHEN** ArgoCD is deployed in staging
- **THEN** it SHALL take IP 192.168.210.71

#### Scenario: Traefik LB IP allocation prod
- **WHEN** Traefik is deployed in prod
- **THEN** it SHALL take IP 192.168.200.70

#### Scenario: ArgoCD LB IP allocation prod
- **WHEN** ArgoCD is deployed in prod
- **THEN** it SHALL take IP 192.168.200.71

### Requirement: Terraform Module Architecture SHALL be DRY
The Terraform code SHALL follow a DRY architecture with a centralized base module.

#### Scenario: Base module structure
- **GIVEN** the terraform folder structure
- **THEN** terraform/modules/base SHALL contain reusable resources
- **AND** terraform/environments/{env} SHALL only contain environment-specific values

#### Scenario: New environment addition
- **WHEN** a new environment is added
- **THEN** only a new folder in terraform/environments/ SHALL be created
- **AND** it SHALL reference the base module without duplicating logic

### Requirement: Infrastructure SHALL be Idempotent
All Terraform operations SHALL be idempotent.

#### Scenario: Repeated apply
- **WHEN** terraform apply is executed three times consecutively
- **THEN** the second and third runs SHALL show "0 to add, 0 to change, 0 to destroy"

### Requirement: Infrastructure SHALL be Replaceable
Major updates SHALL use destroy/recreate, not incremental changes.

#### Scenario: Talos version upgrade
- **WHEN** Talos version is updated
- **THEN** the cluster SHALL be fully recreated via destroy/recreate
- **AND** state SHALL be backed up before destruction

### Requirement: Network Interface Naming SHALL follow Conventions
Network interfaces SHALL be named according to hardware type.

#### Scenario: Hyper-V interface naming
- **WHEN** a Hyper-V VM starts with Talos
- **THEN** the interface SHALL be named enxv + MAC (ex: enx00155d00cb10)

#### Scenario: Physical interface naming
- **WHEN** a physical server starts with Talos
- **THEN** the interface SHALL be named enpXsY format (ex: enp1s0)

### Requirement: Non-Deployed Environments SHALL remain Unprovisioned
Staging and prod environments SHALL remain in "configured but not provisioned" state until manual activation.

#### Scenario: Staging terraform plan
- **WHEN** terraform plan is executed for staging
- **THEN** it SHALL show resources to create without applying them
- **AND** no resources SHALL exist in actual infrastructure

#### Scenario: Prod terraform plan
- **WHEN** terraform plan is executed for prod
- **THEN** it SHALL show resources to create without applying them
- **AND** no resources SHALL exist in actual infrastructure

### Requirement: DNS Records SHALL be Pre-Configured
All required DNS records SHALL be created before cluster provisioning.

#### Scenario: Dev DNS records
- **WHEN** dev cluster is provisioned
- **THEN** DNS records for *.dev.truxonline.com SHALL point to 192.168.208.70

#### Scenario: Test DNS records
- **WHEN** test cluster is provisioned
- **THEN** DNS records for *.test.truxonline.com SHALL point to 192.168.209.70

### Requirement: Terraform SHALL Use 3-Level Architecture with Environment Module
Le code Terraform SHALL utiliser 3 niveaux : environments/ → modules/environment/ → modules/shared/{talos,cilium,argocd}.

#### Scenario: Level 1 - Environment configuration
- **WHEN** defining a new environment (dev/test/staging/prod)
- **THEN** `terraform/environments/{env}/main.tf` SHALL call `modules/environment`
- **AND** it SHALL contain only variables and module call (no resources)

#### Scenario: Level 2 - Environment orchestration
- **WHEN** deploying infrastructure
- **THEN** `modules/environment/main.tf` SHALL orchestrate Talos, Cilium and ArgoCD
- **AND** it SHALL reference `modules/shared` for common values
- **AND** it SHALL NOT duplicate logic from shared modules

#### Scenario: Level 3 - Shared modules
- **WHEN** creating cluster resources
- **THEN** `modules/{talos,cilium,argocd}` SHALL be reusable building blocks
- **AND** they SHALL receive all values from `modules/shared/variables.tf`

#### Scenario: DRY principle enforcement
- **WHEN** updating chart versions
- **THEN** it SHALL be changed in `modules/shared/locals.tf` only
- **AND** all environments SHALL automatically use new version

### Requirement: Backend State SHALL be MinIO on NAS
Terraform state SHALL be stored in MinIO S3-compatible backend hébergé sur Synology.

#### Scenario: MinIO backend configuration
- **WHEN** terraform init runs
- **THEN** it SHALL connect to nas.truxonline.com:9000
- **AND** it SHALL use bucket "terraform-state-vixens"
- **AND** state locking SHALL be enabled via MinIO

### Requirement: Terraform SHALL Wait for Kubernetes API
Terraform SHALL validate cluster readiness before deploying apps.

#### Scenario: API health check
- **WHEN** Talos cluster is created
- **THEN** Terraform SHALL wait 90s initial delay for Talos bootstrap
- **AND** it SHALL perform 60 attempts every 10s for /healthz endpoint
- **AND** it SHALL timeout after 10 minutes if API is unavailable

#### Scenario: Control plane stability verification
- **WHEN** API is responding
- **THEN** Terraform SHALL wait for 3 consecutive successful checks
- **AND** it SHALL validate kube-apiserver, controller-manager, scheduler pods
- **AND** it SHALL timeout after 20 minutes if control plane is not stable

### Requirement: Virtual Machines SHALL be Created on Hyper-V
All virtualized nodes SHALL be provisioned via Hyper-V sur Windows Server 2022.

#### Scenario: Hyper-V VM creation
- **WHEN** a VM is created for dev/test/staging
- **THEN** it SHALL be created on Hyper-V host avec specs NiPoGi
- **AND** it SHALL have static MAC assignment
- **AND** it SHALL have two NICs: one for management VLAN, one for services VLAN

#### Scenario: Hyper-V VM configuration
- **WHEN** a Hyper-V VM is configured for Talos
- **THEN** it SHALL have Secure Boot disabled
- **AND** it SHALL have virtual TPM disabled
- **AND** it SHALL boot from Talos ISO/image

### Requirement: Physical Nodes SHALL be Bare Metal NiPoGi
Prod nodes SHALL be physical NiPoGi mini PC avec Talos installé directement.

#### Scenario: Physical server preparation
- **WHEN** a NiPoGi is prepared for prod
- **THEN** it SHALL boot from Talos ISO via USB (PXE unavailable)
- **AND** it SHALL be installed to local SSD 512Go
- **AND** it SHALL have 16Go DDR4 RAM

### Requirement: Network Infrastructure SHALL be UniFi UDM SE
Le switch et routeur SHALL être l'UniFi Dream Machine SE avec configuration VLAN.

#### Scenario: UDM SE VLAN configuration
- **WHEN** VLANs are configured
- **THEN** UDM SE SHALL have VLANs 111, 200, 208, 209, 210 defined
- **AND** it SHALL have DHCP relay for VLANs 208-210
- **AND** it SHALL have firewall rules blocking inter-VLAN sauf nécessaire

### Requirement: NAS Configuration SHALL be Synology DS1821+
Tout stockage réseau SHALL venir du Synology DS1821+ avec DSM 7.2.2-72806 Update 4.

#### Scenario: Synology iSCSI configuration
- **WHEN** CSI driver needs storage
- **THEN** it SHALL connect to nas.truxonline.com:3260
- **AND** it SHALL use iSCSI LUNs for dynamic provisioning

#### Scenario: Synology NFS configuration
- **WHEN** manual mounts are needed
- **THEN** it SHALL use NFSv4 shares on nas.truxonline.com
- **AND** it SHALL be mounted as legacy only

## Reference Data

### Complete IP Plan

| Cluster | VIP | Nodes (IP/MAC/Disk) | ArgoCD LB | Traefik LB | VLAN Mgmt | VLAN Svc | Status |
|---------|-----|---------------------|-----------|------------|-----------|----------|--------|
| **vixens-dev** | 192.168.111.160 | obsy: 192.168.111.162 / 00:15:5D:00:CB:10 / /dev/sda&lt;br&gt;onyx: 192.168.111.164 / 00:15:5D:00:CB:11 / /dev/sda&lt;br&gt;opale: 192.168.111.163 / 00:15:5D:00:CB:0B / /dev/sda | 192.168.208.71 | 192.168.208.70 | 111 | 208 | Active |
| **vixens-test** | 192.168.111.170 | citrine: 192.168.111.172 / 00:15:5D:00:CB:1A / /dev/sda&lt;br&gt;carny: 192.168.111.173 / 00:15:5D:00:CB:18 / /dev/sda&lt;br&gt;celestite: 192.168.111.174 / 00:15:5D:00:CB:19 / /dev/sda | 192.168.209.71 | 192.168.209.70 | 111 | 209 | Deployed with issues |
| **vixens-stg** | 192.168.111.180 | serpentina: 192.168.111.182 / 00:15:5D:00:77:01 / /dev/sda&lt;br&gt;spinelia: 192.168.111.183 / 00:15:5D:00:77:02 / /dev/sda&lt;br&gt;saphira: 192.168.111.184 / 00:15:5D:00:77:00 / /dev/sda | 192.168.210.71 | 192.168.210.70 | 111 | 210 | Configured, not deployed |
| **vixens-prod** | 192.168.111.190 | perla: 192.168.111.65 / 68:1d:ef:4d:d6:a9 / /dev/nvme0n1&lt;br&gt;peridot: 192.168.111.60-63 / 68:1d:ef:56:d7:bb / /dev/nvme0n1&lt;br&gt;purpuria: 192.168.111.66 / 00:e1:4f:68:0d:f8 / /dev/sda | 192.168.200.71 | 192.168.200.70 | 200 | 200 | Configured, not deployed |

### Terraform Providers & Versions

| Provider | Min Version | Usage |
|----------|-------------|-------|
| **talos** | &gt;= 0.7.0 | Cluster provisioning |
| **helm** | &gt;= 2.11.0 | Helm chart management |
| **kubectl** | &gt;= 1.14.0 | Kubernetes resources |
| **random** | &gt;= 3.5.0 | Random password generation |

### Helm Charts & Versions

| Chart | Version | Environments | Purpose |
|-------|---------|--------------|---------|
| **cilium** | 1.16.5 | all | CNI & network policies |
| **talos** | 0.7.0 | all | Talos Linux config |
| **traefik** | &gt; 20.0.0 | all | Ingress controller |
| **argocd** | &gt; 5.0.0 | all | GitOps |
| **cert-manager** | &gt; 1.12.0 | all | Certificate management |
| **cert-manager-webhook-gandi** | Compatible | all | Gandi DNS challenge |
| **nfs-subdir-external-provisioner** | &gt; 4.0.0 | all | Dynamic NFS volumes |

### Hardware Specifications

| Component | Spec | Quantity | Use |
|-----------|------|----------|-----|
| **NiPoGi Mini PC** | Intel Twin Lake N150 (updated N100), 16GB DDR4, 512GB NVMe SSD | 6 total | 3x Hyper-V host + 3x prod bare metal |
| **Synology NAS** | DS1821+, DSM 7.2.2-72806 Update 4, RAID 5/SHR | 1 | iSCSI + NFS + MinIO backend |

### Network Infrastructure

| Device | Model | Firmware | Purpose |
|--------|-------|----------|---------|
| **Router/Switch** | UniFi Dream Machine SE | Latest | VLAN, firewall, DHCP relay |

### Synology NAS Configuration

| Parameter | Value | Notes |
|-----------|-------|-------|
| **Model** | DS1821+ | 8-bay |
| **DSM Version** | 7.2.2-72806 Update 4 | Latest |
| **Network** | Bonded 2x1GbE | LACP |
| **Volumes** | RAID 5 ou SHR | À préciser |
| **NFS** | Enabled, v4 | For legacy mounts |
| **iSCSI** | Enabled, port 3260 | For CSI driver |
| **MinIO** | Docker container, port 9000 | S3 backend |

### Hyper-V Host Configuration

| Parameter | Value | Notes |
|-----------|-------|-------|
| **OS** | Windows Server 2022 Standard | Host unique |
| **CPU** | Intel Twin Lake N150 | 4 cores/8 threads |
| **RAM** | 16GB DDR4 | Partagé avec VMs |
| **Storage** | 512GB NVMe SSD | VMs sur SSD |
| **Virtual Switches** | vSwitch-VLAN111, vSwitch-VLAN208, etc. | External |

### Hyper-V VM Configuration

```powershell
# Configuration standard pour VMs Talos
New-VM -Name "talos-dev-01" -MemoryStartupBytes 4GB -Generation 2
Set-VMProcessor -VMName "talos-dev-01" -Count 2
Set-VMMemory -VMName "talos-dev-01" -DynamicMemoryEnabled $false
Set-VMSecurity -VMName "talos-dev-01" -EnableSecureBoot Off
Set-VMSecurity -VMName "talos-dev-01" -VirtualizationBasedSecurityOptOut $true

# NIC Management VLAN 111
Add-VMNetworkAdapter -VMName "talos-dev-01" -SwitchName "vSwitch-VLAN111"
Set-VMNetworkAdapter -VMName "talos-dev-01" -StaticMacAddress "00-15-5D-00-CB-10"

# NIC Services VLAN 208
Add-VMNetworkAdapter -VMName "talos-dev-01" -SwitchName "vSwitch-VLAN208"
Set-VMNetworkAdapterVlan -VMName "talos-dev-01" -VMNetworkAdapterName "Network Adapter 2" -Access -VlanId 208