# Promtail

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.0.0  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Agent)

### Méthode Automatique (Command Line)
```bash
kubectl get daemonset promtail -n monitoring
# Attendu: Desired = Current = Ready
```

### Méthode Manuelle
1. Vérifier que les logs d'un nouveau pod apparaissent dans Grafana (Loki) quelques secondes après démarrage.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Loki` (Destination des logs)
- **Particularités :** Déployé via DaemonSet. Monte `/var/log` et `/var/lib/docker/containers` (ou équivalent containerd) de l'hôte.
