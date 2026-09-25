# Multi-agent orchestration

Vixens n'impose pas d'orchestrateur d'agent particulier.

Les humains, Claude, Gemini, ChatGPT, OpenCode ou tout autre agent utilisent le **même contrat de dépôt** : [WORKFLOW.md](../../WORKFLOW.md) et [AGENTS.md](../../AGENTS.md).

## État partagé

La coordination passe par des objets visibles et auditables :

- GitHub Issues pour le backlog ;
- branches et commits pour le travail ;
- Draft/normal Pull Requests pour l'avancement ;
- checks GitHub pour la validation ;
- `main` pour l'état désiré dev ;
- tags de release pour les snapshots ;
- ArgoCD pour l'état de réconciliation.

Aucune mémoire locale d'un agent ne doit être nécessaire pour comprendre l'état du travail.

## Règle de concurrence

Avant d'agir, un agent doit :

1. relire le `main` courant ;
2. lire l'issue ciblée ;
3. vérifier les PR ouvertes sur le même scope ;
4. repartir du commit courant ;
5. limiter sa PR au scope annoncé.

Cela permet à plusieurs agents de travailler sans dépendre d'un serveur MCP, d'un tracker local ou d'une mémoire partagée propriétaire.

## Outils

Chaque agent choisit les outils disponibles dans son environnement pour lire, rechercher, éditer et tester. Ces outils ne sont pas des dépendances du projet et ne doivent pas être cités comme prérequis dans les procédures canoniques.
