# Workflow du Projet Vixens

Ce document décrit le processus de développement, de validation et de promotion du code au sein du projet.

## 1. Méthodologie de Sprint et Focus

Le projet avance par "sprints", où chaque sprint correspond à un `OBJECTIF` unique défini dans le fichier `DEFINITIONS.md`.

**La règle d'or est le focus unique :** tout le travail de développement, y compris l'assistance par l'IA, doit se concentrer exclusivement sur les tâches de l'objectif actuellement "En cours". Aucun travail sur un objectif futur ne doit être entrepris avant que l'objectif actuel n'ait atteint sa "Definition of Done".

## 2. Modèle de Branches (GitFlow)

Le projet utilise un modèle GitFlow où chaque environnement persistant est représenté par une branche Git dédiée.

- **`main`**: Reflète l'état de l'environnement de **production**. Cette branche est protégée. Toute modification ne peut provenir que d'une Pull Request validée depuis la branche `staging`.
- **`staging`**: Reflète l'état de l'environnement de **staging**. Sert à la validation finale avant production.
- **`test`**: Reflète l'état de l'environnement de **test**. Sert à la validation des fonctionnalités et des configurations spécifiques à l'environnement (overlays).
- **`dev`**: La branche principale d'intégration. Toutes les nouvelles fonctionnalités y sont fusionnées avant d'être promues.
- **`feature/*`**, **`fix/*`**, **`docs/*`**: Branches de travail éphémères. Elles sont créées à partir de `dev` et doivent être supprimées après la fusion de leur Pull Request dans `dev`.

## 3. Processus de Promotion

Le code est promu d'un environnement à l'autre en suivant strictement ce chemin, via des Pull Requests (PR) :

`feature/*` -> **PR** -> `dev` -> **PR** -> `test` -> **PR** -> `staging` -> **PR** -> `main`

Chaque PR doit être validée par des contrôles automatisés (CI) avant de pouvoir être fusionnée. Le passage de `dev` à `test` peut être automatisé après la réussite des tests de la CI.

## 4. Conventions de Nommage

Pour maintenir un historique clair et centré sur les versions, une convention de nommage stricte est appliquée aux commits et aux Pull Requests.

## 5. Pull Requests (PR) : Le Journal de Bord du Changement

La Pull Request est plus qu'une simple demande de fusion. C'est l'historique complet, auditable et traçable d'une modification.

**Toute PR doit obligatoirement contenir dans sa description :**

1.  **Titre Humoristique et Version :** Suivant la convention de nommage.
2.  **Référence à l'Objectif :** `Relates to OBJECTIF-XX`.
3.  **Description du Changement :** Le "quoi" et le "pourquoi".
4.  **Procédure de Test :** Une description des tests manuels ou automatisés effectués.
5.  **Résultats des Tests et Journal :**
    - Un résumé des résultats des tests.
    - En cas d'échecs durant le développement, les erreurs et les étapes de résolution doivent être postées en commentaire de la PR. Ceci constitue notre journal d'erreurs.
6.  **Validation de Non-Régression :** Une checklist ou une affirmation confirmant que les fonctionnalités existantes n'ont pas été impactées négativement.```

### Versioning

La version suit le format `Majeur.Medium.Mineur` :
- **Majeur**: La phase majeure du projet (ex: 1 = Infrastructure Terraform, 2 = Déploiement ArgoCD).
- **Medium**: La sous-phase ou la fonctionnalité majeure au sein de la phase (ex: 1.1 = Bootstrap Talos, 1.2 = Configuration Cilium).
- **Mineur**: L'itération d'implémentation ou la correction.

### Sujet des Commits et Titre des Pull Requests

Le sujet doit impérativement respecter le format suivant :

**`v<M.m.p>: <type>(<scope>) - "Titre humoristique"`**

- **`v<M.m.p>:`** : Le numéro de version, préfixé par `v`. (ex: `v1.1.1:`)
- **`<type>(<scope>)`** : Un type et un scope issus de la norme Conventional Commits.
  - `type`: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`.
  - `scope`: Le domaine impacté (ex: `terraform`, `argocd`, `cilium`, `hyper-v`).
- **`"Titre humoristique"`** : Une description courte, personnelle et évocatrice du changement, entre guillemets.

**Exemples :**
- `v1.1.1: feat(talos) - "Et la lumière fut !"`
- `v1.2.4: fix(cilium) - "Le câble réseau était débranché"`
- `v0.1.2: docs(workflow) - "On écrit les règles du jeu"`

### Corps des Commits et Description des Pull Requests

Le corps est utilisé pour fournir le contexte et les détails techniques qui ne figurent pas dans le sujet.

- Il doit contenir une description des changements techniques.
- Il doit faire référence à l'objectif concerné du projet (`Relates to OBJECTIF-01`).

**Exemple Complet :**

v1.1.1: feat(talos) - "Et la lumière fut !"

    Implémentation de la ressource talos_machine_configuration pour les control planes.

    Le bootstrap attend désormais que l'API Kubernetes soit disponible avant de se terminer.
