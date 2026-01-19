# Cert-Manager

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.14.4 |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Infrastructure)

### Méthode Automatique (Command Line)
```bash
# Vérifier que les composants Cert-Manager sont en ligne
kubectl get pods -n cert-manager
# Attendu: Pods cert-manager, cainjector, webhook en statut Running
```

### Méthode Manuelle
1. Créer une Ingress avec `cert-manager.io/cluster-issuer: letsencrypt-staging`.
2. Vérifier la création de la ressource `CertificateRequest`: `kubectl get certificaterequest -A`.
3. Vérifier que le challenge DNS (via webhook Gandi) réussit et que le secret TLS est créé.

## Notes Techniques
- **Namespace :** `cert-manager`
- **Dépendances :**
    - `Infisical` (Secrets pour DNS Challenge via `cert-manager-webhook-gandi`)
- **Particularités :** Déployé via Helm Chart. Utilise `cert-manager-webhook-gandi` pour la validation DNS.