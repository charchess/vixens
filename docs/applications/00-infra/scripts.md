# Internal toolset (`scripts/`)

Le répertoire `scripts/` contient des outils techniques réutilisables. Ils restent volontairement indépendants d'un command runner spécifique.

## Organisation

- `analysis/` — audits et analyses ponctuelles ;
- `infra/` — helpers infrastructure / ArgoCD ;
- `lib/` — bibliothèques partagées ;
- `reports/` — génération de rapports ;
- `testing/` — tests fonctionnels / techniques ;
- `utils/` — utilitaires CLI ;
- `validation/` — validation GitOps et conformité.

## Utilisation

Les scripts sont appelés directement ou par GitHub Actions lorsqu'ils constituent un contrôle CI.

Exemples :

```bash
# Rapports consolidés
scripts/reports/generate-all.sh

# Validation applicative
python3 scripts/validation/validate.py <app> dev

# Un générateur de rapport individuel
python3 scripts/reports/generate_lint_report.py --help
```

Les contrôles obligatoires d'une PR doivent être implémentés dans `.github/workflows/` ; un script local peut fournir l'implémentation, mais le workflow GitHub porte le garde-fou partagé.

## Principes

- un script utile doit pouvoir être lancé sans Serena/Just/outil d'agent ;
- les chemins doivent être relatifs au dépôt ou paramétrables lorsque possible ;
- pas de secret en clair ;
- pas de mutation persistante du cluster comme substitut à GitOps ;
- documenter les entrées, sorties et effets de bord des scripts non triviaux.
