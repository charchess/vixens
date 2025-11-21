# Repository Operations

## MODIFIED Requirements

### Requirement: Application Directory Organization

The repository SHALL organize application code into functional domains with numeric prefixes to indicate deployment order and criticality.

#### Scenario: Deploying infrastructure before applications
**GIVEN** a fresh Kubernetes cluster
**WHEN** deploying applications via ArgoCD
**THEN** infrastructure apps (00-infra) SHALL be deployed first
**AND** storage providers (01-storage) SHALL be deployed after infrastructure
**AND** application domains (10+) SHALL be deployed after storage is ready

#### Scenario: Finding related applications
**GIVEN** a developer wants to find all media-related applications
**WHEN** browsing the repository
**THEN** all media apps SHALL be grouped under `apps/20-media/` directory
**AND** the numeric prefix SHALL indicate it's an application domain (not infrastructure)

### Requirement: Obsolete Code Removal

Obsolete or unmaintained code SHALL be removed from the repository to prevent confusion and reduce maintenance burden.

#### Scenario: Removing superseded implementations
**GIVEN** an obsolete `synology-csi-talos` implementation
**AND** a working replacement `synology-csi` (zebernst fork)
**WHEN** the replacement is validated and operational
**THEN** the obsolete directory SHALL be deleted from the repository
**AND** no ArgoCD Applications SHALL reference the deleted code

## ADDED Requirements

### Requirement: Functional Domain Categories

Applications SHALL be organized into predefined functional domains using numeric prefixes: 00-09 (infrastructure), 10-19 (home), 20-29 (media), 30-39 (productivity), 40-49 (network), 50-59 (backup), 99 (test).

#### Scenario: Categorizing a new application
**GIVEN** a new Jellyfin media server application
**WHEN** adding it to the repository
**THEN** it SHALL be placed in `apps/20-media/jellyfin/` directory
**AND** it SHALL NOT be placed in infrastructure or test directories

#### Scenario: Expanding an existing domain
**GIVEN** the home domain with existing `homeassistant` app
**WHEN** adding a new Zigbee2MQTT bridge
**THEN** it SHALL be added as `apps/10-home/zigbee2mqtt/`
**AND** the numeric prefix (10) SHALL remain consistent across all home domain apps
