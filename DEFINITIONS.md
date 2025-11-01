# Backlog des Objectifs du Projet Vixens

Ce document est le backlog de haut niveau du projet. Il liste tous les objectifs planifiés, en cours et terminés.

## Rôle de ce Fichier

Ce fichier sert de **table des matières** pour les sprints de développement. Il fournit une vue d'ensemble mais ne contient pas les détails techniques.

Pour chaque objectif listé ci-dessous, une **spécification détaillée** incluant le contexte, les tâches à réaliser et la **"Definition of Done" (DoD)** se trouve dans un fichier dédié dans le dossier `/docs/objectives/`. La DoD est le contrat qui définit quand un objectif est considéré comme "terminé".

---

## Backlog

### OBJECTIF-01: Création du code Terraform pour le cluster `dev`
- **Statut:** `[ ] En attente`
- **Description sommaire:** Mettre en place le code Terraform nécessaire pour provisionner l'infrastructure socle (VMs Hyper-V) et y installer un cluster Talos fonctionnel pour l'environnement de développement.
- **Spécification détaillée:** [./docs/objectives/01-infra-bootstrap.md](./docs/objectives/01-infra-bootstrap.md)

### OBJECTIF-02: Amorçage de GitOps avec ArgoCD et Cilium
- **Statut:** `[x] Terminé`
- **Description sommaire:** Déployer ArgoCD et Cilium sur le cluster `dev` via Terraform pour initier la boucle GitOps. ArgoCD deviendra ensuite `self-managed`.
- **Spécification détaillée:** (Fichier à créer : `02-gitops-bootstrap.md`)