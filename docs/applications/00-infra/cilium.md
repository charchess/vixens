# Cilium CNI

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version  |
|---------------|---------|-----------|-------|----------|
| Dev           | [x]     | [x]       | [x]   | v1.18.3  |
| Test          | [ ]     | [ ]       | [ ]   | -        |
| Staging       | [ ]     | [ ]       | [ ]   | -        |
| Prod          | [ ]     | [ ]       | [ ]   | -        |

## Validation
**URL :** N/A (CNI Infrastructure)

### Méthode Automatique (Command Line)
```bash
# Vérifier que les pods Cilium sont en ligne
kubectl get pods -n kube-system -l k8s-app=cilium
# Attendu: DaemonSet cilium avec pods en statut Running sur tous les nœuds

# Vérifier l'état de Cilium
cilium status --wait
# Attendu: All components healthy

# Vérifier la connectivité réseau
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -- ping -c 3 8.8.8.8
# Attendu: Ping successful
```

### Méthode Manuelle
1. Vérifier que tous les nœuds du cluster sont en état Ready
2. Vérifier que les pods des applications peuvent communiquer entre eux
3. Vérifier que la résolution DNS fonctionne (nslookup kubernetes.default)

## Notes Techniques
- **Namespace :** `kube-system`
- **Type :** DaemonSet
- **Dépendances :** 
    - Aucune (infrastructure de base)
- **Particularités :** 
    - CNI principal du cluster (Container Network Interface)
    - Remplace kube-proxy (eBPF datapath)
    - Fournit Network Policies, LoadBalancer (L2), Service Mesh
    - PriorityClass: `system-node-critical`

## Configuration DNS Proxy

**Problème résolu :** Timeouts DNS avec CoreDNS en mode transparent

### Symptômes
- Timeouts intermittents sur les requêtes DNS
- Logs CoreDNS montrant des timeouts upstream
- Applications en erreur au démarrage (résolution DNS échouée)

### Solution
Désactivation du mode transparent du DNS proxy Cilium :
```yaml
dnsProxy:
  enableDNSProxy: true
  dnsProxyTransparentMode: false  # Désactivé pour éviter les timeouts
```

**Raison :** Le mode transparent peut causer des conflits avec CoreDNS dans certaines configurations réseau (notamment avec Talos Linux).

**Alternative testée :** `enableDNSProxy: false` désactive complètement le proxy DNS de Cilium (fonctionne aussi mais perd les features DNS de Cilium).

### Validation de la configuration
```bash
# Vérifier la config DNS proxy
kubectl get cm cilium-config -n kube-system -o yaml | grep -A 2 dns-proxy

# Tester la résolution DNS depuis un pod
kubectl run -it --rm dns-test --image=busybox:1.28 --restart=Never -- nslookup kubernetes.default
# Attendu: Résolution successful sans timeout
```

## Références
- **Documentation officielle :** https://docs.cilium.io/
- **DNS Proxy documentation :** https://docs.cilium.io/en/stable/network/kubernetes/dns/
- **Troubleshooting DNS :** https://docs.cilium.io/en/stable/operations/troubleshooting/
- **ADR connexes :** À créer si changement de configuration majeur
