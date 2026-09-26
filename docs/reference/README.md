# Reference documentation

Références techniques et spécifications du projet Vixens.

## Chaîne d'autorité

Lorsqu'une documentation se contredit, utiliser cet ordre :

1. [`WORKFLOW.md`](../../WORKFLOW.md) — workflow GitOps canonique ;
2. [`AGENTS.md`](../../AGENTS.md) — contraintes supplémentaires pour les agents ;
3. workflows GitHub, manifests et policies courants — comportement exécutable ;
4. architecture, ADR et références actives ;
5. documents explicitement `SUPERSEDED`, rapports et archives — provenance uniquement.

Une ancienne référence conservée pour l'historique ne redevient jamais normative par simple existence dans Git.

## Architecture et standards

- [Vixens Master Architecture](../architecture.md)
- [Application Standard](app-golden-standard.md) — conventions applicatives actuelles ;
- [Quality Standards](quality-standards.md)
- [Resource Standards](RESOURCE_STANDARDS.md)

## Workflow et collaboration

- [WORKFLOW.md](../../WORKFLOW.md) — workflow canonique ;
- [AGENTS.md](../../AGENTS.md) — règles multi-agent ;
- [Task Management](../guides/task-management.md) — conventions actuelles GitHub Issues ;
- [Dependency Management](dependency-management.md) — responsabilité Renovate et politique d'auto-merge ;
- [Multi-Agent Orchestration](multi-agent-orchestration.md) — principes de coordination agnostiques des outils ;
- [Workflow State Machine](workflow-state-machine.md) — référence historique de l'ancien workflow local.

L'ancienne proposition de formalisme Archon du 2025-12-30 est archivée sous `docs/archive/2025-12-30-archon-task-formalism.md`. Elle ne définit plus le workflow courant.

## Kubernetes / GitOps

- [ArgoCD Sync Waves](argocd-sync-waves.md)
- [Deployment Standard](../procedures/deployment-standard.md) — procédure et structure courantes ;
- [Guaranteed QoS Sizing](guaranteed-qos-sizing.md) — référence spécialisée ;
- [Sync Waves Implementation Plan](sync-waves-implementation-plan.md) — historique du déploiement des sync waves.

## Références superseded

Ces fichiers restent accessibles pour préserver les anciens liens et la provenance, mais ne doivent pas être suivis comme instructions actives :

- [Application Deployment Standard](application-deployment-standard.md) — **SUPERSEDED** ;
- [Application Scoring Model](APPLICATION_SCORING_MODEL.md) — **SUPERSEDED**.

## Principe

Une **reference** active décrit un contrat ou un mécanisme. Un **guide/procedure** explique une manière de travailler. Les documents explicitement marqués `SUPERSEDED` ou les archives peuvent décrire d'anciens outils et anciennes politiques, mais ne définissent plus le comportement courant.
