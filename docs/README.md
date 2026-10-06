# Vixens Documentation

Cet espace documente l'architecture, les procédures et les décisions du dépôt Vixens.

## Commencer ici

1. [WORKFLOW.md](../WORKFLOW.md) — workflow GitOps canonique.
2. [AGENTS.md](../AGENTS.md) — règles complémentaires pour assistants et agents.
3. [guides/task-management.md](guides/task-management.md) — backlog GitHub Issues.
4. [guides/adding-new-application.md](guides/adding-new-application.md) — ajout d'une application.

## Guides

- [Adding a New Application](guides/adding-new-application.md)
- [GitOps Workflow](guides/gitops-workflow.md)
- [Task Management](guides/task-management.md)
- [Secret Management](guides/secret-management.md)
- [Terraform Workflow](guides/terraform-workflow.md)

> La procédure de promotion production canonique est définie dans `WORKFLOW.md` et exécutée par `.github/workflows/promote-prod.yaml`.

## Références

- [Architecture](architecture.md)
- [Application Status](STATUS.md)
- [ADR](adr/)
- [Applications](applications/)
- [Troubleshooting](troubleshooting/)
- [Procedures](procedures/)
- [Reports](reports/)
- [Audits](audits/)

## Où mettre l'information

| Information | Emplacement canonique |
|---|---|
| Workflow de contribution / promotion | `WORKFLOW.md` |
| Contraintes agents / reprise à froid | `AGENTS.md` |
| Besoin et critères d'acceptation | GitHub Issues |
| Planification (`Status`, `Priority`, `Target`) | GitHub Project `vixens roadmap` |
| Architecture actuelle | `docs/architecture.md` et documentation applicative active |
| Décisions et historique | `docs/adr/` |
| Procédures récurrentes | `docs/guides/` / `docs/procedures/` |
| Incidents et runbooks | `docs/troubleshooting/` |
| Particularités d'une application | `docs/applications/` |
| État du backlog | GitHub Issues |
| État désiré Kubernetes | manifests Git sur `main` |
| État runtime observé | cluster, pour diagnostic/validation uniquement |

## Principes documentaires

1. **Une source de vérité par sujet** : les autres documents font un lien au lieu de recopier.
2. **Le comportement exécutable courant prime sur les vieux guides** : lorsqu'un document actif contredit les workflows/manifests courants, il doit être corrigé ou marqué historique ; une divergence live du cluster ne devient pas pour autant le desired state.
3. **Les ADR restent historiques** : une décision remplacée est marquée `Superseded`, pas réécrite comme si elle n'avait jamais existé.
4. **Les outils retirés ne doivent plus apparaître dans les guides actifs**.
5. **Toute modification d'un contrat opérationnel inclut sa mise à jour documentaire dans la même PR**.

Les anciens rapports peuvent contenir des références à des outils désormais retirés ; ils ne constituent pas des instructions opérationnelles.
