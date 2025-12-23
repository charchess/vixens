# Cilium LoadBalancer Config

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | -       |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Service Infrastructure)

### Méthode Automatique (Command Line)
```bash
kubectl get ciliumloadbalancerippool
# Attendu: Liste des pools IP configurés (ex: pool-dev)
```

### Méthode Manuelle
1. Vérifier qu'un service de type LoadBalancer obtient une IP externe.

## Notes Techniques
- **Namespace :** `kube-system`
- **Dépendances :**
    - `Cilium` (Installé via Terraform)
- **Particularités :** Configuration pure (CRDs `CiliumLoadBalancerIPPool` et `CiliumL2AnnouncementPolicy`). Ne déploie pas de pods, configure le mode L2 de Cilium.
