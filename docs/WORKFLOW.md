# Workflow du Projet Vixens

Ce document est le guide unique pour le développement, la validation et la promotion du code.

## 1. Principes de Travail

- **Focus Sprint**: Un seul `OBJECTIF` est traité à la fois. Aucun travail sur un objectif futur n'est autorisé.
- **Spécifications**: Chaque `OBJECTIF` est défini dans un fichier dédié dans `/docs/objectives/`.
- **Validation Continue**: Chaque tâche doit inclure une section `test:`. Modifier une tâche impose de re-valider **toutes** les tâches "done" de l'objectif pour garantir la non-régression.

## 2. GitFlow & Promotion

Le code est promu via des Pull Requests (PR) en suivant un chemin strict.

### Branches
- **Environnements**: `main` (prod), `staging`, `test`, `dev`.
- **Travail**: `feature/*`, `fix/*`, `docs/*` (basées sur `dev`).

### Chemin de Promotion
`feature/*` -> **PR** -> `dev` -> **PR** -> `test` -> **PR** -> `staging` -> **PR** -> `main`

### Qualité Pré-PR
Toute PR doit passer ces validations pour être acceptée :
1.  **Linting YAML**:
    ```bash
    find apps argocd -name "*.yaml" -o -name "*.yml" | xargs yamllint -c .yamllint
    ```
2.  **Tests**: Unitaires et intégration doivent réussir.
3.  **Linting & Type-Checking**: Spécifique au langage, sans aucune erreur.

## 3. Commits & Pull Requests

La clarté de l'historique est primordiale.

### Format du Titre (Commit & PR)
Le titre doit respecter ce format : **`v<M.m.p>: <type>(<scope>) - "Titre humoristique"`**

- **Version `M.m.p`**: `Majeur.Medium.Mineur`.
- **Exemples**:
  - `v1.1.1: feat(talos) - "Et la lumière fut !"`
  - `v0.1.2: docs(workflow) - "On écrit les règles du jeu"`

### Contenu Obligatoire de la PR
La description de la PR sert de journal de bord et doit contenir :
1.  **Référence Objectif**: `Relates to OBJECTIF-XX`.
2.  **Description**: Le "quoi" et le "pourquoi" du changement.
3.  **Procédure de Test**: Comment valider la modification.
4.  **Résultats & Journal**: Résumé des tests, y compris les erreurs résolues.
5.  **Validation de Non-Régression**: Confirmation que rien n'est cassé.