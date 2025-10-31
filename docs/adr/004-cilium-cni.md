# ADR-004: Cilium comme CNI

## Statut
✅ Accepté

## Contexte

Kubernetes nécessite un Container Network Interface (CNI) pour :
- Communication Pod-to-Pod
- Network Policies
- Service LoadBalancing (kube-proxy replacement)

### Alternatives Évaluées

1. **Calico**
   - ✅ Mature, très utilisé en production
   - ✅ NetworkPolicies riches
   - ✅ BGP support natif
   - ❌ Pas de kube-proxy replacement
   - ❌ Moins d'observabilité que Cilium

2. **Flannel**
   - ✅ Simple, léger
   - ✅ Overlay VXLAN out-of-the-box
   - ❌ Pas de NetworkPolicies natives
   - ❌ Fonctionnalités basiques

3. **Weave Net**
   - ✅ Simple, encryption native
   - ❌ Moins performant
   - ❌ Communauté moins active

4. **Cilium**
   - ✅ eBPF-based (performance++)
   - ✅ kube-proxy replacement (moins de hops)
   - ✅ Hubble observability (UI réseau)
   - ✅ NetworkPolicies L3-L7
   - ✅ Multi-cluster mesh natif
   - ❌ Complexité configuration avancée
   - ❌ Kernel Linux 4.19+ requis (OK pour Talos)

## Décision

**Adopter Cilium v1.18+ avec kube-proxy replacement**

### Justifications

1. **Performance eBPF** : Bypass netfilter/iptables, moins de latence
2. **Observabilité Hubble** : Visualisation flux réseau temps réel (apprentissage++)
3. **NetworkPolicies L7** : Filtrage HTTP/gRPC (sécurité fine)
4. **Talos Integration** : Support officiel, déploiement via machineconfig
5. **Future Multi-Cluster** : Cilium Cluster Mesh pour staging/prod

## Conséquences

### Positives
- ✅ **Kube-proxy removal** : Moins de composants, moins de overhead
- ✅ **Hubble UI** : Dashboard réseau pour debugging (qui parle à qui ?)
- ✅ **NetworkPolicies avancées** : L7 filtering, DNS-based policies
- ✅ **Encryption optionnelle** : WireGuard/IPsec pour pod-to-pod

### Négatives
- ⚠️ **Courbe apprentissage** : Concepts eBPF, Cilium CLI, Hubble
  - **Mitigation** : Documentation extensive, communauté Slack active
- ⚠️ **Debugging complexe** : eBPF maps, BPF programs
  - **Mitigation** : `cilium status`, `cilium monitor`, Hubble UI
- ⚠️ **Incompatibilité rare** : Certains kernels old/custom
  - **OK ici** : Talos Linux kernel moderne (6.x)

## Configuration Talos

**Bootstrap sans CNI** (CNI: none) :
```yaml
cluster:
  network:
    cni:
      name: none  # Cilium déployé post-bootstrap
```

**Déploiement Cilium** : Via Terraform Helm provider après bootstrap cluster

```hcl
resource "helm_release" "cilium" {
  name       = "cilium"
  repository = "https://helm.cilium.io/"
  chart      = "cilium"
  version    = "1.16.5"
  namespace  = "kube-system"

  values = [<<EOF
kubeProxyReplacement: true
k8sServiceHost: 192.168.111.160  # VIP Kubernetes
k8sServicePort: 6443

hubble:
  relay:
    enabled: true
  ui:
    enabled: true
  metrics:
    enabled: [dns, drop, tcp, flow, port-distribution, icmp, http]

ipam:
  mode: kubernetes

tunnel: vxlan
EOF
  ]
}
```

## Validation

### Tests de Base

```bash
# Status Cilium
cilium status

# Connectivité inter-pods
cilium connectivity test

# Hubble flows
hubble observe --all
```

### Métriques Prometheus

Cilium expose métriques Prometheus :
- `cilium_endpoint_state` : Statut endpoints
- `cilium_policy_l7_total` : Requêtes HTTP filtrées
- `cilium_drop_count_total` : Paquets droppés

## Hubble UI

**Accès** : Via Ingress (Phase 2)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hubble-ui
  namespace: kube-system
spec:
  rules:
    - host: hubble.dev.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: hubble-ui
                port:
                  number: 80
```

## Évolution Future

**Phase 2+** :
- NetworkPolicies strictes (deny-all par défaut)
- Cilium Cluster Mesh (multi-cluster networking)
- Encryption WireGuard (pod-to-pod chiffré)
- Egress Gateway (contrôle trafic sortant)

## Références

- [Cilium Documentation](https://docs.cilium.io/)
- [Talos + Cilium](https://www.talos.dev/latest/kubernetes-guides/network/deploying-cilium/)
- [Hubble Observability](https://docs.cilium.io/en/stable/observability/)
- [eBPF Introduction](https://ebpf.io/)

---

**Date** : 2025-10-30
**Auteur** : Infrastructure Team
**Révisé** : N/A
