# Traefik

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v25.0.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Ingress Controller)

### Méthode Automatique (Command Line)
```bash
# Vérifier que les pods Traefik sont en ligne
kubectl get pods -n traefik
# Attendu: Pods traefik en statut Running

# Vérifier l'IP externe du LoadBalancer
kubectl get svc traefik -n traefik
# Attendu: EXTERNAL-IP assignée (ex: 192.168.111.x)
```

### Méthode Manuelle
1. Vérifier qu'une application exposée via Ingress est accessible (ex: whoami).
2. Vérifier les logs pour s'assurer qu'il n'y a pas d'erreurs de certificats ou de configuration.

## Notes Techniques
- **Namespace :** `traefik`
- **Dépendances :**
    - `Cilium` (LoadBalancer)
- **Particularités :** Ingress Controller principal du cluster. Gère le routage HTTP/HTTPS.