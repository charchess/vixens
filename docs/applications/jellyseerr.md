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
curl -I -k https://jellyseerr.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Faire une demande de média.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - `Jellyfin`
    - `Sonarr` / `Radarr`
- **Particularités :** Gestionnaire de demandes de contenu (Overseerr fork pour Jellyfin).
