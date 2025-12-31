# Task Management with Archon

This guide explains how to create, update, and manage tasks in the Archon project management system.

---

## Overview

Archon MCP is the **PRIMARY** task management system for this project. All tasks are tracked in Archon, not in TodoWrite or markdown files.

**Archon URL:** http://localhost:3000 (when running locally)

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

Tasks use a **dual priority system**: priority label + task_order numeric value.

### Priority Labels

| Priority | Range | Description | Use Case |
|----------|-------|-------------|----------|
| **p0** | 90-100 | Critical blockers | Production down, security breach |
| **p1** | 70-89 | High priority | Infrastructure issues, important features |
| **p2** | 40-69 | Medium priority | Standard features, improvements |
| **p3** | 10-39 | Low priority | Nice-to-have, research |

### Task Order (0-100)

Higher number = higher execution priority

**Guidelines:**
- **90-100:** Critical work (P0 blockers)
- **70-89:** High priority work (P1)
- **40-69:** Medium priority work (P2)
- **10-39:** Low priority work (P3)

**Example:**
```
task_order: 95, priority: p0  → Critical blocker
task_order: 85, priority: p1  → High priority infrastructure
task_order: 50, priority: p2  → Standard feature
task_order: 20, priority: p3  → Nice-to-have research
```

---

## Task Workflow

### Task Status Flow

```
todo → doing → review → done
```

| Status | Meaning | Next Action |
|--------|---------|-------------|
| `todo` | Not started | Pick up and move to doing |
| `doing` | Currently working | Complete and move to review |
| `review` | Completed, needs validation | Test and move to done |
| `done` | Completed and validated | Archive |

**IMPORTANT:** Only ONE task should be in `doing` status at a time.

---

## Creating Tasks

### Using Archon MCP Tools

```python
# Create task
manage_task(
    action="create",
    project_id="15926c59-1ba7-4e9e-b77a-aa4b22a91a6a",  # Vixens project
    title="feat: deploy jellyseerr in media namespace",
    description="""## Context
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
""",
    status="todo",
    assignee="Coding Agent",
    priority="p2",
    task_order=45,
    feature="media-library"
)
```

### Description Template

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

---

## Finding Tasks

### List All Tasks

```python
# Get all tasks
find_tasks(per_page=50)

# Search by keyword
find_tasks(query="jellyfin")

# Filter by status
find_tasks(filter_by="status", filter_value="todo")
find_tasks(filter_by="status", filter_value="doing")

# Filter by feature
find_tasks(filter_by="feature", filter_value="media-library")

# Get specific task (full details)
find_tasks(task_id="550e8400-e29b-41d4-a716-446655440000")
```

### Get Current Work

```python
# What am I working on?
find_tasks(filter_by="status", filter_value="doing")

# What's ready to work on? (no blockers)
# Note: Use `bd ready` in beads workflow if available
find_tasks(filter_by="status", filter_value="todo")
```

---

## Updating Tasks

### Start Working on Task

```python
manage_task(
    action="update",
    task_id="...",
    status="doing"
)
```

### Complete Task

```python
# Mark for review
manage_task(
    action="update",
    task_id="...",
    status="review"
)

# Or mark as done (if no review needed)
manage_task(
    action="update",
    task_id="...",
    status="done"
)
```

### Change Priority

```python
manage_task(
    action="update",
    task_id="...",
    priority="p1",
    task_order=85
)
```

### Reassign Task

```python
manage_task(
    action="update",
    task_id="...",
    assignee="User"  # or "Coding Agent" or "Archon"
)
```

---

## Feature Tags

Group related tasks with feature labels:

| Feature | Description | Examples |
|---------|-------------|----------|
| `architecture-cleanup` | Architecture refactoring | DRY violations, tech debt |
| `monitoring` | Observability & dashboards | Prometheus, Grafana, alerts |
| `infrastructure` | Terraform, Talos, Cilium | Cluster upgrades, resource limits |
| `databases` | PostgreSQL, backups | CloudNativePG, backup strategy |
| `cluster-services` | General K8s services | Linkwarden, Docspell, tools |
| `media-library` | Media management | Jellyfin, *arr stack |
| `downloads` | Download managers | qBittorrent, Pyload |
| `security` | Security tools | CrowdSec, Trivy, Kyverno |
| `home-automation` | Home Assistant | MQTT, integrations |
| `storage` | PVC, CSI, backups | Synology CSI, Velero |
| `automation` | CI/CD, automation | Renovate, scripts |
| `documentation` | Docs, guides | Restructuring, ADRs |
| `gitops-workflow` | GitOps processes | Branch strategy, promotion |

---

## Task Management Cycle

### Standard Workflow

1. **Find work:**
   ```python
   find_tasks(filter_by="status", filter_value="todo", per_page=10)
   ```

2. **Start task:**
   ```python
   manage_task("update", task_id="...", status="doing")
   ```

3. **Research before coding:**
   ```python
   # Search knowledge base
   rag_search_knowledge_base(query="kubernetes pvc", match_count=5)

   # Find code examples
   rag_search_code_examples(query="kustomize overlays", match_count=3)
   ```

4. **Implement solution**

5. **Complete task:**
   ```python
   manage_task("update", task_id="...", status="review")
   ```

6. **Get next task:**
   ```python
   find_tasks(filter_by="status", filter_value="todo", per_page=10)
   ```

---

## Best Practices

### DO ✅
- Use conventional commit style for task titles
- Write detailed descriptions with acceptance criteria
- Set appropriate priority and task_order
- Tag tasks with feature labels
- Mark tasks as `doing` before starting work
- Complete ONE task at a time
- Move to `review` when done (not `done` immediately)

### DON'T ❌
- Use TodoWrite (use Archon instead)
- Create tasks with empty descriptions
- Skip acceptance criteria
- Have multiple tasks in `doing` status
- Use vague titles ("Fix stuff", "Update things")
- Forget to update task status

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

## Troubleshooting

### Too many TODO tasks (50+)

**Solution:** Review and clean up:
```python
# Find old tasks
find_tasks(filter_by="status", filter_value="todo", per_page=100)

# Archive or delete irrelevant tasks
manage_task("delete", task_id="...")
```

### Task stuck in "doing" for days

**Solution:** Either complete it or break it down:
```python
# Complete if done
manage_task("update", task_id="...", status="review")

# Or create smaller subtasks and close original
```

### Can't find the right task

**Solution:** Use search and filters:
```python
# Search by keyword
find_tasks(query="media jellyfin")

# Filter by feature
find_tasks(filter_by="feature", filter_value="media-library")
```

---

## Related Documentation

- [Reference: Task Formalism](../reference/task-formalism.md) - Complete formalism spec
- [GitOps Workflow](gitops-workflow.md) - Promoting changes
- [Adding New Application](adding-new-application.md) - Implementation guide

---

## Archon MCP Tools Reference

```python
# Task Management
find_tasks(query=None, task_id=None, filter_by=None, filter_value=None, per_page=10)
manage_task(action, task_id=None, project_id=None, title=None, description=None, status=None, assignee=None, task_order=None, priority=None, feature=None)

# Knowledge Base
rag_search_knowledge_base(query, source_id=None, match_count=5)
rag_search_code_examples(query, source_id=None, match_count=5)
rag_get_available_sources()
rag_read_full_page(page_id=None, url=None)

# Project Management
find_projects(project_id=None, query=None, page=1, per_page=10)
manage_project(action, project_id=None, title=None, description=None)
```

---

**Last Updated:** 2025-12-30
