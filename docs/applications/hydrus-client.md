# Hydrus Client (Web)

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://hydrus-web.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://hydrus-web.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** `Hydrus Server`
- **Particularités :** Interface Web pour Hydrus.
