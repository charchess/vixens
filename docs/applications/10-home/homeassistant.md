# Home Assistant

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 2026.1.3 |
| Prod          | [x]     | [x]       | [x]   | 2026.1.3 |

## Validation
**URL :** https://homeassistant.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://homeassistant.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://homeassistant.dev.truxonline.com | grep "Home Assistant"
# Attendu: Présence de "Home Assistant"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la page de login s'affiche ("Home Assistant").
3. Vérifier que la connexion WebSocket ne retourne pas d'erreur (pas de bandeau "Connection lost").

## Notes Techniques
- **Namespace :** `homeassistant`
- **Dépendances :**
    - `Infisical` (Secret `homeassistant-config`)
    - `Reloader` (Redémarrage auto sur modif config)
    - `PostgreSQL` (Cluster partagé, via `homeassistant-postgresql-credentials`)
    - `Traefik` (Ingress)
- **Particularités :** Utilise `hostNetwork: true` pour la découverte mDNS. Configuration montée via `subPath`.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
