# Hydrus Server

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://hydrus.[env].truxonline.com (API/Web)

### Méthode Automatique (Curl)
```bash
curl -I -k https://hydrus.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Vérifier la connexion depuis le client Hydrus.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** PVC Storage
- **Particularités :** Serveur pour Hydrus Network (Image Booru personnel).
