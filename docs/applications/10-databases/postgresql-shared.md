# PostgreSQL Shared Cluster

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 17.6    |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | 17.6    |

## Validation
**URL :** N/A (Database Service)

### Méthode Automatique (Command Line)
```bash
# Vérifier le statut du cluster via le plugin cnpg
kubectl cnpg status postgresql-shared -n databases
# Attendu: Status "Cluster in healthy state"

# Vérifier que le service RW est accessible
kubectl get svc postgresql-shared-rw -n databases
# Attendu: Service présent avec ClusterIP
```

### Méthode Manuelle
1. Se connecter à un pod (via plugin cnpg ou psql).
2. Vérifier la création des bases de données applicatives (authentik, docspell, etc.) via `\l`.

## Notes Techniques
- **Namespace :** `databases`
- **Dépendances :**
    - `CloudNativePG` (Operator)
    - `Synology-CSI` (Stockage PVC `synelia-iscsi-retain`)
- **Particularités :** Cluster PostgreSQL mutualisé. Gestion déclarative des utilisateurs et des rôles via le bloc `managed.roles` de CNPG. Chaque application dispose de sa propre base et de son propre utilisateur synchronisé depuis Infisical. Backup S3 configuré (MinIO/Synology).