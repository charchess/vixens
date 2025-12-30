# Jellyseerr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://jellyseerr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://jellyseerr.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://jellyseerr.dev.truxonline.com | grep "Jellyseerr"
# Attendu: Présence de "Jellyseerr"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Faire une demande de média et vérifier qu'elle est transmise à Sonarr/Radarr.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - `Jellyfin`
    - `Sonarr` / `Radarr`
- **Particularités :** Gestionnaire de demandes de contenu (Overseerr fork pour Jellyfin).