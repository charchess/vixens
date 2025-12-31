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
# Vérifier que les composants VPA sont en ligne
kubectl get pods -n monitoring -l app.kubernetes.io/name=vpa
# Attendu: Pods recommender, updater, admission-controller en statut Running
```

### Méthode Manuelle
1. Créer un objet `VerticalPodAutoscaler` de test pour un Deployment existant.
2. Attendre quelques minutes et vérifier qu'il génère des recommandations : `kubectl get vpa <nom> -o yaml`.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Metrics Server`
- **Particularités :** Déployé via Helm Chart `vpa` de Fairwinds. Utilisé principalement par Goldilocks pour les recommandations.