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

# 🏠 Environnement Vixens - Infrastructure GitOps
> **Note IA-Ready** : Fichier autonome pour déploiement 100 % GitOps via ArgoCD.

---

## 🌐 Topologie Réseau
- **Interne (non routé)** : `192.168.111.0/24`
- **Externe** : `192.168.200.0/24`
- **Poste de gestion** : `grenat 192.168.111.64`
- **Nodes** :
  - `jade 192.168.111.63` (controlplane)
  - `ruby 192.168.111.66` (controlplane)
  - `emy 192.168.111.65` (controlplane)
- **NAS** : `synology 192.168.111.69`

---

## ⚙️ Stack
- **OS** : Talos v1.10.7
- **K8s** : v1.30.0
- **CNI** : Cilium v1.18.1 (manuel)
- **LB** : MetalLB v0.14.5 / L2 / `192.168.200.70-80`
- **Ingress** : Traefik v3.1.2
- **Storage** : Longhorn v1.7.1
- **GitOps** : ArgoCD (App-of-Apps, Helm uniquement pour ArgoCD)

---

## 📁 Structure du dépôt

vixens/
├── ENVIRONNEMENT.md
├── base/                 # Manifestes natifs YAML
│   ├── argocd/
│   ├── metallb/
│   ├── traefik/
│   ├── longhorn/
│   ├── monitoring/
│   └── ...
├── clusters/
│   └── vixens/
│       └── root-app.yaml   # Seul point d’entrée
└── scripts/
├── validate-yaml.sh
└── generate-config.sh



---

## 🧩 Flux GitOps (App-of-Apps)
1. **Un seul `kubectl apply` initial** :
   ```bash
   kubectl apply -f clusters/vixens/root-app.yaml

    Tout le reste est géré par ArgoCD (aucune commande manuelle).

📍 Index des fichiers clés
Table

Rôle	Chemin complet	Notes
Application racine	clusters/vixens/root-app.yaml	Unique point d’entrée
Namespace ArgoCD	base/argocd/namespace.yaml	Déployé via ArgoCD
Config MetalLB	base/metallb/configmap.yaml	Pool IP externe
RBAC Traefik	base/traefik/rbac.yaml	ClusterRole + Binding
DaemonSet Longhorn	base/longhorn/daemonset.yaml	hostPath requis
🔐 Placeholders à remplacer
Table

Variable	Fichier	Exemple
TODO@example.com	base/traefik/deployment.yaml	admin@vixens.local
192.168.200.70-80	base/metallb/configmap.yaml	192.168.200.70-192.168.200.80
🛠️ Diagnostics
bash


# Vérifier le sync ArgoCD
argocd app list
kubectl get applications -n argocd
kubectl get events --sort-by='.lastTimestamp'

✅ Scripts de validation IA-ready
bash


# Syntaxe YAML
find . -name "*.yaml" -o -name "*.yml" | xargs yq eval '.' > /dev/null

# Chemins référencés existants
grep -r "path:" clusters/ | cut -d'"' -f2 | xargs ls -la

🎯 TODO

    Intégrer cert-manager et Cilium en GitOps
    Configurer les pools de stockage Longhorn
    Sécuriser les dashboards (à venir)

