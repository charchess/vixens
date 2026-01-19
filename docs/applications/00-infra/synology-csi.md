# Synology CSI Driver

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | edge    |
| Prod          | [x]     | [x]       | [x]   | -       |

## Validation
**URL :** N/A (Storage Driver)

### Méthode Automatique (Command Line)
```bash
# Vérifier que les pods du driver sont en ligne
kubectl get pods -n synology-csi
# Attendu: Pods controller et node en statut Running (tous)
```

### Méthode Manuelle
1. Créer une PVC de test utilisant la StorageClass `synelia-iscsi-retain`.
2. Créer un Pod qui monte cette PVC.
3. Vérifier que la PVC passe en statut `Bound` et que le Pod démarre (Mount successful).

## Notes Techniques
- **Namespace :** `synology-csi`
- **Dépendances :**
    - `Infisical` (Secret `client-info-secret` via `synology-csi-secrets`)
- **Particularités :** Image `ghcr.io/zebernst/synology-csi:edge`. Configure le stockage iSCSI sur le NAS Synology. StorageClass par défaut: `synelia-iscsi-retain`.