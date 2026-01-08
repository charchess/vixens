# Task Management with Beads

This guide explains how to create, update, and manage tasks using Beads and the Just workflow orchestrator.

---

## Overview

**Beads (bd)** is the **PRIMARY** and **ONLY** task management system for this project. All tasks are tracked in Beads (`.beads/` directory), NOT in TodoWrite, Archon Task Management, or markdown files.

**Workflow Orchestration:** Just (`WORKFLOW.just`) provides convenient commands for task management and workflow automation.

---

## Quick Start

```bash
# Entry point: Resume current work or see available tasks
just resume

# Work on a specific task (full orchestration)
just work <task_id>

# Manual operations (if needed)
bd list --status open               # List all open tasks
bd show <task_id>                   # View task details
bd update <task_id> --status in_progress --assignee coding-agent
bd close <task_id>                  # Mark complete
```

---

## Task Formalism

Tasks follow conventional commit style for consistency and searchability.

### Title Format

```
<type>[(<scope>)]: <action> <object> [<context>]
```

**Examples:**
```
feat: deploy firefly-iii in finance namespace
fix: resolve alertmanager crashloop
refactor(tech-debt): centralize http redirect middleware
docs: update deployment guide
chore: configure homepage api keys
infra: upgrade talos to v1.11.0
security: deploy crowdsec with traefik
monitor: integrate apps with grafana
research: evaluate vaultwarden postgresql migration
```

### Task Types

| Type | Purpose | Examples |
|------|---------|----------|
| `feat` | New feature/service | Deploy app, add functionality |
| `fix` | Bug fix | Resolve errors, fix issues |
| `refactor` | Code/architecture refactoring | Cleanup, reorganization |
| `docs` | Documentation | Write guides, update docs |
| `chore` | Maintenance tasks | Configuration, cleanup |
| `infra` | Infrastructure changes | Terraform, Talos, cluster |
| `security` | Security enhancements | Deploy tools, policies |
| `monitor` | Monitoring/observability | Dashboards, metrics |
| `research` | Investigation/evaluation | Feasibility studies |
| `perf` | Performance optimization | Resource tuning |
| `misc` | Other (use sparingly) | Uncategorized tasks |

---

## Priority System

Beads uses simple priority levels for task ordering.

### Priority Levels

| Priority | Description | Use Case |
|----------|-------------|----------|
| **urgent** | Critical blockers | Production down, security breach |
| **high** | High priority | Infrastructure issues, important features |
| **normal** | Medium priority (default) | Standard features, improvements |
| **low** | Low priority | Nice-to-have, research |

**Usage:**
```bash
bd create "feat: deploy app" --priority urgent
bd update <task_id> --priority high
```

---

## Task Workflow

### Task Status Flow

```
open → in_progress → closed
```

| Status | Meaning | Next Action |
|--------|---------|-------------|
| `open` | Not started | Pick up and move to in_progress |
| `in_progress` | Currently working | Complete and close |
| `closed` | Completed | Archive |

**IMPORTANT:** Only ONE task should be in `in_progress` status at a time (especially for `coding-agent`).

### Assignee Convention

- `coding-agent` = Claude Code (AI)
- `user` = Human user

---

## Using Just Workflow

### just resume

**Entry point for all development work.**

```bash
just resume
```

**What it does:**
- Checks for tasks in `in_progress` assigned to `coding-agent`
- Shows current task if found
- Lists open tasks if no work in progress

**Example output:**
```
🔥 RESUME: beads-abc123 | feat: deploy jellyseerr
   Commande: just work beads-abc123
```

### just work <task_id>

**Full workflow orchestration for a task.**

```bash
just work beads-abc123
```

**What it does:**
1. **Phase 1 (Prerequisites):** Checks for technical requirements (PVC RWO, tolerations)
2. **Phase 2 (Documentation):** Identifies relevant app documentation
3. **Phase 3 (Implementation):** Guides you through development (you use Serena/Archon RAG here)
4. **Phase 4 (Validation):** Runs `scripts/validate.py` automatically
5. **Closes task** if validation passes

**Use this for standard development workflows.** Manual operations only when needed.

### just burst <title>

**Quick idea capture.**

```bash
just burst "feat: investigate backup solution"
```

Creates a task immediately without blocking workflow.

---

## Creating Tasks

### Using Beads CLI

```bash
# Create basic task
bd create "feat: deploy jellyseerr in media namespace" \
  --assignee coding-agent \
  --priority normal \
  --label media

# Create with full description
bd create "feat: deploy jellyseerr" \
  --assignee coding-agent \
  --priority high \
  --description "Deploy Jellyseerr with Radarr/Sonarr integration" \
  --label media,deployment
```

### Task Description Template

Use structured markdown for clarity:

```markdown
## Context
[Why this task exists]

## Current State
[What's the situation now]

## Target State
[What we want to achieve]

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3

## Dependencies
[What blocks this task or what it blocks]

## Impact
[Scope and importance]

## References
[Links to docs, ADRs, related tasks]
```

**Example:**
```bash
bd create "feat: deploy jellyseerr" --description "$(cat <<'EOF'
## Context
Need media request management tool.

## Current State
- No request management system
- Users manually request media

## Target State
- Jellyseerr deployed in media namespace
- Integrated with Radarr/Sonarr
- Public ingress configured

## Acceptance Criteria
- [ ] Create Kustomize base and overlays
- [ ] Configure Ingress for dev/prod
- [ ] Integrate with Infisical for secrets
- [ ] Connect to Radarr and Sonarr
- [ ] Add ArgoCD application manifest
- [ ] Verify deployment in dev
- [ ] Promote to prod

## Dependencies
Requires Radarr and Sonarr to be operational.
EOF
)"
```

---

## Finding Tasks

### List Tasks

```bash
# List all open tasks
bd list --status open

# List in-progress tasks
bd list --status in_progress

# List completed tasks
bd list --status closed

# Filter by assignee
bd list --assignee coding-agent --status open

# Filter by label
bd list --label media --status open

# Limit results
bd list --status open --limit 20
```

### View Task Details

```bash
# Show full task details
bd show <task_id>

# Show in JSON format
bd show <task_id> --json
```

### Search Tasks

```bash
# Beads search (if available)
bd list --status open | grep jellyfin

# Or use interactive filtering
bd list --status open --limit 50
```

---

## Updating Tasks

### Start Working on Task

```bash
# Manual start
bd update <task_id> --status in_progress --assignee coding-agent

# Or use just work (recommended)
just work <task_id>
```

### Complete Task

```bash
# Close task
bd close <task_id>

# Close with reason
bd close <task_id> --reason "Deployed and validated in dev"

# Close multiple tasks
bd close <task_id1> <task_id2> <task_id3>
```

### Change Priority

```bash
bd update <task_id> --priority high
```

### Reassign Task

```bash
bd update <task_id> --assignee user
```

### Add Notes

```bash
bd update <task_id> --notes "Blocked by missing DNS records"
```

---

## Labels & Organization

Group related tasks with labels:

| Label | Description | Examples |
|-------|-------------|----------|
| `architecture-cleanup` | Architecture refactoring | DRY violations, tech debt |
| `monitoring` | Observability & dashboards | Prometheus, Grafana, alerts |
| `infrastructure` | Terraform, Talos, Cilium | Cluster upgrades, resource limits |
| `databases` | PostgreSQL, backups | CloudNativePG, backup strategy |
| `cluster-services` | General K8s services | Linkwarden, Docspell, tools |
| `media` | Media management | Jellyfin, *arr stack |
| `downloads` | Download managers | qBittorrent, Pyload |
| `security` | Security tools | CrowdSec, Trivy, Kyverno |
| `home-automation` | Home Assistant | MQTT, integrations |
| `storage` | PVC, CSI, backups | Synology CSI, Velero |
| `automation` | CI/CD, automation | Renovate, scripts |
| `documentation` | Docs, guides | Restructuring, ADRs |
| `gitops-workflow` | GitOps processes | Branch strategy, promotion |

**Usage:**
```bash
bd create "feat: deploy app" --label media,deployment
bd list --label media --status open
```

---

## Task Management Cycle

### Standard Workflow

1. **Resume work:**
   ```bash
   just resume
   ```

2. **Start task:**
   ```bash
   just work <task_id>  # Full orchestration
   # OR manually:
   bd update <task_id> --status in_progress --assignee coding-agent
   ```

3. **Research before coding (MANDATORY):**
   ```python
   # Search knowledge base (Archon RAG - documentation only)
   rag_search_knowledge_base(query="kubernetes pvc", match_count=5)

   # Find code examples
   rag_search_code_examples(query="kustomize overlays", match_count=3)
   ```

4. **Implement solution** (use Serena for code editing)

5. **Complete task:**
   ```bash
   bd close <task_id>
   # Or let `just work` close it automatically
   ```

6. **Get next task:**
   ```bash
   just resume
   ```

---

## Best Practices

### DO ✅
- Use `just resume` as your entry point
- Use `just work <task_id>` for standard development
- Use conventional commit style for task titles
- Write detailed descriptions with acceptance criteria
- Set appropriate priority
- Tag tasks with labels
- Complete ONE task at a time
- Let `just work` handle validation automatically

### DON'T ❌
- Use TodoWrite (use Beads instead)
- Use Archon for task management (Archon = RAG only)
- Create tasks with empty descriptions
- Skip acceptance criteria
- Have multiple tasks in `in_progress` for `coding-agent`
- Use vague titles ("Fix stuff", "Update things")
- Forget to update task status
- Skip the research phase (Archon RAG)

---

## Task Granularity

**For feature-specific projects:**
Create detailed implementation tasks (3-5 subtasks per feature):
- Setup & planning
- Implementation
- Testing
- Documentation

**For codebase-wide projects:**
Create feature-level tasks (1 task = 1 feature)

**Rule of thumb:** Each task should represent 30 minutes to 4 hours of work.

---

## Dependencies & Blocking

Beads supports explicit dependencies between tasks.

```bash
# Add dependency (task1 depends on task2)
bd dep add <task1> <task2>

# List blocked tasks
bd blocked

# Show what blocks a task
bd show <task_id>  # Dependencies section shows blockers
```

---

## Sync & Collaboration

```bash
# Sync with git remote
bd sync

# Check sync status
bd sync --status
```

**IMPORTANT:** Beads uses `.beads/` directory which is git-tracked. Changes are automatically committed.

**At session end (MANDATORY):**
```bash
bd sync  # Commit and push beads changes
```

---

## Project Health

```bash
# Project statistics
bd stats

# Check for issues (sync problems, missing hooks)
bd doctor

# List blocked tasks
bd blocked
```

---

## Troubleshooting

### Too many open tasks (50+)

**Solution:** Review and clean up:
```bash
bd list --status open --limit 100
bd close <task_id1> <task_id2> ...  # Close completed/obsolete tasks
```

### Task stuck in "in_progress" for days

**Solution:** Either complete it or break it down:
```bash
# Complete if done
bd close <task_id>

# Or create smaller subtasks and close original
bd create "feat: subtask 1" --assignee coding-agent
bd close <original_task_id> --reason "Split into smaller tasks"
```

### Can't find the right task

**Solution:** Use filters and labels:
```bash
# Filter by label
bd list --label media --status open

# Filter by assignee
bd list --assignee coding-agent --status open

# Check blocked tasks
bd blocked
```

### Lost track of current work

**Solution:**
```bash
just resume  # Always your entry point
```

---

## Integration with Archon (RAG Only)

**IMPORTANT:** Archon is used ONLY for knowledge base search (RAG), NOT for task management.

**Before implementing any task, use Archon RAG:**

```python
# Search documentation (keep queries SHORT - 2-5 keywords)
rag_search_knowledge_base(query="talos networking", match_count=5)

# Find code examples
rag_search_code_examples(query="terraform cilium", match_count=3)

# Get available documentation sources
rag_get_available_sources()

# Search specific source
rag_search_knowledge_base(
  query="keywords",
  source_id="src_xxx",
  match_count=5
)
```

**Query Tips:**
- ✅ Good: "cilium l2 ipam", "talos vip", "kustomize overlay"
- ❌ Bad: "how to configure Cilium L2 announcements with IPAM in Kubernetes"

---

## Related Documentation

- [WORKFLOW.md](../../WORKFLOW.md) - Master workflow (RÈGLE MAÎTRE)
- [CLAUDE.md](../../CLAUDE.md) - Tool configuration and usage
- [Reference: Task Formalism](../reference/task-formalism.md) - Complete formalism spec
- [GitOps Workflow](gitops-workflow.md) - Promoting changes
- [Adding New Application](adding-new-application.md) - Implementation guide

---

## Command Reference

### Beads CLI

```bash
# Task Operations
bd create <title> [--assignee=<assignee>] [--priority=<priority>] [--label=<labels>] [--description=<desc>]
bd list [--status=<status>] [--assignee=<assignee>] [--label=<label>] [--limit=<n>]
bd show <task_id> [--json]
bd update <task_id> [--status=<status>] [--assignee=<assignee>] [--priority=<priority>] [--notes=<notes>]
bd close <task_id> [<task_id2> ...] [--reason=<reason>]

# Dependencies
bd dep add <task> <depends-on>
bd blocked

# Project Management
bd ready          # Show tasks ready to work (no blockers)
bd stats          # Project statistics
bd doctor         # Check for issues
bd sync           # Sync with git remote
bd sync --status  # Check sync status
```

### Just Commands

```bash
just resume              # Entry point: find/resume current work
just work <task_id>      # Full workflow orchestration
just burst <title>       # Quick idea capture
```

### Archon RAG (Knowledge Base Only)

```python
rag_search_knowledge_base(query, source_id=None, match_count=5)
rag_search_code_examples(query, source_id=None, match_count=5)
rag_get_available_sources()
rag_read_full_page(page_id=None, url=None)
rag_list_pages_for_source(source_id, section=None)
```

---

## Session Close Protocol

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

**Last Updated:** 2026-01-08
