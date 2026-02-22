# Multi-Agent Workflow Guide

**For AI Agents (Claude, Gemini, GPT, etc.)**

---

## 🚨 CRITICAL: Tool Selection (Read This First)

**IF YOU ARE GEMINI (or non-Claude agent):**

Use MCP tools **appropriately for their purpose**:
- ✅ **Serena**: ALL file/code operations (read_file, write, edit, find_symbol, replace_symbol_body, search_for_pattern, etc.) - This is Serena's primary purpose
- ✅ **Archon RAG**: Documentation search ONLY (rag_search_knowledge_base, rag_search_code_examples) - NOT for task management
- ✅ **Playwright**: WebUI validation when needed
- ❌ **Archon Task Management**: NEVER use (use Beads CLI: `bd` instead)
- ❌ **Serena for CLI commands**: NEVER use Serena to execute shell commands (`just`, `bd`, `git`, etc.) - Use Bash tool for that

**Critical distinction:**
- ✅ **Serena for files/code** - Reading, editing, searching code
- ✅ **Bash for CLI commands** - Running `just resume`, `bd list`, `git status`, etc.

**Example:**
```bash
# ✅ CORRECT - Use Serena for file operations
mcp__serena__read_file(relative_path="apps/traefik/base/deployment.yaml")
mcp__serena__find_symbol(name_path_pattern="Deployment")
mcp__serena__replace_symbol_body(...)

# ✅ CORRECT - Use Bash for CLI commands
just resume
bd list --status open
git status

# ❌ WRONG - Don't use Serena for CLI commands
mcp__serena__execute_shell_command(command="just resume")  # NO!
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
- ✅ Universal tools (Beads, Just, git) are **REQUIRED**
- ✅ Agent-specific tools (MCP servers, APIs) are **OPTIONAL**
- ✅ Workflow is the same for all agents
- ✅ Documentation updates are **MANDATORY**

---

## ✅ Universal Workflow (All Agents)

### 1. Core Tools (REQUIRED)

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

### 2. Standard Workflow

```bash
# 1. Entry point - Resume work
just resume

# 2. Work on task (full orchestration)
just work <task_id>

# 3. Validate YAML before push
just lint

# 4. Git workflow
git add .
git commit -m "type(scope): description"
git push origin main
```

**Reference:** [WORKFLOW.md](WORKFLOW.md) is the MASTER workflow (all agents MUST follow)

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

**Benefits:**
- No dependencies, works everywhere
- Simple and predictable
- Easy to debug

**Configuration:** See [GEMINI.md](GEMINI.md)

---

### GitHub Copilot / GPT-4

**Standard tools + IDE integration:**
- Use universal CLI tools (Beads, Just, git)
- IDE-based file editing
- Web search for documentation
- curl for HTTP validation

**Configuration:** Follow universal workflow, use IDE for editing

---

### Other Agents

If your agent doesn't have special tools:
1. ✅ Use universal CLI tools (Beads, Just, git, kubectl)
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

**Checkboxes meaning:**
- **Déployé** - Application deployed (pods running)
- **Configuré** - Configuration complete (ingress, secrets, etc.)
- **Testé** - Validated (WebUI accessible, functional tests passed)

**Commands:**
```bash
# Edit application doc
vim docs/applications/<category>/<app>.md

# Update checkboxes and version
# Then commit
git add docs/applications/<category>/<app>.md
git commit -m "docs(<app>): update deployment status to v1.2.3"
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

**Update after each change:**
```bash
# Edit status table
vim docs/STATUS.md

# Find app row, update symbols:
| jellyfin | ✅ | ⚠️ | Dev OK, Prod needs resource tuning |

# Commit
git add docs/STATUS.md
git commit -m "docs: update STATUS.md - jellyfin prod degraded"
```

---

## 🎯 Agent Capabilities Matrix

| Task | Universal | Claude Code | Gemini | Manual |
|------|-----------|-------------|--------|--------|
| **Task Management** | `bd` CLI | ✅ | ✅ | ✅ |
| **File Editing** | vim/nano | Serena MCP | Standard I/O | ✅ |
| **Code Search** | grep/rg | Serena symbols | grep/rg | ✅ |
| **Doc Search** | Web search | Archon RAG | Web search | ✅ |
| **YAML Validation** | yamllint | ✅ | ✅ | ✅ |
| **WebUI Validation** | curl | Playwright MCP | curl | Browser |
| **Git Operations** | git CLI | ✅ | ✅ | ✅ |

**Key takeaway:** All agents can complete tasks using universal tools. Agent-specific tools are optimizations.

---

## 🚨 Mandatory Checklist (All Agents)

Before closing a task, EVERY agent MUST complete this checklist:

```bash
[ ] 1. Code changes made
[ ] 2. yamllint passed (just lint)
[ ] 3. Kustomize build OK (kustomize build apps/<app>/overlays/dev)
[ ] 4. Git committed to dev branch
[ ] 5. Pushed to remote (git push origin main)
[ ] 6. ArgoCD synced (verify in cluster)
[ ] 7. Application validated (curl or Playwright or manual)
[ ] 8. docs/applications/<app>.md updated ⭐ MANDATORY
[ ] 9. docs/STATUS.md updated ⭐ MANDATORY
[ ] 10. Task closed (bd close <task_id>)
```

**⭐ CRITICAL:** Steps 8-9 (documentation) are NOT optional. If you deployed it, document it.

---

## 🔄 Fallback Strategies

### If Agent Can't Use Just

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

### If Agent Can't Validate WebUI

**Use curl for HTTP checks:**
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

**Or ask user:**
```bash
echo "⚠️ Manual validation needed: https://app.dev.truxonline.com"
echo "Please verify:"
echo "  1. Page loads correctly"
echo "  2. Login works"
echo "  3. Main functionality is accessible"
```

### If Agent Can't Search Knowledge Base

**Use web search with specific queries:**
```bash
# Example searches
"Talos Linux networking configuration"
"ArgoCD sync waves documentation"
"Kustomize overlays best practices"
"Kubernetes ingress TLS configuration"
"Cilium L2 announcements setup"
```

**Official documentation sources:**
- Talos: https://www.talos.dev/
- Kubernetes: https://kubernetes.io/docs/
- ArgoCD: https://argo-cd.readthedocs.io/
- Kustomize: https://kustomize.io/
- Cilium: https://docs.cilium.io/

### If Agent Can't Edit Symbols (Serena Alternative)

**Use standard file operations:**
```bash
# Read file
cat apps/traefik/base/deployment.yaml

# Edit with sed/awk (for simple changes)
sed -i 's/replicas: 1/replicas: 2/' apps/traefik/base/deployment.yaml

# Or use vim/nano for complex edits
vim apps/traefik/base/deployment.yaml

# Verify changes
git diff apps/traefik/base/deployment.yaml
```

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

**Rule:** When in doubt, follow WORKFLOW.md. It's the source of truth.

---

## 🎓 Learning Path for New Agents

### Phase 1: Read Documentation (15 min)

1. **[README.md](README.md)** - Project overview
2. **[WORKFLOW.md](WORKFLOW.md)** - Master workflow (MUST READ)
3. **[docs/guides/task-management.md](docs/guides/task-management.md)** - Beads workflow

### Phase 2: Environment Setup (10 min)

```bash
# Install required tools
brew install steveyegge/tap/bd just kubectl yamllint

# Verify installation
bd --version && just --version && kubectl version --client

# Clone repository
git clone https://github.com/charchess/vixens.git
cd vixens
```

### Phase 3: First Task (Practice)

```bash
# Find a simple documentation task
bd list --status open --label documentation

# Work on it
just work <task_id>

# Follow the workflow
# Update docs
# Commit and push
```

### Phase 4: Real Development

```bash
# Find a real development task
bd list --status open --label feat

# Apply full workflow
just work <task_id>

# Don't forget documentation updates!
```

---

## 🧪 Validation Examples

### Example 1: Deploy Jellyfin to Dev

**Agent actions:**
```bash
# 1. Create/start task
bd create "feat: deploy jellyfin to dev" --assignee coding-agent
bd update <task_id> --status in_progress

# 2. Create application structure
mkdir -p apps/20-media/jellyfin/{base,overlays/{dev,prod}}

# 3. Write manifests
# (agent uses Serena/standard file ops)

# 4. Validate YAML
just lint

# 5. Test build
kustomize build apps/20-media/jellyfin/overlays/dev

# 6. Commit and push
git add apps/20-media/jellyfin
git commit -m "feat(jellyfin): deploy to dev"
git push origin main

# 7. Wait for ArgoCD sync (30-60s)
kubectl -n argocd get application jellyfin

# 8. Validate deployment
kubectl get pods -n media-stack -l app=jellyfin
curl -I https://jellyfin.dev.truxonline.com

# 9. ⭐ UPDATE DOCUMENTATION (MANDATORY)
vim docs/applications/20-media/jellyfin.md
# Mark: Dev [x] Déployé [x] Configuré [x] Testé

vim docs/STATUS.md
# Update: | jellyfin | ✅ | 💤 | Dev working, Prod not deployed |

git add docs/
git commit -m "docs(jellyfin): update deployment status"
git push origin main

# 10. Close task
bd close <task_id> --reason "Deployed and validated in dev"
```

**What each agent would use:**
- **Claude Code:** Serena for file editing, Archon RAG for docs, Playwright for validation
- **Gemini:** Standard file I/O, web search, curl for validation
- **Other:** Same as Gemini (universal tools)

---

## 🛡️ Safety & Best Practices

### DO ✅

- Follow WORKFLOW.md strictly
- Use `just lint` before every push
- Update documentation after EVERY deployment
- Test in dev before promoting to prod
- Close tasks only after documentation is updated
- Ask user if unsure about requirements

### DON'T ❌

- Skip documentation updates (docs/applications/*.md + docs/STATUS.md)
- Push without running `just lint`
- Commit directly to main branch
- Close tasks without validation
- Deploy to prod without dev validation
- Assume configuration without checking existing patterns

### Common Mistakes to Avoid

1. **Forgetting documentation updates** - Most common issue
2. **Not running yamllint** - Will fail GitHub Actions
3. **Wrong branch** - Always work on `dev`, never `main`
4. **Missing validation** - Always test after deployment
5. **Incomplete tasks** - Don't close until fully done

---

## 📞 Getting Help

### For Agents

1. **Read docs first:** [WORKFLOW.md](WORKFLOW.md), [docs/guides/](docs/guides/)
2. **Check examples:** Browse `docs/applications/` for patterns
3. **Ask user:** If requirements unclear, use AskUserQuestion
4. **Search knowledge base:** Archon RAG (Claude) or web search (others)

### For Users

1. **Check agent progress:** `bd list --status in_progress`
2. **View task details:** `bd show <task_id>`
3. **Manual intervention:** If agent blocked, complete task manually

---

## 🔄 Continuous Improvement

This guide evolves. If you (agent or human) encounter:
- Missing information
- Unclear instructions
- Better patterns

**Create a task:**
```bash
bd create "docs: improve AGENTS.md - [description]" --label documentation
```

---

## 📊 Success Metrics

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

## 🎯 Philosophy

**Universal > Specific**
- Prefer universal CLI tools over agent-specific APIs
- Agent-specific tools are optimizations, not requirements
- If it works in bash, it works for all agents

**Documentation > Memory**
- Write it down, don't rely on memory
- Update docs immediately, not later
- Documentation is code, treat it seriously

**Validation > Trust**
- Always validate after changes
- Use automated tests when possible
- Manual validation as fallback

---

**Last Updated:** 2026-01-08

---

🤖 **Any AI agent can successfully work on this project using this guide.**

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
