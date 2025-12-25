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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://homepage.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://homepage.dev.truxonline.com | grep "Homepage"
# Attendu: Contenu de la page d'accueil
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que le dashboard d'accueil s'affiche avec les icônes des services.
3. Vérifier que les widgets dynamiques (ex: météo, ressources) affichent des données et non des erreurs API.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `Infisical` (API Keys pour widgets services externes comme HomeAssistant)
- **Particularités :** Dashboard statique/dynamique configuré via ConfigMap (`settings.yaml`, `services.yaml`, etc.). Utilise `initContainer` pour copier la config initiale.