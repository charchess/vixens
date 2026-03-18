# Multi-Agent Workflow Guide

**For AI Agents (Claude, Gemini, GPT, etc.)**

---

## 🚨 CRITICAL: Tool Selection (Read This First)

**IF YOU ARE GEMINI (or non-Claude agent):**

Use MCP tools **appropriately for their purpose**:
- ✅ **Serena**: ALL file/code operations (read_file, write, edit, find_symbol, replace_symbol_body, search_for_pattern, etc.) - This is Serena's primary purpose
- ✅ **Archon RAG**: Documentation search ONLY (rag_search_knowledge_base, rag_search_code_examples) - NOT for task management
- ✅ **Playwright**: WebUI validation when needed
- ❌ **Archon Task Management**: NEVER use (use `gh issue` CLI instead)
- ❌ **Serena for CLI commands**: NEVER use Serena to execute shell commands (`just`, `gh`, `git`, etc.) - Use Bash tool for that

**Critical distinction:**
- ✅ **Serena for files/code** - Reading, editing, searching code
- ✅ **Bash for CLI commands** - Running `just gh-resume`, `gh issue list`, `git status`, etc.

**Example:**
```bash
# ✅ CORRECT - Use Serena for file operations
mcp__serena__read_file(relative_path="apps/traefik/base/deployment.yaml")
mcp__serena__find_symbol(name_path_pattern="Deployment")
mcp__serena__replace_symbol_body(...)

# ✅ CORRECT - Use Bash for CLI commands
just gh-resume
gh issue list --state open
git status

# ❌ WRONG - Don't use Serena for CLI commands
mcp__serena__execute_shell_command(command="just gh-resume")  # NO!
```

**IF YOU ARE CLAUDE CODE:**
- ✅ Use Serena intensively (symbols, AST operations)
- ✅ Use Archon RAG extensively
- ✅ Use Playwright for all WebUI validation
- ✅ Use universal CLI tools as fallback

**Why this matters:** MCP tools are optimizations. Gemini should use them lightly (file access, docs, validation) but rely primarily on bash for code work.

---

## 🎯 Purpose

This guide ensures **any AI agent** can work on this project using universal tools and processes, with optional agent-specific optimizations.

**Core Philosophy:**
- ✅ Universal tools (GitHub Issues via `gh`, Just, git) are **REQUIRED**
- ✅ Agent-specific tools (MCP servers, APIs) are **OPTIONAL**
- ✅ Workflow is the same for all agents
- ✅ Documentation updates are **MANDATORY**

---

## ✅ Universal Workflow (All Agents)

### 1. Core Tools (REQUIRED)

All agents MUST have access to these CLI tools:

| Tool | Purpose | Install |
|------|---------|---------|
| **gh** | Task management (GitHub Issues) | `brew install gh` |
| **Just** | Workflow orchestrator | `brew install just` |
| **git** | Version control | `brew install git` |
| **kubectl** | Kubernetes CLI | `brew install kubectl` |
| **yamllint** | YAML validation | `brew install yamllint` |
| **kustomize** | Kustomize build/validate | `brew install kustomize` |

**Verification:**
```bash
gh --version
just --version
git --version
kubectl version --client
yamllint --version
kustomize version
```

### 2. Standard Workflow

```bash
# 1. Entry point - Resume work
just gh-resume

# 2. Start an issue (creates branch + Draft PR)
just gh-start <issue-number>

# 3. Work on the task, commit incrementally
git add . && git commit -m "wip: description"
git push

# 4. Validate YAML before finalizing
just lint

# 5. Mark PR as ready when done
just gh-done <pr-number>
```

**Reference:** [WORKFLOW.md](WORKFLOW.md) is the MASTER workflow (all agents MUST follow)

---

## 📋 Task Management (GitHub Issues)

**Task tracking is done via GitHub Issues.** No external tool required — `gh` CLI is the only interface.

### Key Commands

```bash
# See work in progress (Draft PRs + in-progress issues + P0/P1 queue)
just gh-resume

# Start working on an issue
just gh-start 42       # creates feat/42-<slug> branch + Draft PR + labels issue in-progress

# List all open tasks by priority
just gh-tasks

# Create a new issue
gh issue create --title "feat(app): description" --label "priority:p2,type:feat"

# View issue details
gh issue view 42

# Search issues
gh issue list --state open --label "priority:p0"
gh issue list --state open --search "kyverno"

# Close an issue (usually automatic via PR merge "Closes #N")
gh issue close 42 --reason "completed"
```

### Labels

| Label | Values |
|-------|--------|
| Priority | `priority:p0` `priority:p1` `priority:p2` `priority:p3` |
| Type | `type:feat` `type:fix` `type:chore` `type:research` `type:docs` `type:refactor` `type:security` `type:eval` |
| Status | `status:in-progress` `status:blocked` |

### Branch Naming Convention

```
feat/<issue-number>-<short-slug>     # new features
fix/<issue-number>-<short-slug>      # bug fixes
refactor/<issue-number>-<short-slug> # refactoring
```

Example: `feat/42-deploy-jellyfin`

### Work State = Git State

The real work state is always in git, not in an external tool:

| Signal | Meaning |
|--------|---------|
| Branch `feat/<n>-*` exists | Work started |
| Draft PR open | Work in progress |
| Commits on branch | Progress (readable via `git log`) |
| PR ready (non-draft) | Ready for review/merge |
| PR merged → issue auto-closed | Done |

**Crash recovery:** `gh pr list --search "is:draft"` → checkout the branch → `git log --oneline -5` → continue.

---

## 🔧 Agent-Specific Tools (Optional Optimizations)

### Claude Code

**MCP Servers available:**
- **Serena** - Symbol-aware code editing (AST manipulation)
- **Archon** - RAG knowledge base (Talos/K8s/ArgoCD docs indexed)
- **Playwright** - WebUI validation (browser automation)

**Benefits:**
- Faster code navigation (symbol search vs grep)
- Instant documentation lookup (RAG vs web search)
- Automated WebUI testing (Playwright vs manual curl)

**Configuration:** See [CLAUDE.md](CLAUDE.md)

**Fallback if MCP unavailable:** Use universal tools below

---

### Gemini

**Standard tools (no special setup):**
- **File editing:** Standard read/write operations
- **Documentation:** Web search for official docs
- **Validation:** curl/wget for HTTP testing
- **Code search:** grep/rg for pattern matching

**Configuration:** See [GEMINI.md](GEMINI.md)

---

### Other Agents

If your agent doesn't have special tools:
1. ✅ Use universal CLI tools (`gh`, Just, git, kubectl)
2. ✅ Use standard file operations (read/write via shell)
3. ✅ Search web for documentation when needed
4. ✅ Use curl for HTTP validation
5. ✅ Ask user for manual validation if needed

**Key principle:** If you can run shell commands, you can complete the full workflow.

---

## 📚 Documentation Update Protocol (MANDATORY)

### When to Update Documentation

**EVERY deployment MUST update documentation. No exceptions.**

| Event | Action |
|-------|--------|
| ✅ Deploy to dev | Update `docs/applications/<app>.md` + `docs/STATUS.md` |
| ✅ Deploy to prod | Update `docs/applications/<app>.md` + `docs/STATUS.md` |
| ✅ Change configuration | Update `docs/applications/<app>.md` |
| ✅ Discover issue | Update `docs/STATUS.md` (mark ⚠️ or ❌) |
| ✅ Fix issue | Update `docs/STATUS.md` (mark ✅) |
| ✅ Infrastructure change | Update `docs/RECETTE-TECHNIQUE.md` |

### Application Documentation

**Location:** `docs/applications/<category>/<app>.md`

**Update the deployment table:**

```markdown
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.2.3  |
| Prod          | [x]     | [x]       | [ ]   | v1.2.2  |
```

**Template:** [docs/templates/application-doc-template.md](docs/templates/application-doc-template.md)

---

### Status Dashboard

**Location:** `docs/STATUS.md`

**Quick status symbols:**
- ✅ **Working** - Deployed, tested, no issues
- ⚠️ **Degraded** - Working but needs attention (resources, config)
- ❌ **Broken** - Not working, needs immediate fix
- 🚧 **WIP** - Work in progress
- 💤 **Paused** - Intentionally not deployed

---

## 🎯 Agent Capabilities Matrix

| Task | Universal | Claude Code | Gemini | Manual |
|------|-----------|-------------|--------|--------|
| **Task Management** | `gh issue` CLI | ✅ | ✅ | ✅ |
| **File Editing** | vim/nano | Serena MCP | Standard I/O | ✅ |
| **Code Search** | grep/rg | Serena symbols | grep/rg | ✅ |
| **Doc Search** | Web search | Archon RAG | Web search | ✅ |
| **YAML Validation** | yamllint | ✅ | ✅ | ✅ |
| **WebUI Validation** | curl | Playwright MCP | curl | Browser |
| **Git Operations** | git CLI | ✅ | ✅ | ✅ |

---

## 🚨 Mandatory Checklist (All Agents)

Before closing a task, EVERY agent MUST complete this checklist:

```bash
[ ] 1. Code changes made on feature branch (feat/<n>-<slug>)
[ ] 2. yamllint passed (just lint)
[ ] 3. Kustomize build OK (kustomize build apps/<app>/overlays/dev)
[ ] 3b. Kustomize kinds diff OK: compare kinds before/after any kustomization.yaml change
[ ] 4. Git committed and pushed to feature branch
[ ] 5. PR merged to main (gh pr merge <n> --squash --auto)
[ ] 6. ArgoCD synced (verify in cluster)
[ ] 7. Application validated (curl or Playwright or manual)
[ ] 8. docs/applications/<app>.md updated ⭐ MANDATORY
[ ] 9. docs/STATUS.md updated ⭐ MANDATORY
[ ] 10. GitHub Issue closed (auto via PR merge "Closes #N", or gh issue close <n>)
```

**⭐ CRITICAL:** Steps 8-9 (documentation) are NOT optional. If you deployed it, document it.

**⭐ KUSTOMIZE DIFF RULE:** After any `kustomization.yaml` change (add/remove resources, patches, components), run:
```bash
kustomize build apps/<app>/overlays/<env> | grep '^kind:' | sort
```
Compare kinds before/after. A missing `kind` means a resource was silently dropped.

---

## 🏗️ Kustomize Structure (Key Patterns)

### Shared Components

Located in `apps/_shared/components/`. Apply via `components:` in kustomization.yaml.

| Component | Purpose | Where to reference |
|-----------|---------|-------------------|
| `base/` | Pod securityContext (runAsNonRoot, fsGroup, seccompProfile) | `app/base/kustomization.yaml` |
| `revision-history-limit/` | revisionHistoryLimit: 3 | `app/base/kustomization.yaml` |
| `dev-hibernate/` | replicas: 0 in dev | `overlays/dev/kustomization.yaml` |
| `sync-wave/wave-N/` | ArgoCD sync wave | `overlays/prod/kustomization.yaml` |
| `goldilocks/enabled/` | VPA recommendations | `overlays/prod/kustomization.yaml` |
| `infisical/env-prod/` | Prod secrets injection | `overlays/prod/kustomization.yaml` |
| `poddisruptionbudget/` | PDB for prod HA | `overlays/prod/kustomization.yaml` |
| `nometrics/` | Disable metrics scraping | `app/base/kustomization.yaml` |

### Overlay Rules

- **`app/base/kustomization.yaml`**: env-agnostic components (base, revision-history-limit, nometrics)
- **`overlays/dev/`**: dev-only (dev-hibernate)
- **`overlays/prod/`**: prod-only (sync-wave, goldilocks, infisical/env-prod, poddisruptionbudget)

### Maturity Tiers

`Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum`

"Elite" does not exist — use the correct tier name above.

---

## 🔄 Fallback Strategies

### If Agent Can't Use Just

```bash
# Manual equivalent of "just gh-resume"
gh pr list --state open --search "is:draft"
gh issue list --label "status:in-progress"
gh issue list --label "priority:p0" --state open

# Manual workflow
gh issue edit <n> --add-label "status:in-progress"
git checkout -b feat/<n>-<slug>
# ... do the work ...
just lint
gh pr create --draft --title "..." --body "Closes #<n>"
# when done:
gh pr ready <pr-n>
gh pr merge <pr-n> --squash --auto
```

### If Agent Can't Validate WebUI

```bash
# Check HTTP → HTTPS redirect
curl -I http://app.dev.truxonline.com

# Check HTTPS access
curl -L -k https://app.dev.truxonline.com | grep "expected-text"

# Check response time
time curl -o /dev/null -s -w '%{http_code}\n' https://app.dev.truxonline.com
```

### If Agent Can't Search Knowledge Base

**Official documentation sources:**
- Talos: https://www.talos.dev/
- Kubernetes: https://kubernetes.io/docs/
- ArgoCD: https://argo-cd.readthedocs.io/
- Kustomize: https://kustomize.io/
- Cilium: https://docs.cilium.io/

---

## 📖 Documentation Hierarchy (All Agents)

**Priority Order:**

1. **[WORKFLOW.md](WORKFLOW.md)** - ⭐ MASTER workflow (overrides everything)
2. **[AGENTS.md](AGENTS.md)** - This file (multi-agent guide)
3. **Agent-specific guides:**
   - [CLAUDE.md](CLAUDE.md) - Claude Code optimizations
   - [GEMINI.md](GEMINI.md) - Gemini patterns
4. **[docs/README.md](docs/README.md)** - Documentation hub
5. **[docs/guides/](docs/guides/)** - Specific guides

---

## 🎓 Learning Path for New Agents

### Phase 1: Read Documentation (15 min)

1. **[README.md](README.md)** - Project overview
2. **[WORKFLOW.md](WORKFLOW.md)** - Master workflow (MUST READ)

### Phase 2: Environment Setup (10 min)

```bash
# Install required tools
brew install gh just kubectl yamllint kustomize

# Authenticate gh
gh auth login

# Verify installation
gh --version && just --version && kubectl version --client

# Clone repository
git clone https://github.com/charchess/vixens.git
cd vixens
```

### Phase 3: First Task

```bash
# See what's open
just gh-resume

# Start a task
just gh-start 42

# Follow WORKFLOW.md for task execution
# Update docs, commit, push, done
just gh-done <pr-number>
```

---

## 🧪 Validation Example: Deploy Jellyfin to Dev

```bash
# 1. Start the issue
just gh-start 42   # creates feat/42-deploy-jellyfin + Draft PR

# 2. Create application structure
mkdir -p apps/20-media/jellyfin/{base,overlays/{dev,prod}}

# 3. Write manifests + add shared components in base/kustomization.yaml

# 4. Validate
just lint
kustomize build apps/20-media/jellyfin/overlays/dev | grep '^kind:' | sort

# 5. Commit and push
git add apps/20-media/jellyfin
git commit -m "feat(jellyfin): deploy to dev"
git push

# 6. Wait for ArgoCD sync
kubectl -n argocd get application jellyfin

# 7. Validate deployment
kubectl get pods -n media -l app=jellyfin
curl -I https://jellyfin.dev.truxonline.com

# 8. ⭐ UPDATE DOCUMENTATION (MANDATORY)
# Update docs/applications/20-media/jellyfin.md and docs/STATUS.md
git add docs/ && git commit -m "docs(jellyfin): update deployment status" && git push

# 9. Finalize — PR ready, auto-close issue via "Closes #42" in PR body
just gh-done <pr-number>
```

---

## 🛡️ Safety & Best Practices

### DO ✅

- Use `just gh-resume` at the start of every session
- Create a feature branch + Draft PR immediately when starting work
- Run `just lint` before every push
- Update documentation after EVERY deployment
- Test in dev before promoting to prod
- Use `Closes #N` in PR body to auto-close issues on merge

### DON'T ❌

- Skip documentation updates (docs/applications/*.md + docs/STATUS.md)
- Push without running `just lint`
- Push directly to `main` — branch protection requires PRs
- Close issues without validation
- Deploy to prod without dev validation
- Use "Elite" as a maturity tier name (correct: Bronze/Silver/Gold/Platinum/Emerald/Diamond/Orichalcum)

### Common Mistakes to Avoid

1. **Forgetting documentation updates** - Most common issue
2. **Not running yamllint** - Will fail GitHub Actions
3. **Pushing directly to main** - Always use feature branches via `just gh-start`
4. **Missing validation** - Always test after deployment
5. **Kustomize regression** - Always compare kinds before/after kustomization.yaml changes

---

## 📞 Getting Help

### For Agents

1. **Read docs first:** [WORKFLOW.md](WORKFLOW.md), [docs/guides/](docs/guides/)
2. **Check examples:** Browse `docs/applications/` for patterns
3. **Ask user:** If requirements unclear, ask before implementing
4. **Search knowledge base:** Archon RAG (Claude) or web search (others)

### For Users

1. **Check agent progress:** `gh pr list --search "is:draft"`
2. **View task details:** `gh issue view <n>`
3. **Manual intervention:** If agent blocked, complete task manually

---

## 🔄 Continuous Improvement

This guide evolves. If you (agent or human) encounter missing information, unclear instructions, or better patterns:

```bash
gh issue create --title "docs: improve AGENTS.md - [description]" \
  --label "priority:p3,type:docs"
```

---

## 📊 Success Metrics

**Good agent behavior:**
- ✅ Uses `just gh-resume` at session start
- ✅ Creates feature branch + Draft PR for every task
- ✅ Updates documentation consistently
- ✅ Validates before closing tasks
- ✅ Asks for clarification when unsure

**Poor agent behavior:**
- ❌ Skips documentation updates
- ❌ Closes tasks without validation
- ❌ Doesn't run yamllint
- ❌ Pushes directly to main
- ❌ Uses wrong maturity tier names

---

## 🎯 Philosophy

**Single Source of Truth**
- GitHub Issues = tasks (never lose data, git-backed)
- git branches + PRs = work in progress
- No external task databases to sync

**Documentation > Memory**
- Write it down, don't rely on memory
- Update docs immediately, not later
- Documentation is code, treat it seriously

**Validation > Trust**
- Always validate after changes
- Use automated tests when possible
- Manual validation as fallback

---

## 🛠️ Skill Management (Proactive)

**During work, AI agents should proactively:**

### Suggest Creating Skills
When you notice repeated operations (3+ times in a session):
> "I notice we've done [X operation] multiple times. Would you like me to create a skill for this?"

### Suggest Fixing Skills
While executing a skill, watch for commands that fail or missing information.

### Reference
Use skill: `Claude-skill-creator` for detailed guidance on creating/maintaining skills.

---

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** — `gh issue create` for anything needing follow-up
2. **Run quality gates** (if code changed) — `just lint`, kustomize builds
3. **Push all branches** and ensure Draft PRs are open for in-progress work
4. **PUSH TO REMOTE** — This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Update prod-stable tag** if changes are production-ready:
   ```bash
   git tag -f prod-stable origin/main
   git push origin refs/tags/prod-stable --force
   ```
6. **Verify** — All changes committed AND pushed

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing — that leaves work stranded locally
- Draft PRs are your "I was here" marker — always open one when starting work

---

**Last Updated:** 2026-03-18

---

🤖 **Any AI agent can successfully work on this project using this guide.**
