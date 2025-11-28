# Plan de Recette Technique - Vixens Homelab

Ce document fournit des étapes de validation technique détaillées pour chaque composant de l'infrastructure Vixens. Il est destiné à être utilisé pour des audits de non-régression complets, en vérifiant que la configuration déployée correspond exactement à celle définie dans le code.

## Prérequis

- Accès au terminal avec `kubectl`, `terraform` et `talosctl` installés et configurés.
- Accès aux fichiers `kubeconfig` et aux variables Terraform de l'environnement cible.
- Cloner le dépôt Git du projet pour comparer l'état `live` à l'état `desired`.

---

## 1. Infrastructure (Terraform & Talos)

### 1.1. Validation du Plan Terraform

Depuis le répertoire de l'environnement (ex: `terraform/environments/dev`), exécutez un plan Terraform.

```bash
terraform -chdir=terraform/environments/dev plan
```

**Résultat Attendu :**
- Le plan s'exécute sans erreur.
- La sortie indique **"No changes. Your infrastructure matches the configuration."**
- Si des changements sont détectés, ils doivent être justifiés et attendus (par exemple, une mise à jour de version de provider).

### 1.2. Validation des Nœuds Talos

1. **Vérifier la version de Talos :**
   ```bash
   talosctl --nodes <IP-node-1> version
   ```
   Comparez la version affichée avec celle définie dans `terraform/modules/shared/locals.tf`.

2. **Vérifier la santé des nœuds :**
   ```bash
   talosctl --nodes <IP-node-1> health
   ```
   **Résultat Attendu :** Tous les services (`etcd`, `kubelet`, etc.) doivent être `[healthy]`.

3. **Vérifier la configuration du cluster :**
   Récupérez la configuration appliquée et comparez-la avec le fichier `talosconfig` généré par Terraform.
   ```bash
   talosctl --nodes <IP-node-1> get machineconfig -o yaml
   ```
   **Résultat Attendu :** La configuration (version de Kubernetes, CNI, etc.) doit correspondre à ce qui est défini dans les fichiers `.tf` du module `talos`.

---

## 2. Kubernetes & CNI (Cilium)

### 2.1. État des Nœuds Kubernetes

```bash
kubectl get nodes -o wide --kubeconfig <path-to-kubeconfig>
```
**Résultat Attendu :**
- Tous les nœuds sont à l'état `Ready`.
- La version de Kubernetes (`VERSION`) correspond à celle définie dans Talos.
- Les adresses IP internes (`INTERNAL-IP`) sont correctes.

### 2.2. État de Cilium

1. **Vérifier les pods Cilium :**
   ```bash
   kubectl -n kube-system get pods -l k8s-app=cilium --kubeconfig <path-to-kubeconfig>
   ```
   **Résultat Attendu :** Tous les pods Cilium (un par nœud) sont `Running` et `1/1`.

2. **Vérifier le statut de connectivité :**
   ```bash
   kubectl -n kube-system exec <cilium-pod-name> -- cilium status --brief
   ```
   **Résultat Attendu :**
   - `KVStore:` : `Ok`
   - `Kubernetes:`: `Ok`
   - `Cilium:`: `Ok`
   - La section `IP Address Management` indique le pool d'IP utilisé.

### 2.3. Validation du Load Balancer (Cilium L2)

```bash
kubectl -n kube-system get cm cilium-config -o yaml | grep "l2-announcements-cidrs" -A 1
```
**Résultat Attendu :** Le CIDR affiché correspond à celui défini dans `terraform/modules/cilium/main.tf` pour l'environnement concerné.

---

## 3. GitOps (ArgoCD)

### 3.1. Santé de l'Application Racine (App-of-Apps)

```bash
kubectl -n argocd get app vixens-app-of-apps -o yaml --kubeconfig <path-to-kubeconfig>
```
**Résultat Attendu :**
- `status.health.status` est `Healthy`.
- `status.sync.status` est `Synced`.
- `spec.source.path` pointe vers le bon répertoire d'overlay (ex: `argocd/overlays/dev`).

### 3.2. Synchronisation des Applications Enfants

```bash
kubectl -n argocd get applications --kubeconfig <path-to-kubeconfig>
```
**Résultat Attendu :**
- Toutes les applications listées sont `Synced` et `Healthy`.
- Aucune application n'est à l'état `OutOfSync` ou `Progressing` de manière prolongée.

### 3.3. Validation du "Self-Heal" (Test Technique)

1. Modifiez une ressource gérée par ArgoCD, par exemple, ajoutez une annotation à l'Ingress `whoami`.
   ```bash
   kubectl -n whoami edit ingress whoami-ingress --kubeconfig <path-to-kubeconfig>
   # Ajoutez une annotation: annotations: { "test.vixens.io/delete-me": "true" }
   ```
2. Surveillez l'application `whoami` dans l'interface d'ArgoCD ou via CLI.
   ```bash
   kubectl -n argocd get app whoami --kubeconfig <path-to-kubeconfig>
   ```
   Elle doit passer à l'état `OutOfSync`.
3. Attendez le cycle de synchronisation (ou forcez-le).
**Résultat Attendu :** L'annotation est automatiquement supprimée et l'application revient à l'état `Synced` et `Healthy`.

---

## 4. Sécurité, Secrets & Accès

### 4.1. Validation de la Redirection HTTPS

Inspectez le middleware Traefik responsable de la redirection.

```bash
kubectl -n traefik get middleware traefik-http-to-https -o yaml --kubeconfig <path-to-kubeconfig>
```

**Résultat Attendu :**
- La ressource `Middleware` nommée `traefik-http-to-https` existe.
- Sa spécification (`spec`) doit contenir `redirectScheme` avec `scheme: https` et `permanent: true`.
- Ce middleware doit être référencé par les `entryPoints` (comme `web`) dans la configuration statique de Traefik.

### 4.2. Validation du Bootstrap des Secrets Terraform

Vérifiez que le secret initial, nécessaire au démarrage d'Infisical ou d'autres composants critiques, a bien été créé par Terraform.

```bash
# Le nom et le namespace peuvent varier. Adaptez si nécessaire.
kubectl -n infisical get secret infisical-bootstrap-secret -o yaml --kubeconfig <path-to-kubeconfig>
```

**Résultat Attendu :**
- Le secret existe dans le namespace attendu.
- Il contient les clés (`data`) requises (ex: `INFISICAL_TOKEN`).
- Les valeurs ne sont pas vides.

### 4.3. Validation des Secrets Synchronisés par Infisical

1. Choisissez un secret géré par l'opérateur, par exemple le secret pour le webhook Gandi.
2. Vérifiez la ressource `InfisicalSecret` :
   ```bash
   kubectl -n cert-manager get infisicalsecret gandi-credentials -o yaml
   ```
3. Vérifiez le secret Kubernetes natif généré :
   ```bash
   kubectl -n cert-manager get secret gandi-credentials -o yaml
   ```
**Résultat Attendu :**
- Le secret Kubernetes `gandi-credentials` existe.
- Il contient une clé `api-key`.
- La valeur est encodée en Base64. Le décoder doit révéler un token, pas le chemin du secret Infisical.

### 4.4. Validation des Certificats (cert-manager)

1. Vérifiez le `ClusterIssuer`.
   ```bash
   kubectl get clusterissuer gandi -o yaml
   ```
   **Résultat Attendu :** Le statut (`status.conditions`) indique `Ready: True`.

2. Inspectez un certificat géré, par exemple celui de Traefik.
   ```bash
   kubectl -n traefik get certificate traefik-dashboard-tls -o yaml
   ```
**Résultat Attendu :**
- Le statut (`status.conditions`) indique `Ready: True`.
- La `spec.secretName` correspond bien au secret TLS utilisé par l'Ingress Traefik.

---

## 5. Applications

### 5.1. Traefik

Vérifiez les `IngressRoute` et `Middleware` appliqués.
```bash
kubectl get ingressroutes,middlewares -A --kubeconfig <path-to-kubeconfig>
```
**Résultat Attendu :** La liste correspond aux ressources définies dans `apps/traefik/` et les applications (comme `apps/traefik-dashboard/`).

### 5.2. Synology CSI

1. **Vérifier les pods du driver :**
   ```bash
   kubectl -n synology-csi get pods
   ```
   **Résultat Attendu :** Les pods `synology-csi-node` (un par nœud) et `synology-csi-controller` sont `Running`.

2. **Vérifier la `StorageClass` :**
   ```bash
   kubectl get sc synology-iscsi-sc -o yaml
   ```
   **Résultat Attendu :** Le provisioner est `iscsi.csi.synology.com` et les paramètres correspondent à ceux définis dans `apps/synology-csi/base/storageclass.yaml`.

3. **Vérifier le secret de configuration :**
   Le job `create-csi-config` doit avoir créé un ConfigMap ou un Secret avec les informations du NAS.
   ```bash
   kubectl -n synology-csi get secret client-info -o yaml
   ```
   **Résultat Attendu :** Le secret existe et contient le fichier `client-info.yml` avec les informations correctes.