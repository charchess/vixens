# Infrastructure Specification - Update VIP and Node Documentation

## Purpose
This change updates the production VIP to resolve conflict and documents node naming conventions with IP assignments across VLANs for all environments.

## Requirements

### Requirement: Production VIP SHALL be Updated to Resolve Conflict
The production cluster VIP SHALL be changed to eliminate IP conflict with test environment.

#### Scenario: VIP change for production
- **WHEN** provisioning cluster prod
- **THEN** VIP SHALL be 192.168.111.190
- **AND** the old VIP 192.168.111.170 SHALL NOT be used

### Requirement: Node Names SHALL be Documented with Role and Environment
All nodes SHALL retain their current names and be documented with their roles and IPs across management and service VLANs.

#### Scenario: Dev control plane nodes documentation
- **WHEN** referencing dev cluster nodes
- **THEN** nodes SHALL be named obsy, onyx, opale
- **AND** they SHALL be documented as control plane nodes
- **AND** they SHALL have IPs in both VLAN 111 (mgmt) and VLAN 208 (services)

#### Scenario: Test control plane nodes documentation
- **WHEN** referencing test cluster nodes
- **THEN** nodes SHALL be named citrine, carny, celesty
- **AND** they SHALL be documented as control plane nodes
- **AND** they SHALL have IPs in both VLAN 111 (mgmt) and VLAN 209 (services)

#### Scenario: Staging control plane nodes documentation
- **WHEN** referencing staging cluster nodes
- **THEN** nodes SHALL be named serpentina, spinelia, saphira
- **AND** they SHALL be documented as control plane nodes
- **AND** they SHALL have IPs in both VLAN 111 (mgmt) and VLAN 210 (services)

#### Scenario: Prod control plane nodes documentation
- **WHEN** referencing prod cluster nodes
- **THEN** nodes SHALL be named perla, peridot, purpuria
- **AND** they SHALL be documented as control plane nodes
- **AND** they SHALL have IPs in both VLAN 200 (mgmt) and VLAN 200 (services)

### Requirement: IP Assignments SHALL Follow VLAN Segmentation
Each environment's nodes SHALL have IPs in both management and service VLANs as defined in Reference Data.

#### Scenario: Dev IP allocation consistency
- **WHEN** checking dev node obsy
- **THEN** it SHALL have IP 192.168.111.162 in VLAN 111
- **AND** it SHALL have IP 192.168.208.162 in VLAN 208

#### Scenario: Test IP allocation consistency
- **WHEN** checking test node citrine
- **THEN** it SHALL have IP 192.168.111.172 in VLAN 111
- **AND** it SHALL have IP 192.168.209.172 in VLAN 209

#### Scenario: Staging IP allocation consistency
- **WHEN** checking staging node serpentina
- **THEN** it SHALL have IP 192.168.111.182 in VLAN 111
- **AND** it SHALL have IP 192.168.210.182 in VLAN 210

#### Scenario: Prod IP allocation consistency
- **WHEN** checking prod node perla
- **THEN** it SHALL have IP 192.168.111.192 in VLAN 200
- **AND** it SHALL have IP 192.168.200.192 in VLAN 200

## Reference Data

### Complete IP Plan After Changes

#### Dev Cluster (vixens-dev)
- **VIP**: 192.168.111.160
- **Nodes**:
  - **obsy**: Control Plane, MAC=00:15:5D:00:CB:10, VLAN111=192.168.111.162, VLAN208=192.168.208.162, Disk=/dev/sda
  - **onyx**: Control Plane, MAC=00:15:5D:00:CB:11, VLAN111=192.168.111.164, VLAN208=192.168.208.164, Disk=/dev/sda
  - **opale**: Control Plane, MAC=00:15:5D:00:CB:0B, VLAN111=192.168.111.163, VLAN208=192.168.208.163, Disk=/dev/sda
- **ArgoCD LB**: 192.168.208.71 (VLAN 208)
- **Traefik LB**: 192.168.208.70 (VLAN 208)
- **Status**: Active

#### Test Cluster (vixens-test)
- **VIP**: 192.168.111.170
- **Nodes**:
  - **citrine**: Control Plane, MAC=00:15:5D:00:CB:1A, VLAN111=192.168.111.172, VLAN209=192.168.209.172, Disk=/dev/sda
  - **carny**: Control Plane, MAC=00:15:5D:00:CB:18, VLAN111=192.168.111.173, VLAN209=192.168.209.173, Disk=/dev/sda
  - **celesty**: Control Plane, MAC=00:15:5D:00:CB:19, VLAN111=192.168.111.174, VLAN209=192.168.209.174, Disk=/dev/sda
- **ArgoCD LB**: 192.168.209.71 (VLAN 209)
- **Traefik LB**: 192.168.209.70 (VLAN 209)
- **Status**: Deployed with issues

#### Staging Cluster (vixens-stg)
- **VIP**: 192.168.111.180
- **Nodes**:
  - **serpentina**: Control Plane, MAC=00:15:5D:00:77:01, VLAN111=192.168.111.182, VLAN210=192.168.210.182, Disk=/dev/sda
  - **spinelia**: Control Plane, MAC=00:15:5D:00:77:02, VLAN111=192.168.111.183, VLAN210=192.168.210.183, Disk=/dev/sda
  - **saphira**: Control Plane, MAC=00:15:5D:00:77:00, VLAN111=192.168.111.184, VLAN210=192.168.210.184, Disk=/dev/sda
- **ArgoCD LB**: 192.168.210.71 (VLAN 210)
- **Traefik LB**: 192.168.210.70 (VLAN 210)
- **Status**: Configured, not deployed

#### Prod Cluster (vixens)
- **VIP**: 192.168.111.190 **(CHANGED)**
- **Nodes**:
  - **perla**: Control Plane, MAC=68:1d:ef:4d:d6:a9, VLAN200=192.168.111.192, VLAN200=192.168.200.192, Disk=nvme0n1 **(CHANGED)**
  - **peridot**: Control Plane, MAC=68:1d:ef:56:d7:bb, VLAN200=192.168.111.193, VLAN200=192.168.200.193, Disk=nvme0n1 **(CHANGED)**
  - **purpuria**: Control Plane, MAC=00:e1:4f:68:0d:f8, VLAN200=192.168.111.194, VLAN200=192.168.200.194, Disk=sda **(CHANGED)**
- **ArgoCD LB**: 192.168.200.71 (VLAN 200)
- **Traefik LB**: 192.168.200.70 (VLAN 200)
- **Status**: Configured, not deployed

### Cluster Network Configuration

| Cluster | Service CIDR | Pod CIDR | Cluster Domain | Cluster DNS |
|---------|--------------|----------|----------------|-------------|
| **vixens-dev** | 10.96.0.0/12 | 10.244.0.0/16 | cluster.local | 10.96.0.10 |
| **vixens-test** | 10.96.0.0/12 | 10.244.0.0/16 | cluster.local | 10.96.0.10 |
| **vixens-stg** | 10.96.0.0/12 | 10.244.0.0/16 | cluster.local | 10.96.0.10 |
| **vixens-prod** | 10.96.0.0/12 | 10.244.0.0/16 | cluster.local | 10.96.0.10 |

### Hardware & Node Specifications

| Environment | Node Type | CPU | RAM | Disk | NIC Count | Role |
|-------------|-----------|-----|-----|------|-----------|------|
| **dev** | Virtual (Hyper-V) | 4 cores | 8GB | 100GB SSD | 2 (mgmt+svc) | Control Plane |
| **test** | Virtual (Hyper-V) | 4 cores | 8GB | 100GB SSD | 2 (mgmt+svc) | Control Plane |
| **staging** | Virtual (Hyper-V) | 6 cores | 12GB | 200GB SSD | 2 (mgmt+svc) | Control Plane |
| **prod** | Physical NiPoGi | 8+ cores | 32GB+ | 500GB NVMe | 2 (bonded 10Gb) | Control Plane |
