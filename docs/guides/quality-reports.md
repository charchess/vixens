# Quality reports

Vixens conserve plusieurs générateurs de rapports sous `scripts/reports/`. Ils sont des outils d'analyse ; ils ne pilotent pas le workflow GitOps et ne constituent pas une source de vérité parallèle.

## Génération consolidée

Lancer :

```bash
scripts/reports/generate-all.sh
```

Le script remplace l'ancienne recette `just reports` et orchestre les générateurs existants sans imposer de command runner particulier.

Par défaut, il cherche :

```text
.secrets/dev/kubeconfig-dev
.secrets/prod/kubeconfig-prod
```

Les chemins peuvent être surchargés :

```bash
VIXENS_DEV_KUBECONFIG=/path/to/dev \
VIXENS_PROD_KUBECONFIG=/path/to/prod \
  scripts/reports/generate-all.sh
```

Une étape qui nécessite un cluster est ignorée si le kubeconfig correspondant n'est pas disponible.

## Rapports principaux

Selon les données disponibles, le script met à jour notamment :

- `docs/reports/STATE-ACTUAL-dev.md`
- `docs/reports/STATE-ACTUAL-prod.md`
- `docs/reports/STATE-ACTUAL.md` (compatibilité historique)
- `docs/reports/APP-VERSIONS.md`
- `docs/reports/LINT-REPORT.md`
- `docs/reports/CONFORMITY-dev.md`
- `docs/reports/CONFORMITY-prod.md`
- `docs/reports/STATUS.md`
- `docs/reports/MANAGEMENT-REPORT.md`

Les fichiers JSON intermédiaires sont supprimés à la fin du run.

## Générateurs unitaires

Les outils peuvent aussi être lancés directement. Exemples :

```bash
python3 scripts/reports/generate_lint_report.py \
  --paths apps argocd \
  --output docs/reports/LINT-REPORT.md \
  --fail-threshold 0
```

```bash
KUBECONFIG=.secrets/prod/kubeconfig-prod \
python3 scripts/reports/generate_app_versions.py \
  --output docs/reports/APP-VERSIONS.md
```

```bash
python3 scripts/reports/conformity_checker.py \
  --actual docs/reports/STATE-ACTUAL-prod.md \
  --desired docs/reports/STATE-DESIRED.md \
  --output docs/reports/CONFORMITY-prod.md
```

## CI vs rapports locaux

Les contrôles qui doivent **bloquer une PR** appartiennent à `.github/workflows/` : syntaxe YAML, rendu Kustomize, sécurité, structure ArgoCD, etc.

Les rapports de `scripts/reports/` servent plutôt à :

- inventorier l'état courant ;
- analyser ressources/VPA ;
- produire des vues de conformité ;
- documenter une situation ;
- aider au triage et au capacity planning.

Un rapport local ne remplace donc pas un check CI requis.

## Maintenance

Lorsqu'un rapport n'est plus utilisé :

1. vérifier qu'aucun workflow/script actif ne le consomme ;
2. retirer son générateur et ses références dans la même PR ;
3. conserver au besoin un exemplaire historique dans `docs/reports/trash/` plutôt que maintenir une chaîne morte.

La création de tâches issues d'un rapport se fait dans GitHub Issues, conformément à `docs/guides/task-management.md`.
