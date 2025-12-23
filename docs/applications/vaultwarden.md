# Vaultwarden

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://vaultwarden.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://vaultwarden.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Bitwarden s'affiche.

## Notes Techniques
- **Namespace :** `services` (A vérifier)
- **Dépendances :**
    - `Infisical` (Admin Token)
- **Particularités :** Serveur Bitwarden léger (Rust). Utilise SQLite (sur PVC) par défaut (ou Postgres si configuré).
