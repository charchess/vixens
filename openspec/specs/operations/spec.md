# Operations Specification - Vixens

## Purpose
Définit les processus opérationnels, les procédures de validation et les bonnes pratiques pour maintenir l'infrastructure et les clusters Kubernetes.

## Requirements

### Requirement: Architecture Decisions SHALL be Documented
All major technical decisions SHALL be captured in Architecture Decision Records (ADRs).

#### Scenario: ADR creation
- **WHEN** a new pattern is adopted (ex: 3-level Terraform, Cilium capabilities)
- **THEN** it SHALL be documented in `docs/adr/`
- **AND** it SHALL reference related specs and implementation
- **AND** it SHALL include context, decision, consequences (positive/negative)

#### Scenario: ADR review
- **WHEN** reviewing a change affecting architecture
- **THEN** ADRs SHALL be updated if the decision changes
- **AND** the old ADR SHALL be marked as "Superseded" with reference to new one

### Requirement: Infrastructure SHALL be Tested via Destroy/Recreate
Each environment SHALL be tested with at least 5 destroy/recreate cycles before promotion.

#### Scenario: Fresh cluster validation
- **WHEN** testing dev environment
- **THEN** destroy/recreate SHALL complete successfully in under 30 minutes
- **AND** all system apps SHALL become healthy automatically
- **AND** ArgoCD SHALL bootstrap without manual intervention
- **AND** no errors SHALL appear in Talos system logs

#### Scenario: Destroy safety
- **WHEN** running terraform destroy
- **THEN** it SHALL prompt for confirmation
- **AND** state backup SHALL be created automatically
- **AND** it SHALL NOT destroy NAS volumes (iSCSI LUNs) or MinIO bucket

### Requirement: Operational Commands SHALL be Documented
All commands needed for daily operations SHALL be in `docs/commands.md`.

#### Scenario: Command discovery
- **WHEN** a new operation pattern emerges
- **THEN** it SHALL be added to `docs/commands.md` within 24h
- **AND** it SHALL include context, example, and expected output

#### Scenario: Command validation
- **WHEN** a command is documented
- **THEN** it SHALL be tested on at least 2 environments before merging