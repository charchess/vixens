# Home Assistant

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 2026.1.3 |
| Prod          | [x]     | [x]       | [ ]   | 2026.9.2 |

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
- **Restauration DataAngel :** `python_packages/**` est exclu du restore S3 en production. Les dépendances Python sont reconstruites localement par `install-python-deps`, afin qu’un objet S3 obsolète ne bloque pas le démarrage.
- **Init Python :** `install-python-deps` réserve 256 MiB et est plafonné à 512 MiB, car NumPy/SciPy/Shapely dépassent le budget VPA générique pendant l’installation.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
