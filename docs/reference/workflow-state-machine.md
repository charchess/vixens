# Workflow state machine — historical reference

> **Status: Superseded**
>
> L'ancienne state machine locale en phases 0–6 était orchestrée par `justfile` et des outils désormais retirés du workflow canonique.

Le workflow actuel est défini dans [WORKFLOW.md](../../WORKFLOW.md) et repose sur des états observables dans GitHub/Git/ArgoCD :

```text
GitHub Issue
  ↓
branche + Pull Request
  ↓
CI
  ↓
main
  ↓
ArgoCD dev + validation
  ↓
promotion GitHub Actions
  ↓
ArgoCD prod + validation
```

Cette approche évite de maintenir une deuxième state machine locale susceptible de diverger de l'état réel du dépôt.

Pour les règles agents, voir [AGENTS.md](../../AGENTS.md).
