# AdGuard Home

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://adguard.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://adguard.dev.truxonline.com
# Attendu: HTTP 200 (Login Page)
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface s'affiche.
3. Tester la résolution DNS via l'IP du LoadBalancer (UDP 53).

## Notes Techniques
- **Namespace :** `networking`
- **Dépendances :** Aucune
- **Particularités :** Serveur DNS bloqueur de publicités. Expose les ports DNS (53 UDP/TCP) via IngressRouteUDP/TCP sur le LoadBalancer.
