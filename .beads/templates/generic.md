# Generic Task Template

Use this template when creating tasks with `bd create`.

---

## Context

[Why does this task exist? What problem are we solving?]

---

## Current State

[What's the situation now? What exists today?]

---

## Target State

[What do we want to achieve? What will exist after completion?]

---

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3
- [ ] Criterion 4
- [ ] Criterion 5

---

## Implementation Plan (Optional)

1. **Step 1:** [First action]
2. **Step 2:** [Second action]
3. **Step 3:** [Third action]

---

## Dependencies

[What blocks this task? What tasks must be completed first?]

**Blockers:**
- `beads-xxx` - [Task that blocks this one]

**Blocks:**
- `beads-yyy` - [Task that is blocked by this one]

---

## Impact

**Scope:** [dev-only | dev+prod | infrastructure | documentation]

**Priority:** [urgent | high | normal | low]

**Affected Components:**
- Component 1
- Component 2

---

## Documentation Updates Required

**CRITICAL:** All deployments MUST update documentation.

### Application Documentation
- [ ] Update `docs/applications/<category>/<app>.md`
- [ ] Mark checkboxes: Déployé, Configuré, Testé
- [ ] Update version number

### Status Dashboard
- [ ] Update `docs/STATUS.md`
- [ ] Set appropriate status symbol (✅⚠️❌🚧💤)
- [ ] Add/update notes column

### Recipe Files (if infrastructure changes)
- [ ] Update `docs/RECETTE-FONCTIONNELLE.md` (functional tests)
- [ ] Update `docs/RECETTE-TECHNIQUE.md` (technical validation)

---

## Validation Checklist

### Pre-Implementation
- [ ] Task reviewed and understood
- [ ] Dependencies checked (no blockers)
- [ ] Research completed (Archon RAG / web search)
- [ ] Existing patterns analyzed (similar apps)
- [ ] Branch verified (`git branch --show-current` = "dev")

### Implementation
- [ ] Code changes made
- [ ] YAML validation passed (`just lint`)
- [ ] Kustomize build successful
- [ ] Changes committed with conventional format
- [ ] Pushed to dev branch (`git push origin dev`)

### Post-Deployment
- [ ] ArgoCD synced successfully
- [ ] Pods running (`kubectl get pods -n <namespace>`)
- [ ] Ingress configured (`kubectl get ingress -A`)
- [ ] WebUI accessible (curl or manual validation)
- [ ] Functional validation completed (RECETTE-FONCTIONNELLE.md)
- [ ] Technical validation completed (RECETTE-TECHNIQUE.md)

### Documentation
- [ ] Application documentation updated (docs/applications/<app>.md)
- [ ] Status dashboard updated (docs/STATUS.md)
- [ ] Recipe files updated (if applicable)
- [ ] Changes committed and pushed

### Task Closure
- [ ] All acceptance criteria met
- [ ] All validation checks passed
- [ ] All documentation updated
- [ ] Task closed in Beads (`bd close <task_id>`)
- [ ] Beads changes synced (`bd sync`)

---

## References

[Links to related documentation, ADRs, similar tasks, external docs]

- [Related ADR](../docs/adr/XXX-description.md)
- [Similar task](beads-xxx)
- [External documentation](https://example.com/docs)

---

## Notes

[Additional notes, warnings, special considerations]

---

## Usage

### Create a task from this template

```bash
# Interactive creation
bd create "feat: deploy jellyfin" \
  --assignee coding-agent \
  --priority normal \
  --label media,deployment \
  --description "$(cat .beads/templates/generic.md)"

# Then edit the task to fill in the template
bd edit <task_id>
```

### Quick task creation (without template)

```bash
# For simple tasks, use inline description
bd create "fix: resolve alertmanager crashloop" \
  --assignee coding-agent \
  --priority high \
  --description "## Context
Alertmanager is crashlooping in dev cluster.

## Acceptance Criteria
- [ ] Crashloop resolved
- [ ] Pod running and healthy
- [ ] Alerts functional"
```

---

## Task Types Reference

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

---

## Priority Levels

| Priority | Description | Use Case |
|----------|-------------|----------|
| **urgent** | Critical blockers | Production down, security breach |
| **high** | High priority | Infrastructure issues, important features |
| **normal** | Medium priority (default) | Standard features, improvements |
| **low** | Low priority | Nice-to-have, research |

---

**Last Updated:** 2026-01-08
