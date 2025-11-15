# Procédure de Rebuild Complet du Cluster Dev

**Date de création** : 2025-11-15
**Dernière validation** : 2025-11-15
**Durée totale** : ~40 minutes
**Environnement** : dev (obsy, onyx, opale)

---

## Contexte

Cette procédure documente le processus complet de destruction et recréation du cluster dev depuis zéro. Elle garantit la reproductibilité de l'infrastructure et valide que tous les composants peuvent être recréés automatiquement.

## Prérequis

### Fichiers de configuration
- `terraform/environments/dev/` : Configuration Terraform
- `.secrets/dev/gandi-credentials.yaml` : Credentials Gandi API
- `.secrets/dev/client-info.yml` : Configuration Synology CSI

### Outils requis
- `terraform` >= 1.5.0
- `kubectl` >= 1.30.0
- `talosctl` >= 1.11.0
- Accès aux nodes via VLAN 208 (maintenance)

---

## Procédure de Rebuild

### Étape 1 : Terraform Destroy (5-10 minutes)

**Objectif** : Détruire complètement l'infrastructure existante.

```bash
cd /root/vixens/terraform/environments/dev

# 1. Vérifier l'état actuel
terraform state list

# 2. Lancer le destroy
terraform destroy -auto-approve
```

**Problème connu** : Si Cilium Helm release bloque (API server inaccessible) :

```bash
# Retirer Cilium du state et recommencer
terraform state rm 'module.environment.module.cilium.helm_release.cilium'
terraform destroy -auto-approve
```

**Résultat attendu** :
- 20 ressources détruites
- Nodes en mode maintenance (accessible via VLAN 208)
- Fichiers kubeconfig/talosconfig supprimés

---

### Étape 2 : Terraform Apply (15-25 minutes)

**Objectif** : Recréer l'infrastructure complète depuis zéro.

```bash
cd /root/vixens/terraform/environments/dev

# 1. Vérifier la configuration
terraform validate

# 2. Créer le cluster
terraform apply -auto-approve
```

**Phases de déploiement** :
1. **Talos configuration** (5-10 min) : Configuration des 3 control planes
2. **Bootstrap** (2 min) : Initialisation du cluster Kubernetes
3. **Cilium CNI** (10-15 min) : Déploiement du réseau eBPF
4. **ArgoCD** (5 min) : Bootstrap GitOps

**Ressources créées** :
- 3 × Talos control planes (obsy, onyx, opale)
- Cluster Kubernetes v1.34.0
- Cilium v1.18.3 (CNI + L2 Announcements + Hubble)
- ArgoCD v7.7.7 (avec root-app)
- Fichiers kubeconfig-dev et talosconfig-dev

**Validation** :
```bash
# Vérifier les nodes
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
kubectl get nodes

# Devrait afficher :
# NAME    STATUS   ROLES           AGE   VERSION
# obsy    Ready    control-plane   Xm    v1.34.0
# onyx    Ready    control-plane   Xm    v1.34.0
# opale   Ready    control-plane   Xm    v1.34.0
```

---

### Étape 3 : Créer les Namespaces Manquants (1 minute)

**Problème** : Le chart Helm cert-manager ne crée pas automatiquement son namespace.

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Créer le namespace cert-manager
kubectl create namespace cert-manager

# Note : traefik namespace est créé automatiquement par ArgoCD
```

---

### Étape 4 : Forcer la Synchronisation ArgoCD (2-5 minutes)

**Objectif** : Synchroniser manuellement les applications qui n'ont pas démarré.

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# 1. Redémarrer le repo-server si nécessaire (résout les erreurs EOF)
kubectl delete pod -n argocd -l app.kubernetes.io/name=argocd-repo-server

# 2. Forcer la sync de cert-manager
argocd app sync cert-manager --insecure --server 192.168.208.71 --plaintext

# Alternative avec kubectl :
kubectl patch application cert-manager -n argocd --type merge -p '{"operation":{"initiatedBy":{"username":"admin"},"sync":{"revision":"dev"}}}'
```

**Attendre** : 2-3 minutes pour que cert-manager se déploie complètement.

---

### Étape 5 : Appliquer les Secrets Manuels (TEMPORAIRE - 1 minute)

**⚠️ LIMITATION ACTUELLE** : Les secrets ne sont pas gérés par GitOps et doivent être appliqués manuellement.

#### Secret 1 : Gandi API Credentials (cert-manager)

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Appliquer le secret Gandi
kubectl apply -f /root/vixens/.secrets/dev/gandi-credentials.yaml
```

**Contenu attendu** (`gandi-credentials.yaml`) :
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gandi-credentials
  namespace: cert-manager
type: Opaque
stringData:
  api-token: "YOUR_GANDI_API_TOKEN"
```

#### Secret 2 : Synology CSI Configuration

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Créer le ConfigMap depuis le fichier
kubectl create configmap synology-client-info \
  --from-file=client-info.yml=/root/vixens/.secrets/dev/client-info.yml \
  -n synology-csi \
  --dry-run=client -o yaml | kubectl apply -f -
```

**Contenu attendu** (`client-info.yml`) :
```yaml
---
clients:
  - host: "192.168.111.69"
    port: 5000
    https: false
    username: "talos-csi"
    password: "YOUR_SYNOLOGY_PASSWORD"
```

---

### Étape 6 : Validation Finale (5-10 minutes)

#### 6.1 Vérifier l'état des pods

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Tous les pods doivent être Running
kubectl get pods -A

# Compter les pods Running (attendu : 32+)
kubectl get pods -A --field-selector=status.phase=Running --no-headers | wc -l
```

#### 6.2 Vérifier les applications ArgoCD

```bash
kubectl get applications -n argocd

# Attendu :
# NAME                         SYNC STATUS   HEALTH STATUS
# argocd                       Synced        Progressing
# cert-manager                 Synced        Healthy
# cert-manager-webhook-gandi   Synced        Healthy
# cilium-lb                    Synced        Healthy
# traefik                      Synced        Progressing
# ...
```

#### 6.3 Vérifier les certificats TLS

```bash
# Attendre 2-5 minutes après l'application des secrets
kubectl get certificates -A

# Vérifier les ClusterIssuers
kubectl get clusterissuers
```

#### 6.4 Tester les services exposés

```bash
# ArgoCD
curl -I http://192.168.208.71

# Whoami (après sync)
curl http://whoami.dev.truxonline.com

# Traefik Dashboard
curl http://traefik.dev.truxonline.com/dashboard/
```

---

## Résultat Final Attendu

### Infrastructure
- ✅ 3 control planes Talos (obsy, onyx, opale) - Ready
- ✅ Cluster Kubernetes v1.34.0 opérationnel
- ✅ Cilium v1.18.3 avec eBPF datapath sain
- ✅ ArgoCD v7.7.7 gérant 12+ applications

### Applications
- ✅ cert-manager + webhook-gandi : Healthy
- ✅ traefik : Progressing → Healthy
- ✅ cilium-lb : Healthy
- ✅ whoami, nfs-storage, homeassistant, mail-gateway : Synced

### Métriques
- **Pods Running** : 32+
- **Nodes Ready** : 3/3
- **Applications Synced** : 12/12
- **Durée totale** : 35-45 minutes

---

## Problèmes Connus et Solutions

### Problème 1 : Cilium Helm Release Timeout lors du Destroy

**Symptôme** :
```
Error: Error uninstalling release: dial tcp 192.168.111.160:6443: i/o timeout
```

**Cause** : L'API server n'est plus accessible car Cilium (CNI) a déjà été partiellement détruit.

**Solution** :
```bash
# Retirer Cilium du state et recommencer
terraform state rm 'module.environment.module.cilium.helm_release.cilium'
terraform destroy -auto-approve
```

---

### Problème 2 : Namespace cert-manager Non Créé

**Symptôme** :
```
namespaces "cert-manager" not found
```

**Cause** : Le chart Helm cert-manager ne crée pas automatiquement son namespace.

**Solution** :
```bash
kubectl create namespace cert-manager
```

**Amélioration future** : Ajouter `createNamespace: true` dans l'Application ArgoCD ou créer le namespace via Terraform.

---

### Problème 3 : ArgoCD Repo-Server EOF Errors

**Symptôme** :
```
error reading from server: EOF
```

**Cause** : Le repo-server peut avoir des problèmes de connexion après un redémarrage du cluster.

**Solution** :
```bash
kubectl delete pod -n argocd -l app.kubernetes.io/name=argocd-repo-server
# Attendre 30 secondes pour le redémarrage
```

---

## Limitations Actuelles (À Résoudre)

### 🔴 CRITIQUE : Gestion des Secrets Non Automatisée

**Problème** :
- Les secrets Gandi et Synology CSI sont stockés en clair dans `.secrets/dev/`
- Ils doivent être appliqués manuellement après chaque rebuild
- Ils sont commités dans Git (risque de sécurité)

**Impact** :
- La procédure de rebuild n'est pas entièrement automatisée
- Risque d'oubli lors des déploiements
- Non conforme aux bonnes pratiques de sécurité

**Solutions à étudier** :
1. **Sealed Secrets** (Bitnami)
2. **SOPS** (Mozilla) + age encryption
3. **External Secrets Operator** + backend (Minio, Vault)
4. **ArgoCD Vault Plugin**

**Voir** : `docs/procedures/secrets-management-strategy.md` (à créer)

---

### 🟡 MOYEN : Namespace cert-manager Non Automatique

**Solution future** :
- Ajouter `createNamespace: true` dans ArgoCD Application
- Ou créer via ressource Kubernetes dédiée dans Terraform

---

### 🟢 MINEUR : Temps de Déploiement Cilium

**Observation** : Cilium peut prendre 15-20 minutes à se déployer sur un cluster fresh.

**Acceptable** : C'est normal pour un déploiement eBPF complet avec Hubble.

---

## Checklist de Validation

Avant de considérer le rebuild comme réussi, vérifier :

- [ ] Les 3 nodes sont Ready
- [ ] Tous les pods kube-system sont Running
- [ ] Tous les pods Cilium sont Running (1/1)
- [ ] ArgoCD est accessible (http://192.168.208.71)
- [ ] cert-manager pods sont Running
- [ ] cert-manager-webhook-gandi est Running
- [ ] Les secrets sont appliqués (gandi-credentials, synology-client-info)
- [ ] Les ClusterIssuers sont créés
- [ ] Au moins 32 pods sont Running au total
- [ ] Aucun pod en CrashLoopBackOff ou Error

---

## Commandes de Diagnostic

### Vérifier l'état global
```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev

# Vue d'ensemble
kubectl get nodes
kubectl get pods -A | grep -v Running

# Applications ArgoCD
kubectl get applications -n argocd

# Secrets
kubectl get secrets -n cert-manager
kubectl get configmaps -n synology-csi
```

### Déboguer un pod qui ne démarre pas
```bash
# Logs
kubectl logs -n <namespace> <pod-name>

# Événements
kubectl describe pod -n <namespace> <pod-name>

# Vérifier les secrets montés
kubectl get pod -n <namespace> <pod-name> -o yaml | grep -A 10 volumes
```

### Forcer la resynchronisation ArgoCD
```bash
# Toutes les applications
argocd app sync --insecure --server 192.168.208.71 --plaintext -l app.kubernetes.io/part-of=vixens

# Application spécifique
argocd app sync <app-name> --insecure --server 192.168.208.71 --plaintext
```

---

## Amélirations Futures

1. **Automatisation complète des secrets** (PRIORITÉ HAUTE)
   - Implémenter Sealed Secrets ou SOPS
   - Supprimer `.secrets/` de Git
   - Intégrer dans le flux GitOps

2. **Script de rebuild automatisé**
   - Créer `scripts/rebuild-cluster.sh`
   - Inclure toutes les étapes manuelles
   - Ajouter validation automatique

3. **Tests de validation automatiques**
   - Script de smoke tests
   - Vérification des certificats
   - Tests de connectivité réseau

4. **Documentation des rollback**
   - Procédure de retour en arrière
   - Backup/restore etcd (si nécessaire)

---

**Auteur** : Claude Code
**Version** : 1.0
**Dernière mise à jour** : 2025-11-15
