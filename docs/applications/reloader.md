# Reloader

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.0.118|
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Controller)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est en ligne
kubectl get pods -n tools -l app=reloader
# Attendu: Pod en statut Running
```

### Méthode Manuelle
1. Modifier un ConfigMap annoté avec `reloader.stakater.com/auto: "true"`.
2. Vérifier que les pods utilisant ce ConfigMap redémarrent (Rolling Restart).
3. Vérifier les logs de Reloader : `kubectl logs -n tools -l app=reloader`.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :** Aucune
- **Particularités :** Surveille les ConfigMaps et Secrets pour redémarrer automatiquement les Deployments associés.