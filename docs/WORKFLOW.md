# Workflow du Projet Vixens

Ce document décrit le processus de développement, de validation et de promotion du code au sein du projet.

## 1. Méthodologie de Sprint et Focus

Le projet avance par "sprints", où chaque sprint correspond à un `OBJECTIF` unique défini dans le fichier `DEFINITIONS.md`.

**La règle d'or est le focus unique :** tout le travail de développement, y compris l'assistance par l'IA, doit se concentrer exclusivement sur les tâches de l'objectif actuellement "En cours". Aucun travail sur un objectif futur ne doit être entrepris avant que l'objectif actuel n'ait atteint sa "Definition of Done".

**Chaque `OBJECTIF` est obligatoirement accompagné d'un fichier de spécification dédié dans le dossier `/docs/objectives/` qui détaille le périmètre et les critères de succès.**

## 2. Modèle de Branches (GitFlow)

Le projet utilise un modèle GitFlow où chaque environnement persistant est représenté par une branche Git dédiée.

- **`main`**: Reflète l'état de l'environnement de **production**. Cette branche est protégée. Toute modification ne peut provenir que d'une Pull Request validée depuis la branche `staging`.
- **`staging`**: Reflète l'état de l'environnement de **staging**.
- **`test`**: Reflète l'état de l'environnement de **test**.
- **`dev`**: La branche principale d'intégration.
- **`feature/*`**, **`fix/*`**, **`docs/*`**: Branches de travail éphémères créées à partir de `dev`.

## 3. Processus de Promotion

Le code est promu d'un environnement à l'autre en suivant strictement ce chemin, via des Pull Requests (PR) :

`feature/*` -> **PR** -> `dev` -> **PR** -> `test` -> **PR** -> `staging` -> **PR** -> `main`

## 4. Conventions de Nommage

Pour maintenir un historique clair et centré sur les versions, une convention de nommage stricte est appliquée aux commits et aux Pull Requests.

### 4.1. Versioning

La version suit le format `Majeur.Medium.Mineur` :
- **Majeur**: La phase majeure du projet (ex: 1 = Infrastructure Terraform).
- **Medium**: La sous-phase ou la fonctionnalité majeure (ex: 1.1 = Bootstrap Talos).
- **Mineur**: L'itération d'implémentation ou la correction.

### 4.2. Sujet des Commits et Titre des Pull Requests

Le sujet doit impérativement respecter le format suivant :
**`v<M.m.p>: <type>(<scope>) - "Titre humoristique"`**

**Exemples :**
- `v1.1.1: feat(talos) - "Et la lumière fut !"`
- `v0.1.2: docs(workflow) - "On écrit les règles du jeu"`

### 4.3. Corps des Commits et Description des Pull Requests

Le corps est utilisé pour fournir le contexte et les détails techniques.
- Il doit contenir une description des changements techniques.
- Il doit faire référence à l'objectif concerné (`Relates to OBJECTIF-01`).

## 5. Pull Requests (PR) : Le Journal de Bord du Changement

La Pull Request est l'historique complet et auditable d'une modification. **Toute PR doit obligatoirement contenir dans sa description :**

1.  **Titre Humoristique et Version :** Suivant la convention de nommage.
2.  **Référence à l'Objectif :** `Relates to OBJECTIF-XX`.
3.  **Description du Changement :** Le "quoi" et le "pourquoi".
4.  **Procédure de Test :** Une description des tests effectués.
5.  **Résultats des Tests et Journal :** Un résumé des résultats, y compris les erreurs rencontrées et résolues.
6.  **Validation de Non-Régression :** Une confirmation que les fonctionnalités existantes ne sont pas impactées.