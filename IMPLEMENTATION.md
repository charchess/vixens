# Plan d'Implémentation pour la Refactorisation Terraform

## Introduction

Ce document détaille les étapes techniques pour la refactorisation du code Terraform du projet. L'objectif est de centraliser la logique, d'éliminer la duplication de code, d'améliorer la maintenabilité et de suivre les meilleures pratiques Terraform.

---

### Action 1 : Centraliser la logique d'infrastructure (Principe DRY)

**Objectif :** Créer un module `base` réutilisable et transformer les répertoires d'environnement en simples "lanceurs".

**Étapes d'implémentation :**

1.  **Créer le répertoire `base` :**
    ```bash
    mkdir -p terraform/base
    ```

2.  **Déplacer la logique commune :**
    Copiez le contenu de `terraform/environments/dev` (sauf `terraform.tfvars`) dans `terraform/base`.
    ```bash
    mv terraform/environments/dev/*.tf terraform/base/
    ```
    Le répertoire `terraform/base` contiendra maintenant : `main.tf`, `providers.tf`, `versions.tf`, `variables.tf`, etc.

3.  **Nettoyer les répertoires d'environnement :**
    Supprimez tous les fichiers `.tf` des répertoires `terraform/environments/dev` et `terraform/environments/test`. Ne conservez que les fichiers `terraform.tfvars`.

4.  **Créer le `main.tf` des environnements :**
    Dans chaque répertoire d'environnement (`dev` et `test`), créez un nouveau fichier `main.tf` qui appellera le module `base`.

    **Contenu pour `terraform/environments/dev/main.tf` :**
    ```terraform
    # Ce fichier lance l'infrastructure pour l'environnement de développement.
    # Il appelle le module 'base' qui contient toute la logique commune.

    module "dev_environment" {
      source = "../../base"

      # Variables spécifiques à l'environnement
      # Ces valeurs sont lues depuis le fichier terraform.tfvars
      cluster_name          = var.cluster_name
      talos_version         = var.talos_version
      kubernetes_version    = var.kubernetes_version
      cluster_endpoint      = var.cluster_endpoint
      control_plane_nodes   = var.control_plane_nodes
      worker_nodes          = var.worker_nodes
      # ... et toutes les autres variables nécessaires au module base
    }
    ```
    *Note : Il faudra créer un fichier `variables.tf` dans chaque environnement pour déclarer les variables utilisées dans ce `main.tf`, qui seront elles-mêmes alimentées par `terraform.tfvars`.*

---

### Action 2 : Mettre en place un backend distant

**Objectif :** Stocker l'état Terraform de manière centralisée et sécurisée.

**Étapes d'implémentation :**

1.  **Créer le fichier `backend.tf` :**
    Dans `terraform/base`, créez un fichier `backend.tf`. Nous utiliserons le backend `http` comme exemple, qui est un backend standard et flexible.

    **Contenu de `terraform/base/backend.tf` :**
    ```terraform
    terraform {
      backend "http" {
        # La configuration (adresse, etc.) sera fournie dynamiquement
        # lors de l'initialisation pour chaque environnement.
      }
    }
    ```

2.  **Initialiser le backend pour chaque environnement :**
    Depuis chaque répertoire d'environnement (`environments/dev`, `environments/test`), exécutez `terraform init` en passant la configuration du backend.

    **Exemple pour l'environnement `dev` :**
    ```bash
    cd terraform/environments/dev

    terraform init \
      -backend-config="address=https://my-terraform-state-server.com/state/vixens-dev" \
      -backend-config="lock_address=https://my-terraform-state-server.com/lock/vixens-dev" \
      -backend-config="unlock_address=https://my-terraform-state-server.com/unlock/vixens-dev" \
      -backend-config="username=my-user" \
      -backend-config="password=my-secret-password"
    ```
    *Note : Si vous n'avez pas de serveur de backend HTTP, les alternatives sont Terraform Cloud (recommandé) ou un bucket S3 (si vous utilisez AWS).*

---

### Action 3 : Améliorer la définition des variables

**Objectif :** Rendre le code auto-documenté et plus facile à utiliser.

**Étapes d'implémentation :**

1.  **Éditer `terraform/base/variables.tf` :**
    Pour chaque variable, ajoutez une `description` détaillée, un `type` strict, et une `default` si pertinent.

    **Exemple avant/après pour la variable `cluster_name` :**

    *Avant :*
    ```terraform
    variable "cluster_name" {}
    ```

    *Après :*
    ```terraform
    variable "cluster_name" {
      description = "Le nom unique du cluster Talos et Kubernetes. Ce nom est utilisé pour préfixer de nombreuses ressources."
      type        = string
      # Pas de valeur par défaut ici, car le nom doit être défini pour chaque environnement.
    }

    variable "talos_version" {
      description = "La version de Talos Linux à déployer sur les nœuds."
      type        = string
      default     = "v1.11.3"
    }
    ```

---

### Action 4 : Améliorer la configuration des providers

**Objectif :** Centraliser et dynamiser la configuration des providers pour éviter les chemins en dur et les configurations multiples.

**Étapes d'implémentation :**

1.  **Modifier `terraform/base/providers.tf` :**
    Utilisez les sorties du module `talos` pour configurer les providers `helm` et `kubectl`.

    **Contenu de `terraform/base/providers.tf` :**
    ```terraform
    provider "talos" {
      # La configuration est héritée de l'appelant
    }

    provider "helm" {
      kubernetes {
        host                   = module.talos.kubernetes_host
        client_certificate     = base64decode(module.talos.kubernetes_client_certificate)
        client_key             = base64decode(module.talos.kubernetes_client_key)
        cluster_ca_certificate = base64decode(module.talos.kubernetes_ca_certificate)
      }
    }

    provider "kubectl" {
      host                   = module.talos.kubernetes_host
      client_certificate     = base64decode(module.talos.kubernetes_client_certificate)
      client_key             = base64decode(module.talos.kubernetes_client_key)
      cluster_ca_certificate = base64decode(module.talos.kubernetes_ca_certificate)
    }
    ```

2.  **Nettoyer les sous-modules :**
    Supprimez les blocs `provider` des fichiers `.tf` dans les sous-modules (ex: `terraform/modules/argocd/versions.tf`). Les modules hériteront automatiquement de la configuration du module `base` qui les appelle.

---

## Conclusion

En suivant ce plan, le code Terraform sera transformé en une base de code robuste, maintenable et évolutive. La logique étant centralisée, les risques d'erreurs entre environnements sont éliminés, et la création de nouveaux environnements devient une tâche triviale.
