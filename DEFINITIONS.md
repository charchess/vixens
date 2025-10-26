# Définitions et Cadre du Projet "Vixens"

Ce document sert de source de vérité pour les objectifs, les règles et le cadre de travail du projet d'infrastructure GitOps "Vixens".

## 1. Objectif Principal (La "Big Picture")

L'objectif est de mettre en place une infrastructure entièrement déclarative et reproductible, gérée via les principes GitOps. Cette infrastructure hébergera plusieurs clusters Kubernetes (Talos Linux) pour les environnements de développement, test, pré-production et production.

L'automatisation doit permettre de passer d'un dépôt Git vide à un ensemble de clusters fonctionnels et prêts à accueillir des applications.

## 2. Description des Environnements

| Nom       | Branche Git | Objectif                                       | Cycle de vie                               |
|-----------|-------------|------------------------------------------------|--------------------------------------------|
| `dev`     | `dev`       | Développement et tests initiaux des features   | Mises à jour continues                     |
| `test`    | `test`      | Intégration et validation des features (QA)    | Reçoit les features validées de `dev`      |
| `staging` | `staging`   | Pré-production. Miroir de la `prod`            | Reçoit les versions stables de `test`      |
| `prod`    | `main`      | Production. Environnement des utilisateurs finaux | Reçoit les versions validées de `staging` |

## 3. Definition of Done (DoD) - Qu'est-ce que "fini" veut dire ?

Ce premier projet sera considéré comme terminé lorsque les points suivants seront atteints et validés :

-   [ ] **Infrastructure as Code** : Le code Terraform dans `terraform/environments/dev` permet de provisionner un cluster Talos fonctionnel sur des VMs.
-   [ ] **Bootstrap Automatisé** : `terraform apply` installe et configure ArgoCD sur le cluster nouvellement créé.
-   [ ] **Principe de l'App-of-Apps** : ArgoCD est configuré avec une application racine (`app-of-apps`) qui gère le déploiement de toutes les autres applications du cluster.
-   [ ] **Gestion par ArgoCD** :
    -   [ ] ArgoCD est configuré en mode `self-managed` (il se gère lui-même).
    -   [ ] Le CNI Cilium est déployé et géré par ArgoCD.
    -   [ ] L'Ingress Controller Traefik est déployé et géré par ArgoCD.
-   [ ] **Workflow GitOps Fonctionnel** :
    -   [ ] Un commit sur la branche `dev` déclenche une mise à jour sur le cluster `dev`.
    -   [ ] Une application d'exemple (`my-app-example`) est déployée avec des configurations différentes (overlays) sur au moins 2 environnements (ex: `dev` et `test`).
-   [ ] **Gestion des Secrets** : Le secret management est en place avec SOPS. Un secret chiffré dans Git est correctement déchiffré et utilisé par une application dans le cluster.
-   [ ] **Documentation Initiale** :
    -   [ ] Le `README.md` explique comment lancer le projet.
    -   [ ] Le `docs/setup-guide.md` explique comment configurer un poste de travail pour contribuer.

## 4. Workflow de Travail

1.  **Développement** : Tout nouveau développement se fait sur une branche de feature (ex: `feature/setup-traefik`) créée depuis `dev`.
2.  **Pull Request (PR)** : Une fois le développement terminé, une PR est ouverte vers la branche `dev`.
3.  **Promotion** : Pour promouvoir une version d'un environnement à l'autre, une PR est ouverte :
    -   `dev` -> `test`
    -   `test` -> `staging`
    -   `staging` -> `main` (pour la production)
4.  **Pas de `commit` direct** : Aucun commit ne doit être fait directement sur les branches `dev`, `test`, `staging` et `main`. Tout passe par des PR.

## 5. Principes et Décisions Clés

-   **Git est l'unique source de vérité.** Aucune modification manuelle n'est autorisée sur les clusters (`kubectl apply -f ...` est interdit).
-   **Séparation des responsabilités** : `Terraform` gère l'infrastructure physique/virtuelle (les "murs"), `ArgoCD` gère le logiciel qui tourne dessus (les "meubles").
-   **Secrets** : Tous les secrets stockés dans Git DOIVENT être chiffrés avec SOPS.