# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

---

# 🚨 RÈGLES ABSOLUES — LIRE EN PREMIER

## 1. GITOPS : JAMAIS DE KUBECTL PATCH/APPLY SUR LE CLUSTER

**LES DEUX CLUSTERS (prod ET dev) ont ArgoCD `selfHeal: true`.**

Tout `kubectl apply/patch/edit` sur des ressources gérées **est annulé en ~3 minutes** par ArgoCD.

```bash
# ❌ INTERDIT — sera revert par ArgoCD
kubectl patch application X ...
kubectl apply -f deployment.yaml
kubectl edit deployment X

# ✅ CORRECT — Fix dans Git → ArgoCD sync automatique
vim apps/<app>/base/deployment.yaml
git add . && git commit -m "fix(...): ..."
git push
```

**EXCEPTION légitime** : `kubectl annotate application X argocd.argoproj.io/refresh=hard` (déclenche juste un fetch Git, ne modifie rien).

**EXCEPTION d'urgence** (config recovery ONLY) : disable/re-enable auto-sync — **TOUJOURS re-enable immédiatement après**.

## 2. TASK MANAGEMENT : GITHUB ISSUES (pas Beads)

**Beads est déprécié.** Le task management est via GitHub Issues + Just.

```bash
just gh-resume                    # Voir le travail en cours
just gh-start <issue-number>      # Créer branch + Draft PR
just gh-done <pr-number>          # PR ready + auto-merge
gh issue list --state open        # Lister les issues
```

## 3. LIRE AGENTS.md EN DÉBUT DE SESSION

**[AGENTS.md](AGENTS.md)** contient le guide multi-agent complet avec workflow universel, checklist, et règles de sécurité.

## 4. SKILLS DISPONIBLES

Les skills opencode ont été portés comme commandes Claude Code :

| Commande | Usage |
|----------|-------|
| `/vixens-argocd-safety` | **TOUJOURS lire avant toute opération ArgoCD** |
| `/vixens-troubleshoot` | Debug cluster (avec règles GitOps) |
| `/vixens-gitops` | Workflow PR, promote prod, rollback |
| `/vixens-cluster` | Operations kubectl read-only |
| `/vixens-maturity` | Système 7 tiers, upgrade path |
| `/vixens-kubernetes-patterns` | Patterns K8s validés |
| `/vixens-app-patterns` | Templates déploiement nouvelle app |
| `/vixens-infisical` | Gestion secrets Infisical |

---

# 📋 WORKFLOW - RÉFÉRENCE MAÎTRE

**[WORKFLOW.md](WORKFLOW.md)** et **[AGENTS.md](AGENTS.md)** sont les références MAÎTRES. En cas de conflit, ils ont toujours raison.

**Ordre de priorité de la documentation :**
1. WORKFLOW.md — Master workflow
2. AGENTS.md — Guide multi-agent
3. Ce fichier (CLAUDE.md)
4. docs/README.md

---

# 📚 DOCUMENTATION HUB

**Complete documentation index:** **[docs/README.md](docs/README.md)**

### Quick Links for Common Tasks

- **🆕 Adding a new application?** → [docs/guides/adding-new-application.md](docs/guides/adding-new-application.md)
- **🚀 Pushing to production?** → [docs/guides/gitops-workflow.md](docs/guides/gitops-workflow.md)
- **📋 Managing tasks?** → [AGENTS.md](AGENTS.md) (GitHub Issues + Just)
- **🔍 Looking for app documentation?** → [docs/applications/](docs/applications/)
- **❓ Troubleshooting an issue?** → `/vixens-troubleshoot`
- **🏗️ Architecture decisions?** → [docs/adr/](docs/adr/)
- **⭐ Application quality standards?** → [docs/reference/quality-standards.md](docs/reference/quality-standards.md)

**IMPORTANT:** Documentation is organized in `docs/` with clear categories. Always check `docs/README.md` first.

---

# 🚨 TASK MANAGEMENT : GITHUB ISSUES

**NEVER use TodoWrite or Beads (deprecated).**

---

## Tool Configuration

### 1. GitHub Issues + Just - Task Management (PRIMARY)

**Task management = GitHub Issues via `gh` CLI + Just orchestration.**

```bash
# Entry point
just gh-resume                        # Voir PRs draft + issues in-progress

# Workflow standard
just gh-start <issue-number>          # Crée branch feat/<n>-<slug> + Draft PR
just gh-done <pr-number>              # PR ready + auto-merge

# Issues
gh issue list --state open            # Lister issues ouvertes
gh issue view <n>                     # Détail d'une issue
gh issue create --title "..." --label "priority:p2,type:feat"
```

**Branching convention :** `feat/<n>-<slug>`, `fix/<n>-<slug>`, `refactor/<n>-<slug>`

### 2. Just - Workflow Orchestration

**Commands defined in `justfile`:**
```bash
just gh-resume              # Voir travail en cours
just gh-start <issue-n>     # Démarrer une issue
just gh-done <pr-n>         # Finaliser un PR
just gh-tasks               # Lister issues par priorité
just lint                   # Valider YAML avant push
```

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

**IMPORTANT - Serena Usage Rules:**
- ✅ **DO use Serena for:** Code editing, file reading, symbol analysis, pattern search
- ❌ **DO NOT use Serena for:** Executing local commands (kubectl, git, bd, just, etc.)
- ❌ **DO NOT use execute_shell_command:** Use Bash tool for shell commands instead
- Serena = Code operations ONLY
- Bash = System operations (git, kubectl, terraform, etc.)

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

**Vixens** is a multi-cluster Kubernetes homelab following GitOps best practices.

**IMPORTANT:** This repository contains the GitOps application layer. Infrastructure provisioning (Terraform, Talos) is in the **[terravixens](../terravixens)** repository.

## Repository Separation

**vixens** (this repo):
- ArgoCD applications and GitOps manifests
- Kustomize overlays per environment
- Application documentation
- Development workflow (Beads, Just)

**terravixens** (infrastructure repository):
- Terraform infrastructure code
- Talos cluster provisioning
- Cilium CNI bootstrap
- ArgoCD initial deployment
- Infrastructure documentation

**Why separated?**
- Clear layer separation (infrastructure vs applications)
- Independent versioning
- Smaller repo for app changes
- Infrastructure changes don't trigger app CI

**Config files (kubeconfig/talosconfig):**
- **Source of truth:** Generated by Terraform in terravixens
- **Copied to:** `/root/vixens/.secrets/{dev,prod}/` for convenience
- **Purpose:** Agents working on vixens can access clusters without switching repos
- **Regeneration:** Copy from terravixens after `terraform apply` (see `.secrets/README.md`)

**Core Stack:**
- OS: Talos Linux v1.12.4 (immutable, API-driven)
- Kubernetes: v1.34.0
- Infrastructure: Terraform + Talos provider
- GitOps: ArgoCD v3.3.3 (App-of-Apps pattern)
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

### Focus on Current Task

**CRITICAL DISCIPLINE:** Always focus on the task at hand.

**Rules:**
1. ✅ **Work on ONE task at a time** (marked as `in_progress` in Beads)
2. ✅ **Complete the current task** before starting a new one
3. ✅ **Resist scope creep** - If you discover new work, create a Beads task for it
4. ✅ **Close the task** when done before moving to the next
5. ❌ **Do NOT start unrelated work** even if you notice issues
6. ❌ **Do NOT refactor unrelated code** unless explicitly asked

**When you discover new work:**
```bash
# Create a task for later
bd create --title="fix: discovered issue in <component>" --type=bug --priority=2

# Continue with current task
# Do NOT switch to the new task immediately
```

**Benefits:**
- Clear progress tracking
- Easier debugging (know what changed)
- Better git history
- Reduced cognitive load
- Fewer merge conflicts

### Task-Driven Development (GitHub Issues + Just)

**ALWAYS follow this cycle:**

1. **Check current work:**
   ```bash
   just gh-resume   # Shows Draft PRs + in-progress issues
   ```

2. **Start an issue:**
   ```bash
   just gh-start <issue-number>   # Creates branch + Draft PR
   ```

3. **Work, commit, validate:**
   ```bash
   git add . && git commit -m "feat(...): ..."
   git push
   just lint   # YAML validation before finalizing
   ```

4. **Finalize:**
   ```bash
   just gh-done <pr-number>   # PR ready + auto-merge
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

### Task Management (GitHub Issues)

```bash
# Entry point
just gh-resume

# Start an issue (creates branch + Draft PR)
just gh-start <issue-number>

# Validate YAML
just lint

# Finalize (PR ready + auto-merge)
just gh-done <pr-number>

# List issues
gh issue list --state open
gh issue list --label "priority:p0"
```

### Terraform (Phase 1)

**Infrastructure is in the [terravixens](../terravixens) repository.**

```bash
# Working directory
# Go to terravixens repository:
# cd terraform/environments/dev

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
# Set environment (for agents working on vixens)
export KUBECONFIG=/root/vixens/.secrets/dev/kubeconfig-dev
export TALOSCONFIG=/root/vixens/.secrets/dev/talosconfig-dev

# Alternative: direct access from terravixens (for infrastructure work)
# export KUBECONFIG=terravixens:terraform/environments/dev/kubeconfig-dev
# export TALOSCONFIG=terravixens:terraform/environments/dev/talosconfig-dev

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
terraform -chdir=<terravixens_path>/terraform/environments/dev plan  # Should show no changes
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

### Task Management
- Tasks tracked via **GitHub Issues** (repo: charchess/vixens)
- Workflow orchestrated via Just (`justfile`) : `just gh-resume`, `just gh-start <n>`, `just gh-done <n>`
- NEVER use TodoWrite, Beads (deprecated), or Archon for task management

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

**MANDATORY for ALL work in this repository:**

#### 1. DRY (Don't Repeat Yourself)
- **Never duplicate code/configuration** - create shared resources instead
- Use Kustomize bases and overlays for common patterns
- Extract common values to `_shared/` directory
- If you copy-paste, you're doing it wrong

**Examples:**
- Shared Helm values in `apps/_shared/helm-values/`
- Common Kustomize components in `apps/_shared/components/`
- Reusable Terraform modules in `terravixens/terraform/modules/`

#### 2. State of the Art
- **Follow industry best practices** for Kubernetes/GitOps
- Use latest stable versions when upgrading
- Follow Kubernetes resource best practices (limits, probes, labels)
- Implement proper observability (metrics, logs, traces)
- Follow 12-factor app principles

**Required Standards:**
- All services must have health checks (Elite tier requires liveness probe)
- All services must expose metrics
- All services must have proper resource requests/limits
- All ingress must use HTTPS with proper certificates

#### 3. GitOps First
- **Everything in Git** - no manual kubectl apply in production
- All changes go through PR review
- ArgoCD is the source of truth for cluster state
- Infrastructure as Code with Terraform
- Declarative configuration only

**Exceptions:**
- Dev environment: kubectl allowed for troubleshooting (must be consolidated to GitOps after)
- Emergency production fixes: document and backport to Git immediately

#### 4. Best Practices
- **Security:** Follow security best practices (least privilege, secrets management, network policies)
- **Reliability:** Implement proper HA, backups, monitoring
- **Performance:** Resource optimization, efficient storage usage
- **Maintainability:** Clear documentation, consistent naming, logical structure

**Enforcement:**
- Validation scripts check for common issues
- PR reviews ensure compliance
- Documentation must be updated with code changes
- ADRs document architectural decisions

---

## Quick Reference

**For new features/services:**
1. `just gh-resume` — voir le travail en cours
2. `just gh-start <issue-number>` — créer branch + Draft PR
3. Implémenter les changements
4. `just lint` — valider YAML
5. Update `docs/applications/<app>.md` + `docs/STATUS.md` (MANDATORY)
6. `just gh-done <pr-number>` — merge

**For troubleshooting:**
1. `/vixens-argocd-safety` — lire AVANT toute opération ArgoCD
2. `/vixens-troubleshoot` — guide debug avec règles GitOps
3. DIAGNOSE avec read-only kubectl, FIX dans Git

**Pour kubectl :**
- ✅ Read-only (`get`, `describe`, `logs`) : toujours OK
- ✅ `kubectl annotate application X argocd.argoproj.io/refresh=hard` : OK (safe)
- ❌ `kubectl apply/patch/edit` sur ressources gérées : JAMAIS (revert ArgoCD)

**Tips:**
- Secret `infisical-universal-auth` est dans le namespace `argocd` pour tous les operators
- Créer ADRs pour les décisions architecturales
- HTTP → HTTPS redirect à configurer sur tous les services publics
- PRs uniquement vers `main` (trunk-based)

---

## GitOps Workflow (Pure Trunk-Based)

**Branch:** `main` (single source of truth)

**Environment Differentiation:**
- **Dev**: ArgoCD watches `main` branch (HEAD)
- **Prod**: ArgoCD watches `prod-stable` Git tag (`prod-working` pour backup config fonctionnelle)

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

# OR use prod-working (last known working config)
git tag -f prod-stable prod-working
git push origin prod-stable --force
```

**Key Benefits:**
- No merge conflicts between branches
- Linear Git history
- Faster dev deployment (no PR bottleneck)
- Industry standard practice (Google, Netflix, Spotify)

See [ADR-017](docs/adr/017-pure-trunk-based-single-branch.md) for details. Supersedes ADR-008/009.

---

**Last Updated:** 2026-01-12 (terraform moved to terravixens)
