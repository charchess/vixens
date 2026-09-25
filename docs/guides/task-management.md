# Task management with GitHub Issues

GitHub Issues est le système canonique de suivi des tâches pour Vixens.

Le backlog, l'état d'avancement et les échanges liés à une tâche doivent être visibles dans GitHub ; aucun outil local ne constitue une source de vérité parallèle.

## Créer une issue

Utiliser un titre court de type Conventional Commit :

```text
<type>(<scope>): <description>
```

Exemples :

```text
feat(homeassistant): expose application metrics
fix(grafana): allow Loki datasource egress
chore(repo): remove legacy workflow tooling
docs(openbao): document secret rotation
security(traefik): protect dashboard access
```

Commande :

```bash
gh issue create --title "chore(repo): description" --label "priority:p2"
```

## Description recommandée

```markdown
## Contexte
Pourquoi cette tâche existe.

## État actuel
Ce qui existe ou ce qui ne fonctionne pas.

## État cible
Le résultat attendu.

## Acceptance
- [ ] critère vérifiable 1
- [ ] critère vérifiable 2

## Dépendances / risques
Liens vers issues, PR, ADR ou dépendances externes.
```

Une issue doit être compréhensible sans dépendre d'une conversation privée ou d'un état local.

## Priorités

Vixens utilise les labels GitHub de priorité :

- `priority:p0` — incident critique / blocage majeur ;
- `priority:p1` — important ;
- `priority:p2` — normal ;
- `priority:p3` — faible / opportuniste.

Utiliser les labels existants du dépôt plutôt que créer des variantes synonymes.

## Cycle de vie

Le cycle normal est :

```text
Issue ouverte
   ↓
branche dédiée + PR
   ↓
CI verte
   ↓
merge main
   ↓
validation dev
   ↓
promotion prod si nécessaire
   ↓
validation prod
   ↓
Issue fermée
```

Une PR peut utiliser `Closes #123` lorsqu'elle clôt réellement l'intégralité de l'issue après merge. Pour les changements qui nécessitent une validation dev/prod après merge, préférer garder l'issue ouverte jusqu'à cette validation.

## Travailler sur une issue

Toujours relire l'état GitHub courant avant de commencer :

```bash
gh issue view <number>
gh pr list --state open
git fetch origin
git switch main
git pull --ff-only
git switch -c fix/<issue>-<slug>
```

Le scope de la branche et de la PR doit correspondre à l'issue. Éviter les nettoyages opportunistes sans rapport ; créer une issue séparée lorsque le nouveau travail mérite son propre cycle de validation.

## Dépendances

GitHub Issues n'impose pas un graphe de dépendances propriétaire. Utiliser des liens explicites dans la description ou les commentaires :

```text
Blocked by #123
Depends on #456
Follow-up: #789
```

Cela garde les relations lisibles aussi bien pour un humain que pour un agent.

## Recherche et triage

```bash
# backlog
gh issue list --state open

# priorités
gh issue list --state open --label "priority:p1"

# détail
gh issue view <number>

# recherche textuelle
gh issue list --search "openbao"
```

## Fermer une issue

Fermer après satisfaction des critères d'acceptation et des validations nécessaires :

```bash
gh issue close <number> --reason completed
```

Si le travail n'est plus souhaité ou est remplacé, utiliser une raison et un commentaire explicites plutôt que prétendre qu'il a été réalisé.

## Bonnes pratiques

- Une issue décrit un résultat vérifiable, pas une liste vague d'intentions.
- Les critères d'acceptation sont testables.
- Les changements à risque sont découpés en petites PR indépendantes.
- Les décisions d'architecture durables vont dans un ADR ; l'issue peut y faire référence.
- Les incidents et procédures de récupération vont dans les runbooks.
- Les informations de tâche ne doivent pas être dupliquées dans une base locale spécifique à un outil.
- Les agents doivent consulter les issues et PR actuelles avant d'agir afin d'éviter les collisions.

## Références

- [WORKFLOW.md](../../WORKFLOW.md) — workflow GitOps canonique.
- [AGENTS.md](../../AGENTS.md) — contraintes complémentaires pour les agents.
- [GitOps Workflow](gitops-workflow.md) — détails GitOps historiques/complémentaires ; `WORKFLOW.md` fait autorité en cas de divergence.
