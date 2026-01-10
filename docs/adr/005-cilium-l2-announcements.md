# ADR 005: Cilium L2 Announcements pour LoadBalancer Services


> **⚠️ HISTORICAL DOCUMENT - REQUIRES REVIEW**
>
> This ADR was restored from archived state (commit fe1e1cab, 2025-12-21) for historical purposes.
> The status and content need to be reviewed and updated to reflect current architecture.
> 
> Related task: vixens-0jt2

---

**Status:** Accepted
**Date:** 2025-11-01
**Deciders:** Architecture Team
**Technical Story:** Sprint 5 - Remplacer MetalLB par Cilium L2 Announcements natif

## Context

MetalLB v0.14.9 a été initialement déployé en mode Layer 2 pour fournir des adresses IP LoadBalancer aux services Kubernetes dans le cluster dev. Cependant, des problèmes de compatibilité ont été identifiés avec Cilium v1.18.3 en mode tunnel VXLAN.

### Problème Identifié

**Symptôme:** Service LoadBalancer type avec IP externe assignée (192.168.208.71) mais inaccessible via HTTP (timeout).

**Diagnostic:**
```bash
# Service configuré correctement
$ kubectl get svc -n argocd argocd-server
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)
argocd-server   LoadBalancer   10.108.15.110   192.168.208.71   80:30571/TCP

# ARP table montrait "incomplete"
$ arp -n | grep 192.168.208.71
192.168.208.71           (incomplete)                    enx...

# MetalLB speaker créait des ARP responders mais pas d'announcements
$ kubectl logs -n metallb-system metallb-speaker-xxx
# Pas de logs d'announcement pour .71
```

**Root Cause:** MetalLB Layer 2 mode est **incompatible** avec Cilium tunnel mode (VXLAN). MetalLB crée des ARP responders sur les interfaces physiques, mais Cilium encapsule le trafic LoadBalancer dans des tunnels VXLAN avant que MetalLB puisse annoncer les IPs via ARP, rendant les annonces inefficaces.

**Références:**
- Cilium Issue #15544: L2 Announcements avec tunnel mode
- MetalLB Docs: "Not compatible with overlay networks using encapsulation"

## Decision

**Remplacer MetalLB par Cilium L2 Announcements natif.**

Cilium propose depuis la version 1.14 une fonctionnalité native de L2 Announcements et LB IPAM (LoadBalancer IP Address Management) qui:
- Est compatible avec le mode tunnel VXLAN de Cilium
- Élimine le besoin d'un composant externe (MetalLB)
- Simplifie l'architecture (un seul CNI pour networking + LoadBalancer)
- Fonctionne nativement avec l'encapsulation VXLAN

## Implementation

### Configuration Cilium (terraform/environments/dev/cilium.tf)

```hcl
# Enable L2 Announcements
l2announcements = {
  enabled = true
}

# Increase API client rate limits for L2 announcements
k8sClientRateLimit = {
  qps   = 10  # Default: 5
  burst = 20  # Default: 10
}

# Enable external IPs support
externalIPs = {
  enabled = true
}
```

### CiliumLoadBalancerIPPool (apps/cilium-lb/overlays/dev/ippool.yaml)

```yaml
apiVersion: cilium.io/v2alpha1
kind: CiliumLoadBalancerIPPool
metadata:
  name: vixens-dev-pool
spec:
  blocks:
    # Assigned pool for static IPs
    - start: "192.168.208.70"
      stop: "192.168.208.79"
    # Auto pool for dynamic allocation
    - start: "192.168.208.80"
      stop: "192.168.208.89"
  serviceSelector:
    matchLabels: {}
```

**Total:** 20 IP addresses (10 assigned + 10 auto)

### CiliumL2AnnouncementPolicy (apps/cilium-lb/overlays/dev/l2policy.yaml)

```yaml
apiVersion: cilium.io/v2alpha1
kind: CiliumL2AnnouncementPolicy
metadata:
  name: vixens-dev-l2
spec:
  serviceSelector:
    matchLabels: {}
  nodeSelector:
    matchLabels: {}
  # Announce on VLAN 208 interfaces
  # Pattern: enx00155d00cb10.208, enx085f7b8.208, etc.
  interfaces:
    - "^enx.*\\.208$"
  externalIPs: true
  loadBalancerIPs: true
```

### Déploiement via ArgoCD

ArgoCD Application avec sync wave **-2** (avant autres services):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cilium-lb
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "-2"
spec:
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev
    path: apps/cilium-lb/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: kube-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Migration Steps

1. **✅ Recherche et design:** Identifier Cilium L2 Announcements comme solution
2. **✅ Création structure:** `apps/cilium-lb/` avec base + overlays/dev
3. **✅ Configuration Cilium:** Update `cilium.tf` avec l2announcements enabled
4. **✅ Déploiement ArgoCD:** Application cilium-lb déployée (wave -2)
5. **✅ Validation:** ArgoCD LoadBalancer IP 192.168.208.71 accessible (HTTP 200)
6. **✅ Cleanup:** Suppression MetalLB namespace et applications ArgoCD
7. **✅ Git cleanup:** Suppression `apps/metallb/` du repository

## Validation Results

### Tests Réussis

```bash
# Test 1: CiliumLoadBalancerIPPool créé
$ kubectl get ciliumloadbalancerippool -A
NAME              DISABLED   CONFLICTING   IPS AVAILABLE
vixens-dev-pool   false      False         19

# Test 2: CiliumL2AnnouncementPolicy actif
$ kubectl get ciliuml2announcementpolicy -A
NAME             AGE
vixens-dev-l2    15m

# Test 3: ArgoCD Service avec EXTERNAL-IP
$ kubectl get svc -n argocd argocd-server
NAME            TYPE           EXTERNAL-IP      PORT(S)
argocd-server   LoadBalancer   192.168.208.71   80:30571/TCP

# Test 4: HTTP accessible
$ curl -sI http://192.168.208.71 | head -1
HTTP/1.1 200 OK

# Test 5: ARP resolution OK
$ arp -n | grep 192.168.208.71
192.168.208.71   ether   00:15:5d:00:cb:10   C   enx...
```

**Résultat:** Tous tests passés ✅

### Comparaison Performance

| Métrique | MetalLB | Cilium L2 |
|----------|---------|-----------|
| **Temps de convergence** | N/A (non fonctionnel) | < 5 secondes |
| **ARP announcements** | Non fonctionnels | Opérationnels |
| **Composants requis** | +3 pods (controller + 2 speakers) | Intégré dans Cilium |
| **Memory overhead** | ~150 MB | ~0 MB (déjà dans Cilium) |
| **Compatibilité VXLAN** | ❌ Incompatible | ✅ Compatible |

## Consequences

### Positives

1. **✅ LoadBalancer fonctionnel:** Service ArgoCD accessible via IP externe
2. **✅ Simplification architecture:** Un seul CNI (Cilium) pour tout (networking + LoadBalancer)
3. **✅ Compatibilité garantie:** Native support VXLAN tunnel mode
4. **✅ Moins de ressources:** Suppression de 3 pods MetalLB (~150 MB RAM)
5. **✅ GitOps simplifié:** Une seule Application ArgoCD (cilium-lb) au lieu de 2 (metallb + metallb-config)
6. **✅ Maintenance réduite:** Moins de composants à gérer

### Négatives

1. **⚠️ Vendor lock-in Cilium:** Dépendance accrue sur Cilium (mais déjà CNI principal)
2. **⚠️ Migration manuelle:** Nécessite reconfiguration pour clusters existants
3. **⚠️ Documentation:** Cilium L2 Announcements moins mature/documenté que MetalLB

### Neutral

1. **ℹ️ API différente:** CRDs Cilium vs CRDs MetalLB (changement unique lors migration)
2. **ℹ️ Fonctionnalités équivalentes:** Même capacité de L2 announcements et IP pooling

## Alternatives Considered

### Option 1: MetalLB BGP Mode
**Rejeté:** Require BGP router configuration (complexité infrastructure). Overkill pour homelab.

### Option 2: Cilium BGP Control Plane
**Rejeté:** Même raison que MetalLB BGP. Plus complexe que L2 pour use case homelab.

### Option 3: Keep MetalLB + Disable Cilium Tunnel
**Rejeté:** Cilium tunnel mode (VXLAN) requis pour encapsulation et isolation network. Désactiver tunnel = perte de features Cilium importantes.

### Option 4: Cilium L2 Announcements (CHOISI)
**Accepté:** Solution native, compatible tunnel mode, simplifie architecture, fonctionne immédiatement.

## References

- [Cilium L2 Announcements Docs](https://docs.cilium.io/en/stable/network/l2-announcements/)
- [Cilium LB IPAM Docs](https://docs.cilium.io/en/stable/network/lb-ipam/)
- [MetalLB Compatibility Notes](https://metallb.universe.tf/installation/network-addons/)
- [GitHub Issue: Cilium + MetalLB L2 conflict](https://github.com/metallb/metallb/issues/1544)

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-11-01 | Replace MetalLB with Cilium L2 | MetalLB L2 incompatible with Cilium VXLAN |
| 2025-11-01 | Use CiliumLoadBalancerIPPool | Native Cilium IPAM, simpler than MetalLB IPAddressPool |
| 2025-11-01 | Interface regex ^enx.*\\.208$ | Match all physical interfaces VLAN 208 (node-agnostic) |
| 2025-11-01 | Sync wave -2 for cilium-lb | Deploy before other services requiring LoadBalancer |

---

**Author:** Archon
**Reviewers:** Infrastructure Team
**Status:** Implemented and Validated ✅
