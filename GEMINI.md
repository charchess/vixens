# GEMINI.md

This file provides guidance to Gemini (and other standard AI agents) when working with this repository.

---

## 🎯 CRITICAL: MCP Tools Usage Guidelines

**GEMINI: You CAN and SHOULD use MCP tools for their intended purpose.**

### ✅ CORRECT MCP Usage

**Serena (File & Code Operations):**
- ✅ **Use Serena for ALL file/code operations** - This is its primary purpose
- ✅ `read_file`, `list_dir` - File access
- ✅ `find_symbol`, `replace_symbol_body` - Symbol operations
- ✅ `search_for_pattern`, `create_text_file` - Code search/edit
- ✅ ALL Serena file operations are encouraged

**Archon (Documentation RAG):**
- ✅ `rag_search_knowledge_base` - Search documentation (Talos, K8s, ArgoCD)
- ✅ `rag_search_code_examples` - Find code patterns
- ❌ `manage_task`, `manage_project` - Use Beads CLI (`bd`) instead

**Playwright (WebUI Validation):**
- ✅ `browser_navigate`, `browser_snapshot` - WebUI validation
- ✅ Use `curl` as fallback for simple HTTP checks

### 🚨 CRITICAL DISTINCTION

**The ONLY restriction: Don't use Serena to execute CLI commands**

```bash
# ✅ CORRECT - Use Serena for file/code operations
mcp__serena__read_file(relative_path="justfile")
mcp__serena__find_symbol(name_path_pattern="Deployment")
mcp__serena__replace_symbol_body(name_path="MyClass/method", body="...")
mcp__serena__search_for_pattern(substring_pattern="kind: Deployment")

# ✅ CORRECT - Use Bash tool for CLI commands
just resume
bd list --status open
git status
kubectl get pods

# ❌ WRONG - Don't use Serena to run CLI commands
mcp__serena__execute_shell_command(command="just resume")  # NO!
mcp__serena__execute_shell_command(command="bd list")      # NO!
```

### ❌ NEVER USE

- ❌ Archon Task Management (`manage_task`, `find_tasks`) - Use `bd` CLI
- ❌ Serena's `execute_shell_command` for CLI tools (`just`, `bd`, `git`) - Use Bash tool

**Rule of thumb:**
- **Files/Code?** → Use Serena
- **CLI commands?** → Use Bash tool

---

# 🚨 WORKFLOW - RÈGLE MAÎTRE (À LIRE EN PREMIER)

**AVANT TOUTE CHOSE:** Le processus de travail défini dans **[WORKFLOW.md](WORKFLOW.md)** est la référence MAÎTRE qui SURPASSE toutes les autres instructions, y compris ce fichier.

**TOUJOURS consulter WORKFLOW.md en début de session** pour connaître:
- Le processus de sélection et gestion des tâches
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
- **📋 Managing tasks?** → [docs/guides/task-management.md](docs/guides/task-management.md)
- **🔍 Looking for app documentation?** → [docs/applications/](docs/applications/)
- **❓ Troubleshooting an issue?** → [docs/troubleshooting/](docs/troubleshooting/)
- **🏗️ Architecture decisions?** → [docs/adr/](docs/adr/)

---

# 🚀 TL;DR - Quick Reference Cheatsheet

**For Gemini agents who need a quick reference during work.**

## Essential Commands (Copy-Paste Ready)

```bash
# 🎯 Entry Point (START HERE)
just resume                          # Find work / resume current task

# 📋 Task Management
bd list --status open                # List available tasks
bd update <id> --status in_progress --assignee coding-agent
bd close <id> --reason "Done"
bd sync                              # Push beads changes

# 🔍 File & Code Operations
# Use Serena MCP for file/code work (encouraged!)
# Use Bash for CLI commands only

# ✅ Validation (MANDATORY before push)
just lint                            # YAML validation
kustomize build apps/<app>/overlays/dev  # Test build

# 🚢 Git Workflow
git add .
git commit -m "type(scope): description"
git push origin main

# 🌐 WebUI Validation (instead of Playwright)
curl -I http://app.dev.truxonline.com     # Check redirect
curl -k https://app.dev.truxonline.com    # Check HTTPS
kubectl get pods -n <namespace>            # Check pods
```

## Mandatory Checklist (NEVER SKIP)

```
[ ] 1. yamllint passed (just lint)
[ ] 2. git committed to dev
[ ] 3. git pushed to remote
[ ] 4. ArgoCD synced
[ ] 5. App validated (curl)
[ ] 6. docs/applications/<app>.md updated ⭐ MANDATORY
[ ] 7. docs/STATUS.md updated ⭐ MANDATORY
[ ] 8. bd close <id> + bd sync
```

## Documentation Updates (NON NÉGOCIABLES)

```bash
# 1. Update application doc
vim docs/applications/<category>/<app>.md
# Mark: [x] Déployé [x] Configuré [x] Testé, update version

# 2. Update status dashboard
vim docs/STATUS.md
# Symbols: ✅ (OK) ⚠️ (Degraded) ❌ (Broken) 🚧 (WIP) 💤 (Paused)

# 3. Commit docs
git add docs/ && git commit -m "docs(<app>): update deployment status"
```

## Common Patterns (Learn by Example)

```bash
# Find existing patterns before implementing
cat apps/20-media/radarr/base/kustomization.yaml   # Copy structure
grep -r "kind: Ingress" apps/                       # Find ingress examples
diff apps/<app>/overlays/dev/kustomization.yaml \
     apps/<app>/overlays/prod/kustomization.yaml    # Compare envs
```

## Web Search Queries (instead of Archon RAG)

```
"Talos Linux [topic] configuration"
"ArgoCD [topic] documentation"
"Kustomize [topic] best practices"
"Kubernetes [topic] example"
```

**Official docs:** talos.dev, kubernetes.io/docs, argo-cd.readthedocs.io, kustomize.io, docs.cilium.io

## Key Differences vs Claude

- ✅ Same Serena (use for ALL file/code operations)
- ✅ Same Archon RAG (documentation search)
- ✅ Same Playwright (WebUI validation)
- ✅ Same Beads, Just, git workflow
- ❌ Only restriction: Don't use Serena's execute_shell_command for CLI (`just`, `bd`, etc.) - Use Bash tool

**Remember:** Serena is for files/code, Bash is for CLI commands. Use tools for their intended purpose.

---

# 🔧 Standard Tools (No Special Setup Required)

Gemini and other standard AI agents use **universal CLI tools only**. No MCP servers or special integrations required.

## Core Workflow Tools (REQUIRED)

All agents MUST have access to these CLI tools:

| Tool | Purpose | Install |
|------|---------|---------|
| **Beads (bd)** | Task management | `brew install steveyegge/tap/bd` |
| **Just** | Workflow orchestrator | `brew install just` |
| **git** | Version control | `brew install git` |
| **kubectl** | Kubernetes CLI | `brew install kubectl` |
| **yamllint** | YAML validation | `brew install yamllint` |

**Verification:**
```bash
bd --version
just --version
git --version
kubectl version --client
yamllint --version
```

---

# 🎯 Task Management with Beads (PRIMARY)

**CRITICAL:** Beads (bd) is the **ONLY** task management system for this project.

## Quick Start

```bash
# Entry point - Resume work
just resume

# Work on task (full orchestration)
just work <task_id>

# Manual operations (if needed)
bd list --status open               # List all open tasks
bd show <task_id>                   # View task details
bd update <task_id> --status in_progress --assignee coding-agent
bd close <task_id>                  # Mark complete
```

## Task Status Flow

```
open → in_progress → closed
```

**Assignee Convention:**
- `coding-agent` = AI agent (you)
- `user` = Human user

**IMPORTANT:** Only ONE task should be in `in_progress` status at a time.

---

# 📋 Standard Workflow

## 1. Start Your Session

```bash
# Always start with this
just resume
```

This shows your current work or lists open tasks.

## 2. Begin a Task

```bash
# Full orchestration (recommended)
just work <task_id>

# OR manually:
bd update <task_id> --status in_progress --assignee coding-agent
```

## 3. Research Phase (BEFORE Coding)

**Use web search for documentation:**

```bash
# Search official documentation sites
# Example queries:
# - "Talos Linux networking configuration"
# - "ArgoCD sync waves documentation"
# - "Kustomize overlays best practices"
# - "Kubernetes ingress TLS configuration"
# - "Cilium L2 announcements setup"
```

**Official documentation sources:**
- Talos: https://www.talos.dev/
- Kubernetes: https://kubernetes.io/docs/
- ArgoCD: https://argo-cd.readthedocs.io/
- Kustomize: https://kustomize.io/
- Cilium: https://docs.cilium.io/

**Search existing code patterns:**
```bash
# Find similar applications
grep -r "kind: Deployment" apps/

# Search for patterns
grep -r "middleware" apps/

# Find Kustomize overlays
find apps/ -name "kustomization.yaml"
```

## 4. Code Analysis

**Use standard file operations:**

```bash
# Read file
cat apps/traefik/base/deployment.yaml

# List directory structure
ls -la apps/20-media/jellyfin/

# Search for patterns
grep -r "storageClass" apps/

# Compare environments
diff apps/traefik/overlays/dev/kustomization.yaml \
     apps/traefik/overlays/prod/kustomization.yaml
```

## 5. Implementation

**Standard file editing:**

```bash
# Simple changes with sed
sed -i 's/replicas: 1/replicas: 2/' apps/traefik/base/deployment.yaml

# For complex edits, use your standard editing capabilities
# Read the file, understand it, then write the modified version

# Verify changes
git diff apps/traefik/base/deployment.yaml
```

## 6. Validation

```bash
# YAML validation (MANDATORY before push)
just lint
# OR manually:
yamllint -c yamllint-config.yml apps/**/*.yaml

# Kustomize build test
kustomize build apps/<app>/overlays/dev

# Dry-run (if kubectl access available)
kubectl apply --dry-run=client -k apps/<app>/overlays/dev
```

## 7. Commit and Push

```bash
# Stage changes
git add apps/<app>/

# Commit with conventional format
git commit -m "feat(<app>): description"
# Types: feat, fix, refactor, docs, chore, infra, security, monitor

# Push to dev branch
git push origin main
```

**IMPORTANT:** ArgoCD will automatically sync changes to the dev cluster.

## 8. Post-Deployment Validation

### Infrastructure Validation

```bash
# Check ArgoCD sync status
kubectl -n argocd get applications

# Check pods
kubectl get pods -n <namespace>

# Check ingress
kubectl get ingress -A
```

### WebUI Validation (Use curl)

```bash
# Check HTTP → HTTPS redirect
curl -I http://app.dev.truxonline.com
# Expected: HTTP 301/302/308

# Check HTTPS access
curl -L -k https://app.dev.truxonline.com | grep "expected-text"
# Expected: Text found

# Check response time
time curl -o /dev/null -s -w '%{http_code}\n' https://app.dev.truxonline.com
# Expected: 200
```

**If you can't validate WebUI, ask user:**
```bash
echo "⚠️ Manual validation needed: https://app.dev.truxonline.com"
echo "Please verify:"
echo "  1. Page loads correctly"
echo "  2. Login works (if applicable)"
echo "  3. Main functionality is accessible"
```

## 9. Documentation Update (MANDATORY)

**CRITICAL:** EVERY deployment MUST update documentation.

### Application Documentation

**Location:** `docs/applications/<category>/<app>.md`

**Update the deployment table:**

```markdown
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|------------|
| Dev           | [x]     | [x]       | [x]   | v1.2.3     |
| Prod          | [x]     | [x]       | [ ]   | v1.2.2     |
```

**Edit the file:**
```bash
# Use your standard editing capabilities
# Read docs/applications/<category>/<app>.md
# Update checkboxes and version
# Write the modified file

# Commit
git add docs/applications/<category>/<app>.md
git commit -m "docs(<app>): update deployment status to v1.2.3"
```

### Status Dashboard

**Location:** `docs/STATUS.md`

**Update status symbols:**
- ✅ **Working** - Deployed, tested, no issues
- ⚠️ **Degraded** - Working but needs attention
- ❌ **Broken** - Not working, needs immediate fix
- 🚧 **WIP** - Work in progress
- 💤 **Paused** - Intentionally not deployed

**Edit the status table:**
```bash
# Update app row with appropriate symbols
# Example: | jellyfin | ✅ | ⚠️ | Dev OK, Prod needs resource tuning |

git add docs/STATUS.md
git commit -m "docs: update STATUS.md - jellyfin prod degraded"
```

## 10. Complete Task

```bash
# Close task
bd close <task_id>

# OR let `just work` close it automatically after validation

# Sync Beads changes
bd sync
```

---

# 🚨 Mandatory Checklist (Before Closing Task)

```bash
[ ] 1. Code changes made
[ ] 2. yamllint passed (just lint)
[ ] 3. Kustomize build OK (kustomize build apps/<app>/overlays/dev)
[ ] 4. Git committed to dev branch
[ ] 5. Pushed to remote (git push origin main)
[ ] 6. ArgoCD synced (verify in cluster)
[ ] 7. Application validated (curl or manual)
[ ] 8. docs/applications/<app>.md updated ⭐ MANDATORY
[ ] 9. docs/STATUS.md updated ⭐ MANDATORY
[ ] 10. Task closed (bd close <task_id>)
```

**⭐ CRITICAL:** Steps 8-9 (documentation) are NOT optional. If you deployed it, document it.

---

# 🔍 Finding Information

## Search Code Patterns

```bash
# Find applications with ingress
grep -r "kind: Ingress" apps/

# Find all kustomization files
find apps/ -name "kustomization.yaml"

# Search for specific patterns
grep -r "storageClass: synology" apps/

# Find all services
grep -r "kind: Service" apps/
```

## Compare Environments

```bash
# Compare dev vs prod
diff apps/<app>/overlays/dev/kustomization.yaml \
     apps/<app>/overlays/prod/kustomization.yaml

# Check what changed between branches
git diff dev..main -- apps/<app>/
```

## Read Documentation

```bash
# View application documentation
cat docs/applications/<category>/<app>.md

# View guides
cat docs/guides/adding-new-application.md

# View status dashboard
cat docs/STATUS.md

# View architecture decisions
ls docs/adr/
cat docs/adr/008-trunk-based-gitops-workflow.md
```

---

# 📖 Repository Overview

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

**Multi-Cluster Setup:**

| Environment | Nodes | VIP | Status |
|-------------|-------|-----|--------|
| **Dev** | daphne, diva, dulce (3 CP HA) | 192.168.111.160 | ✅ Active |
| **Prod** | Physical nodes (3) | 192.168.111.200 | 📅 Phase 3 |📅 Phase 3 |

**Repository Structure:**

```
vixens/
├── apps/                # Kubernetes applications (Kustomize)
│   ├── 00-infra/       # Infrastructure (ArgoCD, Traefik, Cilium)
│   ├── 10-home/        # Home automation (Home Assistant, Mosquitto)
│   ├── 20-media/       # Media stack (Jellyfin, *arr apps)
│   ├── 40-network/     # Network services (AdGuard, DNS)
│   └── ...
├── argocd/             # ArgoCD App-of-Apps (self-management)
├── terraform/          # Infrastructure as Code (Talos clusters)
├── docs/               # Documentation
├── scripts/            # Automation scripts
├── .beads/             # Task management (git-tracked)
└── WORKFLOW.md         # Master workflow (START HERE)
```

---

# 🛠️ Essential Commands Reference

## Task Management (Beads)

```bash
# Entry point
just resume

# Full workflow
just work <task_id>

# Manual operations
bd list --status open               # List open tasks
bd list --status in_progress        # Active work
bd show <task_id>                   # View details
bd update <task_id> --status in_progress --assignee coding-agent
bd close <task_id>                  # Mark complete
bd sync                             # Sync with git remote
```

## Code Search

```bash
# Find patterns
grep -r "pattern" apps/

# Find files
find apps/ -name "*.yaml"

# Compare files
diff file1 file2

# Search with context
grep -C 5 "pattern" file
```

## YAML Validation

```bash
# Via Just
just lint

# Manual
yamllint -c yamllint-config.yml apps/**/*.yaml

# Specific directory
yamllint -c yamllint-config.yml apps/traefik/
```

## Kustomize

```bash
# Build overlay
kustomize build apps/<app>/overlays/dev

# Test build
kustomize build apps/<app>/overlays/dev | kubectl apply --dry-run=client -f -
```

## Kubernetes Operations

```bash
# Set environment
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
export TALOSCONFIG=/root/vixens/terraform/environments/dev/talosconfig-dev

# Check cluster
kubectl get nodes
kubectl get pods -A
kubectl -n argocd get applications

# Check specific app
kubectl get pods -n <namespace>
kubectl get ingress -A
kubectl describe pod <pod-name> -n <namespace>
```

## Git Operations

```bash
# Check status
git status
git branch --show-current  # MUST be "dev"

# Stage and commit
git add <files>
git commit -m "type(scope): description"

# Push
git push origin main

# Compare branches
git diff dev..main -- apps/
```

## Terraform (Infrastructure)

```bash
# Working directory
cd terraform/environments/dev

# Validate
terraform fmt -recursive
terraform validate
terraform plan

# Apply (caution!)
terraform apply
```

---

# 🚫 What NOT to Do

## DON'T ❌

- Skip documentation updates (docs/applications/*.md + docs/STATUS.md)
- Push without running `just lint`
- Commit directly to main branch (always use dev)
- Close tasks without validation
- Deploy to prod without dev validation
- Use TodoWrite (use Beads instead)
- Assume configuration without checking existing patterns
- Skip the research phase
- Have multiple tasks in `in_progress` for `coding-agent`

## DO ✅

- Follow WORKFLOW.md strictly
- Use `just lint` before every push
- Update documentation after EVERY deployment
- Test in dev before promoting to prod
- Close tasks only after documentation is updated
- Ask user if unsure about requirements
- Use Beads for ALL task management
- Research existing patterns before implementing
- Validate after deployment (kubectl + curl)

---

# 🔄 Fallback Strategies

## If Just is Not Available

```bash
# Manual equivalent of "just resume"
bd list --status in_progress --assignee coding-agent

# Manual equivalent of "just work"
bd update <task_id> --status in_progress
# ... do the work ...
yamllint -c yamllint-config.yml apps/**/*.yaml
bd close <task_id>

# Manual equivalent of "just lint"
yamllint -c yamllint-config.yml apps/**/*.yaml
```

## If Kubectl is Not Available

```bash
# Ask user to verify deployment
echo "⚠️ kubectl not available. Please verify:"
echo "  kubectl get pods -n <namespace>"
echo "  kubectl -n argocd get application <app>"
```

## If Web Access Fails

```bash
# Use curl for basic checks
curl -I https://app.dev.truxonline.com

# Ask user for manual validation
echo "⚠️ Manual validation needed: https://app.dev.truxonline.com"
```

---

# 📊 Documentation Hierarchy

**Priority Order:**

1. **[WORKFLOW.md](WORKFLOW.md)** - ⭐ MASTER workflow (overrides everything)
2. **[AGENTS.md](AGENTS.md)** - Multi-agent guide (universal patterns)
3. **[GEMINI.md](GEMINI.md)** - This file (Gemini-specific guidance)
4. **[docs/README.md](docs/README.md)** - Documentation hub
5. **[docs/guides/](docs/guides/)** - Specific guides

**Rule:** When in doubt, follow WORKFLOW.md. It's the source of truth.

---

# 🎓 Learning Path

## Phase 1: Read Documentation (15 min)

1. **[README.md](README.md)** - Project overview
2. **[WORKFLOW.md](WORKFLOW.md)** - Master workflow (MUST READ)
3. **[docs/guides/task-management.md](docs/guides/task-management.md)** - Beads workflow

## Phase 2: Environment Setup (5 min)

```bash
# Verify tools are available
bd --version
just --version
git --version
kubectl version --client
yamllint --version
```

## Phase 3: First Task (Practice)

```bash
# Find a simple documentation task
bd list --status open --label documentation

# Work on it
just work <task_id>

# Follow the workflow
# Update docs
# Commit and push
```

## Phase 4: Real Development

```bash
# Find a real development task
bd list --status open --label feat

# Apply full workflow
just work <task_id>

# Don't forget documentation updates!
```

---

# 🧪 Example Workflow: Deploy Jellyfin to Dev

**Full step-by-step example:**

```bash
# 1. Create/start task
bd create "feat: deploy jellyfin to dev" --assignee coding-agent
bd update <task_id> --status in_progress

# 2. Research phase
# Search web: "Jellyfin Kubernetes deployment"
# Look at existing apps: ls apps/20-media/

# 3. Create application structure
mkdir -p apps/20-media/jellyfin/{base,overlays/{dev,prod}}

# 4. Write manifests (use existing patterns)
# Copy from similar app, modify as needed
cat apps/20-media/radarr/base/kustomization.yaml  # Example

# 5. Validate YAML
just lint

# 6. Test build
kustomize build apps/20-media/jellyfin/overlays/dev

# 7. Commit and push
git add apps/20-media/jellyfin
git commit -m "feat(jellyfin): deploy to dev"
git push origin main

# 8. Wait for ArgoCD sync (30-60s)
kubectl -n argocd get application jellyfin

# 9. Validate deployment
kubectl get pods -n media-stack -l app=jellyfin
curl -I https://jellyfin.dev.truxonline.com

# 10. ⭐ UPDATE DOCUMENTATION (MANDATORY)
# Edit docs/applications/20-media/jellyfin.md
# Mark: Dev [x] Déployé [x] Configuré [x] Testé

# Edit docs/STATUS.md
# Update: | jellyfin | ✅ | 💤 | Dev working, Prod not deployed |

git add docs/
git commit -m "docs(jellyfin): update deployment status"
git push origin main

# 11. Close task
bd close <task_id> --reason "Deployed and validated in dev"
bd sync
```

---

# 🚨 Session Close Protocol

**CRITICAL:** Before saying "done" or "complete", you MUST run this checklist:

```bash
[ ] 1. git status              # Check what changed
[ ] 2. git add <files>         # Stage code changes
[ ] 3. bd sync                 # Commit beads changes
[ ] 4. git commit -m "..."     # Commit code
[ ] 5. bd sync                 # Commit any new beads changes
[ ] 6. git push                # Push to remote
```

**NEVER skip this.** Work is not done until pushed.

---

# 📞 Getting Help

## For AI Agents (You)

1. **Read docs first:** [WORKFLOW.md](WORKFLOW.md), [docs/guides/](docs/guides/)
2. **Check examples:** Browse `docs/applications/` for patterns
3. **Ask user:** If requirements unclear, ask for clarification
4. **Search web:** For official documentation (Talos, K8s, ArgoCD, etc.)

## For Users

1. **Check agent progress:** `bd list --status in_progress`
2. **View task details:** `bd show <task_id>`
3. **Manual intervention:** If agent blocked, complete task manually

---

# 🎯 Success Metrics

**Good agent behavior:**
- ✅ Follows WORKFLOW.md strictly
- ✅ Updates documentation consistently
- ✅ Validates before closing tasks
- ✅ Uses universal tools (Beads, Just, git)
- ✅ Asks for clarification when unsure

**Poor agent behavior:**
- ❌ Skips documentation updates
- ❌ Closes tasks without validation
- ❌ Doesn't run yamllint
- ❌ Assumes requirements without asking
- ❌ Commits to wrong branch

---

# 🔑 Key Differences from Claude Code

| Feature | Claude Code | Gemini (You) |
|---------|-------------|--------------|
| **Task Management** | Beads CLI | Beads CLI (same) |
| **File Editing** | Serena MCP (symbols) | Standard I/O |
| **Code Search** | Serena symbols | grep/find |
| **Doc Search** | Archon RAG | Web search |
| **YAML Validation** | yamllint | yamllint (same) |
| **WebUI Validation** | Playwright MCP | curl + user |
| **Git Operations** | git CLI | git CLI (same) |

**Key Takeaway:** You use the same universal tools (Beads, Just, git), but rely on standard file I/O and web search instead of specialized MCP servers. The workflow is identical.

---

# 🛡️ Safety & Best Practices

## Branch Strategy

```
dev (development) → main (production)
```

- **ALWAYS work on dev branch**
- **NEVER commit directly to main**
- Promotion to main via GitHub Actions (user-triggered)

## Validation Flow

1. **Local:** yamllint + kustomize build
2. **Git:** Push to dev branch
3. **ArgoCD:** Auto-sync to dev cluster
4. **Infrastructure:** kubectl checks
5. **WebUI:** curl or manual validation
6. **Documentation:** Update docs/applications/*.md + docs/STATUS.md
7. **Task:** Close in Beads

## Common Mistakes to Avoid

1. **Forgetting documentation updates** - Most common issue
2. **Not running yamllint** - Will fail GitHub Actions
3. **Wrong branch** - Always work on `dev`, never `main`
4. **Missing validation** - Always test after deployment
5. **Incomplete tasks** - Don't close until fully done

---

# 📚 Related Documentation

- **[WORKFLOW.md](WORKFLOW.md)** - Master workflow (READ FIRST)
- **[AGENTS.md](AGENTS.md)** - Multi-agent guide (universal patterns)
- **[docs/README.md](docs/README.md)** - Documentation hub
- **[docs/guides/task-management.md](docs/guides/task-management.md)** - Beads workflow
- **[docs/guides/adding-new-application.md](docs/guides/adding-new-application.md)** - Deployment guide
- **[docs/guides/gitops-workflow.md](docs/guides/gitops-workflow.md)** - GitOps patterns
- **[docs/STATUS.md](docs/STATUS.md)** - Status dashboard

---

**Last Updated:** 2026-01-08

---

🤖 **Gemini agents can successfully work on this project using standard CLI tools and this guide.**

prends connaissance de @AGENTS.md
