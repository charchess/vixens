# Hydrus Client (Web)

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.0.1  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | v2.0.1  |

## Validation
**URL :** https://hydrus.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://hydrus.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://hydrus.dev.truxonline.com
# Attendu: Page noVNC chargée (HTTP 200)
```

## Notes Techniques
- **Namespace :** `media`
- **Standard Elite** : QoS Guaranteed (Requests=Limits), Intégrité sqlite3, Litestream HA, Métriques Prometheus.
- **Resources** : 1 CPU / 2Gi RAM (Guaranteed).
- **Storage** : PVC iSCSI de **40Gi** (augmenté pour gérer les volumineux sets de WAL Litestream).
- **Optimisation** : Restauration de la base `caches` désactivée en init-container pour accélérer le boot.
- **PriorityClass** : `vixens-medium`.
- **Authentication** : Intégration SSO via Authentik ForwardAuth (`authentik-forward-auth`).
- **Health Probes** : Liveness (120s delay) et Readiness (30s delay) configurées.

---
> ⚠️ **HIBERNATION DEV**
> Cette application est activée par défaut en `dev` pour les tests actuels.