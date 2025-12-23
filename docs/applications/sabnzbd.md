# Sabnzbd

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://sabnzbd.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://sabnzbd.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier la file d'attente de téléchargement.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** NFS Storage (`/volume3/Downloads`)
- **Particularités :** Client de téléchargement Usenet.
