# Tasks - Align Documentation Systems

## Phase 1: Audit and Inventory

- [ ] Create documentation inventory:
  - [ ] List all files in `docs/` with categories (ADR, architecture, runbook, procedure)
  - [ ] List all OpenSpec changes (active + archived)
  - [ ] List all Archon tasks (use `mcp__archon__find_tasks`)
  - [ ] Create comparison matrix showing overlaps

- [ ] Identify duplicate information:
  - [ ] Find content duplicated in docs + OpenSpecs
  - [ ] Find content duplicated in docs + Archon tasks
  - [ ] Document which source is more authoritative/up-to-date

- [ ] Map relationships:
  - [ ] Which ADRs relate to which OpenSpecs?
  - [ ] Which OpenSpecs relate to which Archon tasks?
  - [ ] Which runbooks relate to which OpenSpecs?

## Phase 2: Define Information Hierarchy

- [ ] Create documentation standards document:
  - [ ] Create `docs/DOCUMENTATION_STANDARDS.md`
  - [ ] Define what goes in OpenSpecs (requirements, specs, proposals)
  - [ ] Define what goes in ADRs (architectural decisions, rationale)
  - [ ] Define what goes in runbooks (operational procedures)
  - [ ] Define what goes in Archon (task tracking, execution status)

- [ ] Define cross-reference standards:
  - [ ] How to link from ADR to OpenSpec
  - [ ] How to link from OpenSpec to ADR
  - [ ] How to reference Archon tasks
  - [ ] How to link from runbooks to specs

- [ ] Create workflow documentation:
  - [ ] Create `docs/WORKFLOW.md`
  - [ ] Document new feature workflow (OpenSpec → Archon → Implementation → ADR)
  - [ ] Document documentation update workflow
  - [ ] Document OpenSpec lifecycle (propose → implement → archive)

## Phase 3: OpenSpec Integration

- [ ] Add OpenSpec references to ADRs:
  - [ ] Review each ADR in `docs/adr/`
  - [ ] Add "Related OpenSpec" section where applicable
  - [ ] Link to archived OpenSpecs for implemented features
  - [ ] Example: ADR 007 links to `add-feature-infisical` archive

- [ ] Add ADR references to OpenSpec proposals:
  - [ ] Review active OpenSpecs
  - [ ] Add "Related ADRs" section to proposals
  - [ ] Link to ADRs that provide context or rationale

- [ ] Create OpenSpec archive index:
  - [ ] Create `openspec/archives/INDEX.md`
  - [ ] List all archived changes with:
    - Archive date
    - Original proposal summary
    - Related ADRs
    - Related documentation
  - [ ] Keep index updated (manual or script)

## Phase 4: CLAUDE.md Enhancement

- [ ] Update CLAUDE.md infrastructure status:
  - [ ] Add "OpenSpec" column to infrastructure status tables
  - [ ] Link each component to its implementation OpenSpec
  - [ ] Example: "Infisical ✅ Running | [OpenSpec](openspec/archives/2025-11-20-add-feature-infisical/)"

- [ ] Add OpenSpec workflow section:
  - [ ] Document how OpenSpecs are used in vixens project
  - [ ] Explain relationship to ADRs and Archon
  - [ ] Link to DOCUMENTATION_STANDARDS.md

- [ ] Update repository structure diagram:
  - [ ] Add `openspec/` directory to structure
  - [ ] Show relationship between docs, openspec, and code

## Phase 5: Archon Integration

- [ ] Review current Archon tasks:
  - [ ] List all tasks: `mcp__archon__find_tasks`
  - [ ] Identify tasks related to OpenSpecs
  - [ ] Check for orphaned tasks (no OpenSpec)

- [ ] Create Archon task naming standard:
  - [ ] Prefix format: `[openspec-id] Task description`
  - [ ] Example: `[propagate-infisical-multi-env] Create test overlay`
  - [ ] Update existing tasks to follow standard

- [ ] Link OpenSpecs to Archon tasks:
  - [ ] Add "Archon Tasks" section to OpenSpec proposals
  - [ ] List related Archon task IDs
  - [ ] Update when creating/completing tasks

- [ ] Establish sync workflow:
  - [ ] When creating OpenSpec: Create Archon tasks from `tasks.md`
  - [ ] When completing Archon task: Mark OpenSpec task complete
  - [ ] Before archiving OpenSpec: Verify all Archon tasks closed
  - [ ] Document this workflow in WORKFLOW.md

## Phase 6: Documentation Consolidation

- [ ] Create documentation indexes:
  - [ ] Create `docs/adr/000-index.md` listing all ADRs
  - [ ] Create `docs/runbooks/README.md` listing all runbooks
  - [ ] Create `docs/procedures/README.md` listing all procedures
  - [ ] Update each index when adding new docs

- [ ] Identify redundant documentation:
  - [ ] Review `docs/plans/` - can these be replaced with OpenSpec proposals?
  - [ ] Review `docs/sprints/` - can these be replaced with Archon tasks?
  - [ ] Review validation reports - can these be incorporated into ADRs?
  - [ ] Create list of files to archive or delete

- [ ] Archive or delete redundant documentation:
  - [ ] Create `docs/archives/` for historical docs
  - [ ] Move redundant files to archives
  - [ ] Update any references to moved files
  - [ ] Delete truly redundant files (after confirming with user)

- [ ] Standardize remaining documentation:
  - [ ] Ensure all ADRs follow same format
  - [ ] Ensure all runbooks follow same format
  - [ ] Add cross-references where needed
  - [ ] Fix any broken internal links

## Phase 7: Automation Tools

- [ ] Create OpenSpec archive indexer:
  - [ ] Create `scripts/generate-openspec-index.sh`
  - [ ] Script should:
    - Scan `openspec/archives/` directory
    - Extract metadata from each archive
    - Generate `openspec/archives/INDEX.md`
  - [ ] Test script and validate output
  - [ ] Document usage in WORKFLOW.md

- [ ] Create documentation link checker:
  - [ ] Create `scripts/check-doc-links.sh`
  - [ ] Script should:
    - Find all Markdown files in docs/ and openspec/
    - Extract all internal links
    - Verify link targets exist
    - Report broken links
  - [ ] Test script and fix any found broken links
  - [ ] Add to CI/CD pipeline (future)

- [ ] Create ADR <-> OpenSpec linker:
  - [ ] Create `scripts/link-adr-openspec.sh`
  - [ ] Script should:
    - Scan ADRs for keywords (infisical, traefik, etc.)
    - Find related OpenSpecs by keyword
    - Add "Related OpenSpec" section to ADRs
    - Add "Related ADR" section to OpenSpecs
  - [ ] Test script manually first
  - [ ] Document usage

- [ ] Create status dashboard generator (optional):
  - [ ] Create `scripts/generate-status.sh`
  - [ ] Script should:
    - Query Kubernetes cluster status
    - Extract OpenSpec status
    - Extract Archon task status
    - Generate STATUS.md or update CLAUDE.md
  - [ ] Test in dev environment
  - [ ] Document usage

## Phase 8: Workflow Documentation

- [ ] Update WORKFLOW.md with new procedures:
  - [ ] Document new feature workflow:
    1. Create OpenSpec proposal
    2. Reference relevant ADRs
    3. Create Archon tasks
    4. Implement following OpenSpec
    5. Mark tasks complete
    6. Archive OpenSpec
    7. Create/update ADR
    8. Update CLAUDE.md
  - [ ] Document documentation update workflow
  - [ ] Document OpenSpec lifecycle
  - [ ] Add examples and templates

- [ ] Create quick reference guide:
  - [ ] "Where does X go?" decision tree
  - [ ] Example: Requirements → OpenSpec, Rationale → ADR, Status → Archon
  - [ ] Add to DOCUMENTATION_STANDARDS.md

- [ ] Train AI agents on new workflow:
  - [ ] Update CLAUDE.md with workflow instructions
  - [ ] Update AGENTS.md with OpenSpec instructions
  - [ ] Add examples of correct cross-referencing
  - [ ] Test with sample scenario

## Phase 9: Migration

- [ ] Migrate existing documentation:
  - [ ] Add OpenSpec references to all existing ADRs
  - [ ] Create missing ADRs for architectural decisions
  - [ ] Archive obsolete planning documents
  - [ ] Consolidate duplicated information

- [ ] Migrate active OpenSpecs:
  - [ ] Add ADR references to active OpenSpecs
  - [ ] Create Archon tasks for OpenSpecs without them
  - [ ] Update task naming to follow standard

- [ ] Update ROADMAP.md:
  - [ ] Link each sprint/milestone to related OpenSpecs
  - [ ] Add status indicators with OpenSpec links
  - [ ] Remove duplicated implementation details (keep high-level only)

## Phase 10: Validation

- [ ] Run link checker:
  - [ ] Execute `scripts/check-doc-links.sh`
  - [ ] Fix all broken links
  - [ ] Verify cross-references work

- [ ] Verify consistency:
  - [ ] Check that OpenSpec requirements match ADR decisions
  - [ ] Check that Archon tasks match OpenSpec tasks
  - [ ] Check that CLAUDE.md status matches reality

- [ ] Test new workflow:
  - [ ] Create a test OpenSpec following new workflow
  - [ ] Create Archon tasks from OpenSpec
  - [ ] Implement and archive OpenSpec
  - [ ] Verify all documentation updated correctly
  - [ ] Document any workflow issues encountered

## Phase 11: Continuous Improvement

- [ ] Schedule documentation reviews:
  - [ ] Quarterly review of all documentation
  - [ ] Verify links still work
  - [ ] Update stale information
  - [ ] Archive obsolete documentation

- [ ] Monitor workflow adoption:
  - [ ] Track if new OpenSpecs follow workflow
  - [ ] Identify common mistakes or confusion
  - [ ] Update WORKFLOW.md based on learnings

- [ ] Evolve automation:
  - [ ] Identify manual steps that could be automated
  - [ ] Enhance scripts based on feedback
  - [ ] Consider CI/CD integration for link checking
