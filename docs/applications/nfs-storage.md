# NFS Storage Namespace

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | -       |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Namespace)

### Méthode Automatique (Command Line)
```bash
kubectl get ns nfs-storage
# Attendu: Active
```

### Méthode Manuelle
1. Vérifier la présence du namespace.

## Notes Techniques
- **Namespace :** `nfs-storage`
- **Dépendances :** Aucune
- **Particularités :** Namespace dédié aux ressources NFS manuelles ou futures.
