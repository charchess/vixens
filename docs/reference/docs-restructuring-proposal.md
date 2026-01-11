# Documentation Restructuring Proposal

**Date:** 2025-12-30
**Status:** PROPOSAL
**Priority:** P1 (High)

---

## Current State Analysis

### Problems Identified

1. **Root-level chaos:** 6 orphan files at docs/ root
2. **Poor organization:** No clear categorization
3. **Missing guides:** No quick-start guides for common tasks
4. **Inconsistent naming:** UPPERCASE vs lowercase, dates in filenames
5. **applications/ overload:** 62 application files in one directory
6. **Incomplete procedures:** Only 1 procedure documented
7. **No onboarding guide:** New contributors lost

### Current Structure (Problematic)

```
docs/
├── ARGOCD-SYNC-WAVES.md                      ❌ Uppercase, should be reference
├── archon-task-formalism-proposal.md          ❌ Should be in guides/
├── archon-tasks-migration-plan.md             ❌ Should be in reports/
├── archon-tasks-migration-summary.md          ❌ Should be in reports/
├── code-review-report-2025-12-30.md           ❌ Should be in reports/
├── goldilocks-resource-recommendations-...md  ❌ Should be in reports/
├── adr/                                       ✅ OK
│   ├── 007-renovate-trunk-based-workflow.md
│   ├── 008-trunk-based-gitops-workflow.md
│   └── 009-simplified-two-branch-workflow.md
├── applications/                              ⚠️  62 files - too many
│   ├── adguard-home.md
│   ├── argocd.md
│   └── ... (60 more files)
├── implementation/                            ⚠️  Unclear purpose
│   ├── README.md
│   └── sync-waves-implementation-plan.md
├── procedures/                                ❌ Only 1 file
│   └── deployment-standard.md
└── troubleshooting/                           ⚠️  Only 2 files
    ├── 2024-12-25-cluster-redeploy-analysis.md
    └── 2024-12-25-final-status.md
```

---

## Proposed Structure

### New Organization

```
docs/
├── README.md                          # 🆕 Documentation index with quick links
│
├── guides/                            # 🆕 Step-by-step HOW-TO guides
│   ├── README.md
│   ├── adding-new-application.md      # 🆕 CRITICAL - How to add apps
│   ├── task-management.md             # 🆕 Archon task workflow
│   ├── gitops-workflow.md             # 🆕 Push to prod process
│   ├── terraform-workflow.md          # 🆕 Infrastructure changes
│   ├── secret-management.md           # 🆕 Infisical workflow
│   └── troubleshooting-guide.md       # 🆕 Common issues
│
├── reference/                         # 🆕 Technical references
│   ├── README.md
│   ├── argocd-sync-waves.md           # Moved from ARGOCD-SYNC-WAVES.md
│   ├── task-formalism.md              # Moved from archon-task-formalism-proposal.md
│   ├── kustomize-patterns.md          # 🆕 Common patterns
│   ├── overlay-strategy.md            # 🆕 Dev/Prod overlays
│   └── naming-conventions.md          # 🆕 Files, resources, namespaces
│
├── applications/                      # Grouped by category
│   ├── README.md                      # 🆕 Application index
│   ├── 00-infra/                      # 🆕 Infrastructure apps
│   │   ├── argocd.md
│   │   ├── cert-manager.md
│   │   ├── cilium-lb.md
│   │   ├── synology-csi.md
│   │   └── traefik.md
│   ├── 02-monitoring/                 # 🆕 Monitoring stack
│   │   ├── alertmanager.md
│   │   ├── goldilocks.md
│   │   ├── grafana.md
│   │   ├── loki.md
│   │   ├── prometheus.md
│   │   └── vpa.md
│   ├── 10-databases/                  # 🆕 Database services
│   │   ├── cloudnative-pg.md
│   │   ├── postgresql-shared.md
│   │   └── redis-shared.md
│   ├── 20-media/                      # 🆕 Media applications
│   │   ├── birdnet-go.md
│   │   ├── jellyfin.md
│   │   ├── jellyseerr.md
│   │   ├── radarr.md
│   │   ├── sonarr.md
│   │   └── ...
│   ├── 40-network/                    # 🆕 Network services
│   │   ├── adguard-home.md
│   │   ├── external-dns.md
│   │   └── gluetun.md
│   ├── 50-services/                   # 🆕 General services
│   │   ├── homeassistant.md
│   │   ├── vaultwarden.md
│   │   └── ...
│   └── 70-tools/                      # 🆕 Tools & utilities
│       ├── changedetection.md
│       ├── docspell.md
│       ├── homepage.md
│       └── ...
│
├── procedures/                        # Step-by-step operational procedures
│   ├── README.md
│   ├── deployment-standard.md         # Existing
│   ├── backup-restore.md              # 🆕 Backup/restore procedure
│   ├── disaster-recovery.md           # 🆕 DR procedure
│   ├── cluster-upgrade.md             # 🆕 Upgrade procedure
│   ├── secret-rotation.md             # 🆕 Secret rotation
│   └── certificate-renewal.md         # 🆕 TLS cert renewal
│
├── adr/                               # Architecture Decision Records
│   ├── README.md                      # 🆕 ADR index
│   ├── 001-...md                      # 🆕 Earlier ADRs (from memory)
│   ├── 007-renovate-trunk-based-workflow.md
│   ├── 008-trunk-based-gitops-workflow.md
│   ├── 009-simplified-two-branch-workflow.md
│   ├── 010-shared-resources-organization.md     # 🆕 From code review
│   ├── 011-namespace-ownership-strategy.md      # 🆕 From code review
│   ├── 012-middleware-management.md             # 🆕 From code review
│   └── template.md                    # 🆕 ADR template
│
├── reports/                           # 🆕 Analysis reports & audits
│   ├── README.md
│   ├── 2024-12-25-cluster-redeploy-analysis.md  # Moved from troubleshooting/
│   ├── 2025-12-29-goldilocks-recommendations.md # Moved + renamed
│   ├── 2025-12-30-code-review.md                # Moved + renamed
│   ├── 2025-12-30-archon-migration-plan.md      # Moved + renamed
│   └── 2025-12-30-archon-migration-summary.md   # Moved + renamed
│
├── troubleshooting/                   # Incident logs & post-mortems
│   ├── README.md
│   ├── 2024-12-25-final-status.md     # Existing
│   ├── common-issues.md               # 🆕 Quick fixes
│   └── post-mortems/                  # 🆕 Detailed incident analysis
│       └── 2024-12-25-cluster-rebuild.md
│
└── templates/                         # 🆕 File templates
    ├── README.md
    ├── adr-template.md
    ├── application-doc-template.md
    ├── procedure-template.md
    └── troubleshooting-template.md
```

---

## Key Improvements

### 1. Clear Categorization

| Directory | Purpose | Target Audience |
|-----------|---------|-----------------|
| `guides/` | How-to guides for common tasks | Everyone |
| `reference/` | Technical references & specifications | Developers |
| `procedures/` | Step-by-step operational procedures | Operators |
| `adr/` | Architecture decisions & rationale | Architects |
| `reports/` | Analysis reports & audits | Management |
| `troubleshooting/` | Incident logs & fixes | Support |
| `templates/` | Document templates | Contributors |

### 2. Critical New Guides

**guides/adding-new-application.md** - Most important!
```markdown
# Adding a New Application

## Prerequisites
- Application name decided (lowercase, kebab-case)
- Namespace determined
- Category identified (00-infra, 20-media, etc.)

## Step-by-Step Process

### 1. Create Base Structure
[Detailed steps with examples]

### 2. Configure Overlays
[Dev/Prod overlay setup]

### 3. Manage Secrets
[Infisical integration]

### 4. Create ArgoCD Application
[ArgoCD app manifest]

### 5. Validation
[Testing checklist]
```

**guides/task-management.md**
- Archon workflow
- Task formalism (conventional commits)
- Priority mapping
- Feature tagging

**guides/gitops-workflow.md**
- Trunk-based workflow
- dev → main promotion
- PR process
- Validation steps

### 3. Application Docs Reorganization

Instead of 62 files in one directory, group by category matching `apps/` structure:
- `00-infra/` (5 apps)
- `02-monitoring/` (8 apps)
- `10-databases/` (3 apps)
- `20-media/` (15+ apps)
- `40-network/` (3 apps)
- `50-services/` (10+ apps)
- `70-tools/` (10+ apps)

### 4. Naming Conventions

| Type | Format | Example |
|------|--------|---------|
| Guide | `<topic>.md` | `adding-new-application.md` |
| Reference | `<topic>.md` | `argocd-sync-waves.md` |
| Procedure | `<action>-<object>.md` | `backup-restore.md` |
| ADR | `NNN-<decision>.md` | `010-shared-resources.md` |
| Report | `YYYY-MM-DD-<topic>.md` | `2025-12-30-code-review.md` |
| Template | `<type>-template.md` | `adr-template.md` |

---

## Migration Plan

### Phase 1: Create New Structure (30 min)
- [ ] Create new directories
- [ ] Create README files for each directory
- [ ] Create templates/

### Phase 2: Move Existing Files (15 min)
- [ ] Move ARGOCD-SYNC-WAVES.md → reference/argocd-sync-waves.md
- [ ] Move archon-task-formalism-proposal.md → reference/task-formalism.md
- [ ] Move code-review-report-2025-12-30.md → reports/2025-12-30-code-review.md
- [ ] Move goldilocks-... → reports/2025-12-29-goldilocks-recommendations.md
- [ ] Move archon-tasks-migration-* → reports/
- [ ] Move troubleshooting/2024-12-25-cluster-redeploy-analysis.md → reports/
- [ ] Reorganize applications/ into subdirectories

### Phase 3: Create Critical Guides (2-3 hours)
- [ ] guides/adding-new-application.md (HIGH PRIORITY)
- [ ] guides/task-management.md
- [ ] guides/gitops-workflow.md
- [ ] guides/secret-management.md
- [ ] guides/terraform-workflow.md

### Phase 4: Create Missing ADRs (1-2 hours)
- [ ] adr/010-shared-resources-organization.md
- [ ] adr/011-namespace-ownership-strategy.md
- [ ] adr/012-middleware-management.md
- [ ] adr/README.md (index)
- [ ] adr/template.md

### Phase 5: Create docs/README.md (30 min)
Documentation hub with quick links to all sections

### Phase 6: Update References (30 min)
- [ ] Update CLAUDE.md links
- [ ] Update WORKFLOW.md links
- [ ] Update root README.md

---

## Success Criteria

✅ **Discoverability:**
- New team member finds "how to add an application" in < 30 seconds
- Task formalism easily accessible
- GitOps workflow documented end-to-end

✅ **Organization:**
- No orphan files at docs/ root
- Clear directory structure
- Consistent naming

✅ **Completeness:**
- All critical workflows documented
- All ADRs from code review created
- Templates available

✅ **Maintainability:**
- Clear ownership (README in each dir)
- Easy to find what needs updating
- Template-driven documentation

---

## Immediate Next Steps

1. **Get user validation** on proposed structure
2. **Execute Phase 1-2** (structure + move files) - Quick wins
3. **Execute Phase 3** (create guides/adding-new-application.md) - CRITICAL
4. **Create Archon task** for remaining phases

---

## Questions for User

1. Approve proposed structure?
2. Priority order: Should we focus on `guides/adding-new-application.md` first?
3. Any additional guides needed?
4. Keep troubleshooting/2024-12-25-final-status.md or archive?
