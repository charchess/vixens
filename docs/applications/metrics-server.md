# Metrics Server

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Infrastructure)

### Méthode Automatique (Command Line)
```bash
kubectl top nodes
# Attendu: Affichage de la consommation CPU/RAM des noeuds
```

### Méthode Manuelle
1. Exécuter `kubectl top pods -A`.

## Notes Techniques
- **Namespace :** `kube-system`
- **Dépendances :** Aucune
- **Particularités :** Installé via manifeste officiel (upstream). Patché pour `kubelet-insecure-tls` (certificats auto-signés Talos) et tolérance control-plane.
