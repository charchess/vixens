# Production promotion

La production Vixens est promue depuis un snapshot dev immuable ; elle n'est jamais construite depuis une branche de production séparée.

La référence canonique reste [WORKFLOW.md](../../WORKFLOW.md). Ce guide détaille les commandes opérationnelles.

## Modèle

```text
main
  ↓ auto-tag-dev.yaml
dev-vYYYY.MM.<PR>
  ↓ promote-prod.yaml
prod-vYYYY.MM.<PR>   (immuable)
  ↓
prod-stable          (alias mutable suivi par ArgoCD prod)
```

Après validation explicite, `prod-working` peut être déplacé manuellement vers une release `prod-v*` connue comme bonne.

## 1. Vérifier la version dev

Après le merge d'une PR vers `main`, `.github/workflows/auto-tag-dev.yaml` crée automatiquement :

```text
dev-vYYYY.MM.<PR>
```

Exemple pour la PR #3416 :

```text
dev-v2026.09.3416
```

Lister les versions récentes :

```bash
git fetch --tags --force
git tag -l 'dev-v*' --sort=-version:refname | head
```

Ne pas recréer manuellement un tag dev déjà produit par le workflow.

## 2. Promouvoir

Déclencher le workflow GitHub :

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Puis suivre son exécution :

```bash
gh run list --workflow=promote-prod.yaml --limit=1
gh run watch
```

Le workflow :

1. vérifie que `dev-v<version>` existe ;
2. résout le commit immuable correspondant ;
3. crée `prod-v<version>` sur le même commit ;
4. déplace `prod-stable` vers ce commit ;
5. génère et attache un SBOM à la release GitHub ;
6. laisse ArgoCD détecter le nouveau `prod-stable`.

`prod-stable` ne doit pas être déplacé manuellement lors d'une promotion normale.

## 3. Vérifier la production

Après le workflow :

```bash
kubectl -n argocd get applications
```

Pour une application ciblée, vérifier `Synced` et `Healthy`, puis réaliser le smoke test pertinent.

Vérifier aussi les tags :

```bash
git fetch --tags --force
git rev-parse 'prod-stable^{commit}'
git rev-parse 'prod-vYYYY.MM.<PR>^{commit}'
```

Les deux commits doivent être identiques.

## 4. Marquer un état connu-bon

`prod-working` est un marqueur de secours **manuel**. Il n'est jamais mis à jour automatiquement après une promotion.

Après avoir confirmé qu'une release fonctionne réellement :

```bash
gh workflow run mark-prod-working.yaml -f version=vYYYY.MM.<PR>
gh run watch
```

Le workflow vérifie que `prod-v<version>` existe, puis déplace `prod-working` vers son commit avec un tag annoté.

Vérification :

```bash
git fetch --tags --force
git rev-parse 'prod-working^{commit}'
git rev-parse 'prod-vYYYY.MM.<PR>^{commit}'
```

## Rollback

La récupération doit viser une release Git connue, pas une réparation permanente faite directement dans le cluster.

Deux références permettent de retrouver un état :

- `prod-working` : dernier état explicitement déclaré connu-bon ;
- `prod-v*` : historique immuable de toutes les promotions.

Le déplacement de `prod-stable` en rollback est une opération exceptionnelle. Il doit être explicite, auditable et suivi d'une vérification ArgoCD/production. Ne pas automatiser `prod-working` : sa valeur vient précisément de la validation humaine explicite.

## Checklist

Avant promotion :

- dev synchronisé et sain ;
- comportement modifié validé ;
- CI de la PR verte ;
- tag `dev-v...` présent ;
- migrations ou breaking changes compris.

Après promotion :

- workflow `promote-prod.yaml` vert ;
- `prod-v...` et `prod-stable` pointent vers le même commit ;
- ArgoCD prod `Synced/Healthy` ;
- smoke tests applicatifs réussis ;
- si la release est confirmée connue-bonne, déplacer explicitement `prod-working`.

## Source de vérité

- `.github/workflows/auto-tag-dev.yaml` — création des snapshots dev ;
- `.github/workflows/promote-prod.yaml` — promotion ;
- `.github/workflows/mark-prod-working.yaml` — marqueur manuel known-good ;
- `WORKFLOW.md` — règles de contribution et de promotion.
