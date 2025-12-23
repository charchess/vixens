# Prowlarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://prowlarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://prowlarr.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier le statut des indexeurs.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - `Gluetun` (Proxy pour contourner les blocages)
- **Particularités :** Gestionnaire d'indexeurs (Torrent/Usenet). Synchronise les indexeurs vers Sonarr/Radarr/etc.
