# Workflow State Machine (Justfile)

> **Version:** 1.0
> **Date:** 2026-01-29
> **Status:** Active
> **Reference:** `justfile`

## Overview

Le workflow Vixens est géré par une **Machine à États** implémentée dans le `justfile`. Ce système force une progression séquentielle rigoureuse (Phases 0 à 6) pour garantir l'intégrité du cluster et le respect du cycle GitOps.

---

## Les 7 Phases du Workflow

### Phase 0: SELECTION (Compréhension)
- **Objectif :** Identifier l'application et l'objectif.
- **Action Agent :** Lire le titre et la description de la tâche Beads.
- **Commande :** `just start <task_id>` (initialise la phase 0).
- **Blocage :** Interdiction de modifier des fichiers à cette étape.

### Phase 1: PREREQS (Analyse Technique)
- **Objectif :** Identifier les contraintes d'infrastructure.
- **Vérifications :** 
    - PVC avec `ReadWriteOnce` ? (Nécessite `strategy: Recreate`).
    - Déploiement sur control-plane ? (Nécessite `tolerations`).
- **Validation :** L'agent doit noter ces contraintes dans les notes de la tâche.

### Phase 2: DOCUMENTATION (Contexte)
- **Objectif :** Charger et comprendre l'existant.
- **Action Agent :** Lire `docs/applications/<category>/<app>.md` et utiliser Archon RAG pour trouver des patterns similaires.
- **Garde-fou :** Empêche de coder "à l'aveugle" sans connaître l'historique de l'application.

### Phase 3: IMPLEMENTATION (Codage)
- **Objectif :** Réaliser les modifications de code.
- **Règles :** 
    - **Scope Limité :** Uniquement l'application citée dans la tâche.
    - **DRY :** Réutiliser `apps/_shared/`.
- **Blocage :** Impossible de passer à la phase suivante si `git status` est vide.

### Phase 4: DEPLOYMENT (GitOps Sync) ⭐ CRITIQUE
- **Objectif :** Pousser les changements et synchroniser ArgoCD.
- **Actions :** `git add` + `git commit` + `git push origin main`.
- **Vérification Automatique :** La commande `just next` vérifie que l'application est bien `Synced` et `Healthy` dans ArgoCD avant de valider la phase.
- **Note sur l'Hibernation :** Si l'app est commentée dans Kustomize, l'agent est bloqué et doit la décommenter manuellement.

### Phase 5: VALIDATION (Test Réel)
- **Objectif :** Vérifier que l'application fonctionne une fois déployée.
- **Action :** Exécution de `python3 scripts/validation/validate.py <app> dev`.
- **Blocage :** La tâche ne peut pas être fermée si le script de validation échoue (Exit Code != 0).

### Phase 6: FINALIZATION (Documentation & Close)
- **Objectif :** Clôturer le cycle documentaire.
- **Actions :** 
    - Mettre à jour `docs/applications/<app>.md` (cases à cocher).
    - Mettre à jour `docs/STATUS.md`.
    - Committer la documentation.
- **Finalisation :** `just close <task_id>` (ferme la tâche Beads).

---

## Commandes de Navigation

| Commande | Usage |
| :--- | :--- |
| `just resume` | Affiche la phase actuelle et les instructions spécifiques. |
| `just next <id>` | Tente de passer à la phase suivante (exécute les contrôles). |
| `just reset-phase <id> <N>` | Réinitialise la tâche à la phase N (utile en cas d'erreur). |
| `just wait-argocd <app>` | Helper pour attendre que l'app soit Synced/Healthy. |

---

## Règles d'Or pour les Agents

1. **Pas de Saut de Phase :** Chaque étape doit être validée techniquement.
2. **GitOps Strict :** Jamais de `kubectl apply` direct. Tout passe par `git push` vers `main`.
3. **Validation Post-Déploiement :** On ne valide pas le code, on valide l'application *en vie* sur le cluster.
4. **Hibernation :** Toujours vérifier si une application doit être re-hibernée après test.

---
*Ce document est la référence pour l'orchestration multi-agent du cluster Vixens.*
