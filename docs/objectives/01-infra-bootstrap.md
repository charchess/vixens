# Spécification de l'Objectif 01 : Création du module Terraform Talos et bootstrap du cluster `dev`

## 1. Contexte et Finalité

Cet objectif constitue la fondation de tout le projet. Le but est de configurer un cluster Kubernetes Talos entièrement fonctionnel pour l'environnement de développement (`dev`) sur des machines virtuelles (VMs) **préexistantes**.

Le cœur de ce sprint est la création d'un **module Terraform réutilisable** (`talos-cluster`) qui encapsulera toute la logique de configuration et de bootstrap de Talos. Ce module sera ensuite utilisé pour provisionner l'environnement `dev`.

La source de vérité pour toutes les configurations techniques (noms, adresses IP, VLANs) est le document [ARCHITECTURE.md](../ARCHITECTURE.md).

## 2. Prérequis OBLIGATOIRES

Avant d'exécuter `terraform apply`, l'infrastructure suivante doit être manuellement mise en place et fonctionnelle :
- Les 3 VMs (`obsy`, `opale`, `onyx`) sont créées dans Hyper-V.
- Leurs interfaces réseau sont configurées avec les adresses IP statiques et les VLANs décrits dans le document d'architecture.
- Les 3 VMs sont démarrées sur l'ISO de Talos et sont en **mode maintenance**.
- La machine `grenat` (où Terraform est exécuté) a une connectivité réseau vers les adresses IP du VLAN 111 des 3 VMs.

## 3. Périmètre Technique et Tâches à Réaliser

- **[ ] Développement du module Terraform `talos-cluster` :**
  - Dans `/terraform/modules/talos-cluster/`, créer le code Terraform nécessaire pour :
    - Accepter en variables d'entrée (inputs) la liste des nœuds, le nom du cluster, le VIP de l'endpoint, etc.
    - Configurer le provider `siderolabs/talos`.
    - Générer les configurations machine Talos (`machineconfig`) pour un ensemble de nœuds `controlplane`.
    - Lancer le processus de bootstrap du cluster Talos.
    - Exposer en sortie (outputs) les fichiers `talosconfig` et `kubeconfig`.

- **[ ] Utilisation du module pour l'environnement `dev` :**
  - Dans `/terraform/environments/dev/`, écrire le code `main.tf` qui va **instancier** le module `talos-cluster`.
  - Créer un fichier `terraform.tfvars` pour passer au module les valeurs spécifiques au cluster `dev` (noms des nœuds, IPs, etc.).

## 4. Définition de Terminé (Definition of Done)

Cet objectif sera considéré comme "terminé" si et seulement si tous les critères suivants sont validés :

- **[ ] Le code est modulaire :** Toute la logique de configuration de Talos est bien contenue dans le module `/terraform/modules/talos-cluster/`. Le code dans `/terraform/environments/dev/` est minimaliste et se contente d'appeler le module.
- **[ ] `terraform apply` s'exécute avec succès :** La commande `terraform apply` lancée depuis `/terraform/environments/dev/` se termine sans aucune erreur.
- **[ ] L'accès via `talosctl` est fonctionnel :** La commande `talosctl --talosconfig=./talosconfig health -n 192.168.111.11` (et pour les autres nœuds) retourne un état de santé `[HEALTHY]`.
- **[ ] L'accès via `kubectl` est fonctionnel :** La commande `kubectl --kubeconfig=./kubeconfig get nodes -o wide` retourne la liste des 3 nœuds avec le statut `Ready` et leurs adresses IP.
- **[ ] Les secrets ne sont pas dans Git :** Les fichiers générés (`talosconfig`, `kubeconfig`) ainsi que les fichiers d'état Terraform sont bien listés dans `.gitignore`.
- **[ ] Le dépôt est propre :** Après l'exécution de `terraform apply`, `git status` n'indique aucune modification non commitée.
