# MCP Tools Guide - Vixens

Ce guide documente les outils MCP disponibles pour les agents intervenant sur l'infrastructure **Vixens**. L'utilisation correcte de ces outils est cruciale pour le respect du workflow GitOps et la stabilité du cluster.

---

## 🏗️ 1. Serena (Outils de Code Sémantiques)
**Usage :** Opérations sur le code source, analyse sémantique et édition.

| Outil | Quand l'utiliser ? |
|-------|-------------------|
| `find_symbol` | Pour localiser une ressource K8s (Deployment, Service) par son nom sans chercher dans tout le repo. |
| `get_symbols_overview` | Pour comprendre la structure d'un dossier `base` ou `overlay` en un coup d'œil. |
| `replace_content` | **Prioritaire** pour les modifications YAML. Utiliser le mode `regex` pour des remplacements précis. |
| `search_for_pattern` | Pour trouver des modèles existants (ex: "comment sont configurés les ingress ailleurs ?"). |
| `execute_shell_command` | **INTERDIT** pour les outils CLI (`just`, `bd`, `git`). Utiliser l'outil Bash standard à la place. |

---

## 🌐 2. Playwright (Validation Web & Navigateur)
**Usage :** Validation fonctionnelle des applications après déploiement.

| Outil | Quand l'utiliser ? |
|-------|-------------------|
| `browser_navigate` | Pour vérifier qu'une UI (HA, Vaultwarden, AdGuard) est accessible en HTTPS. |
| `browser_snapshot` | Pour valider le contenu du DOM (ex: vérifier que la mire de login s'affiche). |
| `browser_take_screenshot` | Pour fournir une preuve visuelle de la validation au "User". |

---

## 🐙 3. GitHub (Workflow GitOps & Collaboration)
**Usage :** Gestion du cycle de vie des Pull Requests et promotion.

| Outil | Quand l'utiliser ? |
|-------|-------------------|
| `create_pull_request` | Systématiquement pour toute modification sur `main`. |
| `pull_request_read` | Pour vérifier l'état des checks de validation avant de tenter un merge. |
| `merge_pull_request` | Une fois les checks "Green", pour fusionner sur `main`. |
| `push_files` | Pour pousser plusieurs fichiers corrigés en un seul commit atomique. |

---

## 🔧 4. Terraform (Registry & Documentation)
**Usage :** Recherche de versions et de bonnes pratiques pour l'IaC (Talos/K8s).

| Outil | Quand l'utiliser ? |
|-------|-------------------|
| `get_provider_details` | Pour vérifier les arguments valides d'une ressource (ex: Cilium, Talos). |
| `search_modules` | Pour trouver des modules Terraform officiels lors de l'ajout de nouvelles capacités. |

---

## 📚 5. Context7 (Documentation Librairies)
**Usage :** Recherche de documentation API à jour.

| Outil | Quand l'utiliser ? |
|-------|-------------------|
| `get-library-docs` | Pour obtenir la syntaxe exacte d'une commande (ex: flags rclone, options litestream). |

---

## 🎨 6. Nano Banana (Génération d'Images & Diagrammes)
**Usage :** Documentation visuelle et architecture.

| Outil | Quand l'utiliser ? |
|-------|-------------------|
| `generate_diagram` | Pour illustrer des flux complexes (ex: flux de restauration Litestream -> S3 -> PVC). |

---

## 🐚 7. Outils Core & Beads (CLI Shell)
**Usage :** Orchestration et gestion des tâches.

| Méthode | Quand l'utiliser ? |
|---------|-------------------|
| `run_shell_command` | **Obligatoire** pour `just`, `git`, `kubectl`, `yamllint` et `bd`. |
| `bd` (via shell) | **Seul outil autorisé** pour la gestion des tâches (Beads). Ne jamais utiliser Archon pour cela. |

---

## 🚨 Règles d'Or pour l'Agent
1. **Analyse avant Action** : Utiliser `serena__read_file` ou `kubectl get` pour valider l'état réel avant de modifier quoi que ce soit.
2. **GitOps Strict** : Pas de `kubectl apply` direct si une alternative GitOps existe (sauf nettoyage d'objets immuables orphelins).
3. **Validation Double** : Toujours lancer `just lint` (via shell) et vérifier le rendu `kustomize build` avant de pousser.
4. **Heredoc Shell** : Éviter les longs `cat <<EOF` via `run_shell_command` s'ils contiennent des variables `$`, préférer `write_file`.
