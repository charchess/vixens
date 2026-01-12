# Reports Directory

**Consolidated application state tracking and historical analysis reports for the Vixens project.**

---

## 📊 Current Reports (Living Documents)

These reports are actively maintained and reflect the current state of the infrastructure:

### **[Application Status Dashboard](../STATUS.md)**
**Purpose:** High-level overview of all applications across environments.
**Updated:** On every significant infrastructure change.

### **[AUDIT-CONFORMITY.md](AUDIT-CONFORMITY.md)** - Application Audit & Conformity
**Purpose:** Detailed technical audit results and conformity scores.
**Updated:** Automated audits.

---

### **[validation/](validation/)** - Validation Results
**Purpose:** Latest functional and technical validation reports (Recettes).
**Contains:**
- **[RECETTE-FONCTIONNELLE.md](validation/RECETTE-FONCTIONNELLE.md)**
- **[RECETTE-TECHNIQUE.md](validation/RECETTE-TECHNIQUE.md)**

---

### **[audits/](audits/)** - Technical Audits
**Purpose:** Deep-dive analysis on resources, conformity, and architecture.

---

### **[STATE-ACTUAL.md](STATE-ACTUAL.md)** - Production Reality
**Purpose:** Detailed technical state of all applications in production
**Updated:** After deployments, resource changes, or incidents
**Contains:**
- Complete resource specifications (CPU/Memory requests/limits)
- VPA recommendations and current profile assignments
- Priority classes and sync waves
- Backup profiles (Litestream)
- Detected issues (OOM risk, CPU throttling, missing limits)

**Use For:**
- Troubleshooting performance issues
- Capacity planning
- Resource optimization
- VPA analysis

---

### **[STATE-DESIRED.md](STATE-DESIRED.md)** - Reference Standard
**Purpose:** Single source of truth for desired application configurations
**Updated:** When architectural decisions are made
**Contains:**
- Target resource profiles for all applications
- Desired priority classes, sync waves, backup profiles
- Target conformity scores
- Decision history and rationale
- Applications marked for removal

**Use For:**
- New application deployments (copy standards)
- Conformity analysis (compare against STATE-ACTUAL)
- Architectural decision tracking
- Automation input (generate patches from this data)

**Workflow:**
```
Decision Made → Update STATE-DESIRED → Apply to GitOps → Verify → Update STATE-ACTUAL
```

---

## 📁 Historical Reports (Archived)

Point-in-time analysis reports for reference:

### Infrastructure Analysis
- **[2026-01-01-prod-resource-analysis.md](2026-01-01-prod-resource-analysis.md)** - Production resource deep-dive
- **[2026-01-01-resource-analysis.md](2026-01-01-resource-analysis.md)** - Cross-environment resource comparison
- **[2024-12-25-cluster-redeploy-analysis.md](2024-12-25-cluster-redeploy-analysis.md)** - Cluster rebuild lessons learned

### Code & Quality
- **[2025-12-30-code-review.md](2025-12-30-code-review.md)** - Architecture review (DRY violations, tech debt)
- **[2025-12-29-goldilocks-recommendations.md](2025-12-29-goldilocks-recommendations.md)** - VPA recommendations

### Migrations
- **[2025-12-30-archon-migration-summary.md](2025-12-30-archon-migration-summary.md)** - Task format migration (50 tasks)
- **[2025-12-30-archon-migration-plan.md](2025-12-30-archon-migration-plan.md)** - Migration strategy

### Maintenance
- **[2025-12-30-serena-memories-cleanup.md](2025-12-30-serena-memories-cleanup.md)** - Serena memory cleanup

---

## 🗄️ Legacy Reports (Superseded)

These reports have been replaced by the consolidated STATUS/STATE-* system:

- ~~**audits/APP_AUDIT.md**~~ → Replaced by **AUDIT-CONFORMITY.md** + **STATE-ACTUAL.md**
- ~~**audits/ULTIMATE-AUDIT.md**~~ → Replaced by **STATE-ACTUAL.md**

**Note:** Legacy files are retained for historical reference but should not be actively maintained.

---

## 🔄 Report Update Workflow

### When to Update Reports

| Trigger                          | Update STATUS.md | Update STATE-ACTUAL.md | Update STATE-DESIRED.md |
|:---------------------------------|:----------------:|:----------------------:|:-----------------------:|
| **New application deployed**     | ✅               | ✅                     | ✅                      |
| **Resource limits changed**      | ✅               | ✅                     | If intentional decision |
| **Application status change**    | ✅               | ✅                     | -                       |
| **Architectural decision made**  | -                | -                      | ✅                      |
| **Conformity gap detected**      | ✅               | ✅                     | Review needed           |
| **GitOps repair/incident**       | ✅               | ✅                     | -                       |

### Update Process

**For STATUS.md:**
1. Update status matrix (OK/NOK/Hibernate/Absent)
2. Recalculate conformity scores (compare ACTUAL vs DESIRED)
3. Update priority actions if critical issues detected
4. Update last updated date

**For STATE-ACTUAL.md:**
1. Query production cluster (kubectl, VPA, ArgoCD)
2. Update resource specifications table
3. Update detected issues section
4. Cross-reference with GitOps manifests
5. Update last updated date

**For STATE-DESIRED.md:**
1. Document decision in Decision History section
2. Update application row(s) with new standards
3. Add rationale for non-obvious changes
4. Reference related tasks (Beads IDs) or ADRs
5. Update last updated date

---

## 📝 Report Types

| Type           | Purpose                              | Examples                                      |
|:---------------|:-------------------------------------|:----------------------------------------------|
| **Dashboard**  | Quick status overview                | STATUS.md                                     |
| **State**      | Technical configuration details      | STATE-ACTUAL.md, STATE-DESIRED.md             |
| **Analysis**   | Deep-dive investigations             | Resource analysis, VPA recommendations        |
| **Code Review**| Architecture analysis                | DRY violations, tech debt identification      |
| **Migration**  | System transitions                   | Task format migration, tool adoption          |
| **Audit**      | Compliance & security                | Secret management, permissions                |
| **Post-Mortem**| Incident analysis                    | Cluster failures, outages                     |

---

## 🤖 Automation Opportunities

### Planned Scripts (See Beads Tasks)

**Report Generation:**
- `scripts/generate-status-report.py` - Auto-generate STATUS.md from cluster state
- `scripts/generate-actual-state.py` - Query cluster and update STATE-ACTUAL.md
- `scripts/conformity-checker.py` - Compare ACTUAL vs DESIRED, output diff report

**Data Collection:**
- `scripts/collect-vpa-recommendations.py` - Fetch VPA data for all apps
- `scripts/collect-resource-usage.py` - Query Prometheus for actual usage
- `scripts/collect-backup-status.py` - Check Litestream MinIO bucket for all apps

**Validation:**
- `scripts/validate-state-consistency.py` - Ensure reports match GitOps manifests
- `scripts/validate-conformity-scores.py` - Recalculate scores against scoring model

---

## 📋 Naming Convention

### Living Documents
```
UPPERCASE.md
```
Examples: `STATUS.md`, `STATE-ACTUAL.md`, `STATE-DESIRED.md`

### Historical Reports
```
YYYY-MM-DD-<topic>.md
```
Examples:
- `2026-01-01-prod-resource-analysis.md`
- `2025-12-30-code-review.md`
- `2025-12-29-goldilocks-recommendations.md`

### Legacy Reports (Superseded)
```
UPPERCASE_WITH_UNDERSCORES.md
```
Examples: `APP_AUDIT.md`, `ULTIMATE-AUDIT.md`

---

**Last Updated:** 2026-01-10
