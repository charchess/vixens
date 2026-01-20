# Internal Toolset (scripts/)

Internal scripts for cluster management, report generation, and validation.

## 📊 État du Déploiement

| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | N/A     |
| Prod          | [x]     | [x]       | [x]   | N/A     |

## 🏗️ Architecture

Le répertoire `scripts/` est organisé par catégories :
- `analysis/`: Outils d'audit des ressources (VPA, priorités).
- `infra/`: Automatisation de l'infrastructure et ArgoCD.
- `lib/`: Bibliothèques partagées.
- `reports/`: Génération de rapports pour la documentation.
- `testing/`: Suites de tests fonctionnels et techniques.
- `utils/`: Utilitaires CLI généraux (`k`, `gp`, `check`).
- `validation/`: Scripts de conformité et validation GitOps.

## 🚀 Utilisation

Les scripts sont principalement invoqués via `just` :
- `just reports`: Génère les rapports d'état.
- `just lint`: Valide les manifests YAML.
- `just start/next/close`: Gère le workflow des tâches.

## ✅ Validation

La validation des scripts consiste en :
1. Vérification de la structure du répertoire.
2. Validation du `justfile` pour s'assurer que les chemins sont corrects.
3. Tests manuels des utilitaires critiques (`k`, `gp`).
