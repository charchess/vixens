# Conventions de Codage du Projet Vixens

Ce document est le guide de style officiel pour tout le code et les configurations du projet. Le respect de ces conventions est obligatoire pour garantir la lisibilité, la cohérence et la maintenabilité du projet.

## 1. Conventions Générales

### 1.1. Langue

- **Code & Commits**: Tout le code (noms de variables, ressources, commentaires) et les messages de commit doivent être rédigés en **Anglais**.
- **Documentation**: La documentation (`.md` files) est rédigée en Français.

### 1.2. Format de Fichier

- **Encodage**: Tous les fichiers texte doivent utiliser l'encodage **UTF-8**.
- **Fins de ligne**: Les fins de ligne doivent être de type Unix (**LF**).

## 2. Conventions Terraform (`.tf`)

### 2.1. Nommage

- Toutes les identifiants (ressources, data sources, variables, outputs) doivent utiliser le style **`snake_case`**.

```hcl
# Bon
resource "talos_machine_configuration" "control_plane" {}
variable "worker_node_count" {}
```

### 2.2. Variables

- Chaque variable déclarée doit obligatoirement inclure `type` et `description`. Un `default` est fortement recommandé.

```hcl
variable "cluster_name" {
  description = "The unique name of the Kubernetes cluster."
  type        = string
  default     = "vixens-dev"
}
```

### 2.3. Tagging des Ressources

- Toutes les ressources qui le supportent doivent être étiquetées avec un ensemble commun de tags pour la traçabilité.

```hcl
locals {
  common_tags = {
    project     = "vixens"
    environment = var.environment
    managed-by  = "terraform"
  }
}
```

## 3. Conventions Kubernetes (`.yaml`)

### 3.1. Structure des Fichiers

- Chaque fichier YAML doit commencer par un commentaire indiquant son chemin complet et par un séparateur de document (`---`).

```yaml
# /kubernetes/apps/my-app/base/deployment.yaml
---
apiVersion: v1
kind: Deployment
...
```

### 3.2. Nommage des Ressources

- La valeur de `metadata.name` doit suivre la convention **`kebab-case`**.

### 3.3. Labels

- Les ressources Kubernetes doivent utiliser les [Labels Recommandés par Kubernetes](https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/) comme base.

```yaml
metadata:
  labels:
    app.kubernetes.io/name: my-app
    app.kubernetes.io/instance: my-app-v1.2.3
    app.kubernetes.io/managed-by: argocd
```
