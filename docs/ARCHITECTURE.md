# Document d'Architecture du Projet Vixens

Ce document est la source de vérité technique pour l'infrastructure du projet Vixens.

## 1. Vue d'ensemble de l'Infrastructure

- **Hyperviseur**: Microsoft Hyper-V
- **Commutateur Virtuel (vSwitch)**: `TRUNK-EXT` (configuré en mode Trunk pour transporter plusieurs VLANs)
- **Stockage**: Disques de 40Go par VM, extensibles.
- **Mémoire**: 4Go de RAM par VM.

## 2. Conception Réseau

L'architecture repose sur une segmentation réseau stricte à l'aide de VLANs. Chaque nœud est "multi-homed" avec deux interfaces réseau : une pour le trafic interne du cluster et une pour l'exposition des services de son environnement.

### 2.1. VLANs

| ID VLAN | Nom           | Description                                                 |
| :------ | :------------ | :---------------------------------------------------------- |
| **111** | `inter-node`  | Réseau privé pour la communication entre les nœuds, etcd, et l'API Kubernetes. |
| **208** | `dev-services`| Réseau pour l'exposition des services de l'environnement `dev`. |
| **209** | `test-services`| Réseau pour l'exposition des services de l'environnement `test`.|

### 2.2. Plan d'Adressage IP

| Environnement | VLAN | Objectif              | Plage IP Nœuds Stat.            | VIP / Endpoint KubeAPI | Passerelle       |
| :------------ | :--- | :-------------------- | :------------------------------ | :--------------------- | :--------------- |
| **Interne**   | 111  | Communication Cluster | (défini par cluster)            | (défini par cluster)   | `192.168.111.1`  |
| **Dev**       | 208  | Services `dev`        | `192.168.208.10` - `192.168.208.19` | N/A                    | `192.168.208.1`  |
| **Test**      | 209  | Services `test`       | `192.168.209.20` - `192.168.209.29` | N/A                    | `192.168.209.1`  |

*Note : Les plages IP statiques doivent être exclues du scope DHCP.*

## 3. Conception des Clusters Kubernetes

### 3.1. Topologie

- Chaque cluster est composé de **3 nœuds**.
- Tous les nœuds ont le rôle de **`controlplane`** (topologie "stacked") pour assurer la haute disponibilité.
- Les workloads applicatifs sont autorisés à tourner sur les nœuds `controlplane` (tolérance `node-role.kubernetes.io/control-plane:NoSchedule` à appliquer).

### 3.2. Nommage des VMs

La convention de nommage est basée sur des noms de pierres précieuses ou semi-précieuses. Les nœuds d'un même environnement partagent la même première lettre.

### 3.3. Configuration du Cluster `dev`

| Nom de VM        | IP Interne (VLAN 111) | IP Externe (VLAN 208) | Rôle        |
| :--------------- | :-------------------- | :-------------------- | :---------- |
| `obsy`           | `192.168.111.11`      | `192.168.208.11`      | ControlPlane |
| `opale`          | `192.168.111.12`      | `192.168.208.12`      | ControlPlane |
| `onyx`           | `192.168.111.13`      | `192.168.208.13`      | ControlPlane |
| **VIP / Endpoint** | **`192.168.111.10`**  | N/A                   | KubeAPI     |

### 3.4. Configuration des Réseaux Internes Kubernetes

- **Pod Network CIDR**: `10.244.0.0/16`
- **Service Network CIDR**: `10.96.0.0/16`

### 3.5. Configuration de MetalLB (par environnement)

- **Cluster `dev` (sur VLAN 208)**:
  - `dynamic-pool`: `192.168.208.50` - `192.168.208.59`
  - `service-pool`: `192.168.208.70` - `192.168.208.79`