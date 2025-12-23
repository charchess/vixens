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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://netvisor.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://netvisor.dev.truxonline.com | grep "Netvisor"
# Attendu: Présence de "Netvisor"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les métriques réseau s'affichent et se mettent à jour.

## Notes Techniques
- **Namespace :** `networking`
- **Dépendances :** DaemonSet sur tous les noeuds.
- **Particularités :** Outil de visualisation réseau interne.