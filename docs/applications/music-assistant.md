# Music Assistant

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://music-assistant.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://music-assistant.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** `Home Assistant` (Optionnel mais recommandé)
- **Particularités :** Agrégateur de sources musicales.
