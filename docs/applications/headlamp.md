# Headlamp

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://headlamp.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://headlamp.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Headlamp s'affiche (Titre: "Headlamp").

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `ServiceAccount` (Permissions cluster-view)
- **Particularités :** Authentification OIDC désactivée temporairement. Accès direct.
