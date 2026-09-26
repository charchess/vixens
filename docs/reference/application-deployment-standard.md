# Application Deployment Standard — SUPERSEDED

**Status:** SUPERSEDED depuis le 2026-09-26

Ce fichier est conservé comme point de compatibilité pour les anciens liens. Il **ne définit plus le standard de déploiement Vixens**.

L'ancienne version de ce document imposait notamment des conventions aujourd'hui retirées ou remplacées : Infisical/`InfisicalSecret`, mutations persistantes via `kubectl apply`, anciens noms de `PriorityClass` et patterns de backup appliqués uniformément. Son contenu historique reste disponible dans l'historique Git.

## Sources actuelles

Utiliser, dans cet ordre :

1. [`WORKFLOW.md`](../../WORKFLOW.md) — workflow GitOps canonique ;
2. [`AGENTS.md`](../../AGENTS.md) — contraintes supplémentaires pour les agents ;
3. workflows GitHub, manifests et policies courants — comportement exécutable ;
4. [`app-golden-standard.md`](app-golden-standard.md) — conventions applicatives actuelles ;
5. [`../procedures/deployment-standard.md`](../procedures/deployment-standard.md) — procédure/structure de déploiement ;
6. [`RESOURCE_STANDARDS.md`](RESOURCE_STANDARDS.md) — ressources et priorités ;
7. [`../guides/secret-management.md`](../guides/secret-management.md) — OpenBao + External Secrets Operator.

Le modèle secrets actif est :

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Kubernetes Secret → workload
```

Toute modification persistante de l'état désiré passe par Git, PR, CI puis ArgoCD.
