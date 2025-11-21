# Operations Specification - File Structure Groups

## ADDED Requirements

### Requirement: Applications SHALL be organized by logical groups

Application directories SHALL be grouped by their role in the infrastructure stack (infrastructure, storage, platform, applications).

#### Scenario: Infrastructure applications in dedicated group
- **WHEN** adding or locating core infrastructure components
- **THEN** they SHALL be in `apps/infrastructure/` directory
- **AND** this includes: cilium-lb, cert-manager, cert-manager-webhook-gandi, traefik
- **AND** these are components required for basic cluster functionality

#### Scenario: Storage applications in dedicated group
- **WHEN** adding or locating storage providers
- **THEN** they SHALL be in `apps/storage/` directory
- **AND** this includes: synology-csi, future CSI drivers
- **AND** these are components providing persistent storage

#### Scenario: Platform services in dedicated group
- **WHEN** adding or locating platform-level services
- **THEN** they SHALL be in `apps/platform/` directory
- **AND** this includes: authentik, monitoring, future platform services
- **AND** these are components enabling application functionality

#### Scenario: Applications in dedicated group
- **WHEN** adding or locating end-user applications
- **THEN** they SHALL be in `apps/applications/` directory
- **AND** this includes: homeassistant, media-stack, future applications
- **AND** these are components serving end users

### Requirement: ArgoCD Application paths SHALL reflect group structure

All ArgoCD Application manifests SHALL reference grouped paths in `spec.source.path`.

#### Scenario: Infrastructure app references grouped path
- **GIVEN** ArgoCD Application for traefik
- **WHEN** Application manifest is deployed
- **THEN** `spec.source.path` SHALL be `apps/infrastructure/traefik/overlays/{env}`
- **AND** path SHALL resolve correctly in Git repository

#### Scenario: Storage app references grouped path
- **GIVEN** ArgoCD Application for synology-csi
- **WHEN** Application manifest is deployed
- **THEN** `spec.source.path` SHALL be `apps/storage/synology-csi/overlays/{env}`
- **AND** path SHALL resolve correctly in Git repository

### Requirement: Group directories SHALL have descriptive README files

Each group directory SHALL contain a README.md explaining its purpose and listing applications.

#### Scenario: Infrastructure group has README
- **WHEN** navigating to `apps/infrastructure/` directory
- **THEN** a `README.md` file SHALL exist
- **AND** it SHALL explain the purpose of infrastructure group
- **AND** it SHALL list all applications in the group
- **AND** it SHALL document deployment order considerations

#### Scenario: Applications group has README
- **WHEN** navigating to `apps/applications/` directory
- **THEN** a `README.md` file SHALL exist
- **AND** it SHALL explain the purpose of applications group
- **AND** it SHALL list all applications in the group

### Requirement: Git history SHALL be preserved during restructure

File moves SHALL use `git mv` to preserve commit history.

#### Scenario: Traefik history preserved after move
- **WHEN** running `git log --follow apps/infrastructure/traefik/`
- **THEN** full commit history SHALL be visible
- **AND** history SHALL include commits from before the move (when it was at `apps/traefik/`)

#### Scenario: Synology CSI history preserved after move
- **WHEN** running `git log --follow apps/storage/synology-csi/`
- **THEN** full commit history SHALL be visible
- **AND** history SHALL include commits from before the move (when it was at `apps/synology-csi/`)

### Requirement: Deployment order SHALL follow group hierarchy

ArgoCD sync waves SHALL reflect group-based deployment order.

#### Scenario: Infrastructure deploys before platform
- **GIVEN** Infrastructure apps with wave 1-2
- **AND** Platform apps with wave 3
- **WHEN** ArgoCD syncs applications
- **THEN** all infrastructure apps SHALL complete before platform apps start
- **AND** dependencies SHALL be satisfied

#### Scenario: Storage deploys before applications
- **GIVEN** Storage apps with wave -1 to 0
- **AND** Application apps with wave 4
- **WHEN** ArgoCD syncs applications
- **THEN** storage providers SHALL be available before applications start
- **AND** PVC creation SHALL succeed
