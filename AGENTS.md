# AGENT.md

Guide d'orientation pour agents AI (Gemini, Claude, etc.) travaillant sur le projet Vixens.

---

## 🚨 RÈGLE MAÎTRE

**[WORKFLOW.md](WORKFLOW.md) est la référence ABSOLUE pour le processus de travail.**

Toutes les instructions de ce fichier sont complémentaires et ne doivent JAMAIS contredire WORKFLOW.md.

---

## 📋 Processus de Travail

### Voir WORKFLOW.md pour le processus complet

**Résumé rapide du cycle de travail :**

1. **Initialisation** → Récupérer les tâches attribuées à "Coding Agent" (Archon : `find_tasks`). **Attention :** Utiliser une pagination suffisante (`per_page=50`) pour ne pas manquer de tâches en cours.
2. **Sélection** :
   - Priorité 1 : Reprendre les tâches `review` (assignées à l'agent).
   - Priorité 2 : Continuer les tâches `doing` (assignées à l'agent).
   - Priorité 3 : Si aucune tâche en cours, **PROPOSER** la liste des tâches `todo` critiques à l'utilisateur et attendre son choix.
3. **Analyse** → Définir "Definition of Done", consulter `docs/applications/<app>.md`.
4. **Exécution** → Passer en "doing", travailler de manière incrémentale.
5. **Prévalidation** → Vérifier la conformité (AGENTS.md, workflow, DoD).
6. **Commit/Push** → Git commit + push vers `dev` UNIQUEMENT.
7. **Validation Dev** → Tester en dev (kubectl + playwright). Validation du DoD complète.
8. **Promotion** → Si le DoD est 100% validé en dev, promouvoir en prod via GitHub Actions.
9. **Validation Prod** → Re-valider le résultat en production.
10. **Finalisation** → Passer en `review` + assignee="User".

---

## 🛠️ Outils Essentiels

### 1. Archon MCP Server (Task & Knowledge Management)
**Système PRIMARY pour la gestion des tâches.**

- **Règles :**
  - Toujours rechercher dans RAG AVANT de coder.
  - Garder les queries courtes (2-5 mots-clés).
  - Status flow : `todo` → `doing` → `review` (Agent) → `review` (User) → `done`.

### 2. Serena MCP Server (Code Analysis)
**Analyse sémantique et édition de code.**
- **Action :** Toujours demander les `initial_instructions` à Serena pour connaître les capacités actuelles.

### 3. Playwright (Validation Web)
**Validation des interfaces web après déploiement.**
- **Fallback :** Si Playwright ne fonctionne pas, utiliser `curl` et informer l'utilisateur.

---

## 📄 Documentation Centralisée dans Archon

**IMPORTANT :** Toute la documentation critique du projet est accessible via Archon MCP Server (`find_documents`).

---

## ⚠️ Règles Impératives

1. **WORKFLOW.md est MAÎTRE** - Toujours suivre le processus défini.
2. **Archon FIRST** - Pas de TodoWrite, gestion via Archon MCP.
3. **RAG avant code** - Rechercher avant d'implémenter.
4. **Git : dev ONLY** - Jamais de push direct vers main.
5. **Proposition de Tâches** - Toujours faire valider le choix d'une nouvelle tâche `todo`.
6. **Validation DoD** - La promotion en prod exige une validation complète du DoD en dev.
