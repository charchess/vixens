# Redis Shared

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 7-alpine|
| Prod          | [x]     | [x]       | [x]   | 7-alpine|

## Validation
**URL :** N/A (Database Service)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est en ligne
kubectl get pods -n databases -l app=redis-shared
# Attendu: Pod en statut Running

# Test de connexion simple (PING)
kubectl exec -it -n databases deploy/redis-shared -- redis-cli ping
# Attendu: PONG
```

### Méthode Manuelle
1. Vérifier que les applications dépendantes (Authentik, Netbox) ne remontent pas d'erreur de connexion Redis.

## Notes Techniques
- **Namespace :** `databases`
- **Dépendances :** Aucune
- **Particularités :** Instance Redis standalone mutualisée. Authentification activée via `REDIS_PASSWORD` (Infisical). Utilisée pour le cache et les files de messages (Celery, etc.) des applications. Priorité `vixens-critical` en production.