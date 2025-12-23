# Synology CSI Driver

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | edge    |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Storage Driver)

### Méthode Automatique (Command Line)
```bash
kubectl get pods -n synology-csi
# Attendu: Pods controller et node en statut Running
```

### Méthode Manuelle
1. Créer une PVC utilisant la StorageClass `synelia-iscsi-retain`.
2. Vérifier que la PVC est bound.

## Notes Techniques
- **Namespace :** `synology-csi`
- **Dépendances :**
    - `Infisical` (Secret `client-info-secret` via `synology-csi-secrets`)
- **Particularités :** Image `ghcr.io/zebernst/synology-csi:edge`. Configure le stockage iSCSI sur le NAS Synology. StorageClass par défaut: `synelia-iscsi-retain`.
