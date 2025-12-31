# Proposition de Formalisme pour les Tâches Archon
**Date:** 2025-12-30
**Auteur:** Claude Sonnet 4.5
**Status:** 📝 DRAFT - Proposition pour review

---

## 🔍 Analyse de l'Existant

### Problèmes Identifiés

**1. Titres Incohérents (8+ patterns différents):**
```
✅ "Deploy X in `namespace` namespace"
❌ "Fix X Error/Issues"
❌ "Implement X Strategy"
❌ "Pin Application Versions"
❌ "Standardize X"
❌ "Intégrer X" (français)
❌ "optimization: X"
❌ "Set X for Y"
```

**2. Metadata Chaos:**
```yaml
task_order: 0 (presque toutes) vs 95, 90, 88, 85...
priority: "low" | "medium" | "high" | "critical" (4 valeurs, pas de critères)
assignee: "User" | "Coding Agent" (pas de convention)
feature: null (la plupart) vs "architecture-cleanup"
```

**3. Descriptions Variables:**
- Certaines: 1 ligne minimaliste
- D'autres: Structure complète (Problem, Current State, Target, Acceptance)
- Pas de template

**4. Pas de Relations:**
- Aucune dépendance entre tâches
- Pas de hiérarchie (epic > story > task)
- Pas de sprints/milestones

---

## 📋 Proposition de Formalisme

### 1. Taxonomie des Types de Tâches

**Format:** `<type>: <action> <object> [context]`

**Types Primaires (préfixe obligatoire):**

| Type | Description | Exemples |
|------|-------------|----------|
| `feat` | Nouvelle fonctionnalité/service | feat: deploy jellyfin in media namespace |
| `fix` | Correction de bug/erreur | fix: argocd server crashloop in prod |
| `refactor` | Refactoring sans changement fonctionnel | refactor: centralize http redirect middleware |
| `perf` | Optimisation performance | perf: reduce frigate memory usage |
| `docs` | Documentation seule | docs: create adr for namespace strategy |
| `chore` | Maintenance, cleanup | chore: remove deprecated readarr deployment |
| `infra` | Infrastructure/Terraform | infra: add cilium operator resource limits |
| `security` | Sécurité | security: enable pod security policies |
| `monitor` | Monitoring/observabilité | monitor: add grafana dashboards for *arr apps |
| `research` | Investigation/étude | research: evaluate velero backup strategy |

**Types Secondaires (optionnel, après `:`):**

| Type | Utilisation |
|------|-------------|
| `(breaking)` | Changement cassant la compatibilité |
| `(critical)` | Urgence production |
| `(tech-debt)` | Dette technique |

**Exemples:**
```
feat: deploy firefly-iii in finance namespace
fix(critical): restore argocd server functionality
refactor(tech-debt): centralize http redirect middleware
docs: create adr-015 shared resources organization
infra: configure backup strategy with velero
research: evaluate crowdsec for traefik integration
```

---

### 2. Structure de Titre Standardisée

**Pattern:** `<type>[(<modifier>)]: <verb> <object> [<context>]`

**Règles:**
1. **Type:** Toujours minuscule (feat, fix, refactor...)
2. **Modifier:** Optionnel, entre parenthèses
3. **Verbe:** Infinitif anglais (deploy, fix, add, remove, update...)
4. **Objet:** Nom du composant/service
5. **Context:** Optionnel (namespace, environment, scope)

**Verbes Standards par Type:**

| Type | Verbes Recommandés |
|------|-------------------|
| `feat` | deploy, add, implement, create |
| `fix` | fix, repair, resolve, restore |
| `refactor` | centralize, factorize, reorganize, extract |
| `docs` | create, update, document, write |
| `chore` | remove, cleanup, archive, standardize |
| `infra` | configure, provision, setup |

**Anti-Patterns à Éviter:**
```
❌ "Deploy Firefly III in `finance` namespace"  # Majuscule, backticks
❌ "Fix ArgoCD Server Error"                     # Pas de type
❌ "optimization: centralize HTTP redirect..."   # Type non-standard
❌ "Intégrer les divers composants"             # Français
❌ "Set Resource Limits for New Apps"           # Type manquant, trop vague
```

**Corrections:**
```
✅ feat: deploy firefly-iii in finance namespace
✅ fix: argocd server crashloop
✅ refactor: centralize http redirect middleware
✅ monitor: integrate components with home-assistant
✅ chore: set resource limits for new apps
```

---

### 3. Template de Description

**Template Obligatoire:**

```markdown
## Context
[Pourquoi cette tâche existe]

## Current State
[Situation actuelle - fichiers, config, problème]

## Target State
[État désiré après la tâche]

## Acceptance Criteria
- [ ] Critère 1
- [ ] Critère 2
- [ ] Critère 3

## Technical Notes
[Détails techniques, références, ADRs]

## Dependencies
[Bloque: task-xxx, Bloqué par: task-yyy]

## Estimated Effort
[XS: <1h, S: 1-2h, M: 2-4h, L: 4-8h, XL: 8-16h, XXL: >16h]

## Impact
[Scope du changement: files changed, apps affected, etc.]
```

**Exemple Rempli:**

```markdown
## Context
71 identical `http-redirect.yaml` files duplicated across all app overlays (8% pure duplication).
Changing HTTPS redirect scheme requires modifying 71 files manually.

## Current State
- apps/20-media/{jellyfin,radarr,...}/overlays/{dev,prod}/http-redirect.yaml (71 files)
- Each file identical except metadata.name

## Target State
- apps/_shared/middlewares/base/http-redirect.yaml (1 file)
- Apps reference via Kustomize components

## Acceptance Criteria
- [ ] Create apps/_shared/middlewares/base/ structure
- [ ] Create single http-redirect.yaml middleware
- [ ] Update all apps to reference shared middleware (71 apps)
- [ ] Delete 70 duplicate files
- [ ] Verify ArgoCD sync success (all environments)
- [ ] Update CONTRIBUTING.md with pattern

## Technical Notes
- Use Kustomize components feature (requires k8s 1.21+)
- Reference: https://kubectl.docs.kubernetes.io/guides/config_management/components/
- Related ADR: docs/adr/015-shared-resources-organization.md

## Dependencies
- Blocks: refactor-arr-deployment-patches (needs shared structure)
- Blocked by: None

## Estimated Effort
M (2-4h)

## Impact
- Files changed: +1, -70
- Apps affected: 71 (all apps with HTTP ingress)
- Risk: LOW (mechanical change)
```

---

### 4. Metadata Standardisée

**Champs Archon:**

| Champ | Format | Règles |
|-------|--------|--------|
| `title` | `<type>: <action> <object>` | 80 chars max |
| `description` | Markdown template | Template obligatoire |
| `status` | `todo|doing|review|done` | Workflow standard |
| `assignee` | `@username` | @user, @claude, @team |
| `task_order` | `0-100` | 0=lowest, 100=highest |
| `priority` | `p0|p1|p2|p3` | P0=critical, P3=nice-to-have |
| `feature` | `kebab-case` | Epic/feature grouping |

**Mapping Priority:**

| Priority | Critères | SLA |
|----------|----------|-----|
| `p0` | Production cassée, sécurité critique | Immédiat |
| `p1` | Feature bloquante, bug majeur | 1-2 jours |
| `p2` | Amélioration importante, tech debt | 1-2 semaines |
| `p3` | Nice-to-have, optimisation mineure | Backlog |

**Mapping task_order (0-100):**

| Range | Usage |
|-------|-------|
| 90-100 | P0 Critical (production down, security) |
| 70-89 | P1 High (blockers, major bugs) |
| 40-69 | P2 Medium (features, improvements) |
| 10-39 | P3 Low (nice-to-have, refactoring) |
| 0-9 | Backlog (future consideration) |

**Assignee Convention:**

| Value | Signification |
|-------|--------------|
| `@user` | Humain (toi) |
| `@claude` | AI agent (moi) |
| `@team-infra` | Équipe infra (si applicable) |
| `@team-dev` | Équipe dev (si applicable) |

**Feature Naming (kebab-case):**

```yaml
# Exemples de features valides:
feature: architecture-cleanup      # Epic: Refactoring architecture
feature: media-stack-deployment   # Epic: Déployer stack media
feature: monitoring-enhancement   # Epic: Améliorer monitoring
feature: security-hardening       # Epic: Sécurité
feature: backup-restore          # Epic: Backup/restore
```

---

### 5. Exemple de Tâches Cohérentes

**AVANT (incohérent):**
```yaml
title: "Deploy Firefly III in `finance` namespace"
description: "- Create Kustomize overlays for the application.\n- Manage secrets via Infisical."
status: todo
assignee: Coding Agent
task_order: 0
priority: medium
feature: null
```

**APRÈS (cohérent):**
```yaml
title: "feat: deploy firefly-iii in finance namespace"
description: |
  ## Context
  Need personal finance management tool for budget tracking.

  ## Current State
  No finance namespace or apps deployed.

  ## Target State
  - finance namespace created
  - firefly-iii deployed with PostgreSQL backend
  - Accessible via https://firefly.dev.truxonline.com

  ## Acceptance Criteria
  - [ ] Create finance namespace
  - [ ] Deploy PostgreSQL instance
  - [ ] Deploy firefly-iii with Kustomize overlays
  - [ ] Configure Infisical secrets
  - [ ] Setup Ingress with TLS
  - [ ] Verify WebUI accessibility

  ## Dependencies
  - Blocked by: None
  - Blocks: None

  ## Estimated Effort
  M (2-4h)

  ## Impact
  - New namespace: finance
  - New apps: firefly-iii, postgresql
  - Files: +15 (kustomize base + overlays)
status: todo
assignee: @claude
task_order: 45
priority: p2
feature: finance-management
```

---

## 🏗️ Hiérarchie des Tâches

**Proposition: 3 Niveaux**

### Niveau 1: Epic (Feature)

**Format:** Groupement logique via `feature` field

```yaml
feature: architecture-cleanup
# Regroupe toutes les tâches de refactoring
```

**Exemples d'Epics:**
- `architecture-cleanup` - Tech debt reduction
- `media-stack` - All media apps deployment
- `monitoring-observability` - Monitoring improvements
- `security-hardening` - Security enhancements
- `backup-disaster-recovery` - Backup/restore strategy

### Niveau 2: Story (Task principale)

**Format:** `<type>: <action> <object>`

Une story est une tâche qui peut être décomposée en sub-tasks.

**Exemple:**
```yaml
title: "refactor: centralize http redirect middleware"
feature: architecture-cleanup
task_order: 95
priority: p0
# Cette story a des sub-tasks implicites dans acceptance criteria
```

### Niveau 3: Sub-task (Acceptance Criteria)

**Format:** Liste dans description

```markdown
## Acceptance Criteria
- [ ] Create apps/_shared/middlewares/base/ structure     # Sub-task 1
- [ ] Create single http-redirect.yaml middleware         # Sub-task 2
- [ ] Update all apps to reference (71 apps)             # Sub-task 3
- [ ] Delete 70 duplicate files                          # Sub-task 4
```

**Note:** Archon ne supporte pas les sub-tasks natives, on utilise donc les checkbox dans description.

---

## 📊 Migration Plan

### Phase 1: Standardiser Nouvelles Tâches (Immédiat)

**Action:** Toutes nouvelles tâches suivent le nouveau format.

**Checklist création:**
- [ ] Titre suit pattern `<type>: <action> <object>`
- [ ] Description utilise template markdown
- [ ] task_order cohérent avec priority
- [ ] feature défini si epic existe
- [ ] assignee utilise @username format

### Phase 2: Nettoyer Tâches Existantes (1-2h)

**Action:** Mettre à jour les tâches actives (status=todo,doing).

**Priorités:**
1. Tâches P0/P1 (urgentes)
2. Tâches optimization récentes
3. Tâches deployment actives

**Exemples de migrations:**

```diff
- title: "Deploy Firefly III in `finance` namespace"
+ title: "feat: deploy firefly-iii in finance namespace"

- task_order: 0
- priority: medium
+ task_order: 45
+ priority: p2

- assignee: Coding Agent
+ assignee: @claude

- feature: null
+ feature: finance-management
```

### Phase 3: Archiver Anciennes Tâches (optionnel)

**Action:** Archiver tâches done anciennes (>1 mois).

```python
# Pseudo-code
for task in find_tasks(status="done", before="2025-11-30"):
    if not task.archived:
        archive_task(task.id)
```

---

## 🔧 Outils et Automation

### Script de Validation (CI)

```python
def validate_task_title(title: str) -> bool:
    """Validate task title follows convention."""
    pattern = r'^(feat|fix|refactor|perf|docs|chore|infra|security|monitor|research)(\([a-z-]+\))?: [a-z]'
    return bool(re.match(pattern, title))

def validate_task_metadata(task: dict) -> list[str]:
    """Validate task metadata."""
    errors = []

    # Priority vs task_order coherence
    if task['priority'] == 'p0' and task['task_order'] < 90:
        errors.append("P0 tasks must have task_order >= 90")

    # Assignee format
    if not task['assignee'].startswith('@'):
        errors.append("Assignee must start with @")

    # Feature kebab-case
    if task['feature'] and not re.match(r'^[a-z][a-z0-9-]*$', task['feature']):
        errors.append("Feature must be kebab-case")

    return errors
```

### Template Generator

```python
def generate_task_template(task_type: str) -> str:
    """Generate task description template."""
    return f"""## Context
[Why this {task_type} task exists]

## Current State
[Current situation]

## Target State
[Desired state after task completion]

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Technical Notes
[Technical details, references]

## Dependencies
- Blocks: None
- Blocked by: None

## Estimated Effort
[XS|S|M|L|XL|XXL]

## Impact
[Scope of change]
"""
```

---

## 📚 Documentation à Créer

**1. CONTRIBUTING.md - Section "Task Management"**
```markdown
# Task Management Guidelines

## Creating a Task

**Title Format:** `<type>: <action> <object> [context]`

**Types:**
- `feat`: New feature/service
- `fix`: Bug fix
- `refactor`: Code refactoring
- `docs`: Documentation
- `chore`: Maintenance
- `infra`: Infrastructure
- `security`: Security
- `monitor`: Monitoring

**Example:** `feat: deploy jellyfin in media namespace`

[... rest of guidelines ...]
```

**2. docs/task-templates/ - Templates par type**

```
docs/task-templates/
├── feat-deploy-app.md
├── fix-production-issue.md
├── refactor-codebase.md
├── docs-create-adr.md
└── README.md
```

**3. .github/ISSUE_TEMPLATE/ - GitHub integration (optionnel)**

Si on synchronise Archon avec GitHub Issues:

```yaml
name: Feature Request
about: Deploy a new application
title: 'feat: deploy [APP_NAME] in [NAMESPACE]'
labels: ['feature', 'deployment']
assignees: ''
```

---

## 🎯 Avantages du Nouveau Formalisme

**1. Cohérence:**
- Tous les titres suivent même pattern
- Descriptions structurées identiques
- Metadata prévisible

**2. Filtrage Efficace:**
```python
# Trouver toutes les tâches de refactoring
find_tasks(query="refactor:")

# Trouver tous les bugs critiques
find_tasks(priority="p0", query="fix:")

# Trouver toutes les tâches d'une epic
find_tasks(feature="architecture-cleanup")
```

**3. Priorisation Claire:**
- task_order cohérent avec priority
- P0 = 90-100 (urgent)
- P3 = 10-39 (backlog)

**4. Onboarding:**
- Template clair pour nouvelles tâches
- Exemples documentés
- Guidelines dans CONTRIBUTING.md

**5. Automation:**
- Validation automatique
- Génération de templates
- CI checks

---

## 🚀 Next Steps

**Décision Requise:**

1. ✅ **Approuver ce formalisme** (ou proposer ajustements)
2. ✅ **Migrer les 8 tâches optimization** créées aujourd'hui
3. ✅ **Créer CONTRIBUTING.md section** task management
4. ✅ **Créer templates** dans docs/task-templates/
5. ✅ **Nettoyer tâches existantes** (todo/doing uniquement)

**Questions Ouvertes:**

1. **Langue:** Tout en anglais ? Ou autoriser français pour descriptions ?
2. **GitHub Sync:** Synchroniser Archon tasks avec GitHub Issues ?
3. **Sprints:** Ajouter notion de sprint/milestone dans Archon ?
4. **Dépendances:** Comment gérer task dependencies (Archon supporte-t-il ?)

---

**Auteur:** Claude Sonnet 4.5
**Version:** 1.0 DRAFT
**Feedback:** Attendu avant implémentation
