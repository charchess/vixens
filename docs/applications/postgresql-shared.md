# PostgreSQL Shared Cluster

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 17.6    |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Database Service)

### Méthode Automatique (Command Line)
```bash
kubectl cnpg status postgresql-shared -n databases
# Attendu: Status "Cluster in healthy state"
```

### Méthode Manuelle
1. Se connecter à un pod (via plugin cnpg ou psql).
2. Vérifier la création des bases de données applicatives (authentik, docspell, etc.).

## Notes Techniques
- **Namespace :** `databases`
- **Dépendances :**
    - `CloudNativePG` (Operator)
    - `Synology-CSI` (Stockage PVC `synelia-iscsi-retain`)
- **Particularités :** Cluster PostgreSQL mutualisé. Chaque application dispose de sa propre base et de son propre utilisateur (gérés via init scripts ou Jobs). Backup S3 configuré (MinIO/Synology).
