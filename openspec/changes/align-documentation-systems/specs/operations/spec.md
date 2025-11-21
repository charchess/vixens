# Operations Specification - Documentation Alignment

## ADDED Requirements

### Requirement: Information SHALL have single authoritative source

Each piece of information SHALL live in exactly one authoritative location, with other locations providing references only.

#### Scenario: Requirements live in OpenSpec
- **GIVEN** A requirement like "Infisical SHALL use isolated paths"
- **WHEN** Documenting this requirement
- **THEN** requirement SHALL be in OpenSpec `specs/` directory
- **AND** ADRs MAY reference the OpenSpec for full requirements
- **AND** runbooks MAY link to OpenSpec for context
- **AND** requirement SHALL NOT be duplicated in multiple documents

#### Scenario: Architecture decisions live in ADRs
- **GIVEN** An architectural decision like "Use Cilium instead of Calico"
- **WHEN** Documenting this decision
- **THEN** decision SHALL be in ADR (e.g., `docs/adr/004-cilium-cni.md`)
- **AND** OpenSpec proposals MAY reference ADR for context
- **AND** decision SHALL NOT be duplicated in CLAUDE.md or other docs

#### Scenario: Task status lives in Archon
- **GIVEN** A task like "Deploy Traefik to test environment"
- **WHEN** Tracking task execution
- **THEN** status SHALL be in Archon task system
- **AND** OpenSpec `tasks.md` MAY list tasks at high level
- **AND** detailed status SHALL NOT be duplicated in multiple documents

### Requirement: Cross-references SHALL use links not duplication

When one document needs information from another, it SHALL link to the authoritative source rather than duplicating content.

#### Scenario: ADR references OpenSpec requirements
- **GIVEN** ADR describing Infisical implementation
- **WHEN** ADR needs to reference requirements
- **THEN** ADR SHALL include "Related OpenSpec" section with link
- **AND** ADR SHALL NOT copy/paste requirements from OpenSpec
- **AND** link SHALL be relative path to OpenSpec archive

#### Scenario: OpenSpec references ADR context
- **GIVEN** OpenSpec proposal for new feature
- **WHEN** Proposal needs architectural context
- **THEN** Proposal SHALL include "Related ADRs" section with links
- **AND** Proposal SHALL NOT duplicate ADR content
- **AND** Links SHALL use relative paths

#### Scenario: CLAUDE.md references OpenSpec status
- **GIVEN** Infrastructure status table in CLAUDE.md
- **WHEN** Showing component deployment status
- **THEN** Status SHALL link to implementation OpenSpec
- **AND** CLAUDE.md SHALL NOT duplicate OpenSpec details
- **AND** Link SHALL point to OpenSpec archive or active change

### Requirement: Documentation indices SHALL be maintained

Each documentation category SHALL have an index file listing all documents in that category.

#### Scenario: ADR index exists and is current
- **WHEN** Navigating to `docs/adr/`
- **THEN** file `000-index.md` SHALL exist
- **AND** index SHALL list all ADR files with titles and summaries
- **AND** index SHALL be updated when new ADRs are added

#### Scenario: Runbook index exists and is current
- **WHEN** Navigating to `docs/runbooks/`
- **THEN** file `README.md` SHALL exist
- **AND** index SHALL list all runbooks with descriptions
- **AND** index SHALL be updated when new runbooks are added

#### Scenario: OpenSpec archive index exists
- **WHEN** Navigating to `openspec/archives/`
- **THEN** file `INDEX.md` SHALL exist
- **AND** index SHALL list all archived changes with dates and summaries
- **AND** index SHALL link to related ADRs and documentation

### Requirement: Documentation workflow SHALL be defined and followed

Project SHALL have documented workflow for creating and updating documentation.

#### Scenario: New feature follows documentation workflow
- **GIVEN** New feature to implement
- **WHEN** Following documentation workflow
- **THEN** workflow SHALL specify:
  1. Create OpenSpec proposal with requirements
  2. Reference relevant ADRs in proposal
  3. Create Archon tasks from OpenSpec tasks.md
  4. Implement feature following OpenSpec
  5. Mark Archon tasks complete
  6. Archive OpenSpec when done
  7. Create/update ADR if architectural decision made
  8. Update CLAUDE.md status
- **AND** workflow SHALL be documented in `docs/WORKFLOW.md`

#### Scenario: Documentation update follows workflow
- **GIVEN** Need to update documentation
- **WHEN** Following update workflow
- **THEN** workflow SHALL specify:
  1. Identify authoritative source (OpenSpec, ADR, runbook)
  2. Update authoritative source only
  3. Update cross-references if needed (links, not content)
  4. Run link checker to validate
- **AND** Workflow SHALL prevent content duplication

### Requirement: Archon tasks SHALL reference OpenSpecs

Archon tasks related to OpenSpecs SHALL include OpenSpec identifier in task name.

#### Scenario: Task name includes OpenSpec ID
- **GIVEN** Archon task for implementing OpenSpec feature
- **WHEN** Creating task in Archon
- **THEN** task name SHALL include OpenSpec ID prefix
- **AND** format SHALL be `[openspec-id] Task description`
- **AND** Example: `[propagate-infisical-multi-env] Create test overlay`

#### Scenario: Finding tasks related to OpenSpec
- **GIVEN** OpenSpec identifier like `deploy-homeassistant`
- **WHEN** Searching Archon tasks
- **THEN** search for `[deploy-homeassistant]` SHALL return all related tasks
- **AND** tasks SHALL be easy to identify and track

### Requirement: Documentation links SHALL be validated

Internal documentation links SHALL be checked regularly to prevent broken references.

#### Scenario: Link checker validates all links
- **WHEN** Running documentation link checker script
- **THEN** script SHALL scan all Markdown files in docs/ and openspec/
- **AND** script SHALL extract all internal links
- **AND** script SHALL verify link targets exist
- **AND** script SHALL report any broken links

#### Scenario: Broken links are fixed
- **GIVEN** Link checker reports broken link
- **WHEN** Fixing broken link
- **THEN** link target SHALL be corrected or removed
- **AND** link checker SHALL pass after fix
- **AND** commit message SHALL reference link checker run

### Requirement: Redundant documentation SHALL be eliminated

Documentation that duplicates information from authoritative sources SHALL be archived or deleted.

#### Scenario: Obsolete planning documents archived
- **GIVEN** Old planning documents in `docs/plans/`
- **WHEN** OpenSpec system provides same functionality
- **THEN** old documents SHALL be moved to `docs/archives/`
- **AND** references SHALL be updated to point to OpenSpecs
- **AND** archived documents SHALL include note explaining migration

#### Scenario: Duplicate sprint tracking archived
- **GIVEN** Sprint status in both `docs/sprints/` and Archon
- **WHEN** Archon provides authoritative task tracking
- **THEN** `docs/sprints/` SHALL be archived
- **AND** ROADMAP.md SHALL link to OpenSpecs instead
- **AND** High-level milestones MAY remain in ROADMAP.md

### Requirement: Documentation standards SHALL be documented

Project SHALL have documented standards for what information goes where.

#### Scenario: Documentation standards document exists
- **WHEN** Contributor needs to know where to put information
- **THEN** document `docs/DOCUMENTATION_STANDARDS.md` SHALL exist
- **AND** document SHALL define information hierarchy
- **AND** document SHALL explain cross-reference standards
- **AND** document SHALL provide decision tree for "where does X go?"

#### Scenario: Standards cover all documentation types
- **GIVEN** Documentation standards document
- **WHEN** Contributor has new information to document
- **THEN** standards SHALL cover:
  - Requirements → OpenSpec specs/
  - Architecture decisions → ADRs
  - Operational procedures → Runbooks
  - Task tracking → Archon
  - Current status → CLAUDE.md + Archon
- **AND** Standards SHALL provide examples for each type
