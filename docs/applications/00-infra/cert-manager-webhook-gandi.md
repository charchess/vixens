# Cert-Manager Webhook Gandi

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.5.2  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Infrastructure)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod webhook est en ligne
kubectl get pods -n cert-manager -l app.kubernetes.io/name=cert-manager-webhook-gandi
# Attendu: Pod en statut Running
```

### Méthode Manuelle
1. Vérifier les logs du pod webhook lors d'une demande de certificat : `kubectl logs -n cert-manager -l app.kubernetes.io/name=cert-manager-webhook-gandi`.
2. S'assurer qu'il n'y a pas d'erreur d'authentification API Gandi (403 Forbidden).

## Notes Techniques
- **Namespace :** `cert-manager`
- **Dépendances :**
    - `cert-manager`
    - `Infisical` (Secret `gandi-credentials`)
- **Particularités :** Permet la validation DNS pour les domaines gérés par Gandi. Le secret API Key est synchronisé depuis Infisical.