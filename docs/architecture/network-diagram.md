# Architecture Réseau Vixens

## Vue d'Ensemble

Infrastructure Kubernetes multi-cluster avec segmentation VLAN pour isolation réseau et sécurité.

### Principes de Design

1. **VLAN 111 (Non Routé)** : Communication inter-nodes + accès storage NAS
   - Optimise la bande passante (trafic cluster isolé)
   - Sécurise l'accès storage (pas d'exposition externe)

2. **VLANs 20X (Routés Internet)** : Exposition services par environnement
   - Isolation par environnement (dev/test/staging/prod)
   - Contrôle d'accès au niveau VLAN

3. **Dual-Interface Virtuelle** : Chaque node Talos a 2 interfaces VLAN sur 1 NIC physique

---

## Topologie VLANs

```
┌─────────────────────────────────────────────────────────────────┐
│                         Physical Network                          │
└─────────────────────────────────────────────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │   Switch (VLAN-Aware)    │
                    └────────────┬────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
   ┌────▼────┐            ┌─────▼─────┐          ┌──────▼──────┐
   │ VLAN 111│            │ VLAN 200  │          │ VLAN 20X    │
   │ Storage │            │   Admin   │          │ Services    │
   └─────────┘            └───────────┘          └─────────────┘
   Non-Routé              Routé (Mgmt)           Routé (Apps)
```

---

## Matrice VLANs

| VLAN | Subnet            | Routage | Usage                          | Clusters      |
|------|-------------------|---------|--------------------------------|---------------|
| 111  | 192.168.111.0/24  | ❌ Non  | Inter-node, Storage (Synology) | Tous          |
| 200  | 192.168.200.0/24  | ✅ Oui  | Administration (Hyper-V, SSH)  | Management    |
| 201  | 192.168.201.0/24  | ✅ Oui  | Services Prod                  | Prod          |
| 208  | 192.168.208.0/24  | ✅ Oui  | Services Dev                   | Dev           |
| 209  | 192.168.209.0/24  | ✅ Oui  | Services Test                  | Test          |
| 210  | 192.168.210.0/24  | ✅ Oui  | Services Staging               | Staging       |

---

## Plan d'Adressage par Cluster

### Cluster DEV (VLAN 208)

| Hostname | Interface VLAN 111 | MAC (111)        | Interface VLAN 208 | MAC (208)        | Rôle         |
|----------|-------------------|------------------|--------------------|------------------|--------------|
| obsy     | 192.168.111.162   | 00:15:5D:00:CB:10| 192.168.208.162    | 00:15:5D:00:CB:0A| Control Plane|
| onyx     | 192.168.111.164   | 00:15:5D:00:CB:11| 192.168.208.164    | 00:15:5D:00:CB:09| Control Plane|
| opale    | 192.168.111.163   | 00:15:5D:00:CB:0F| 192.168.208.163    | 00:15:5D:00:CB:0B| Control Plane|

**VIP Kubernetes** : 192.168.111.160 (VLAN 111)

**MetalLB Pools** :
- Pool Assigned (Traefik) : `192.168.208.70-192.168.208.79`
- Pool Auto : `192.168.208.80-192.168.208.89`

---

### Cluster TEST (VLAN 209)

| Hostname | Interface VLAN 111 | MAC (111)        | Interface VLAN 209 | MAC (209)        | Rôle         |
|----------|-------------------|------------------|--------------------|------------------|--------------|
| carny    | 192.168.111.183   | 00:15:5D:00:CB:1B| 192.168.209.183    | 00:15:5D:00:CB:18| Control Plane|
| celesty  | 192.168.111.182   | 00:15:5D:00:CB:1C| 192.168.209.182    | 00:15:5D:00:CB:19| Control Plane|
| citrine  | 192.168.111.184   | 00:15:5D:00:CB:1D| 192.168.209.184    | 00:15:5D:00:CB:1A| Control Plane|

**VIP Kubernetes** : 192.168.111.180 (VLAN 111)

**MetalLB Pools** :
- Pool Assigned (Traefik) : `192.168.209.70-192.168.209.79`
- Pool Auto : `192.168.209.80-192.168.209.89`

---

### Cluster STAGING (VLAN 210)

**À définir** - Nodes non encore provisionnés

**VIP Kubernetes** : 192.168.111.190 (VLAN 111)

**MetalLB Pools** :
- Pool Assigned (Traefik) : `192.168.210.70-192.168.210.79`
- Pool Auto : `192.168.210.80-192.168.210.89`

---

### Cluster PROD (VLAN 201)

| Hostname | Interface VLAN 111 | Interface VLAN 201 | Rôle         |
|----------|-------------------|--------------------|--------------|
| jade     | 192.168.111.63    | 192.168.201.63     | Control Plane|
| ruby     | 192.168.111.66    | 192.168.201.66     | Control Plane|
| emy      | 192.168.111.65    | 192.168.201.65     | Control Plane|

**VIP Kubernetes** : 192.168.111.170 (VLAN 111)

**MetalLB Pools** :
- Pool Assigned (Traefik) : `192.168.201.70-192.168.201.79`
- Pool Auto : `192.168.201.80-192.168.201.89`

---

## Infrastructure Partagée

### NAS Synology (Synelia)

| Service  | VLAN | IP              | Protocole | Usage                    |
|----------|------|-----------------|-----------|--------------------------|
| Storage  | 111  | 192.168.111.69  | iSCSI/NFS | CSI Driver, PV statiques |

**NFS Exports** :
- `/volume1/content` : Media partagés (Radarr, Sonarr, etc.)
- `/volume1/downloads` : Téléchargements (SABnzbd)

**iSCSI Targets** : Créés dynamiquement par Synology CSI Driver

---

### Poste de Gestion (grenat)

| Interface | VLAN | IP              | Usage                          |
|-----------|------|-----------------|--------------------------------|
| eth0      | 200  | 192.168.200.X   | Admin Hyper-V, SSH             |
| eth1      | 111  | 192.168.111.64  | Accès Kubernetes VIP, Storage  |

**Outils installés** :
- `kubectl` (accès API Kubernetes via VLAN 111)
- `talosctl` (gestion nodes)
- `terraform` (provisioning)
- `argocd` CLI (GitOps management)

---

## Flux Réseau par Use Case

### 1. Déploiement Terraform (depuis grenat)

```
grenat (VLAN 111) → Nodes (VLAN 111)
  └─ talosctl bootstrap, machine config apply
```

### 2. Accès API Kubernetes (depuis grenat)

```
grenat (192.168.111.64) → VIP Kubernetes (192.168.111.160)
  └─ kubectl, helm, argocd CLI
```

### 3. Trafic Inter-Nodes

```
Node A (VLAN 111) ↔ Node B (VLAN 111)
  └─ etcd, kubelet, CNI (Cilium)
```

### 4. Accès Storage Synology

```
Nodes (VLAN 111) → Synelia (192.168.111.69)
  └─ iSCSI/NFS mounts
```

### 5. Exposition Services (Utilisateur → App)

```
User (Internet/LAN) → Traefik LoadBalancer (VLAN 20X) → Pod (VLAN 111)
  └─ HTTP/HTTPS via Ingress
```

**Exemple Dev** :
```
User → 192.168.208.70:443 (Traefik) → Pod ArgoCD (VLAN 111)
```

---

## Sécurité Réseau

### Isolation par VLAN

| Source         | Destination    | Autorisation | Règle Firewall |
|----------------|----------------|--------------|----------------|
| Internet       | VLAN 111       | ❌ Bloqué    | Drop all       |
| Internet       | VLAN 20X:80/443| ✅ Autorisé  | Allow HTTP(S)  |
| VLAN 200       | VLAN 111       | ✅ Autorisé  | Admin access   |
| VLAN 20X       | VLAN 111       | ✅ Autorisé  | Backend access |
| VLAN 111       | VLAN 111       | ✅ Autorisé  | Cluster traffic|

### Ports Exposés par VLAN

**VLAN 111 (Non-Routé)** :
- 6443 : Kubernetes API (VIP)
- 50000 : Talos API
- 2379-2380 : etcd
- 3260 : iSCSI
- 2049 : NFS

**VLAN 20X (Routé)** :
- 80 : HTTP (Traefik)
- 443 : HTTPS (Traefik)

---

## Validation Réseau

### Tests de Connectivité

```bash
# Depuis grenat (VLAN 111)
ping 192.168.111.162  # Node obsy
kubectl --kubeconfig kubeconfig-dev get nodes  # API Kubernetes

# Depuis node (via talosctl)
talosctl --nodes 192.168.111.162 get members
cilium connectivity test  # Test CNI

# Depuis Internet (VLAN 208)
curl https://argocd.dev.example.com  # Service exposé
```

---

## Considérations Futures

1. **BGP Mode MetalLB** : Actuellement L2, envisager BGP pour multi-subnet routing
2. **NetworkPolicies Cilium** : Restreindre trafic inter-namespace
3. **Egress Gateway** : Contrôler trafic sortant des pods
4. **VPN/Bastion** : Accès externe sécurisé aux VLANs non-routés

---

## Diagramme Détaillé Cluster DEV

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Internet / LAN                                │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                    ┌────────▼─────────┐
                    │  Router/Firewall │
                    │   (VLAN Routing) │
                    └────────┬─────────┘
                             │
              ┌──────────────┴───────────────┐
              │                              │
      ┌───────▼────────┐           ┌────────▼────────┐
      │  VLAN 111      │           │  VLAN 208       │
      │  192.168.111/24│           │  192.168.208/24 │
      │  (Non-Routé)   │           │  (Routé)        │
      └───────┬────────┘           └────────┬────────┘
              │                              │
    ┌─────────┼─────────────────────────────┼─────────┐
    │         │                              │         │
┌───▼───┐ ┌──▼────┐ ┌────────┐   ┌──────────▼──────┐  │
│ obsy  │ │ onyx  │ │ opale  │   │   MetalLB Pool  │  │
│ .162  │ │ .164  │ │ .163   │   │   .70-.89       │  │
└───┬───┘ └───┬───┘ └───┬────┘   └─────────┬───────┘  │
    │         │         │                   │          │
    └─────────┴─────────┴───────────────────┘          │
              │                                         │
         ┌────▼────┐                         ┌─────────▼────────┐
         │   VIP   │                         │ Traefik LB       │
         │  .160   │                         │ 192.168.208.70   │
         └─────────┘                         └──────────────────┘
              │                                         │
         ┌────▼────────────────────────────────────────▼────┐
         │          Kubernetes Cluster (CNI: Cilium)        │
         │  Pods communicate via VLAN 111 (internal)        │
         └──────────────────────────────────────────────────┘
              │
         ┌────▼────┐
         │ Synelia │
         │ .69 NAS │
         └─────────┘
```

---

## Changelog

| Date       | Version | Changement                           |
|------------|---------|--------------------------------------|
| 2025-10-30 | 1.0     | Architecture initiale multi-cluster  |
