# Kubernetes Dashboard

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 7.14.0  |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://dashboard.dev.truxonline.com/

### Méthode Automatique (Command Line)
```bash
# Vérifier que tous les composants sont en ligne
kubectl get pods -n kubernetes-dashboard
# Attendu: 5 composants (api, auth, web, metrics-scraper, kong) en Running

# Vérifier l'URL via curl
curl -I -k https://dashboard.dev.truxonline.com/
# Attendu: HTTP 200 ou 302 (redirect vers /#login)
```

### Méthode Manuelle
1. Accéder à l'URL via un navigateur.
2. Générer un token d'accès : `kubectl -n kubernetes-dashboard create token admin-user`.
3. Se connecter et vérifier que les ressources du cluster sont visibles.

## Notes Techniques
- **Namespace :** `kubernetes-dashboard`
- **Chart Helm :** `kubernetes-dashboard/kubernetes-dashboard` (v7.x)
- **Architecture :** Utilise Kong comme passerelle interne.
- **Tolerations :** Nécessite des tolerations control-plane (spécifiques à chaque sous-chart).
- **Ingress :** Traefik pointe sur le service Kong proxy (port 80).
- **Admin User :** Un `ServiceAccount` nommé `admin-user` est créé pour l'accès complet.
