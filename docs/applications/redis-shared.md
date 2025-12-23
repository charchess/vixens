# Redis Shared

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 7-alpine|
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Database Service)

### Méthode Automatique (Command Line)
```bash
kubectl get pods -n databases -l app=redis-shared
# Attendu: Pod en statut Running
```

### Méthode Manuelle
1. `kubectl exec -it -n databases deploy/redis-shared -- redis-cli ping`
2. Attendu: `PONG`

## Notes Techniques
- **Namespace :** `databases`
- **Dépendances :** Aucune
- **Particularités :** Instance Redis standalone mutualisée. Utilisée pour le cache et les files de messages (Celery, etc.) des applications.
