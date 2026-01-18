# NFS Storage Namespace

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Namespace)

### Méthode Automatique (Command Line)
```bash
# Vérifier l'existence et le statut du namespace
kubectl get ns nfs-storage
# Attendu: Status "Active"
```

### Méthode Manuelle
1. Vérifier la présence du namespace.

## Notes Techniques
- **Namespace :** `nfs-storage`
- **Dépendances :** Aucune
- **Particularités :** Namespace dédié aux ressources NFS manuelles ou futures.