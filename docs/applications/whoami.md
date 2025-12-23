# Whoami

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | traefik/whoami |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://whoami.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://whoami.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les informations de la requête (Headers, IP) s'affichent.

## Notes Techniques
- **Namespace :** `whoami`
- **Dépendances :** Aucune
- **Particularités :** Application de test légère pour valider l'Ingress, les certificats et le routage.
