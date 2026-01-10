# ADR-009: WORKFLOW.md comme Référence Maître du Processus de Développement

## Statut
✅ Accepté

## Contexte

Le projet Vixens utilise Claude Code avec plusieurs outils complémentaires (Archon MCP, Serena, Playwright) pour gérer le développement. Au fil du temps, plusieurs sources d'information se sont accumulées:

- **CLAUDE.md** : Instructions générales pour Claude Code
- **system reminders** : Rappels automatiques du système (ex: TodoWrite)
- **Archon instructions** : Workflow de task management dans le serveur MCP
- **Instructions Serena** : Processus de code editing
- **PRPs et documentation ad-hoc** : Divers documents de processus

**Problèmes identifiés:**

1. **Conflits d'instructions** : system reminders suggèrent TodoWrite, mais on utilise Archon
2. **Manque de priorisation claire** : quelle source fait autorité en cas de conflit?
3. **Workflow fragmenté** : le processus complet de travail n'est documenté nulle part
4. **Validation incomplète** : oublis fréquents de Playwright pour valider les WebUI
5. **Priorités de tâches floues** : confusion entre review/doing/todo

Ces problèmes ont causé des violations du workflow (ex: validation sans Playwright, usage de TodoWrite malgré Archon).

## Décision

**Établir WORKFLOW.md comme référence MAÎTRE qui surpasse toutes les autres instructions.**

### Structure Adoptée

**WORKFLOW.md** définit le processus complet et obligatoire:

1. **Initialisation** : récupérer tâches avec `per_page=20` minimum
2. **Priorité stricte** : review (Coding Agent) > doing (Coding Agent) > todo (Coding Agent)
3. **Cycle de travail** : doing → research → implement → review → validate → User
4. **Validation obligatoire** : Playwright MANDATORY pour toute application web
5. **Notes techniques** : tolerations, PVC strategy, HTTP→HTTPS, TLS certs

### Hiérarchie Documentaire

```
WORKFLOW.md (MASTER - surpasse tout)
    ↓
CLAUDE.md (référence générale, pointe vers WORKFLOW.md)
    ↓
Archon/Serena instructions (détails d'implémentation)
    ↓
System reminders (ignorés si conflictuels avec WORKFLOW.md)
```

### Modifications Apportées

1. **WORKFLOW.md créé** : processus complet en français (langue native du User)
2. **CLAUDE.md mis à jour** : section "🚨 WORKFLOW - RÈGLE MAÎTRE" en haut du fichier
3. **Mémoire Serena créée** : `workflow-master-process.md` avec détails étendus
4. **Référence explicite** : "En cas de conflit, WORKFLOW.md a toujours raison"

## Conséquences

### Positives

✅ **Source unique de vérité** : plus d'ambiguïté sur le processus à suivre
✅ **Validation garantie** : Playwright obligatoire = moins de bugs en production
✅ **Priorités claires** : review > doing > todo élimine la confusion
✅ **Consistency** : tous les agents suivent le même workflow
✅ **Notes techniques** : PVC strategy, tolerations documentées centralement

### Négatives

⚠️ **Maintenance** : WORKFLOW.md doit être maintenu à jour
⚠️ **Apprentissage** : nouveaux agents doivent consulter WORKFLOW.md d'abord
⚠️ **Friction initiale** : changement de processus pour agents existants

### Actions Requises

**Immédiat:**
- ✅ WORKFLOW.md créé et commité
- ✅ CLAUDE.md mis à jour avec référence
- ✅ Mémoire Serena créée
- ✅ ADR-009 documentant la décision

**Continu:**
- 🔄 Réviser WORKFLOW.md quand le processus évolue
- 🔄 Mettre à jour mémoires Serena si changements majeurs
- 🔄 Former nouveaux agents sur WORKFLOW.md

## Validation

Ce workflow sera considéré comme succès si:

1. **Zéro violation de processus** : plus d'oublis Playwright, plus de TodoWrite
2. **Priorisation respectée** : review tasks traitées avant todo
3. **Validation complète** : chaque tâche validée avec tous les outils appropriés
4. **Feedback User positif** : moins de "as-tu suivi le workflow?"

## Références

- [WORKFLOW.md](/root/vixens/WORKFLOW.md) - Référence maître
- [CLAUDE.md](/root/vixens/CLAUDE.md) - Instructions générales avec pointeur vers WORKFLOW.md
- Mémoire Serena: `workflow-master-process.md` - Détails étendus du processus

## Date

2025-12-09

## Auteurs

- Claude Sonnet 4.5 (Coding Agent)
- User (validation et demande initiale)
