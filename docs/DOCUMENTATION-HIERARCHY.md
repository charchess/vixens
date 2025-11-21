# Documentation Systems Hierarchy

**Last Updated**: 2025-11-21
**Status**: Active
**Purpose**: Define single source of truth for all project information

---

## System Overview

The vixens project uses three complementary documentation systems:

1. **OpenSpec** (`openspec/`) - Requirements, specifications, change proposals
2. **Documentation** (`docs/`) - Architecture decisions, runbooks, procedures
3. **Archon MCP** - Task management, work-in-progress tracking

Each system has a distinct purpose. This document defines what information belongs where and how they reference each other.

---

## Information Hierarchy (Single Source of Truth)

| Information Type | Primary Location | Cross-References From | Format |
|------------------|------------------|------------------------|--------|
| **Requirements & Specs** | OpenSpec `specs/` | ADRs, CLAUDE.md | Markdown spec files |
| **Change Proposals** | OpenSpec `proposal.md` | Commit messages | OpenSpec format |
| **Implementation Tasks** | OpenSpec `tasks.md` + Archon | CLAUDE.md | Checklist + MCP tasks |
| **Architecture Decisions** | `docs/adr/XXX-title.md` | OpenSpec proposals, CLAUDE.md | ADR format |
| **System Status** | `CLAUDE.md` | - | Prose + tables |
| **Operational Procedures** | `docs/runbooks/` | OpenSpec non-goals, ADRs | Step-by-step guides |
| **Setup Guides** | `docs/procedures/` | ADRs, OpenSpec | Tutorial format |
| **Verification Reports** | `docs/reports/` | OpenSpec completion | Report format |
| **Historical Changes** | OpenSpec `archives/` + `CHANGELOG.md` | ADRs | Archived proposals |
| **Active Work** | Archon tasks | OpenSpec tasks.md | MCP task tracking |

---

## System-Specific Guidelines

### OpenSpec (`openspec/`)

**Purpose**: Define "what should be built" with requirements and specifications.

**Structure**:
```
openspec/
├── changes/
│   └── {change-id}/
│       ├── proposal.md      # Why, what changes, impact
│       ├── tasks.md          # Implementation checklist
│       └── specs/
│           └── {domain}/
│               ├── spec.md   # SHALL/MUST requirements
│               └── delta.md  # Changes from current
└── archives/                 # Completed OpenSpecs (future)
    └── {date}-{change-id}/
```

**When to Create OpenSpec**:
- New features or capabilities
- Breaking changes
- Architecture shifts
- Performance/security improvements
- Multi-file refactoring

**When NOT to Use OpenSpec**:
- Bug fixes (use commit message)
- Documentation updates (edit docs directly)
- Dependency updates (use commit message)
- Configuration tweaks (use commit message)

**Cross-References**:
- Link to related ADRs in `proposal.md` (context)
- Link from CLAUDE.md to active OpenSpecs (status)
- Link from ADRs to archived OpenSpecs (history)

---

### Documentation (`docs/`)

**Purpose**: Explain "how it works" and "how to operate it".

**Structure**:
```
docs/
├── adr/                      # Architecture Decision Records
│   ├── 000-index.md          # Master ADR index
│   └── XXX-title.md          # Individual ADRs
├── reports/                  # Verification and analysis reports
│   └── {type}-{date}.md
├── procedures/               # Setup and deployment guides
│   └── {procedure-name}.md
├── runbooks/                 # Operational procedures (future)
│   └── {system}-management.md
└── DOCUMENTATION-HIERARCHY.md  # This file
```

**ADR Format**:
- **Status**: Draft | Active | Superseded
- **Context**: Why decision needed
- **Decision**: What we decided
- **Consequences**: Impact and trade-offs
- **Related**: Links to OpenSpecs, other ADRs

**When to Create ADR**:
- Architectural decisions (technology choices)
- Design patterns adopted
- Infrastructure strategies
- Security/compliance decisions

**When to Update ADR**:
- Implementation deviates from decision
- New information changes consequences
- Decision is superseded (mark as Superseded, create new ADR)

**Cross-References**:
- Link to OpenSpec archives (history)
- Link from OpenSpec proposals (context)
- Link from CLAUDE.md (documentation)

---

### Archon MCP

**Purpose**: Track "what's in progress" and task execution status.

**Task Naming Convention**:
```
[{openspec-id}] {specific-task-description}

Examples:
[propagate-infisical-multi-env] Create test environment overlay
[verify-clusterissuer-versions] Check dev ClusterIssuers
```

**Task Lifecycle**:
1. Create Archon tasks from OpenSpec `tasks.md`
2. Update task status as work progresses (todo → doing → review → done)
3. Before archiving OpenSpec, verify all Archon tasks are done
4. Archive OpenSpec links to final Archon task status

**When to Create Archon Task**:
- Implementing OpenSpec tasks
- Sprint planning items
- Ongoing maintenance work

**When NOT to Create Archon Task**:
- One-off quick fixes (just commit)
- Documentation updates (just commit)

**Cross-References**:
- Tasks reference OpenSpec ID in name
- CLAUDE.md links to active Archon context

---

### CLAUDE.md

**Purpose**: Provide AI agents with current project status and context.

**Structure**:
```markdown
# CLAUDE.md

## Current Phase
## Architecture (links to ADRs)
## Development Commands
## Current Infrastructure Status (links to OpenSpecs)
## Next Steps (links to OpenSpecs + Archon)
```

**Cross-Reference Pattern**:
```markdown
| Component | Status | Reference |
|-----------|--------|-----------|
| Infisical Operator | ✅ Deployed | [ADR 007](docs/adr/007-infisical-secrets-management.md), [OpenSpec: propagate-infisical-multi-env](openspec/changes/propagate-infisical-multi-env/) |
```

**When to Update**:
- After completing OpenSpec implementation
- After major infrastructure changes
- After creating new ADR
- Sprint transitions

---

## Workflows

### New Feature Workflow

1. **Planning**:
   - Create OpenSpec proposal in `openspec/changes/{change-id}/`
   - Reference relevant ADRs for context
   - Define specs if architectural impact

2. **Execution**:
   - Create Archon tasks from OpenSpec `tasks.md`
   - Implement following OpenSpec requirements
   - Mark Archon tasks complete as you go

3. **Documentation**:
   - Create/update ADR if architectural decision made
   - Create verification report in `docs/reports/` if needed
   - Update procedure docs if operational impact

4. **Completion**:
   - Archive OpenSpec (when available)
   - Update CLAUDE.md with links to ADR + archived OpenSpec
   - Update CHANGELOG.md

### Documentation Update Workflow

1. **Find Authoritative Source**:
   - Check hierarchy table above
   - Locate primary location for information

2. **Update Primary Location ONLY**:
   - Make changes to authoritative source
   - Do NOT duplicate content elsewhere

3. **Update Cross-References**:
   - Add/update links from secondary locations
   - Link to primary, don't copy content

4. **Validate**:
   - Check all links work
   - Ensure no broken references

### Verification/Report Workflow

1. **Create Report**:
   - Place in `docs/reports/{type}-{date}.md`
   - Use descriptive type (e.g., clusterissuer-verification, dns-verification)

2. **Link from OpenSpec**:
   - Reference report in OpenSpec completion section
   - Report validates OpenSpec requirements met

3. **Link from ADR** (if applicable):
   - Add reference to validation report
   - Shows decision was validated in practice

---

## Cross-Reference Examples

### OpenSpec References ADR

**In**: `openspec/changes/propagate-infisical-multi-env/proposal.md`
```markdown
## Context

See [ADR 007: Infisical Secrets Management](../../../docs/adr/007-infisical-secrets-management.md)
for background on Infisical Operator architecture.
```

### ADR References OpenSpec

**In**: `docs/adr/007-infisical-secrets-management.md`
```markdown
## Status

Active - Implemented via [OpenSpec: propagate-infisical-multi-env](../../openspec/changes/propagate-infisical-multi-env/)

## Related Changes

- [OpenSpec: propagate-infisical-multi-env](../../openspec/changes/propagate-infisical-multi-env/) - Multi-environment implementation
```

### CLAUDE.md References Both

**In**: `CLAUDE.md`
```markdown
### Secrets Management (Infisical)

✅ **SOLUTION IMPLÉMENTÉE** - Secrets management via **Infisical Kubernetes Operator**.

**See**:
- [ADR 007: Infisical Secrets Management](docs/adr/007-infisical-secrets-management.md) - Architecture et décisions
- [OpenSpec: propagate-infisical-multi-env](openspec/changes/propagate-infisical-multi-env/) - Multi-env deployment
- [Procedure: Infisical Multi-Env Setup](docs/procedures/infisical-multi-env-setup.md) - Manual steps
```

### Verification Report References OpenSpec

**In**: `docs/reports/clusterissuer-verification-2025-11-20.md`
```markdown
**Related OpenSpec**: [verify-clusterissuer-versions](../../openspec/changes/verify-clusterissuer-versions/)
```

---

## Key Principles

### 1. No Duplication

**❌ Wrong**:
- Copy OpenSpec requirements into ADR
- Duplicate ADR content in CLAUDE.md
- Repeat procedure steps in multiple places

**✅ Right**:
- OpenSpec contains requirements
- ADR links to OpenSpec for requirements
- CLAUDE.md links to both

### 2. Link, Don't Copy

**❌ Wrong**:
```markdown
## Infisical Architecture

Infisical uses Universal Auth with Machine Identities...
(100 lines of duplicated content from ADR)
```

**✅ Right**:
```markdown
## Infisical Architecture

See [ADR 007: Infisical Secrets Management](docs/adr/007-infisical-secrets-management.md)
for complete architecture details.
```

### 3. Update Once, Reference Everywhere

When information changes:
1. Find authoritative source (use hierarchy table)
2. Update ONLY the authoritative source
3. Verify cross-references still work
4. Do NOT update copies (there should be none)

### 4. Use Relative Paths

**✅ All links use relative paths from repository root**:
- From OpenSpec: `../../../docs/adr/007-infisical-secrets-management.md`
- From ADR: `../../openspec/changes/propagate-infisical-multi-env/`
- From CLAUDE.md: `docs/adr/007-infisical-secrets-management.md`

---

## Migration Status

**Completed**:
- ✅ Documentation hierarchy defined
- ✅ Cross-reference patterns established
- ✅ Workflow documentation created
- ✅ ADR 007 follows pattern (links to OpenSpec)
- ✅ CLAUDE.md references ADRs

**In Progress**:
- ⏳ Adding OpenSpec references to all ADRs
- ⏳ Creating ADR index (000-index.md)
- ⏳ Creating runbooks directory structure

**Future**:
- 📅 OpenSpec archival system
- 📅 Automated link checker script
- 📅 Automated status dashboard generator
- 📅 Archon ↔ OpenSpec sync automation

---

## FAQ

**Q: Where do I document a new architectural decision?**
A: Create ADR in `docs/adr/XXX-title.md`. Reference related OpenSpec if it exists.

**Q: Where do I track implementation tasks?**
A: Create OpenSpec `tasks.md` + corresponding Archon tasks with `[openspec-id]` prefix.

**Q: Where do I put verification results?**
A: Create report in `docs/reports/{type}-{date}.md`. Link from OpenSpec completion.

**Q: Where do I update project status?**
A: Update `CLAUDE.md` with links to ADRs and OpenSpecs. Don't duplicate content.

**Q: Can I create documentation outside this hierarchy?**
A: Only for temporary working docs. All permanent docs must follow this hierarchy.

**Q: What if I find duplicate information?**
A: Identify authoritative source, delete duplicate, add cross-reference link instead.

---

**Document Owner**: vixens project
**Review Cycle**: Update when new patterns emerge
**Feedback**: Create OpenSpec for major hierarchy changes
