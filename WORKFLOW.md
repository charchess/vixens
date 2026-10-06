# Vixens workflow

Vixens suit un workflow GitOps centré sur GitHub. Ce document est la référence canonique pour modifier le dépôt.

## Principes

- **Git est la source de vérité** pour l'état désiré Kubernetes.
- **GitHub Issues** est le backlog canonique et porte le besoin ainsi que ses critères d'acceptation.
- **GitHub Project `vixens roadmap`** porte la planification active (`Status`, `Priority`, `Target`) sans remplacer les issues.
- **Pull Requests** est le seul chemin normal vers `main`.
- **GitHub Actions** valide et orchestre les opérations de cycle de vie.
- **ArgoCD** réconcilie les clusters depuis Git ; la CI ne déploie pas directement les manifests.
- Les outils locaux et les assistants IA sont des clients du workflow, jamais des dépendances du workflow.
- Les changements persistants via `kubectl apply`, `kubectl edit` ou `kubectl delete` sont interdits lorsqu'ils doivent être déclarés dans Git.

## Sources de vérité

| Sujet | Source de vérité |
|---|---|
| Processus de contribution / promotion | `WORKFLOW.md` |
| Besoin, scope, critères d'acceptation | GitHub Issue |
| Planification / priorité / release cible | GitHub Project `vixens roadmap` |
| État désiré exécutable | `main`, workflows GitHub et manifests |
| Architecture courante | documentation active + ADR applicables |
| État runtime observé | cluster, en lecture seule pour diagnostic/validation |

Une documentation active qui contredit le comportement exécutable courant doit être corrigée ou explicitement marquée historique. Un état live divergent ne devient pas pour autant le desired state : la correction persistante revient dans Git.

## Avant toute modification

1. Lire l'issue GitHub concernée ou en créer une.
2. Relire le `main` courant : plusieurs humains/agents peuvent travailler en parallèle.
3. Vérifier les PR ouvertes susceptibles de toucher les mêmes fichiers.
4. Identifier le scope exact du changement et les dépendances.
5. Lire la documentation et les ADR pertinents.
6. Pour une issue planifiée, vérifier sa présence et ses champs `Status`, `Priority` et `Target` dans `vixens roadmap`.

Ne jamais partir d'une copie locale supposée à jour sans vérifier l'état GitHub courant. Pour une reprise sans contexte préalable, suivre également la checklist de reprise à froid définie dans `AGENTS.md`.

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
    ├──→ auto-tag-dev.yaml → dev-vYYYY.MM.PR (snapshot immuable candidat)
    ↓
ArgoCD dev auto-sync
    ↓
validation dev du candidat
    ↓
autorisation humaine explicite
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

#### TXO Fabric Operator — politique de tests de non-régression

Le TXO Fabric Operator est un composant à fort blast radius. Toute modification de son comportement doit être accompagnée de tests unitaires/reconciliation adaptés dans la même PR.

Règles minimales :

- toute nouvelle fonctionnalité ou nouvelle branche de comportement de l'operator doit ajouter ou étendre des `*_test.go` couvrant le contrat introduit ;
- toute correction de bug doit ajouter un test de régression qui aurait échoué avant le correctif ;
- les chemins de sécurité, isolation tenant, lifecycle/finalizers, rétention/destruction, credentials, IAM et permissions doivent inclure les cas négatifs/fail-closed pertinents ;
- les réconciliations doivent être testées pour l'idempotence lorsqu'une répétition sans changement ne doit provoquer ni mutation ni rollout ;
- les fonctionnalités multi-tenant/multi-agent doivent tester l'absence de collision et de contamination entre tenants/agents ;
- un refactor ne doit pas réduire silencieusement la couverture des contrats existants ; toute suppression ou adaptation de test doit être justifiée par un changement explicite du contrat ;
- viser le maximum de branches et d'invariants utiles en tests rapides et déterministes, sans écrire des tests tautologiques uniquement pour augmenter un pourcentage de couverture ;
- la couverture Go est publiée comme artefact CI et sert à repérer les zones non testées ; un seuil global arbitraire n'est pas utilisé comme substitut à la couverture par scénario ;
- les comportements impossibles à prouver correctement en TU utilisent envtest/kind lorsqu'une telle couche existe pour le scope ; CSI/TrueNAS, Cilium, ArgoCD et services externes réels restent couverts par la validation runtime/physique appropriée.

La CI de l'operator refuse une PR qui modifie du code Go métier de l'operator sans modification/ajout de tests unitaires correspondants. Les fichiers générés et les PR de pin d'image ne déclenchent pas cette exigence.

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

Le workflow `auto-tag-dev.yaml` crée automatiquement un snapshot dev immuable après chaque push sur `main`.

**Un `dev-v*` prouve l'identité d'un snapshot, pas sa validation.** Le tag peut exister avant qu'ArgoCD ait fini de converger et avant tout test fonctionnel. La validation dev rend un snapshot acceptable comme candidat à une promotion ; elle ne crée pas le tag.

Pour un commit issu d'une PR mergée, le workflow interroge GitHub pour retrouver la PR associée au commit et produit :

```text
dev-vYYYY.MM.<PR>
```

Le numéro de PR ne dépend donc pas du texte du commit squash. Pour un push direct exceptionnel sans PR associée, le workflow utilise un suffixe SHA court :

```text
dev-vYYYY.MM.<short-sha>
```

`dev-latest` est un alias mutable mis à jour atomiquement vers le snapshot dev le plus récent. Les tags `dev-v*` sont les snapshots immuables utilisés comme source des promotions production.

#### Artefacts générés après merge

Certains changements source déclenchent volontairement une seconde PR générée afin de matérialiser un artefact immuable dans GitOps. C'est notamment le cas du TXO Fabric Operator : un changement de source Go mergé sur `main` déclenche `build-txo-fabric-operator.yaml`, qui construit l'image puis ouvre une PR automatique pour pinner `ghcr.io/charchess/txo-fabric-operator:main-<sha>` dans `operator/config/manager/manager.yaml`.

Dans ce cas, le tag dev du changement source est **intermédiaire**. Il ne doit pas être promu en production. Le candidat à la promotion est le `dev-v*` créé après merge de la PR de pin générée, afin que le desired state Git et l'image immuable construite depuis les mêmes sources soient promus ensemble.

### 6. Promotion production

La promotion production passe **uniquement** par GitHub Actions et constitue une décision humaine explicite sur le snapshot choisi :

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

Exécuter cette commande signifie que l'opérateur autorise **ce `dev-v*` précis** pour la production après les validations pertinentes. Un agent automatisé ne doit jamais déduire cette autorisation d'une CI verte, d'un merge, d'un statut Project ou de l'existence du tag ; il peut préparer/recommander la promotion, mais ne l'exécute qu'après instruction explicite de l'utilisateur/opérateur.

Pour un snapshot issu d'un push direct exceptionnel, utiliser sa version SHA telle qu'elle apparaît dans le tag `dev-v*`.

Le workflow de promotion est sérialisé et travaille exclusivement sur le commit exact résolu depuis le tag dev immuable. Il :

1. vérifie que le tag `dev-v*` existe et pointe vers un ancêtre de `main` ;
2. détache le workspace sur ce commit exact ;
3. vérifie les garde-fous de cohérence d'artefacts générés, notamment que le pin d'image TXO Fabric Operator correspond à la dernière source operator du snapshot ;
4. génère le SBOM et les métadonnées de promotion depuis ce commit ;
5. crée ou vérifie le tag immuable `prod-v*` ;
6. publie SBOM et métadonnées dans la GitHub Release ;
7. déplace `prod-stable` atomiquement **en dernier**, puis vérifie le commit cible.

Ne jamais créer ou déplacer `prod-stable` manuellement lors d'une promotion normale.

Le workflow crée :

- `prod-vYYYY.MM.<PR>` : release prod **immuable** ;
- `prod-stable` : alias **mutable** suivi par ArgoCD prod.

Les protections GitHub côté serveur pour les familles de tags immuables sont suivies séparément ; les workflows refusent déjà de réutiliser un tag versionné s'il pointe vers un autre commit.

### 7. `prod-working`

`prod-working` est un marqueur manuel vers un état production connu comme fonctionnel.

- Il n'est **jamais** mis à jour automatiquement.
- Il n'est déplacé qu'avec accord explicite de l'utilisateur/opérateur.
- Avant un chantier de nettoyage ou un changement à risque, il peut être repositionné sur la release prod actuellement validée.
- Les tags `prod-v*` restent la trace historique immuable et permettent de retrouver précisément toute release.
- Son workflow le déplace atomiquement et vérifie ensuite la cible résolue.

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
- `docs/architecture.md` : architecture actuelle (et documents applicatifs actifs liés lorsque le scope l'exige).
- `docs/adr/` : décisions et historique des choix.
- `docs/guides/` : procédures de travail récurrentes.
- `docs/troubleshooting/` : diagnostics et runbooks.
- `docs/applications/` : particularités par application.

Les documents historiques peuvent décrire d'anciens outils ; ils ne remplacent jamais ce workflow canonique.
