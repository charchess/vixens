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
kubectl get pods -n traefik
# Attendu: Pods traefik en statut Running
```

### Méthode Manuelle
1. Vérifier qu'une application exposée via Ingress est accessible.
2. Vérifier l'IP externe du service LoadBalancer traefik.

## Notes Techniques
- **Namespace :** `traefik`
- **Dépendances :**
    - `Cilium` (LoadBalancer)
- **Particularités :** Ingress Controller principal du cluster. Gère le routage HTTP/HTTPS.
