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

Un push direct exceptionnel sans PR associée utilise un suffixe SHA court à la place du numéro de PR.

Après validation explicite, `prod-working` peut être déplacé manuellement vers une release `prod-v*` connue comme bonne.

## 1. Vérifier la version dev

Après le merge d'une PR vers `main`, `.github/workflows/auto-tag-dev.yaml` retrouve la PR associée au commit via l'API GitHub et crée automatiquement :

```text
dev-vYYYY.MM.<PR>
```

Exemple pour la PR #3512 :

```text
dev-v2026.09.3512
```

Pour un push direct exceptionnel sans PR associée, le suffixe est un SHA court, par exemple :

```text
dev-v2026.09.7fe2e79d8
```

Lister les versions récentes :

```bash
git fetch --tags --force
git tag -l 'dev-v*' --sort=-version:refname | head
```

Ne pas recréer manuellement un tag dev déjà produit par le workflow.

## 2. Promouvoir

La commande de promotion est l'autorisation humaine explicite pour **ce candidat
`dev-v*` précis**. Un agent automatisé peut préparer la commande mais ne doit pas
l'exécuter sans instruction explicite de l'utilisateur/opérateur après les
validations pertinentes.

Déclencher le workflow GitHub avec la version exacte du tag dev, sans le préfixe `dev-` :

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Puis suivre son exécution :

```bash
gh run list --workflow=promote-prod.yaml --limit=1
gh run watch
```

Le workflow est sérialisé : deux promotions ne peuvent pas modifier `prod-stable` en parallèle.

Il :

1. vérifie que `dev-v<version>` existe ;
2. résout le commit immuable correspondant et vérifie qu'il appartient à l'historique de `main` ;
3. checkout ce commit exact dans un workspace détaché ;
4. génère depuis ce commit le SBOM et un JSON de métadonnées de promotion ;
5. crée `prod-v<version>` sur le même commit, ou vérifie qu'un tag déjà présent pointe exactement au même endroit ;
6. publie SBOM et métadonnées dans la GitHub Release associée à `prod-v<version>` ;
7. déplace `prod-stable` atomiquement vers ce commit **en dernière étape de mutation prod** ;
8. relit le tag distant et vérifie qu'il résout bien vers le commit attendu.

`prod-stable` ne doit pas être déplacé manuellement lors d'une promotion normale.

Si la génération du SBOM ou la publication de la release échoue, `prod-stable` n'est pas modifié. Si une exécution échoue après création du tag immuable mais avant la bascule, le workflow peut être relancé : il accepte le tag existant uniquement s'il pointe vers le commit attendu.

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

La GitHub Release `prod-v...` doit contenir :

- `sbom-prod-<version>.spdx.json` ;
- `promotion-prod-<version>.json` avec commit, source dev, opérateur et workflow d'origine.

## 4. Marquer un état connu-bon

`prod-working` est un marqueur de secours **manuel**. Il n'est jamais mis à jour automatiquement après une promotion.

Après avoir confirmé qu'une release fonctionne réellement et après autorisation
explicite pour marquer cette release connue-bonne :

```bash
gh workflow run mark-prod-working.yaml -f version=vYYYY.MM.<PR>
gh run watch
```

Le workflow vérifie que `prod-v<version>` existe, puis déplace `prod-working` atomiquement vers son commit et relit la cible pour confirmer l'opération.

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
- release GitHub contient SBOM + métadonnées de promotion ;
- ArgoCD prod `Synced/Healthy` ;
- smoke tests applicatifs réussis ;
- si la release est confirmée connue-bonne, déplacer explicitement `prod-working`.

## Source de vérité

- `.github/workflows/auto-tag-dev.yaml` — création des snapshots dev ;
- `.github/workflows/promote-prod.yaml` — promotion ;
- `.github/workflows/mark-prod-working.yaml` — marqueur manuel known-good ;
- `WORKFLOW.md` — règles de contribution et de promotion.
