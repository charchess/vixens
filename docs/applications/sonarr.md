# Sonarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://sonarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://sonarr.dev.truxonline.com
# Attendu: HTTP 200 (Login ou App)
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier l'accès aux séries TV.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage (`/volume3/Content`, `/volume3/Downloads`)
    - `Prowlarr` (Indexers)
    - `Sabnzbd` / `Transmission` (Download Clients)
- **Particularités :** Gestionnaire de séries TV (PVR).
