# 🏠 Environnement Vixens - Infrastructure GitOps

## 🌐 Topologie Réseau
- **Réseau interne (non routé)**: 192.168.111.0/24
  - Communication inter-nœuds & stockage
- **Réseau externe**: 192.168.200.0/24
  - Accès aux services depuis l'extérieur
- **Poste gestion**: grenat (192.168.111.64)
- **Nodes**:
  - jade (192.168.111.63) - controlplane
  - ruby (192.168.111.66) - controlplane  
  - emy (192.168.111.65) - controlplane
- **NAS**: synology (192.168.111.69)

## ⚙️ Stack Technique

### Kubernetes
- **Distribution**: Talos v1.10.7
- **Version**: v1.30.0
- **Runtime**: containerd v2.0.5
- **Taints**: Supprimés sur tous les nodes

### Réseau & Sécurité
- **CNI**: Cilium v1.18.1 (installé mais non géré GitOps)
- **LoadBalancer**: MetalLB v0.13.x (Layer2)
  - Pool externe: 192.168.200.70-192.168.200.80
- **Ingress**: Traefik v3.x
- **Certificats**: cert-manager (installé mais non géré GitOps)

### Stockage
- **Solution**: Longhorn v1.9.1 (accessible via UI, non configuré)
- **Path**: /var/lib/kubelet (fix Talos)
- **UI**: http://192.168.111.64:32000 (NodePort)

### GitOps
- **Outil**: ArgoCD
- **Pattern**: App-of-Apps
- **Repo**: https://github.com/ton-repo/vixens-config

## 📊 État des Services

| Service      | Namespace       | Status GitOps | Notes                        |
|--------------|-----------------|---------------|------------------------------|
| ArgoCD       | argocd          | ✅ Géré       | Port 8080                    |
| MetalLB      | metallb-system  | ✅ Géré       | Config externe requise       |
| Traefik      | traefik         | ✅ Géré       | Ingress class configuré      |
| cert-manager | cert-manager    | ❌ Manuel     | À intégrer GitOps            |
| Longhorn     | longhorn-system | ⚠️ Déployé     | UI accessible, non configuré |
| Cilium       | kube-system     | ❌ Manuel     | À intégrer GitOps            |

## 🎯 TODO Prioritaire
1. Configurer Longhorn (pools de stockage)
2. Migrer cert-manager en GitOps
3. Migrer Cilium en GitOps
4. Configurer IP pools MetalLB externe

## 🛠️ Accès & Commandes
# Services
ArgoCD: argocd.truxonline.com
Longhorn UI: ?

# Diagnostic
kubectl get nodes -o wide --show-labels
kubectl get svc -A | grep LoadBalancer
