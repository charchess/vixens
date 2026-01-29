# Vixens Infrastructure Brownfield Enhancement PRD

**Version:** 1.0  
**Status:** Draft / Ready for Execution  
**Author:** John (Product Manager)  

---

## 1. Intro Project Analysis and Context

### 1.1. Clarification du Scope & Alerte "Fuite de Contexte"
*   **Repo Actuel :** `vixens` (Infrastructure GitOps, "State Repo").
*   **Contenu :** Définitions ArgoCD, Overlays Kustomize pour diverses applications (`media`, `network`, `infra`).
*   **Problème Résolu :** Suppression de la documentation `docs/architecture.md` (hAIrem) qui polluait le contexte du projet.
*   **Point d'entrée :** Plateforme Kubernetes (Talos/Terraform géré par `terravixens`) fonctionnelle, ArgoCD pointe sur ce repo.

### 1.2. Analyse de l'existant
*   **Source d'analyse :** Fresh Analysis (IDE + Cluster Prod).
*   **État actuel :** Cluster 5 nœuds (v1.34.0), ~50 applications déployées. Majorité Healthy, mais quelques dérives (`OutOfSync`/`Unknown`).
*   **Backlog :** Campagne massive de "Goldification" (Elite status) planifiée dans Beads (~80 tâches).

---

## 2. Enhancement Scope Definition

**Type d'amélioration :**
*   [x] Bug Fix and Stability Improvements (Fixing `Unknown`/`OutOfSync` states).
*   [x] Standardisation & Refactoring (Campagne "Goldification").

**Description :**
Stabiliser le cluster Vixens en résolvant les états incohérents et en standardisant l'ensemble des applications vers le niveau de conformité "Elite".

**Impact Assessment :**
*   **Significant Impact :** Touche à la quasi-totalité des manifestes applicatifs du dépôt.

---

## 3. Goals and Background Context

**Goals :**
*   Atteindre 100% d'applications en statut "Elite".
*   Zéro état `OutOfSync` ou `Unknown` dans ArgoCD.
*   Validation de la résilience de la couche stockage (iSCSI/NFS).

**Background Context :**
Le cluster Vixens nécessite une phase de consolidation après une période de croissance rapide. L'initiative "Goldification" vise à appliquer les meilleures pratiques K8s (probes, ressources, sécurité) de manière systématique.

---

## 4. Requirements

### 4.1. Functional (FR)
*   **FR1 : Standardisation "Goldification" :** Application du standard Elite à toutes les apps (Probes, Resource limits, stable tags).
*   **FR2 : Résolution des États Inconnus :** Correction de `frigate`, `hydrus-client`, `sonarr`, `stirling-pdf-ingress` et `synology-csi-secrets`.
*   **FR3 : Résolution des Dérives GitOps :** Convergence totale de `renovate` et `vixens-app-of-apps` (OutOfSync).
*   **FR4 : Validation du Stockage :** Audit et stress-test du Synology CSI (iSCSI/NFS).
*   **FR5 : Stabilisation DNS :** Investigation des redémarrages fréquents de `external-dns-gandi`.

### 4.2. Non-Functional (NFR)
*   **NFR1 : Documentation Applicative :** Mise à jour des fichiers `docs/applications/*.md`.
*   **NFR2 : GitOps Strict :** Zéro modification manuelle, tout passe par PR et Kustomize build.

### 4.3. Technical Constraints (Real-world Analysis)
*   **Kubernetes Version :** v1.34.0
*   **OS :** Talos Linux v1.12.2 (Kernel 6.18.5)
*   **Nodes :** 3 Control-planes (phoebe, poison, powder), 2 Workers (peach, pearl)
*   **IP Range :** 192.168.111.191 - 192.168.111.195
*   **CNI :** Cilium
*   **Secrets :** Infisical Operator (Critique pour `synology-csi-secrets`)

## 5. Epic Structure

### 5.1. Epic 1: Critical Fixes & Infrastructure Stability (Prio 1)
**Goal :** Stabiliser le socle et corriger les dérives immédiates.

#### Story 1.1 : Diagnostic et Résolution des États ArgoCD Critiques
*   **Cibles :** `frigate`, `hydrus-client`, `sonarr`, `stirling-pdf-ingress`, `renovate`, `vixens-app-of-apps`.
*   **Acceptance Criteria :**
    *   Identification et suppression des ressources orphelines.
    *   Convergence totale de l'App-of-Apps.
    *   Toutes les applications cibles sont `Synced` et `Healthy`.

#### Story 1.2 : Sécurisation de la chaîne de confiance Stockage (Synology CSI)
*   **Cibles :** `synology-csi-secrets` et ressource `InfisicalSecret`.
*   **Acceptance Criteria :**
    *   `synology-csi-secrets` est `Synced` et `Healthy`.
    *   Validation de l'injection des credentials sans intervention manuelle.
    *   Stress-test de bascule de nœud avec remounting iSCSI réussi.

#### Story 1.3 : Stabilisation du Service DNS et Monitoring
*   **Cibles :** `external-dns-gandi` et `netvisor`.
*   **Acceptance Criteria :**
    *   Investigation et correction des redémarrages fréquents (Liveness probes ou Resource limits).
    *   Vérification des logs pour éliminer les erreurs de propagation.

#### Story 1.4 : Création de la documentation d'architecture réelle de Vixens
*   **Goal :** Produire `docs/architecture-vixens.md` basé sur la réalité du cluster (Nodes, CNI, Storage, GitOps flow).

### 5.2. Epic 2: Core Services Goldification (Prio 2)
**Goal :** Porter les services d'infrastructure (Network, Security, Monitoring) au standard Elite.
*   **Story 2.1 :** Goldification Ingress (Traefik) et GitOps (ArgoCD).
*   **Story 2.2 :** Goldification Stockage et Observabilité.
*   **Story 2.3 :** Optimisation des ressources partagées (Redis, VPA).

### 5.3. Epic 3: Application Goldification Campaign (Prio 3)
**Goal :** Standardisation industrielle des applications utilisateurs.
*   **Story 3.1 :** Goldification du domaine Multimédia (Media Stack).
*   **Story 3.2 :** Goldification des services Outils et Maison (Tools & Home).
*   **Story 3.3 :** Déploiement des services de maintenance (Janitorial Services).

---

## 6. Change Log
| Change | Date | Version | Description | Author |
|--------|------|---------|-------------|--------|
| Initial Draft | 2026-01-29 | 1.0 | Initial PRD for Brownfield Enhancement | John (PM) |
| Tech Alignment | 2026-01-29 | 1.1 | Integration of real-world cluster data and specific app targets | John (PM) |
