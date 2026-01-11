# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

---

# 🚨 WORKFLOW - RÈGLE MAÎTRE (À LIRE EN PREMIER)

**AVANT TOUTE CHOSE:** Le processus de travail défini dans **[WORKFLOW.md](WORKFLOW.md)** est la référence MAÎTRE qui SURPASSE toutes les autres instructions, y compris ce fichier.

**TOUJOURS consulter WORKFLOW.md en début de session** pour connaître:
- Le processus de sélection et gestion des tâches (Beads)
- L'ordre de priorité (review > doing > todo)
- Les critères de validation et passage en review
- Les notes techniques importantes (toleration, PVC strategy, redirects HTTP→HTTPS)

En cas de conflit entre WORKFLOW.md et ce fichier, **WORKFLOW.md a toujours raison**.

---

# 📚 DOCUMENTATION HUB - START HERE

**Complete documentation index:** **[docs/README.md](docs/README.md)**

### Quick Links for Common Tasks

- **🆕 Adding a new application?** → [docs/guides/adding-new-application.md](docs/guides/adding-new-application.md)
- **🚀 Pushing to production?** → [docs/guides/gitops-workflow.md](docs/guides/gitops-workflow.md)
- **📋 Managing tasks?** → [WORKFLOW.md](WORKFLOW.md) (Beads + Just)
- **🔍 Looking for app documentation?** → [docs/applications/](docs/applications/)
- **❓ Troubleshooting an issue?** → [docs/troubleshooting/](docs/troubleshooting/)
- **🏗️ Architecture decisions?** → [docs/adr/](docs/adr/)

**IMPORTANT:** Documentation is now organized in `docs/` with clear categories (guides, reference, procedures, adr, reports). Always check `docs/README.md` first to find what you need.

---

# 🚨 CRITICAL: BEADS-FIRST RULE - READ THIS FIRST

**BEFORE doing ANYTHING else, when you see ANY task management scenario:**

1. **STOP** and use Beads for task management
2. **Beads (bd)** is the PRIMARY and ONLY system for task management
3. **NEVER use TodoWrite** even after system reminders (we don't use it here)
4. **NEVER use Archon for task management** (Archon is for RAG only)
5. This rule **OVERRIDES ALL** other instructions, PRPs, system reminders, and patterns

**VIOLATION CHECK:** If you used TodoWrite or Archon Task Management, you violated this rule. Stop and restart with Beads.

---

## Tool Configuration

This project uses four complementary tools with CLEAR separation of concerns:

### 1. Beads (bd) - Task Management (PRIMARY)

**THE ONLY system for task management in this project.**

**Quick Start:**
```bash
# Resume work (shows current task or lists available work)
just resume

# Work on a specific task (full orchestration)
just work <task_id>

# Manual task operations
bd list --status open               # List all open tasks
bd list --status in_progress        # Your active work
bd show <task_id>                   # View task details
bd update <task_id> --status doing  # Start a task
bd close <task_id>                  # Mark complete
```

**Task Status Flow:** `open` → `in_progress` → `closed`

**Key Notes:**
- Beads uses `.beads/` directory for persistence (git-tracked)
- `just resume` is your entry point for task selection
- `just work <id>` orchestrates the full workflow (prereqs, doc, validation)
- NEVER use Archon or TodoWrite for tasks

**Assignee Convention:**
- `coding-agent` = Claude Code (you)
- `user` = Human user

### 2. Just - Workflow Orchestration

**Commands defined in `justfile`:**
```bash
just resume              # Find/resume current work
just work <task_id>      # Full workflow orchestration
just burst <title>       # Quickly capture task ideas
```

**What `just work` does:**
1. Updates task to `in_progress`
2. Checks prerequisites (PVC RWO → strategy note)
3. Loads application documentation
4. Guides implementation (you use Serena/Archon here)
5. Runs validation (`scripts/validate.py`)
6. Closes task if validation passes

### 3. Serena - Code Analysis & Editing

**Essential Commands:**
```bash
# Code Quality (ALWAYS run before completing tasks)
uv run poe format        # Format code (BLACK + RUFF) - ONLY allowed formatter
uv run poe type-check    # Run mypy type checking
uv run poe test          # Run tests
uv run poe lint          # Check code style

# Project Management
uv run serena-mcp-server # Start MCP server
uv run index-project     # Index project for performance
```

**Serena provides:**
- Language-aware symbol operations (find_symbol, replace_symbol_body)
- File manipulation (read, write, edit)
- Pattern search (search_for_pattern)
- Memory persistence (write_memory, read_memory)

**Use Serena for:** Reading/editing code, analyzing symbols, searching patterns.

### 4. Archon - Knowledge Base (RAG ONLY)

**IMPORTANT:** Archon is ONLY used for documentation search and code examples. NOT for task management.

**Core RAG Operations:**
```bash
# Search documentation (2-5 keywords ONLY!)
rag_search_knowledge_base(query="talos networking", match_count=5)

# Find code examples
rag_search_code_examples(query="terraform cilium", match_count=3)

# Get available documentation sources
rag_get_available_sources()

# Search specific source
rag_search_knowledge_base(
  query="keywords",
  source_id="src_xxx",  # From get_available_sources()
  match_count=5
)
```

**Query Tips:**
- ✅ Good: "cilium l2 ipam", "talos vip", "kustomize overlay"
- ❌ Bad: "how to configure Cilium L2 announcements with IPAM in Kubernetes"

**Use Archon for:** Searching Talos/Kubernetes/ArgoCD docs, finding implementation patterns.

### 5. Playwright - WebUI Validation

**Used for validating web service access after deployments.**

Example validation workflow:
```python
# Navigate to service
mcp__playwright__browser_navigate(url="https://argocd.dev.truxonline.com")

# Capture page state
mcp__playwright__browser_snapshot()
```

**When to use:** After deploying services to verify WebUI accessibility (see [RECETTE-FONCTIONNELLE.md](docs/RECETTE-FONCTIONNELLE.md)).

---

## Documentation Hierarchy

**Key Principle:** Information lives in ONE authoritative place. Other locations LINK to it, never duplicate.

**📚 MAIN HUB:** **[docs/README.md](docs/README.md)** - Start here for all documentation

**Key Documentation:**
- **[WORKFLOW.md](WORKFLOW.md)** - Master workflow (Beads + Just)
- **[docs/guides/](docs/guides/)** - How-to guides (adding apps, gitops)
- **[docs/reference/](docs/reference/)** - Technical references (sync-waves, kustomize)
- **[docs/applications/](docs/applications/)** - Application documentation by category
- **[docs/adr/](docs/adr/)** - Architecture Decision Records
- **[docs/procedures/](docs/procedures/)** - Operational procedures
- **[docs/RECETTE-FONCTIONNELLE.md](docs/RECETTE-FONCTIONNELLE.md)** - Functional validation
- **[docs/RECETTE-TECHNIQUE.md](docs/RECETTE-TECHNIQUE.md)** - Technical validation

**IMPORTANT:** Keep RECETTE-*.md up to date when infrastructure or services change.

---

## Repository Overview

**Vixens** is a multi-cluster Kubernetes homelab infrastructure following GitOps best practices.

**Core Stack:**
- OS: Talos Linux v1.11.0 (immutable, API-driven)
- Kubernetes: v1.34.0
- Infrastructure: Terraform + Talos provider
- GitOps: ArgoCD v7.7.7 (App-of-Apps pattern)
- CNI: Cilium v1.18.3 (eBPF, kube-proxy replacement)
- LoadBalancer: Cilium L2 Announcements + LB IPAM
- Ingress: Traefik v3.x
- Storage: Synology CSI (iSCSI)
- Secrets: Infisical (self-hosted)

**Current Phase:** Phase 2 (GitOps Infrastructure) - Active production

---

## Multi-Cluster Architecture

4 independent clusters on dedicated VLANs:

| Environment | Nodes | VLAN Internal | VLAN Services | VIP | Status |
|-------------|-------|---------------|---------------|-----|--------|
| **Dev** | obsy, onyx, opale (3 CP HA) | 111 | 208 | 192.168.111.160 | ✅ Active |
| **Test** | citrine, carny, celesty (3 CP HA) | 111 | 209 | 192.168.111.180 | ⏳ Planned |
| **Staging** | TBD (3 nodes) | 111 | 210 | 192.168.111.190 | 📅 Future |
| **Prod** | Physical nodes (3) | 111 | 201 | 192.168.111.200 | 📅 Phase 3 |

**Dual-VLAN Network Architecture:**
- **VLAN 111** (192.168.111.0/24) - Non-routed, internal (etcd, kubelet, storage)
- **VLAN 20X** (192.168.20X.0/24) - Routed, services (Ingress, LoadBalancer)

**Key Infrastructure:**
- Storage: Synology NAS at 192.168.111.69
- Management: grenat at 192.168.111.64

---

## Repository Structure

```
vixens/
├── terraform/                      # Phase 1: Infrastructure as Code
│   ├── modules/
│   │   ├── shared/                # DRY Module (Single Source of Truth)
│   │   ├── talos/                 # Reusable Talos cluster module
│   │   ├── cilium/                # Cilium CNI module
│   │   └── argocd/                # ArgoCD GitOps module
│   └── environments/
│       ├── dev/                   # Dev cluster (obsy, onyx, opale - 3 CP HA)
│       ├── test/                  # Test cluster (citrine, carny, celesty - 3 CP HA)
│       ├── staging/               # Staging cluster
│       └── prod/                  # Prod cluster
│
├── argocd/                        # Phase 2: ArgoCD self-management
│   ├── base/
│   └── overlays/                  # dev, test, staging, prod
│
├── apps/                          # Phase 2: Infrastructure & Application services
│   ├── 00-infra/                  # Infrastructure (ArgoCD, Traefik, Cilium, etc.)
│   ├── 01-storage/                # Storage (Synology CSI, NFS)
│   ├── 02-monitoring/             # Monitoring (Prometheus, Grafana, Loki)
│   ├── 03-security/               # Security (Authentik)
│   ├── 04-databases/              # Databases (CloudNativePG, MariaDB, etc.)
│   ├── 10-home/                   # Home automation
│   ├── 20-media/                  # Media applications
│   ├── 40-network/                # Network services
│   ├── 60-services/               # General services
│   ├── 70-tools/                  # Tools & utilities
│   ├── 99-test/                   # Test applications
│   └── _shared/                   # Shared resources
│
├── docs/                          # Documentation
│   ├── README.md                  # Documentation hub
│   ├── RECETTE-FONCTIONNELLE.md  # ⚠️ Keep updated - Functional validation
│   ├── RECETTE-TECHNIQUE.md      # ⚠️ Keep updated - Technical validation
│   ├── adr/                       # Architecture Decision Records
│   ├── applications/              # App docs (mirroring apps/ structure)
│   ├── guides/                    # How-to guides
│   ├── procedures/                # Operational procedures
│   ├── reference/                 # Technical references
│   └── troubleshooting/           # Troubleshooting guides
│
├── scripts/                       # Automation scripts
│   ├── validate.py                # Validation script (used by `just work`)
│   └── testing/                   # Test suites
│
├── .beads/                        # Beads task management (git-tracked)
├── .claude/                       # Claude Code configuration
│   └── hooks/                     # Workflow hooks
│
├── WORKFLOW.md                    # Master workflow (RÈGLE MAÎTRE)
├── justfile                       # Just commands (bd + workflow)
└── CLAUDE.md                      # This file
```

---

## Development Workflow

### Task-Driven Development (Beads + Just)

**ALWAYS follow this cycle:**

1. **Check current work:**
   ```bash
   just resume  # Shows current task or lists open tasks
   ```

2. **Start a task:**
   ```bash
   just work <task_id>  # Full orchestration
   ```

3. **What happens during `just work`:**
   - Phase 1: Checks prerequisites (PVC RWO, tolerations)
   - Phase 2: Identifies relevant documentation
   - Phase 3: You implement (use Serena for code, Archon for research)
   - Phase 4: Automatic validation via `scripts/validate.py`

4. **Manual task operations (if needed):**
   ```bash
   bd update <task_id> --status in_progress
   bd close <task_id>
   ```

### Working with Multiple Environments

**IMPORTANT:** Trunk-based workflow with 2 environments:
- **dev**: Development and testing (cluster dev)
- **main/prod**: Production (cluster prod)
- **test/staging**: Planned for future

When working on production, base your analysis on dev environment. Compare differences.

Example workflow:
```bash
# Compare configurations
diff apps/traefik/overlays/dev/kustomization.yaml \
     apps/traefik/overlays/prod/kustomization.yaml

# Check what changed between dev and main
git diff dev..main -- apps/

# Promote to production
gh workflow run promote-prod.yaml -f version=v1.2.3
```

---

## Essential Commands

### Beads (Task Management)

```bash
# Resume workflow
just resume

# Work on task (full orchestration)
just work <task_id>

# Manual task operations
bd list --status open
bd show <task_id>
bd update <task_id> --status in_progress
bd close <task_id>

# Quick idea capture
just burst "Implement feature X"
```

### Terraform (Phase 1)

```bash
# Working directory
cd terraform/environments/dev

# Standard workflow
terraform fmt -recursive
terraform init
terraform validate
terraform plan
terraform apply

# Destroy/recreate (dev/test only!)
terraform destroy -auto-approve
terraform apply -auto-approve
```

**Destroy/Recreate Strategy:**
- Safe for: dev, test (virtualized)
- Dangerous for: staging, prod (physical infrastructure)
- Use when: validating reproducibility, major refactoring
- NOT for: normal development (just apply changes)

### Kubernetes Operations

```bash
# Set environment
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
export TALOSCONFIG=/root/vixens/terraform/environments/dev/talosconfig-dev

# Check cluster
kubectl get nodes -o wide
kubectl get pods -A

# Talos operations
talosctl --nodes 192.168.111.162 --endpoints 192.168.111.162 version
talosctl --nodes 192.168.111.162 health
talosctl --nodes 192.168.111.160 etcd members
```

### Secrets Management (Infisical)

**Instance:** http://192.168.111.69:8085 (self-hosted on NAS)

```bash
# Check InfisicalSecret status
kubectl get infisicalsecret -n cert-manager gandi-credentials-sync -o yaml

# Verify secret sync
kubectl get secret -n cert-manager gandi-credentials -o jsonpath='{.data}' | jq 'keys'

# Force reconciliation
kubectl annotate infisicalsecret gandi-credentials-sync \
  -n cert-manager \
  --overwrite \
  reconcile="$(date +%s)"
```

**Secret Architecture:**
- Project: `vixens`
- Environments: `dev`, `test`, `staging`, `prod`
- Paths: `/cert-manager`, `/synology-csi`, etc.

### Validation & Testing

**Functional Validation (WebUI):**
See [docs/RECETTE-FONCTIONNELLE.md](docs/RECETTE-FONCTIONNELLE.md)

**Technical Validation (Infrastructure):**
See [docs/RECETTE-TECHNIQUE.md](docs/RECETTE-TECHNIQUE.md)

**Automated Validation:**
```bash
# Used by `just work` automatically
python3 scripts/validate.py <app_name> dev
```

**Quick checks:**
```bash
terraform -chdir=terraform/environments/dev plan  # Should show no changes
kubectl get nodes                                  # All should be Ready
kubectl -n argocd get applications                 # All should be Synced+Healthy
```

---

## Important Notes

### Phase 1 (Completed ✅)
- Terraform-managed infrastructure
- Immutable Talos nodes (disposable, reproducible)
- Per-node configuration (disk, network, patches)
- Dual-VLAN required (internal 111 + services 20X)
- HA etcd quorum (3 control planes validated)

### Phase 2 (Current - GitOps Active ✅)
- ArgoCD App-of-Apps managing all services
- Kustomize overlays per environment
- Branch per environment (dev, test, staging, main)
- Auto-sync enabled (git push = automatic deployment)
- Zero manual kubectl commands required

### Beads Task Management
- Tasks tracked in Beads (`.beads/` directory)
- Workflow orchestrated via Just (`justfile`)
- Task status: open → in_progress → closed
- NEVER use TodoWrite or Archon Task Management

### Recipe Files Maintenance
**CRITICAL:** When infrastructure or services change, update:
- [docs/RECETTE-FONCTIONNELLE.md](docs/RECETTE-FONCTIONNELLE.md)
- [docs/RECETTE-TECHNIQUE.md](docs/RECETTE-TECHNIQUE.md)

---

## Technical Notes (Important)

### Controlplane Scheduling
Applications that need to run on control plane nodes require tolerations:
```yaml
tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
```

### Storage Strategy
**PVC with ReadWriteOnce (RWO)** → Deployment must use `strategy: Recreate`:
```yaml
spec:
  strategy:
    type: Recreate
```

### Network Configuration
**HTTP to HTTPS Redirection:** Always configure for public-facing services.

**Certificates:**
- Dev: `letsencrypt-staging` (for testing)
- Prod: `letsencrypt-prod` (production)

**Ingress URLs:**
- Dev: `<app>.dev.truxonline.com`
- Prod: `<app>.truxonline.com`

### Design Principles
- **DRY (Don't Repeat Yourself):** Use shared resources when possible
- **Maintainability:** Clear structure, good documentation
- **State of the Art:** Follow Kubernetes/GitOps best practices
- **Reproducibility:** Everything in git, infrastructure as code

---

## Quick Reference

**For new features/services:**
1. Check Beads tasks first (`just resume`)
2. Research with Archon RAG knowledge base
3. Compare with lower environments (if applicable)
4. Implement changes (use Serena for code editing)
5. Update recipe files if infrastructure/services changed
6. Validate with `just work` (automatic validation)
7. Close task in Beads

**For troubleshooting:**
1. Check [RECETTE-TECHNIQUE.md](docs/RECETTE-TECHNIQUE.md)
2. Compare with working lower environment
3. Verify ArgoCD sync status
4. Check Archon knowledge base for similar issues

**For validation:**
1. Functional: [RECETTE-FONCTIONNELLE.md](docs/RECETTE-FONCTIONNELLE.md) + Playwright
2. Technical: [RECETTE-TECHNIQUE.md](docs/RECETTE-TECHNIQUE.md)
3. Always run: `terraform plan` (should show no changes)

**kubectl apply/edit/delete:**
- Acceptable in DEV for troubleshooting/confirmation
- Must be consolidated with GitOps approach afterward
- NEVER in prod (GitOps only)

**Beads Tips:**
- `find_tasks` has a window of 10 tasks, extend with `--limit` for more
- Secret `infisical-universal-auth` is in namespace `argocd` for all operators
- Remember to signal when secrets need creation in Infisical or DNS updates
- Remember to create ADRs for architectural decisions
- When configuring HTTPS ingress, set up HTTP → HTTPS redirect
- Cannot merge dev to main directly, must PR: dev → test → staging → main

---

## GitOps Workflow (Pure Trunk-Based)

**Branch:** `main` (single source of truth)

**Environment Differentiation:**
- **Dev**: ArgoCD watches `main` branch (HEAD)
- **Prod**: ArgoCD watches `prod-stable` Git tag

**Development Flow:**
1. Create feature branch from `main`
2. Develop and commit changes
3. Create PR to `main`: `gh pr create --base main --head feature/xyz`
4. Merge PR → ArgoCD auto-syncs to **dev cluster**
5. Validate in dev environment

**Production Promotion:**
1. After validation in dev: `gh workflow run promote-prod.yaml -f version=v1.2.3`
2. Workflow moves `prod-stable` tag to current main HEAD
3. ArgoCD auto-syncs to **prod cluster**

**Rollback:**
```bash
# Move prod-stable tag to previous version
git tag -f prod-stable prod-v1.2.2
git push origin prod-stable --force
```

**Key Benefits:**
- No merge conflicts between branches
- Linear Git history
- Faster dev deployment (no PR bottleneck)
- Industry standard practice (Google, Netflix, Spotify)

See [ADR-017](docs/adr/017-pure-trunk-based-single-branch.md) for details. Supersedes ADR-008/009.

---

**Last Updated:** 2026-01-11
