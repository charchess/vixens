# Vixens workflow

Vixens suit un workflow GitOps centré sur GitHub. Ce document est la référence canonique pour modifier le dépôt.

## Principes

- **Git est la source de vérité** pour l'état désiré Kubernetes.
- **GitHub Issues** est le backlog canonique.
- **Pull Requests** est le seul chemin normal vers `main`.
- **GitHub Actions** valide et orchestre les opérations de cycle de vie.
- **ArgoCD** réconcilie les clusters depuis Git ; la CI ne déploie pas directement les manifests.
- Les outils locaux et les assistants IA sont des clients du workflow, jamais des dépendances du workflow.
- Les changements persistants via `kubectl apply`, `kubectl edit` ou `kubectl delete` sont interdits lorsqu'ils doivent être déclarés dans Git.

## Avant toute modification

1. Lire l'issue GitHub concernée ou en créer une.
2. Relire le `main` courant : plusieurs humains/agents peuvent travailler en parallèle.
3. Vérifier les PR ouvertes susceptibles de toucher les mêmes fichiers.
4. Identifier le scope exact du changement et les dépendances.
5. Lire la documentation et les ADR pertinents.

Ne jamais partir d'une copie locale supposée à jour sans vérifier l'état GitHub courant.

## Flux de changement

```text
GitHub Issue
    ↓
feature/fix/chore branch depuis main courant
    ↓
Pull Request
    ↓
CI / review
    ↓
merge vers main
    ↓
ArgoCD dev auto-sync
    ↓
validation dev
    ↓
dev-vYYYY.MM.PR
    ↓
workflow promote-prod.yaml
    ↓
prod-vYYYY.MM.PR + prod-stable
    ↓
ArgoCD prod auto-sync
    ↓
validation prod
```

### 1. Issue

Les changements significatifs sont décrits dans une GitHub Issue avec :

- contexte ;
- état actuel ;
- état cible ;
- critères d'acceptation ;
- dépendances et risques lorsque nécessaire.

Commandes utiles :

```bash
gh issue list --state open
gh issue view <number>
gh issue create --title "chore(repo): description"
```

### 2. Branche

Créer une branche courte depuis le `main` courant :

```bash
git fetch origin
git switch main
git pull --ff-only
git switch -c feat/<issue>-<slug>
```

Préfixes usuels : `feat/`, `fix/`, `chore/`, `docs/`, `refactor/`.

### 3. Pull Request et CI

La PR doit rester limitée au scope de l'issue. Les checks GitHub constituent les garde-fous exécutables du dépôt : rendu YAML/Kustomize, qualité, sécurité et contrôles structurels selon le workflow concerné.

```bash
gh pr create --base main --fill
```

Ne pas contourner un check en modifiant le cluster à la main.

### 4. Dev

Après merge, `main` représente l'état désiré de dev. ArgoCD synchronise automatiquement.

Vérifier au minimum :

```bash
kubectl -n argocd get applications
```

Pour une application ciblée, attendre `Synced` et `Healthy` puis réaliser les tests fonctionnels pertinents.

### 5. Tags dev

Le workflow `auto-tag-dev.yaml` crée automatiquement un tag de version dev basé sur la PR squashée :

```text
dev-vYYYY.MM.<PR>
```

Ces tags sont des snapshots immuables et servent de source à la promotion prod.

### 6. Promotion production

La promotion production passe **uniquement** par GitHub Actions :

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Ne jamais créer ou déplacer `prod-stable` manuellement.

Le workflow crée :

- `prod-vYYYY.MM.<PR>` : release prod **immuable** ;
- `prod-stable` : alias **mutable** suivi par ArgoCD prod.

### 7. `prod-working`

`prod-working` est un marqueur manuel vers un état production connu comme fonctionnel.

- Il n'est **jamais** mis à jour automatiquement.
- Il n'est déplacé qu'avec accord explicite de l'utilisateur/opérateur.
- Avant un chantier de nettoyage ou un changement à risque, il peut être repositionné sur la release prod actuellement validée.
- Les tags `prod-v*` restent la trace historique immuable et permettent de retrouver précisément toute release.

### 8. Validation prod

Après promotion, attendre ArgoCD `Synced/Healthy` puis tester le comportement réellement modifié. Une issue n'est considérée terminée qu'après validation adaptée au changement.

## Rollback

Le rollback doit revenir à un état Git connu plutôt que réparer durablement le cluster à la main.

Références utiles :

- `prod-working` pour le dernier état explicitement déclaré connu-bon ;
- les tags immuables `prod-v*` pour une release précise.

Toute mutation d'urgence du cluster doit être temporaire, documentée et réconciliée rapidement dans Git.

## Règles pour humains et agents

Tous les contributeurs suivent les mêmes règles, quel que soit l'éditeur ou l'assistant utilisé :

1. vérifier le `main` courant avant de modifier ;
2. vérifier les PR concurrentes ;
3. travailler sur une branche dédiée ;
4. limiter le scope ;
5. utiliser les patterns déjà présents avant d'en créer de nouveaux ;
6. ne jamais stocker de valeur secrète en clair dans Git ;
7. ne pas muter durablement le cluster hors GitOps ;
8. laisser CI puis ArgoCD faire leur travail ;
9. mettre à jour la documentation lorsqu'un contrat ou une architecture change ;
10. ne jamais inventer un workflow spécifique à un outil ou à un agent.

## Où documenter quoi

- `README.md` : présentation et démarrage rapide.
- `WORKFLOW.md` : workflow de contribution et de promotion — **ce document**.
- `AGENTS.md` : contraintes supplémentaires utiles aux agents, sans recopier le workflow.
- `docs/architecture/` : architecture actuelle.
- `docs/adr/` : décisions et historique des choix.
- `docs/guides/` : procédures de travail récurrentes.
- `docs/troubleshooting/` : diagnostics et runbooks.
- `docs/applications/` : particularités par application.

Les documents historiques peuvent décrire d'anciens outils ; ils ne remplacent jamais ce workflow canonique.
