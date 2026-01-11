# Workflow Vixens - State Machine GitOps

**Adhérence stricte et totale requise. Pas de raccourcis.**

Ce workflow remplace l'ancien usage d'Archon pour la gestion des tâches. Tout passe par **Beads (bd)** et **Just**.

---

## 🎯 Vue d'Ensemble

Le workflow est un **state machine à 7 phases** (0-6) orchestré par Just. Chaque phase a des objectifs précis et des interdictions claires.

### Commandes Principales

```bash
just resume              # Afficher phase actuelle et instructions
just start <task_id>     # Démarrer une nouvelle tâche (phase 0)
just next <task_id>      # Avancer à la phase suivante (avec validation)
just close <task_id>     # Fermer la tâche (phase 6 uniquement)
```

### Helpers

```bash
just wait-argocd <app>   # Attendre ArgoCD sync (Synced+Healthy)
just promote-prod        # Instructions promotion production
just create-task         # Créer tâche (template guidé)
just lint                # Valider YAML
```

### Assignations Agents

**Qui peut prendre quelles tâches:**

- **`coding-agent`** - Tâches génériques (assignation par défaut)
  - Peut être prise par: Claude, Gemini, autres agents

- **`claude`** - Tâches spécifiques Claude
  - Peut prendre: tâches `claude` ET tâches `coding-agent`

- **`gemini`** - Tâches spécifiques Gemini
  - Peut prendre: tâches `gemini` ET tâches `coding-agent`

**En pratique:**
```bash
# Créer tâche pour tous agents
bd create --title="..." --assignee=coding-agent

# Créer tâche spécifique Claude
bd create --title="..." --assignee=claude

# Créer tâche spécifique Gemini
bd create --title="..." --assignee=gemini

# just resume fonctionne pour tous (claude, gemini, coding-agent)
```

---

## 📋 Les 7 Phases du Workflow

### Phase 0: SELECTION
**Objectif:** Comprendre la tâche

✅ À faire:
- Lire le titre et la description de la tâche
- Identifier l'application ciblée (entre parenthèses dans le titre)
- Comprendre l'objectif de la tâche

❌ Interdit:
- NE PAS commencer à coder
- NE PAS toucher aux fichiers

**Commande:** `just next <task_id>` pour avancer

---

### Phase 1: PREREQS
**Objectif:** Vérifier prérequis techniques

✅ À faire:
- Vérifier si PVC RWO → noter `strategy: Recreate` requis
- Vérifier si controlplane → noter `tolerations` requis
- Identifier les dépendances techniques

❌ Interdit:
- NE PAS modifier de fichiers
- NE PAS coder

**Commande:** `just next <task_id>`

---

### Phase 2: DOCUMENTATION
**Objectif:** Charger documentation de l'application

✅ À faire:
- Lire `docs/applications/<category>/<app>.md`
- Comprendre l'architecture actuelle
- Utiliser Archon RAG pour rechercher patterns similaires

❌ Interdit:
- NE PAS modifier de code
- NE PAS créer de fichiers

**Commande:** `just next <task_id>`

---

### Phase 3: IMPLEMENTATION
**Objectif:** Coder (scope limité à l'application ciblée)

✅ À faire:
- Coder UNIQUEMENT l'application ciblée (dans le titre)
- Utiliser Serena pour édition de code
- Suivre les patterns existants (DRY)
- Respecter GitOps (ZERO kubectl apply direct)

❌ INTERDICTIONS CRITIQUES:
- ❌ Toucher à d'autres applications
- ❌ `kubectl apply/edit/delete` (GitOps only)
- ❌ Créer des duplications (DRY)
- ❌ Fermer la tâche
- ❌ Bypasser la validation
- ❌ Commit/push (c'est phase suivante)

📜 Règles:
- GitOps: Tout passe par Git → ArgoCD sync
- DRY: Réutiliser `apps/_shared/` si applicable
- Scope: UNIQUEMENT l'app dans le titre de la tâche
- NO COMMIT: Attendre phase DEPLOYMENT

**Commande:** `just next <task_id>` (vérifie qu'il y a des changements)

---

### Phase 4: DEPLOYMENT
**Objectif:** Commit + Push via PR + Wait ArgoCD sync ⭐ CRITIQUE

✅ À faire:
1. Créer feature branch: `git checkout -b fix/<app-name>` ou `feat/<app-name>`
2. Commit changements: `git add . && git commit -m "fix(app): description"`
3. Push feature branch: `git push origin fix/<app-name>`
4. Créer PR: `gh pr create --base main --head fix/<app-name>`
5. Merger PR: Auto-merge après checks OU merge manuel
6. ArgoCD auto-sync dev depuis main (après merge)
7. Attendre sync: `just wait-argocd <app_name>`
8. Vérifier: Sync=Synced, Health=Healthy

❌ INTERDICTIONS:
- ❌ Push directement vers `main` (branch protégée, repository rules)
- ❌ Créer des tags manuellement
- ❌ Avancer avant ArgoCD Synced+Healthy
- ❌ `kubectl apply/edit` direct

📜 Règles:
- **Feature branch OBLIGATOIRE** (main protégée par repository rules)
- Naming: `fix/<app>`, `feat/<app>`, `chore/<app>`, `docs/<app>`
- PR required checks: YAML lint, ArgoCD structure, Security
- GitOps: PR merge → ArgoCD auto-sync dev
- Attente: ArgoCD peut prendre 1-3 minutes après merge
- Vérification: Synced + Healthy obligatoires

**Commande:** `just next <task_id>` (vérifie ArgoCD status)

---

### Phase 5: VALIDATION
**Objectif:** Valider APRÈS déploiement

✅ À faire:
1. Validation APRÈS déploiement: `python3 scripts/validate.py <app_name> dev`
2. Vérifier que la validation passe (exit code 0)
3. Corriger les erreurs si échec → retour phase 3

❌ INTERDICTIONS:
- ❌ Valider AVANT ArgoCD sync
- ❌ Avancer sans validation réussie
- ❌ Fermer la tâche manuellement

📜 Règles:
- Validation: Teste l'app DÉPLOYÉE sur cluster dev
- Échec: Retour phase 3 (`just reset-phase <task_id> 3`)
- Succès: Marqué dans notes Beads

**Commande:** `just next <task_id>` (bloque si validation échoue)

---

### Phase 6: FINALIZATION
**Objectif:** Documentation + Close

✅ À faire:
1. Mettre à jour `docs/applications/<category>/<app>.md`
   - Marquer `[x]` pour Déployé, Configuré, Testé
   - Mettre à jour version
2. Mettre à jour `docs/STATUS.md` si nécessaire
   - Symboles: ✅ (OK) ⚠️ (Degraded) ❌ (Broken) 🚧 (WIP) 💤 (Paused)
3. Committer les changements de documentation
4. Vérifier `git push` réussi

🎯 PROMOTION PRODUCTION:
1. Validé sur dev ✅
2. Promouvoir vers prod:
   ```bash
   gh workflow run promote-prod.yaml -f version=vX.Y.Z
   ```
3. Attendre déploiement prod (ArgoCD auto-sync depuis tag `prod-stable`)
4. **Valider prod:**
   ```bash
   # Switch to prod kubeconfig
   export KUBECONFIG=/path/to/prod/kubeconfig
   
   # Validate deployment
   python3 scripts/validate.py <app_name> prod
   just wait-argocd <app_name>  # If needed
   ```
5. Si validation prod OK → **Fermer tâche**
6. Si validation prod échoue → **Rollback et corriger**

**Important:**
- Ne JAMAIS créer tag `prod-stable` manuellement
- Promotion via GitHub Actions workflow uniquement
- **Toujours valider prod avant de fermer**

**Commande:** `just close <task_id>` (vérifie validation dev + déploiement OK)

---

## 🚨 Règles Critiques (Non Négociables)

### GitOps ONLY
- ❌ **ZERO** `kubectl apply/edit/delete` direct
- ✅ Tout passe par Git → ArgoCD auto-sync
- ✅ Dev: Feature branch → PR → merge main → ArgoCD sync
- ✅ Prod: Promotion via workflow → tag `prod-stable` → ArgoCD sync
- ⚠️ **Main branch PROTÉGÉE** (repository rules) → PR obligatoire

### DRY (Don't Repeat Yourself)
- ✅ Réutiliser `apps/_shared/` pour resources communes
- ❌ NE PAS dupliquer code entre applications
- ✅ Suivre patterns existants

### Scope Limité
- ✅ Coder UNIQUEMENT l'app dans le titre de la tâche
- ❌ NE PAS toucher à d'autres applications
- ❌ NE PAS faire du "tant qu'à faire"

### Deployment + Validation OBLIGATOIRES
- ✅ Phase 4 (Deployment) → commit + push + ArgoCD sync
- ✅ Phase 5 (Validation) → `scripts/validate.py` DOIT passer
- ❌ NE PAS fermer sans ces deux étapes

### Production via Workflow UNIQUEMENT
- ❌ JAMAIS créer tag `prod-stable` manuellement
- ❌ JAMAIS push force sur `main`
- ✅ TOUJOURS utiliser workflow: `gh workflow run promote-prod.yaml`
- ✅ Tag `prod-stable` déplacé automatiquement par GitHub Actions

---

## 📊 Workflow Complet (Exemple)

```bash
# 1. Lister tâches disponibles
just resume

# 2. Démarrer une tâche
just start vixens-abc123

# 3. Phase 0: SELECTION
just resume   # Lire instructions
just next vixens-abc123

# 4. Phase 1: PREREQS
just resume   # Vérifier prérequis
just next vixens-abc123

# 5. Phase 2: DOCUMENTATION
just resume   # Charger doc
# Lire docs/applications/<category>/<app>.md
# Utiliser Archon RAG
just next vixens-abc123

# 6. Phase 3: IMPLEMENTATION
just resume   # Instructions de codage
# Coder avec Serena
# Scope limité à l'app ciblée
just next vixens-abc123  # Vérifie qu'il y a des changements

# 7. Phase 4: DEPLOYMENT
just resume
# Créer feature branch
git checkout -b feat/app-feature
git add .
git commit -m "feat(app): description"
git push origin feat/app-feature
# Créer et merger PR
gh pr create --base main --head feat/app-feature --title "feat(app): description"
gh pr merge --auto --squash  # Auto-merge après checks
# ArgoCD auto-sync dev depuis main (après merge)
just wait-argocd <app_name>  # Attendre sync
just next vixens-abc123  # Vérifie ArgoCD status

# 8. Phase 5: VALIDATION
just resume
just next vixens-abc123  # Lance scripts/validate.py automatiquement

# 9. Phase 6: FINALIZATION
just resume
# Mettre à jour docs/applications/<category>/<app>.md
# Mettre à jour docs/STATUS.md
git add docs/ && git commit -m "docs(app): update deployment status"
git push origin main
just close vixens-abc123

# 10. Promotion production (après validation complète en dev)
gh workflow run promote-prod.yaml -f version=v1.2.3
# Déplace prod-stable tag vers HEAD de main → ArgoCD sync prod
```

---

## 🛠️ Outils & Commandes

### Beads (bd) - Task Management
```bash
bd list --status open              # Lister tâches ouvertes
bd show <task_id>                  # Voir détails
bd update <task_id> --notes "..."  # Ajouter notes
bd close <task_id>                 # Fermer (via just close)
bd sync                            # Sync avec remote
```

### Just - Workflow Orchestrator
```bash
just resume              # Point d'entrée (affiche phase actuelle)
just start <task_id>     # Démarrer tâche
just next <task_id>      # Avancer phase (avec validation)
just close <task_id>     # Fermer tâche (phase 6 requis)
just wait-argocd <app>   # Attendre ArgoCD sync
just reset-phase <id> N  # Réinitialiser à phase N (debug)
just lint                # Valider YAML
just create-task         # Créer tâche (template guidé)
```

### Serena - Code Editing (Phase 3)
- `read_file` - Lecture de fichiers
- `list_dir` - Listing de répertoires
- `find_symbol` - Recherche de symbols
- `replace_symbol_body` - Édition symbols
- `search_for_pattern` - Recherche de patterns
- `create_text_file` - Création de fichiers

**All agents:** Utiliser Serena pour TOUTES les opérations fichiers/code. C'est son rôle principal.

**CRITICAL:** Ne PAS utiliser `execute_shell_command` de Serena pour lancer des commandes CLI (`just`, `bd`, `git`) → Utiliser Bash tool

### Archon RAG - Documentation (Phase 2)
- `rag_search_knowledge_base` - Recherche doc (Talos, K8s, ArgoCD)
- `rag_search_code_examples` - Patterns de code

**Note:** NE PAS utiliser Archon Task Management (`manage_task`) → Utiliser `bd` CLI

### Playwright - Validation WebUI (Phase 5, optionnel)
- `browser_navigate` - Navigation WebUI
- `browser_snapshot` - Capture page
- Fallback: `curl -I` pour checks HTTP simples

---

## 📝 Notes Techniques Importantes

### Controlplane Scheduling
Applications sur control plane nécessitent toleration:
```yaml
tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
```

### Storage Strategy
**PVC avec ReadWriteOnce (RWO)** → Deployment doit utiliser `strategy: Recreate`:
```yaml
spec:
  strategy:
    type: Recreate
```

### Network Configuration
- **HTTP → HTTPS redirect:** Systématique pour services publics
- **Certificats:**
  - Dev: `letsencrypt-staging`
  - Prod: `letsencrypt-prod`
- **URLs Ingress:**
  - Dev: `<app>.dev.truxonline.com`
  - Prod: `<app>.truxonline.com`

### Design Principles
- **DRY:** Réutiliser `apps/_shared/` pour resources communes
- **GitOps:** Git est source of truth, ZERO `kubectl apply` direct
- **State of the Art:** Suivre best practices Kubernetes/GitOps
- **Reproducibility:** Tout dans git, infrastructure as code

---

## 🚀 Workflow GitOps (Trunk-Based)

### Branches
- **`main`** - Unique branche (trunk-based development)

### Environnements
- **Dev**: ArgoCD watch `main` branch (HEAD)
- **Prod**: ArgoCD watch `prod-stable` tag

### Flux Réel
1. **Développement sur feature branch** (main est protégée)
   ```bash
   git checkout -b fix/app-name  # ou feat/, chore/, docs/
   # Coder et tester localement
   git add . && git commit -m "fix(app): description"
   git push origin fix/app-name
   ```

2. **Pull Request vers main**
   ```bash
   gh pr create --base main --head fix/app-name
   # Required checks: YAML lint, ArgoCD structure, Security
   ```

3. **Merge PR → ArgoCD auto-sync dev**
   ```bash
   gh pr merge --auto --squash  # Après checks passés
   # ArgoCD détecte le nouveau commit sur main
   # Auto-sync vers cluster dev (1-3 minutes)
   ```

4. **Validation dev** (Phase 5)
   ```bash
   python3 scripts/validate.py <app> dev
   just wait-argocd <app_name>
   ```

5. **Promotion prod**
   ```bash
   gh workflow run promote-prod.yaml -f version=v1.2.3
   # Workflow déplace tag prod-stable vers HEAD de main
   ```

6. **ArgoCD auto-sync prod**
   ```bash
   # ArgoCD prod watch le tag prod-stable
   # Auto-sync vers cluster prod (1-3 minutes)
   ```

7. **Validation prod + Fermeture**
   ```bash
   export KUBECONFIG=/path/to/prod/kubeconfig
   python3 scripts/validate.py <app> prod
   bd close <task_id>  # Si validation OK
   ```

### Règles Critiques
- ❌ **JAMAIS push direct sur `main`** (branch protégée par repository rules)
- ✅ **Feature branch OBLIGATOIRE** pour tous les changements
- ✅ **PR required** avec checks passés
- ❌ JAMAIS push force sur `main`
- ❌ JAMAIS créer tag `prod-stable` manuellement
- ✅ Promotion production via GitHub Actions workflow uniquement
- ✅ **Valider prod avant de fermer** la tâche

Voir [ADR-017](docs/adr/017-pure-trunk-based-single-branch.md) pour détails (supersède ADR-008/009).

---

## 🔍 Debugging & Troubleshooting

### Bloquer en Phase N
```bash
# Réinitialiser à une phase antérieure
just reset-phase <task_id> N

# Exemples
just reset-phase vixens-abc 3  # Retour IMPLEMENTATION
just reset-phase vixens-abc 5  # Retour VALIDATION
```

### Validation Échoue
```bash
# Vérifier erreur
python3 scripts/validate.py <app_name> dev

# Corriger et relancer
just reset-phase <task_id> 3  # Retour phase 3
just resume                    # Réimplémenter
```

### ArgoCD Pas Synced
```bash
# Attendre sync (5 min max)
just wait-argocd <app_name>

# Vérifier manuellement
kubectl -n argocd get application <app_name>
kubectl -n argocd describe application <app_name>
```

---

## 📖 Documentation Associée

- **[AGENTS.md](AGENTS.md)** - Guide multi-agents (Claude, Gemini, etc.)
- **[CLAUDE.md](CLAUDE.md)** - Configuration Claude Code
- **[GEMINI.md](GEMINI.md)** - Configuration Gemini
- **[docs/guides/task-management.md](docs/guides/task-management.md)** - Guide Beads détaillé
- **[docs/guides/gitops-workflow.md](docs/guides/gitops-workflow.md)** - GitOps patterns
- **[docs/guides/adding-new-application.md](docs/guides/adding-new-application.md)** - Déployer nouvelle app

---

**Last Updated:** 2026-01-11

**Version:** 3.0 (Non-Interactive GitOps - Feature Branch Required)
