# ADR-003: Segmentation VLAN Multi-Environnement

## Statut
✅ Accepté

## Contexte

Infrastructure multi-cluster nécessitant :
- Isolation réseau par environnement (dev/test/staging/prod)
- Optimisation bande passante (trafic cluster vs services)
- Sécurité (limitation surface d'attaque externe)
- Accès contrôlé au storage NAS

### Alternatives Évaluées

1. **VLAN Unique (Flat Network)**
   - ✅ Simplicité configuration
   - ❌ Pas d'isolation environnements
   - ❌ Trafic cluster pollue réseau externe
   - ❌ Tous les services exposés sur même subnet

2. **VLANs par Cluster (1 VLAN = 1 Cluster)**
   - ✅ Isolation complète
   - ❌ Complexité routage inter-cluster
   - ❌ Storage NAS doit être multi-VLAN
   - ❌ Nombre de VLANs élevé (4+ clusters)

3. **Dual-VLAN (Interne + Externe par Env)**
   - ✅ VLAN interne partagé (optimisation storage)
   - ✅ VLAN externe par environnement (isolation services)
   - ✅ Contrôle granulaire routage Internet
   - ❌ Configuration réseau plus complexe

## Décision

**Adopter architecture Dual-VLAN avec VLAN interne partagé**

### Justifications

1. **VLAN 111 (Non-Routé, Partagé)** : Communication inter-nodes + storage
   - Bande passante dédiée pour etcd, kubelet, CNI
   - Accès NAS Synology unifié (pas de multi-VLAN NAS)
   - Pas d'exposition Internet (sécurité)

2. **VLANs 20X (Routés, Par Environnement)** : Services exposés
   - Isolation environnements (firewall au niveau VLAN)
   - Exposition sélective (80/443 uniquement)
   - MetalLB pool par VLAN = IPs dédiées par cluster

3. **VLAN 200 (Admin)** : Gestion infrastructure
   - Accès Hyper-V, SSH poste de gestion
   - Hors scope clusters (pas utilisé par Kubernetes)

## Conséquences

### Positives
- ✅ **Isolation Sécurité** : Dev compromis ≠ Prod compromis
- ✅ **Performance** : Trafic cluster (VLAN 111) isolé du trafic user (VLAN 20X)
- ✅ **Apprentissage** : Configuration réseau avancée (VLANs, routing, firewall)
- ✅ **Évolutivité** : Ajout cluster staging = nouveau VLAN 210

### Négatives
- ⚠️ **Complexité Configuration** : Dual-interface sur chaque node
  - **Mitigation** : Terraform automatise la config VLAN
- ⚠️ **VIP sur VLAN 111** : API Kubernetes pas directement accessible depuis Internet
  - **Mitigation** : Poste gestion (grenat) a patte sur VLAN 111
- ⚠️ **Debugging Réseau** : Routing entre VLANs peut causer issues
  - **Mitigation** : Documentation flux réseau détaillée

## Mapping VLANs

| VLAN | Subnet            | Routage | Usage                          | Firewall Rules      |
|------|-------------------|---------|--------------------------------|---------------------|
| 111  | 192.168.111.0/24  | ❌ Non  | Inter-node, Storage            | Drop all from Internet |
| 200  | 192.168.200.0/24  | ✅ Oui  | Admin (Hyper-V, SSH)           | Admin IPs only      |
| 201  | 192.168.201.0/24  | ✅ Oui  | Services Prod                  | Allow 80/443        |
| 208  | 192.168.208.0/24  | ✅ Oui  | Services Dev                   | Allow 80/443        |
| 209  | 192.168.209.0/24  | ✅ Oui  | Services Test                  | Allow 80/443        |
| 210  | 192.168.210.0/24  | ✅ Oui  | Services Staging               | Allow 80/443        |

## Configuration Talos

**Exemple Node Dev (obsy)** :
```yaml
machine:
  network:
    interfaces:
      - interface: eth0
        vlans:
          - vlanId: 111
            addresses:
              - 192.168.111.162/24
            # Pas de gateway (non-routé)
          - vlanId: 208
            addresses:
              - 192.168.208.162/24
            routes:
              - network: 0.0.0.0/0
                gateway: 192.168.208.1  # Gateway sur VLAN services
cluster:
  network:
    podSubnets:
      - 10.244.0.0/16
    serviceSubnets:
      - 10.96.0.0/12
  controlPlane:
    endpoint: https://192.168.111.160:6443  # VIP sur VLAN 111
```

## Flux Réseau

### 1. Utilisateur → Service Kubernetes
```
Internet/LAN → Router (VLAN 208) → Traefik LB (192.168.208.70)
  └─> Cilium encapsulation → Pod (VLAN 111 backend)
```

### 2. Pod → Pod (même cluster)
```
Pod A (VLAN 111) ←→ Cilium (VXLAN/Geneve) ←→ Pod B (VLAN 111)
```

### 3. Node → Storage NAS
```
Node (VLAN 111) → Synology (192.168.111.69)
  └─> iSCSI/NFS mount
```

### 4. Admin → Kubernetes API
```
grenat (192.168.111.64) → VIP (192.168.111.160:6443)
  └─> kubectl, talosctl
```

## Références

- [Talos Network Configuration](https://www.talos.dev/latest/reference/configuration/#networkconfiguration)
- [Cilium Multi-VLAN Support](https://docs.cilium.io/en/stable/network/concepts/routing/)

## Notes de Sécurité

**Firewall Rules à Implémenter** :

| Source         | Destination       | Ports       | Action |
|----------------|-------------------|-------------|--------|
| Internet       | VLAN 111          | *           | DENY   |
| Internet       | VLAN 20X:80/443   | 80/443      | ALLOW  |
| VLAN 200       | VLAN 111          | *           | ALLOW  |
| VLAN 20X       | VLAN 111          | Backend     | ALLOW  |

**NetworkPolicies Kubernetes (Phase Future)** :
- Deny all inter-namespace par défaut
- Allow explicite pour communication apps

---

**Date** : 2025-10-30
**Auteur** : Infrastructure Team
**Révisé** : N/A
