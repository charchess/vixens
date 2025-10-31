### **Plan de Projet VIXENS - Roadmap Complète**

#### **Phase 0 : Initialisation (Pré-requis)**

**Sprint 0 : Préparation de l'Atelier**
*   **Objectif :** Mettre en place les outils et la structure nécessaires au projet.
*   **DoD du Sprint :** L'environnement de travail local est prêt, et le dépôt Git est structuré.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S0.1** | Installer les outils locaux | Les commandes `git`, `terraform`, `kubectl`, `talosctl`, `cilium` sont fonctionnelles. |
| **S0.2** | Structurer le dépôt Git | Le dépôt `vixens` est créé avec l'arborescence de dossiers `terraform/`, `kubernetes/`, etc. |
| **S0.3** | Mettre en place la CI de base | Créer un workflow GitHub Actions qui valide le formatage du code Terraform (`fmt -check`) et YAML (lint) sur chaque Pull Request. | Une PR avec un code mal formaté échoue au check de la CI. |

---

#### **Phase 1 : La Fondation (Environnement DEV)**

**Sprint 1 : Le Socle Infrastructure HA (DEV)**
*   **Objectif :** Obtenir un cluster Kubernetes `DEV` en Haute Disponibilité (3 control planes), provisionné par un module Terraform `talos-cluster` flexible.
*   **DoD du Sprint :** Un cluster de 3 nœuds est `Ready`, le réseau est validé, et l'infrastructure est gérée par un module Terraform robuste.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S1.1** | Module Talos : Gérer une liste de Nœuds Control Plane. | Le module accepte une variable `control_plane_nodes` et génère les configurations via un `for_each`. |
| **S1.2** | Environnement DEV : Instancier le module pour un Control Plane unique. | `terraform apply` réussit. `kubectl get node` affiche 1 nœud. |
| **S1.3** | Module Talos : Intégrer la configuration Cilium. | Après `apply`, le nœud passe à `Ready` et `kubectl get pods -n kube-system` montre les pods Cilium en `Running`. |
| **S1.4** | Module Talos : Ajouter la gestion générique des Nœuds Workers. | Le module accepte une variable `worker_nodes` (peut être vide) sans erreur au `plan`. |
| **S1.5** | Environnement DEV : Configurer le cluster de 3 Control Planes (HA). | `terraform apply` configure les 2 nœuds supplémentaires. `kubectl get nodes` affiche 3 nœuds `Ready`. |
| **S1.6** | Validation complète du réseau | La commande `cilium connectivity test` s'exécute sans erreur. |

**Sprint 2 : Le Cerveau GitOps (DEV)**
*   **Objectif :** Installer et configurer ArgoCD pour qu'il gère le cluster `DEV` via Git.
*   **DoD du Sprint :** Le cluster est entièrement piloté par la branche `dev` du dépôt Git. Un `git push` sur cette branche entraîne une modification dans le cluster.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S2.1** | Déployer le bootstrap d'ArgoCD via Terraform. | `terraform apply` déploie ArgoCD. `kubectl get pods -n argocd` montre tous les pods `Running`. |
| **S2.2** | Créer la structure "App of Apps" dans le dépôt Git. | Créer les fichiers `root-app.yaml` et la structure `kubernetes/clusters/dev/`. `git commit` & `push`. |
| **S2.3** | Appliquer la racine "App of Apps" manuellement. | `kubectl apply -f kubernetes/bootstrap/root-app-dev.yaml`. | Dans l'UI ArgoCD, l'application `root-dev` apparaît. |
| **S2.4** | Configurer l'auto-gestion d'ArgoCD. | Créer l'application `argocd.yaml` dans `kubernetes/clusters/dev/`. | L'application `argocd` apparaît dans l'UI, `Healthy` et `Synced`. |

**Sprint 3 : La Sécurisation des Secrets (DEV)**
*   **Objectif :** Intégrer SOPS dans le workflow GitOps.
*   **DoD du Sprint :** Les secrets Kubernetes sont gérés de manière sécurisée (chiffrés) dans le dépôt Git.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S3.1** | Mettre en place une clé de chiffrement SOPS (Age ou GPG). | Une commande `sops --encrypt --decrypt test.yaml` fonctionne localement. |
| **S3.2** | Configurer ArgoCD pour utiliser SOPS. | Modifier la configuration du `argocd-repo-server` pour intégrer le déchiffrement SOPS. | Les logs du `argocd-repo-server` indiquent que le plugin SOPS est actif. |
| **S3.3** | Créer et déployer un premier secret chiffré. | Créer un `Secret` Kubernetes, chiffrer ses données avec SOPS, et le commiter. | `kubectl get secret <nom-du-secret> -o yaml` montre les données déchiffrées dans le cluster. |

---

#### **Phase 2 : Les Services Essentiels (Environnement DEV)**

**Sprint 4 : La Porte d'Entrée (Réseau)**
*   **Objectif :** Exposer les services du cluster au réseau local.
*   **DoD du Sprint :** L'Ingress Controller est déployé et accessible via une IP externe fixe.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S4.1** | Déployer MetalLB via GitOps. | Créer l'application MetalLB dans `kubernetes/clusters/dev/`. Les pods sont `Running`. |
| **S4.2** | Configurer l'IPAddressPool de MetalLB. | Créer le manifeste `IPAddressPool` pour le réseau `DEV`. ArgoCD l'applique. | `kubectl get ipaddresspool -n metallb-system` montre la ressource configurée. |
| **S4.3** | Déployer Traefik via GitOps. | Créer l'application Traefik. Les pods sont `Running`. |
| **S4.4** | Valider l'exposition de Traefik. | `kubectl get svc -n traefik traefik` affiche une `EXTERNAL-IP` provenant du pool MetalLB. |

**Sprint 5 : Les Certificats Automatiques (TLS)**
*   **Objectif :** Activer le HTTPS automatique pour les applications.
*   **DoD du Sprint :** Une application test est accessible en HTTPS avec un certificat valide généré automatiquement.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S5.1** | Déployer Cert-Manager via GitOps. | Créer l'application Cert-Manager. Les pods sont `Running`. |
| **S5.2** | Créer le secret pour l'API Gandi (avec SOPS). | Créer et commiter le secret chiffré pour l'API Gandi. | Le secret existe et est correctement formaté dans le cluster. |
| **S5.3** | Déployer le `ClusterIssuer` Gandi via GitOps. | Créer le manifeste du `ClusterIssuer` pour le challenge DNS01. | `kubectl describe clusterissuer gandi-dns` affiche un statut `Ready`. |
| **S5.4** | Déployer une application test (`whoami`) avec Ingress. | Créer l'application `whoami` avec son `IngressRoute` Traefik. | `kubectl get certificate -n whoami` montre un certificat `Ready`. Accéder à l'URL en HTTPS montre un cadenas vert. |

**Sprint 6 : Le Stockage Persistant**
*   **Objectif :** Permettre aux applications de stocker des données de manière persistante.
*   **DoD du Sprint :** Les deux types de stockage (CSI Synology et NFS) sont fonctionnels et testés.

| ID Tâche | Tâche (Orientée Code/Config) | Validation de la Tâche |
| :--- | :--- | :--- |
| **S6.1** | Déployer le CSI Synology via GitOps. | Créer l'application CSI. Les pods sont `Running`. |
| **S6.2** | Déployer le provisionneur NFS via GitOps. | Créer l'application NFS. Les pods sont `Running`. |
| **S6.3** | Créer les `StorageClass` via GitOps. | Créer les manifestes pour les `StorageClass` `synology-csi` et `nfs`. | `kubectl get sc` liste les deux nouvelles Storage Classes. |
| **S6.4** | Valider la persistance des données. | Déployer un pod de test qui écrit un fichier dans un PVC de chaque type. Supprimer le pod, le recréer. | Le fichier est toujours présent dans le volume après recréation du pod. |

---

### **Liste Cumulative de Non-Régression (Version Initiale)**

Cette liste s'allongera après chaque sprint. Avant de promouvoir du code vers un nouvel environnement, il faudra s'assurer que tous les tests pertinents de cette liste passent toujours.

**Fondation Infrastructure (Après Sprint 1)**
1.  `terraform plan` dans `environments/dev` ne montre aucun changement sur une infra déployée.
2.  `kubectl get nodes` retourne 3 nœuds, tous en statut `Ready`.
3.  `kubectl get pods -n kube-system` montre tous les pods `cilium-*` en statut `Running`.
4.  La commande `cilium connectivity test` se termine avec 0 test échoué.

**Fondation GitOps (Après Sprint 2)**
5.  L'UI ArgoCD est accessible.
6.  Les applications `root-dev` et `argocd` sont `Healthy` et `Synced`.
7.  Un `git push` sur la branche `dev` est reflété dans le cluster en moins de 3 minutes.

**Fondation Sécurité (Après Sprint 3)**
8.  Un secret de test chiffré avec SOPS dans Git est lisible (déchiffré) via `kubectl get secret`.

**(La liste continuera de s'allonger avec les Sprints 4, 5, 6, etc.)**
