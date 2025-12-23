# Cert-Manager

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.14.4 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Infrastructure)

### Méthode Automatique (Command Line)
```bash
kubectl get pods -n cert-manager
# Attendu: Pods cert-manager, cainjector, webhook en statut Running
```

### Méthode Manuelle
1. Vérifier la création d'un certificat pour une Ingress.
2. `kubectl get certificaterequest -A`

## Notes Techniques
- **Namespace :** `cert-manager`
- **Dépendances :**
    - `Infisical` (Secrets pour DNS Challenge via `cert-manager-webhook-gandi`)
- **Particularités :** Déployé via Helm Chart. Utilise `cert-manager-webhook-gandi` pour la validation DNS.
