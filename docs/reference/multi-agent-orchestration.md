# Multi-Agent Orchestration

> **Version:** 1.0
> **Date:** 2026-01-11
> **Status:** Active

## Overview

Système d'orchestration multi-agent pour Beads + Just, permettant à plusieurs agents (Claude, Gemini, coding-agent) de travailler collaborativement sur des tâches avec attribution intelligente.

---

## Architecture

### 1. Détection Intelligente d'Agent

L'agent actuel est détecté automatiquement selon cette priorité:

1. **Variable d'environnement** `AGENT_NAME` (priorité haute)
2. **Détection contextuelle** (présence de `.claude/` = agent Claude)
3. **Fallback par défaut** (`coding-agent`)

```python
def get_current_agent():
    """Détecter l'agent actuel de manière intelligente"""
    # 1. Env var explicite (priorité)
    agent = os.getenv("AGENT_NAME")
    if agent:
        return agent

    # 2. Détection via Claude Code context
    if os.path.exists("/.claude") or os.path.exists(".claude"):
        return "claude"

    # 3. Default fallback
    return "coding-agent"
```

### 2. Filtrage des Tâches par Agent

Un agent peut voir et prendre:

- ✅ **Ses tâches spécifiques** (`assignee = agent_name`)
- ✅ **Tâches génériques** (`assignee = 'coding-agent'`)
- ✅ **Tâches non assignées** (`assignee = null` ou vide)

```python
def filter_tasks_for_agent(all_tasks, current_agent):
    """Filtrage intelligent des tâches par agent"""
    return [t for t in all_tasks
            if t.get('assignee') in [current_agent, 'coding-agent', None, '']]
```

### 3. Préservation de l'Assignee

Lors du démarrage d'une tâche avec `just start`, l'assignee existant est **préservé**.

**Comportement:**
- Si `assignee` existe → **préservé**
- Si `assignee` est vide/null → **attribué à l'agent actuel**

---

## Agents Disponibles

### Claude (`claude`)

**Capacités:**
- Code analysis & review
- File editing (Serena integration)
- Architecture design
- Documentation writing

**Types de tâches préférés:**
- `feature` - Nouvelles fonctionnalités
- `refactor` - Refactoring/architecture
- `docs` - Documentation

**Détection:**
- Présence de `.claude/` directory
- `AGENT_NAME=claude`

### Gemini (`gemini`)

**Capacités:**
- Workflow automation
- Batch processing
- Task execution at scale

**Types de tâches préférés:**
- `task` - Tâches opérationnelles
- `chore` - Maintenance
- `fix` - Bug fixes

**Détection:**
- `AGENT_NAME=gemini`

### Coding Agent (`coding-agent`)

**Capacités:**
- General purpose
- Fallback for all task types

**Types de tâches préférés:**
- Tous types (agent générique)

**Détection:**
- Default si aucun autre agent détecté

---

## Commandes

### `just agents`

Lister les agents disponibles et leurs capacités.

```bash
$ just agents

🤖 Agents Disponibles:

👉 claude          - Claude Code
   Capacités: Code analysis, File editing, Architecture design, Documentation
   Types préférés: feature, refactor, docs

   gemini          - Gemini Agent
   Capacités: Automation, Workflow execution, Batch processing
   Types préférés: task, chore, fix

   coding-agent    - Generic Coding Agent
   Capacités: General purpose
   Types préférés: all

Agent actuel détecté: claude

💡 Pour changer d'agent:
   export AGENT_NAME=claude
   export AGENT_NAME=gemini
```

### `just workload`

Afficher la charge de travail par agent.

```bash
$ just workload

📊 Charge de Travail par Agent:

🔴 user             8 in_progress, 4 open (total: 12)
🟡 unassigned       1 in_progress, 3 open (total: 4)
🟢 coding-agent     0 in_progress, 43 open (total: 43)

💡 Utilisation:
   just assign <task_id> <agent>  # Réassigner une tâche
   just claim <task_id>            # Prendre une tâche
```

**Indicateurs:**
- 🟢 Disponible (0 tâches in_progress)
- 🟡 Occupé (1 tâche in_progress)
- 🔴 Surchargé (2+ tâches in_progress)

### `just assign <task_id> <agent>`

Réassigner une tâche à un agent spécifique.

```bash
# Assigner à Gemini
$ just assign vixens-abc gemini
✅ Tâche vixens-abc assignée à: gemini

# Assigner à Claude
$ just assign vixens-def claude
✅ Tâche vixens-def assignée à: claude

# Rendre générique (tous les agents peuvent prendre)
$ just assign vixens-ghi coding-agent
✅ Tâche vixens-ghi assignée à: coding-agent
```

**Agents valides:** `claude`, `gemini`, `coding-agent`

### `just claim <task_id>`

Prendre une tâche pour l'agent actuel.

```bash
# Agent actuel détecté: claude
$ just claim vixens-jkl
✅ Tâche vixens-jkl réclamée par: claude
```

**Utilisation:**
- Permet à un agent de prendre une tâche rapidement
- Utilise la détection automatique d'agent
- Équivalent à `just assign <task_id> $(current_agent)`

### `just resume`

Reprendre la tâche en cours avec filtrage intelligent.

**Comportement modifié:**
- Filtre les tâches `in_progress` pour l'agent actuel
- Un agent voit:
  - Ses tâches spécifiques
  - Les tâches `coding-agent` (génériques)
  - Les tâches non assignées

```bash
$ just resume

🔥 TÂCHE EN COURS: vixens-8w8b
📌 Titre: feat(beads): add agent assignment support
📍 Phase actuelle: 0
```

### `just start <task_id>`

Démarrer une tâche avec préservation de l'assignee.

**Comportement:**
- Si tâche déjà assignée → **préserve l'assignee**
- Si tâche non assignée → **attribue à l'agent actuel**

```bash
# Tâche avec assignee existant
$ bd show vixens-abc --json | jq '.[0].assignee'
"gemini"

$ just start vixens-abc
📝 Assignee préservé: gemini
✅ Tâche démarrée en Phase 0: SELECTION

# Tâche non assignée
$ bd show vixens-def --json | jq '.[0].assignee'
null

$ just start vixens-def
📝 Attribution à: claude
✅ Tâche démarrée en Phase 0: SELECTION
```

---

## Workflows

### Workflow 1: Agent Prend une Tâche Générique

```bash
# 1. Voir les tâches disponibles
$ bd list --status open | grep "coding-agent\|null"

# 2. Prendre une tâche pour l'agent actuel
$ just claim vixens-abc

# 3. Démarrer le travail
$ just start vixens-abc
```

### Workflow 2: Réassigner une Tâche à un Agent Spécialisé

```bash
# 1. Identifier une tâche qui nécessite expertise
$ bd show vixens-xyz
# Type: refactor, complexité: haute

# 2. Assigner à Claude (spécialisé en architecture)
$ just assign vixens-xyz claude

# 3. Claude démarre la tâche
$ export AGENT_NAME=claude
$ just start vixens-xyz
```

### Workflow 3: Équilibrer la Charge de Travail

```bash
# 1. Voir la charge actuelle
$ just workload
🔴 claude          3 in_progress, 8 open (total: 11)
🟡 gemini          1 in_progress, 2 open (total: 3)
🟢 coding-agent    0 in_progress, 25 open (total: 25)

# 2. Réassigner des tâches de Claude vers Gemini
$ bd list --status open --assignee claude | head -5
$ just assign vixens-task1 gemini
$ just assign vixens-task2 gemini

# 3. Vérifier le nouveau workload
$ just workload
🟡 claude          3 in_progress, 6 open (total: 9)
🟡 gemini          1 in_progress, 4 open (total: 5)
🟢 coding-agent    0 in_progress, 25 open (total: 25)
```

---

## Configuration

### Changer d'Agent Manuellement

```bash
# Définir l'agent via variable d'environnement
export AGENT_NAME=gemini

# Vérifier la détection
$ just agents
Agent actuel détecté: gemini

# Travailler avec cet agent
$ just resume
```

### Revenir à l'Agent Par Défaut

```bash
# Supprimer la variable
unset AGENT_NAME

# Ou utiliser le default
export AGENT_NAME=coding-agent
```

---

## Bonnes Pratiques

### 1. Attribution Initiale

Lors de la création de tâches, attribuer selon la spécialisation:

```bash
# Tâche d'architecture → Claude
$ bd create --title="refactor(app): redesign module X" \
  --type=refactor \
  --assignee=claude

# Tâche opérationnelle → Gemini
$ bd create --title="chore(infra): update all goldilocks VPA" \
  --type=chore \
  --assignee=gemini

# Tâche générique → coding-agent (ou laisser vide)
$ bd create --title="fix(app): resolve minor bug" \
  --type=fix \
  --assignee=coding-agent
```

### 2. Révision de Workload

Vérifier régulièrement la charge:

```bash
# Quotidien
$ just workload

# Si déséquilibre, réassigner
```

### 3. Claim vs Assign

- **`claim`** - Pour soi-même (rapide)
- **`assign`** - Pour un autre agent (orchestration)

```bash
# Je prends cette tâche
$ just claim vixens-abc

# J'assigne à un autre agent
$ just assign vixens-def gemini
```

---

## Limites Actuelles

### 1. Pas de File d'Attente Intelligente

Les tâches ne sont pas automatiquement routées selon les capacités.

**Workaround:** Utiliser `just assign` manuellement.

**Future:** Auto-routing basé sur `agents.yaml` config.

### 2. Pas de Capacités Configurables

Les capacités sont hardcodées dans le code.

**Workaround:** Modifier le code Python dans `just agents`.

**Future:** Configuration via `.beads/agents.yaml`.

### 3. Détection Agent Limitée

Seuls Claude (via `.claude/`) et env var sont détectés.

**Workaround:** Utiliser `export AGENT_NAME=<agent>`.

**Future:** Détection automatique via API calls, process inspection, etc.

---

## Évolutions Futures

### Phase 3: Capacités Configurables

```yaml
# .beads/agents.yaml
agents:
  claude:
    capabilities:
      - code_analysis
      - architecture_design
    preferred_types: [feature, refactor, docs]
    max_concurrent: 3

  gemini:
    capabilities:
      - automation
      - batch_processing
    preferred_types: [task, chore, fix]
    max_concurrent: 5
```

### Phase 4: Routing Intelligent

Auto-suggestion d'agent selon type de tâche:

```bash
$ bd create --title="refactor(app): redesign X" --type=refactor
💡 Suggestion: Assigner à 'claude' (spécialisé en refactor)
   Accepter? (y/N): y
✅ Tâche créée et assignée à: claude
```

### Phase 5: Métriques et Analytics

```bash
$ just metrics

📊 Métriques (7 derniers jours):
  claude:  15 tâches complétées, temps moyen: 2.3h, taux succès: 93%
  gemini:  28 tâches complétées, temps moyen: 0.8h, taux succès: 96%
```

---

## Troubleshooting

### Problème: Agent non détecté correctement

```bash
$ just agents
Agent actuel détecté: coding-agent  # Mais je suis Claude!

# Solution 1: Env var explicite
$ export AGENT_NAME=claude

# Solution 2: Vérifier présence .claude/
$ ls -la .claude/
```

### Problème: Tâche non visible dans resume

```bash
$ bd show vixens-abc --json | jq '.[0].assignee'
"gemini"

$ just resume  # (agent actuel: claude)
📋 AUCUNE TÂCHE EN COURS.

# Cause: La tâche est assignée à gemini, Claude ne la voit pas
# Solution: Réassigner ou claim
$ just assign vixens-abc claude
$ just resume
```

### Problème: Workload déséquilibré

```bash
$ just workload
🔴 claude          5 in_progress  # Trop!
🟢 gemini          0 in_progress

# Solution: Réassigner certaines tâches
$ bd list --status open --assignee claude
$ just assign vixens-task1 gemini
$ just assign vixens-task2 gemini
```

---

## Références

- [WORKFLOW.md](../../WORKFLOW.md) - Workflow master avec phases
- [Task Management Guide](../guides/task-management.md) - Guide complet Beads
- [ADR-017](../adr/017-pure-trunk-based-single-branch.md) - Trunk-based workflow
- [CLAUDE.md](../../CLAUDE.md) - Instructions pour Claude Code

---

**Version History:**
- 1.0 (2026-01-11) - Architecture initiale avec détection intelligente, helpers d'orchestration

**Maintainers:** Claude Code, Gemini Agent

**Status:** ✅ Active - Used in production
