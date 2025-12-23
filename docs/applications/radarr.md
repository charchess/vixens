# Radarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://radarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://radarr.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier l'accès aux films.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage
    - `Prowlarr`
    - Download Clients
- **Particularités :** Gestionnaire de films.
