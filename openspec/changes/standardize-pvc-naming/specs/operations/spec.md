# Operations Specification - PVC Naming

## ADDED Requirements

### Requirement: PVC Names SHALL Follow Standard Convention

All PersistentVolumeClaim resources SHALL use naming format `<application>-<volume-purpose>` in lowercase kebab-case.

#### Scenario: Creating PVC for new application
**GIVEN** a new application `jellyfin` being deployed
**WHEN** creating PVC for configuration storage
**THEN** PVC name SHALL be `jellyfin-config`
**AND** name SHALL be lowercase with hyphens only
**AND** name SHALL NOT include environment suffix (dev/test/prod)

#### Scenario: LUN traceability in Synology DSM
**GIVEN** PVC `homeassistant-config` in namespace `homeassistant`
**WHEN** Synology CSI creates corresponding LUN
**THEN** LUN description in DSM SHALL be `homeassistant/homeassistant-config`
**AND** operator can search DSM for "homeassistant" to find LUN
**AND** LUN name will be driver-generated UUID (unchangeable)

### Requirement: Purpose Suffix SHALL Describe Volume Content

PVC names SHALL include purpose suffix indicating what data the volume stores (config, data, cache, media, db).

#### Scenario: Multiple PVCs for one application
**GIVEN** application `nextcloud` requires multiple volumes
**WHEN** creating PVCs
**THEN** config PVC SHALL be `nextcloud-config`
**AND** user data PVC SHALL be `nextcloud-data`
**AND** database PVC SHALL be `nextcloud-db`
**AND** each purpose SHALL be clear from name alone
