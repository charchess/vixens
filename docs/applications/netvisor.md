# Netvisor

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://netvisor.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://netvisor.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les métriques réseau s'affichent.

## Notes Techniques
- **Namespace :** `networking`
- **Dépendances :** DaemonSet sur tous les noeuds.
- **Particularités :** Outil de visualisation réseau interne.
