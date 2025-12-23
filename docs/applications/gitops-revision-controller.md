# GitOps Revision Controller

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | Custom  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Controller)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est en ligne
kubectl get pods -n tools -l app=gitops-revision-controller
# Attendu: Pod en statut Running
```

### Méthode Manuelle
1. Vérifier les logs du pod: `kubectl logs -n tools -l app=gitops-revision-controller`.
2. Vérifier qu'il réagit aux changements d'applications ArgoCD (logs de type "Application changed...").
3. Vérifier qu'il met à jour les ConfigMaps cibles (ex: config HomeAssistant).

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :** `Kopf` (Framework Python)
- **Particularités :** Contrôleur maison écrit en Python. Synchronise la révision Git déployée (tag/commit) vers d'autres ressources (ex: injecte la version dans la config HomeAssistant).