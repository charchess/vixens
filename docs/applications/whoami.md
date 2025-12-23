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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://whoami.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://whoami.dev.truxonline.com
# Attendu: HTTP 200, contenu brut texte (Headers, IP)
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les informations de la requête (Headers, IP) s'affichent correctement.
3. Vérifier que le certificat SSL est valide (cadenas).

## Notes Techniques
- **Namespace :** `whoami`
- **Dépendances :** Aucune
- **Particularités :** Application de test légère pour valider l'Ingress, les certificats et le routage.