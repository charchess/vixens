# Vertical Pod Autoscaler (VPA)

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v4.7.0  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Infrastructure)

### Méthode Automatique (Command Line)
```bash
kubectl get pods -n monitoring -l app.kubernetes.io/name=vpa
# Attendu: Pods recommender, updater, admission-controller en statut Running
```

### Méthode Manuelle
1. Créer un objet `VerticalPodAutoscaler` de test.
2. Vérifier qu'il génère des recommandations.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Metrics Server`
- **Particularités :** Déployé via Helm Chart `vpa` de Fairwinds. Utilisé principalement par Goldilocks pour les recommandations.
