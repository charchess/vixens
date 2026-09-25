# Claude Code adapter

Claude Code doit suivre les mêmes règles que tout autre contributeur.

Lire d'abord :

1. [WORKFLOW.md](WORKFLOW.md)
2. [AGENTS.md](AGENTS.md)
3. l'issue GitHub concernée
4. la documentation/les ADR du scope

Les outils disponibles dans Claude Code (édition de fichiers, recherche, navigateur, MCP éventuels, shell, etc.) sont des commodités. Ils ne modifient ni le workflow GitOps ni les règles de contribution.

Points essentiels :

- GitHub Issues est le backlog canonique.
- Toujours repartir du `main` GitHub courant et vérifier les PR concurrentes.
- Toute modification passe par branche + PR + CI.
- Dev suit `main` via ArgoCD.
- La promotion prod passe uniquement par `promote-prod.yaml`.
- Ne pas effectuer de mutation Kubernetes persistante hors GitOps.
- Ne pas dupliquer les règles de `WORKFLOW.md` ici.
