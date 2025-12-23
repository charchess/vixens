# Homepage

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://homepage.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://homepage.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que le dashboard d'accueil s'affiche avec les icônes des services.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `Infisical` (API Keys pour widgets services externes comme HomeAssistant)
- **Particularités :** Dashboard statique/dynamique configuré via ConfigMap (`settings.yaml`, `services.yaml`, etc.). Utilise `initContainer` pour copier la config initiale.
