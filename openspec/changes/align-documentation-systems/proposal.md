# Align Documentation, OpenSpec, and Archon Systems

## Why

The vixens project uses three parallel systems for tracking work:
1. **Documentation** - Markdown files in `docs/` (ADRs, architecture, runbooks)
2. **OpenSpec** - Change proposals with specs and deltas in `openspec/`
3. **Archon MCP** - Task management via MCP server tools

**Current Problems:**
- **Information Duplication**: Same information appears in multiple places with inconsistencies
- **Sync Burden**: Changes require updating docs, OpenSpecs, AND Archon tasks separately
- **Discovery Confusion**: Unclear where to find authoritative information
- **Stale Data**: Documentation gets outdated while OpenSpecs remain current (or vice versa)
- **Workflow Friction**: Too much overhead maintaining three parallel systems

**Example Inconsistencies:**
- ADR 007 describes Infisical implementation, but OpenSpec has different details
- CLAUDE.md has infrastructure status, but no link to corresponding OpenSpecs
- Archon tasks reference outdated documentation paths
- Completed OpenSpecs not reflected in ROADMAP.md

**Vision:**
Establish a **single source of truth** hierarchy with clear relationships:
- **OpenSpec** = Authoritative for "what should be built" (requirements, specs)
- **Documentation** = Authoritative for "how it works" (architecture, procedures)
- **Archon** = Authoritative for "what's in progress" (task tracking, status)

## What Changes

### Phase 1: Establish Information Hierarchy

**Define what goes where:**

| Information Type | Primary Location | Secondary References |
|------------------|------------------|----------------------|
| **Requirements** (SHALL/MUST) | OpenSpec `specs/` | Referenced in ADRs |
| **Architecture Decisions** | ADRs (`docs/adr/`) | Referenced in OpenSpec proposals |
| **Implementation Plans** | OpenSpec `tasks.md` | Synced to Archon tasks |
| **Current Status** | CLAUDE.md + Archon | Links to OpenSpecs |
| **Operational Procedures** | Runbooks (`docs/runbooks/`) | Referenced in OpenSpec non-goals |
| **Historical Changes** | OpenSpec archives + CHANGELOG.md | Summarized in ADRs |

**Key Principles:**
1. **No Duplication**: Information lives in ONE authoritative place
2. **Cross-Reference**: Other locations LINK to authoritative source
3. **Automation**: Use scripts/tools to sync between systems where needed
4. **Living Documents**: Documentation evolves with OpenSpecs, not separately

### Phase 2: OpenSpec → Documentation Integration

**ADRs Reference OpenSpecs:**
- Each ADR should link to corresponding OpenSpec archive
- OpenSpec proposals should reference relevant ADRs for context
- Example: ADR 007 (Infisical) links to archived `add-feature-infisical` OpenSpec

**CLAUDE.md References OpenSpecs:**
- Infrastructure status table links to implementation OpenSpecs
- Example: "Infisical ✅ Running - See [OpenSpec: add-feature-infisical](openspec/archives/2025-11-20-add-feature-infisical/)"

**Runbooks Reference OpenSpecs:**
- Operational procedures link to OpenSpec specs for requirements
- Example: Certificate management runbook links to ClusterIssuer spec

### Phase 3: OpenSpec → Archon Integration

**Automated Task Sync (Future Enhancement):**
- OpenSpec `tasks.md` generates Archon tasks automatically
- Archon task completion updates OpenSpec status
- Bi-directional sync keeps both systems consistent

**Manual Sync (Current Approach):**
- When creating OpenSpec, manually create corresponding Archon tasks
- When completing Archon tasks, mark OpenSpec tasks complete
- Before archiving OpenSpec, verify all Archon tasks closed

**Archon Task Naming Convention:**
- Prefix Archon tasks with OpenSpec ID: `[propagate-infisical-multi-env] Create test overlay`
- Easy to find related tasks via search

### Phase 4: Documentation Structure Standardization

**Consolidate Documentation:**

```
docs/
├── adr/                     # Architecture Decision Records
│   ├── 000-index.md         # Master index linking to all ADRs
│   └── XXX-title.md         # Individual ADRs (link to OpenSpecs)
├── architecture/            # High-level architecture docs
│   ├── README.md            # Architecture overview
│   ├── network-diagram.md
│   └── gitops-workflow.md
├── runbooks/                # Operational procedures
│   ├── README.md            # Runbook index
│   ├── terraform-operations.md
│   ├── dns-management.md
│   └── clusterissuer-management.md
├── procedures/              # Step-by-step guides
│   ├── README.md            # Procedure index
│   └── certificate-management.md
└── ROADMAP.md               # High-level roadmap (links to OpenSpecs)
```

**Eliminate Redundant Documentation:**
- Delete: `docs/plans/` (use OpenSpec proposals instead)
- Delete: `docs/sprints/` (use Archon task tracking instead)
- Delete: Duplicate implementation details (use OpenSpec archives)
- Archive: Old validation reports after incorporating into ADRs

### Phase 5: Cross-Reference Automation

**Create Documentation Tools:**

**Tool 1: OpenSpec Archive Indexer**
- Script: `scripts/generate-openspec-index.sh`
- Generates: `openspec/archives/INDEX.md` with all archived changes
- Links to: Original proposals, specs, completion dates

**Tool 2: Documentation Link Checker**
- Script: `scripts/check-doc-links.sh`
- Validates: All internal links work (OpenSpec refs, ADR refs, etc.)
- Reports: Broken links, missing references

**Tool 3: Status Dashboard Generator**
- Script: `scripts/generate-status.sh`
- Generates: `STATUS.md` with current infrastructure state
- Sources: OpenSpec status, Archon tasks, Kubernetes cluster state
- Updates: CLAUDE.md automatically

**Tool 4: ADR <-> OpenSpec Linker**
- Script: `scripts/link-adr-openspec.sh`
- Adds: "Related OpenSpec" section to ADRs automatically
- Adds: "Related ADR" section to OpenSpec proposals

### Phase 6: Workflow Integration

**New Feature Workflow:**
1. Create OpenSpec proposal with requirements
2. Reference relevant ADRs in proposal (if exists)
3. Create Archon tasks from OpenSpec `tasks.md`
4. Implement feature following OpenSpec
5. Mark Archon tasks complete as you go
6. Archive OpenSpec when complete
7. Create/update ADR if architectural decision made
8. Update CLAUDE.md status (or use automation)
9. Update ROADMAP.md if sprint completed

**Documentation Update Workflow:**
1. Find authoritative source (OpenSpec, ADR, runbook)
2. Update authoritative source ONLY
3. Update cross-references if needed (links, not content)
4. Run link checker to validate
5. Commit with message referencing OpenSpec/ADR/task

## Impact

**Developer Experience:**
- ✅ **Single Source of Truth**: No more guessing where to find information
- ✅ **Less Maintenance**: Update once, reference everywhere
- ✅ **Clear Workflow**: Defined process for new features and documentation

**Documentation Quality:**
- ✅ **Consistency**: Cross-references prevent drift
- ✅ **Freshness**: Authoritative sources stay up-to-date
- ✅ **Discoverability**: Clear indexes and links

**Project Management:**
- ✅ **Visibility**: Easy to see what's planned, in progress, completed
- ✅ **Accountability**: Archon tasks track execution, OpenSpec tracks requirements
- ✅ **History**: Archived OpenSpecs provide audit trail

**Risk:**
- ⚠️ Large refactoring effort required
- ⚠️ Need to train AI agents on new workflow
- ⚠️ Risk of breaking existing references during consolidation
- ⚠️ Automation scripts add maintenance burden
- Mitigation: Incremental migration, keep old docs until confident, document new workflow clearly

## Non-Goals

- Not eliminating any of the three systems (all serve different purposes)
- Not fully automating sync (manual workflow acceptable initially)
- Not creating new documentation system (use existing Markdown)
- Not migrating to external tools (Jira, Notion, etc.)
- Not enforcing strict templates (flexibility for different doc types)
