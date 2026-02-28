# Music Assistant

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://music-assistant.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://music-assistant.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://music-assistant.dev.truxonline.com | grep "Music Assistant"
# Attendu: Présence de "Music Assistant"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge et que les lecteurs audio sont détectés.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** `Home Assistant` (Optionnel mais recommandé)
- **Particularités :** Agrégateur de sources musicales.
- **Sizing :** `medium` (1Gi RAM limit) - Nécessaire pour éviter les OOMKilled.
- **Ports supplémentaires :** 
    - **3483 (TCP/UDP) :** Redirigé via Traefik pour le protocole SlimProto (SlimServer).
    - **8097 (TCP) :** Stream Server (interne/externe).
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
