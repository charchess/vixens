# Promtail

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.6.7  |
| Prod          | [x]     | [x]       | [x]   | v3.6.7  |

## Status Elite ✅
- **PriorityClass:** `vixens-critical`
- **QoS:** **Guaranteed** (Requests = Limits)
- **PSA:** `privileged` (requis pour accès logs hôte)

## Validation
**URL :** N/A (Agent)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le DaemonSet déploie bien un pod par noeud
kubectl get daemonset promtail -n monitoring
# Attendu: Desired = Current = Ready = (Nombre de noeuds)
```

### Méthode Manuelle
1. Vérifier que les logs d'un nouveau pod apparaissent dans Grafana (Loki) quelques secondes après démarrage.
2. Vérifier les logs d'un pod Promtail pour s'assurer de la connexion réussie à Loki (`clients.go: Connect ... success`).

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Loki` (Destination des logs)
- **Particularités :** Déployé via DaemonSet. Monte `/var/log` de l'hôte.

## Known Issues

### High Restart Count Due to Probe Timeouts (Resolved)

**Issue:** DaemonSet experiencing 8-97 restarts with liveness/readiness probe failures.

**Root Cause:**
- DaemonSets run on every node, including control-plane nodes under load
- Original probe timeout (1s) too strict for distributed agents
- Control-plane nodes running ArgoCD (89 apps) + system pods can have response delays

**Resolution (PR #1980):**
- Increased livenessProbe timeout: 1s → 5s
- Increased readinessProbe timeout: 1s → 5s
- Maintains fast failure detection:
  - initialDelaySeconds: 10
  - periodSeconds: 10
  - failureThreshold: 3
  - **Total detection time:** 30s (3 failures * 10s period)

**Rationale:** Industry standard for DaemonSets:
- 5s timeout is common for distributed log agents (Fluent Bit, Filebeat, Promtail)
- 1s timeout appropriate for low-latency services (APIs), not for background agents
- DaemonSets on control-plane nodes need tolerance for scheduler contention

**Status:** ✅ Fixed in production (PR #1980)

**Related:** Part of production cluster health check (2026-03-10)
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
