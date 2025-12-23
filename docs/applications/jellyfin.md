# Jellyfin

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://jellyfin.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://jellyfin.dev.truxonline.com/web/index.html
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Lire une vidéo.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage (`/volume3/Content`)
    - GPU (Intel QuickSync) via Device Plugin (si configuré)
- **Particularités :** Serveur de streaming média.
