# MariaDB Shared

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 11.4    |
| Prod          | [x]     | [x]       | [x]   | 11.4    |

## Validation
**URL :** N/A (Database Service)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est en ligne
kubectl get pods -n databases -l app=mariadb-shared
# Attendu: Pod en statut Running

# Test de connexion simple (PING)
kubectl exec -it -n databases statefulset/mariadb-shared -- mariadb-admin ping -u root -p\$MARIADB_ROOT_PASSWORD
# Attendu: mysqld is alive
```

### Méthode Manuelle
1. Se connecter au pod et vérifier l'accès aux bases de données.

## Notes Techniques
- **Namespace :** `databases`
- **Dépendances :**
    - `Synology-CSI` (Stockage PVC `synelia-iscsi-retain`)
- **Particularités :** Instance MariaDB standalone mutualisée (StatefulSet). Mot de passe root géré via Infisical (`MARIADB_ROOT_PASSWORD`). Priorité `vixens-critical` en production.
