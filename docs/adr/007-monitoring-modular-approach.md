# ADR 007: Adopter une approche modulaire pour le monitoring (Prometheus/Grafana)

**Date:** 2025-12-04
**Statut:** Accepté
**Décideurs:** Claude Code, User

## Contexte

Lors du déploiement de la stack de monitoring pour le cluster Kubernetes, nous avons initialement tenté d'utiliser `kube-prometheus-stack`, un meta-chart Helm populaire qui bundle:
- Prometheus Operator
- Prometheus
- Alertmanager
- Grafana
- kube-state-metrics
- node-exporter

Cette approche a rencontré plusieurs problèmes bloquants:

1. **Complexité de configuration nested**: Configuration imbriquée difficile à gérer (ex: `kubeStateMetrics.kube-state-metrics.tolerations`)
2. **Incompatibilité ServerSideApply**: Les applications ArgoCD multi-sources avec Helm ne fonctionnent pas correctement avec `ServerSideApply=true`
3. **Debugging complexe**: Impossible de diagnostiquer rapidement quel sous-composant pose problème
4. **Ressources fantômes**: ArgoCD voyait les ressources comme "OutOfSync" mais ne les créait pas réellement dans le cluster
5. **Grafana deployment manquant**: Le deployment Grafana n'était jamais créé malgré les syncs répétés

Après 9 commits de debug et plusieurs heures de troubleshooting sans succès, nous avons décidé de reconsidérer l'approche.

## Décision

Nous adoptons une **approche modulaire** pour le monitoring, avec des déploiements séparés pour chaque composant:

### Architecture retenue

```
apps/02-monitoring/
├── prometheus/          # Prometheus standalone (chart: prometheus-community/prometheus)
│   ├── base/
│   │   ├── namespace.yaml
│   │   ├── values.yaml
│   │   └── kustomization.yaml
│   └── overlays/
│       ├── dev/
│       ├── test/
│       ├── staging/
│       └── prod/
│
├── grafana/             # Grafana standalone (futur - chart: grafana/grafana)
│   └── ...
│
└── loki/                # Loki standalone (futur - chart: grafana/loki)
    └── ...
```

### Composants déployés séparément

1. **Prometheus** (chart `prometheus/prometheus` v25.30.1)
   - Inclut: server, alertmanager, node-exporter
   - Persistence: PVC 50Gi (server), 10Gi (alertmanager)
   - Ingress: Traefik IngressRoute

2. **Grafana** (à venir - chart `grafana/grafana`)
   - Déploiement indépendant
   - Configuration datasource Prometheus via ConfigMap
   - Secrets admin via Infisical

3. **kube-state-metrics** (optionnel)
   - Peut être ajouté si nécessaire
   - Déploiement standalone simple

### Configuration simplifiée

**Avant (kube-prometheus-stack):**
```yaml
kubeStateMetrics:
  enabled: true
  kube-state-metrics:  # Nested subchart
    tolerations:
      - key: node-role.kubernetes.io/control-plane
```

**Après (Prometheus standalone):**
```yaml
nodeExporter:
  enabled: true
  tolerations:
    - key: node-role.kubernetes.io/control-plane
      operator: Exists
      effect: NoSchedule
```

### ArgoCD Application

**Sans ServerSideApply** (qui causait des problèmes):
```yaml
syncPolicy:
  automated:
    prune: true
    selfHeal: true
  syncOptions:
    - CreateNamespace=true
    # ServerSideApply retiré
```

## Conséquences

### Avantages ✅

1. **Debugging simple**: Un pod par service = problème isolé facile à identifier
2. **Configuration claire**: Pas de nested subchart hell
3. **Flexibilité**: Versions indépendantes par composant
4. **GitOps-friendly**: Chaque composant a son cycle de vie propre
5. **Réduction complexité**: Moins de dépendances implicites
6. **Compatibilité ArgoCD**: Pas de problèmes ServerSideApply/multi-source

### Inconvénients ❌

1. **Plus de fichiers**: Nécessite plus de structure de répertoires
2. **Intégration manuelle**: Connexion Grafana → Prometheus à configurer manuellement
3. **Pas de Prometheus Operator**: Perte de CRDs ServiceMonitor/PodMonitor (acceptable pour notre use case)
4. **Maintenance**: Mise à jour de plusieurs charts vs un seul

### Risques et mitigations

| Risque | Mitigation |
|--------|-----------|
| Configuration divergente entre envs | DRY via base/ + overlays Kustomize |
| Oubli de mise à jour d'un composant | Monitoring Renovate/Dependabot |
| Complexité réseau entre services | Utilisation de Service DNS Kubernetes standards |

## Alternatives considérées

### Option 1: Continuer avec kube-prometheus-stack
- **Rejeté**: Après 9 commits de debug sans succès
- **Raison**: Complexité excessive pour notre besoin simple

### Option 2: Prometheus Operator standalone
- **Considéré**: Apporte CRDs ServiceMonitor
- **Rejeté**: Complexité intermédiaire non nécessaire pour 3 control-planes

### Option 3: VictoriaMetrics
- **Considéré**: Alternative Prometheus-compatible plus légère
- **Rejeté**: Préférence pour l'écosystème standard Prometheus

## Références

- [Prometheus Helm Chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus)
- [Grafana Helm Chart](https://github.com/grafana/helm-charts/tree/main/charts/grafana)
- [ArgoCD ServerSideApply Known Issues](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-options/#server-side-apply)
- Commits du debugging kube-prometheus-stack: `60121ba` → `3f0a018` (9 commits)

## Notes d'implémentation

### Phase 1 (Complété - 2025-12-04)
- ✅ Suppression kube-prometheus-stack
- ✅ Déploiement Prometheus standalone
- ✅ Configuration control-plane tolerations
- ✅ Persistent storage (Synology CSI)
- ✅ node-exporter: 3/3 Running
- ✅ Alertmanager: 1/1 Running
- ⏳ Prometheus-server: Bloqué montage iSCSI (problème NAS, pas architecture)

### Phase 2 (À venir)
- Grafana standalone deployment
- Connexion Grafana → Prometheus datasource
- Dashboards Grafana (Kubernetes, Node, Prometheus)
- Ingress TLS pour Grafana

### Phase 3 (Optionnel)
- Loki pour les logs
- kube-state-metrics si métriques Kubernetes détaillées nécessaires
- AlertManager rules configuration
