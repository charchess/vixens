# CloudNativePG Operator

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.27.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Operator)

### Méthode Automatique (Command Line)
```bash
kubectl get pods -n cnpg-system
# Attendu: Pod operator en statut Running
```

### Méthode Manuelle
1. Vérifier les logs de l'opérateur.
2. Vérifier que les CRDs `Cluster` sont disponibles.

## Notes Techniques
- **Namespace :** `cnpg-system`
- **Dépendances :** Aucune
- **Particularités :** Opérateur Kubernetes pour gérer les clusters PostgreSQL. Déployé via Helm Chart.
