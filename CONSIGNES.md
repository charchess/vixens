# Guide de Création d'une Nouvelle Application

Ce document explique, étape par étape, comment déployer une nouvelle application dans l'écosystème Vixens en respectant l'architecture GitOps en place.

**Philosophie :** Git est la source unique de vérité. Toute configuration est déclarée sous forme de fichiers YAML. ArgoCD s'occupe de synchroniser l'état désiré (dans Git) avec l'état réel (dans Kubernetes).

---

## Étape 1: Création de la Structure de Fichiers

1.  **Choisissez une catégorie** pour votre application (ex: `10-home`, `70-tools`, `99-test`).

2.  **Créez le dossier de l'application** dans `apps/`.
    La structure doit être la suivante :

    ```
    apps/
    └── <catégorie>/
        └── <nom-de-votre-app>/
            ├── base/
            └── overlays/
                ├── dev/
                ├── test/
                ├── staging/
                └── prod/
    ```

    **Exemple pour une application "FileBrowser" dans la catégorie "tools" :**
    ```
    apps/
    └── 70-tools/
        └── filebrowser/
            ├── base/
            └── overlays/
                ├── dev/
                ├── test/
                ...
    ```

---

## Étape 2: Configuration de Base de l'Application

La configuration de `base` contient les manifestes Kubernetes qui **ne changent pas** d'un environnement à l'autre.

1.  **Créez les manifestes de base** dans le dossier `base/`. Au minimum, vous aurez besoin de :
    *   `deployment.yaml` : Décrit comment lancer les conteneurs de votre application.
    *   `service.yaml` : Expose votre application à l'intérieur du cluster.

2.  **Créez le fichier `kustomization.yaml`** à la racine de `base/`. Ce fichier liste tous les manifestes du dossier.

**Exemple : `apps/70-tools/filebrowser/base/kustomization.yaml`**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: tools # Spécifiez le namespace ici s'il est commun

resources:
  - deployment.yaml
  - service.yaml
  # - ... autres fichiers de base
```

### ⚠️ Point de Vigilance : Vérifier le Contenu de la Base

Avant de passer à l'étape 4, il est **impératif** de vérifier le *contenu* des fichiers dans le dossier `base`. La structure peut être correcte, mais la méthode de déploiement peut être incompatible.

Ce projet utilise **ArgoCD**. Assurez-vous que les ressources dans `base` sont compatibles :

1.  **Cas Idéal : Manifestes Kubernetes Standards**
    *   Les fichiers sont des `Deployment.yaml`, `Service.yaml`, `ConfigMap.yaml`, etc.
    *   **Action :** Vous pouvez suivre ce guide sans modification.

2.  **Cas Incompatible : `HelmRelease` (FluxCD)**
    *   Si vous trouvez un fichier `helm-release.yaml` ou un objet de `kind: HelmRelease`, **ceci n'est PAS compatible** avec notre installation ArgoCD.
    *   **Action :** N'allez PAS à l'étape 4. L'application doit être "traduite" pour qu'ArgoCD puisse la déployer en tant que chart Helm natif.

**En résumé : Ne supposez pas que le contenu de `base` est correct, vérifiez-le toujours.**

---

## Étape 3: Création des Overlays par Environnement

Les `overlays` contiennent les configurations **spécifiques à chaque environnement** (dev, test, etc.). C'est ici que l'on configure les noms de domaine, les certificats, les ressources, etc.

1.  **Pour chaque environnement** (commençons par `dev`), créez un fichier `kustomization.yaml` dans le dossier correspondant.

**Exemple : `apps/70-tools/filebrowser/overlays/dev/kustomization.yaml`**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Important : On importe la configuration de base
resources:
  - ../../base

# Ici, on listera les fichiers spécifiques à l'environnement dev
# Par exemple, pour l'accès externe :
# resources:
#   - ingress.yaml
```

C'est dans ce dossier d'overlay que nous ajouterons les configurations avancées (voir section suivante).

---

## Étape 4: Déclaration de l'Application dans ArgoCD

Pour qu'ArgoCD déploie votre application, vous devez la déclarer dans le modèle "App-of-Apps".

1.  **Créez un fichier de définition d'application ArgoCD** pour chaque environnement. Placez-le dans `argocd/overlays/<env>/apps/`.

**Exemple : `argocd/overlays/dev/apps/filebrowser.yaml`**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: filebrowser
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: dev # La branche Git correspondant à l'environnement
    path: apps/70-tools/filebrowser/overlays/dev # Le chemin vers l'overlay de l'app
  destination:
    server: https://kubernetes.default.svc
    namespace: tools # Le namespace où déployer l'application
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

2.  **Ajoutez une référence** à ce nouveau fichier dans le `kustomization.yaml` principal de l'environnement.

**Exemple : `argocd/overlays/dev/kustomization.yaml`**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../base
  - apps/cert-manager.yaml
  - apps/traefik.yaml
  - apps/filebrowser.yaml # <--- AJOUTEZ VOTRE APPLICATION ICI
```

Une fois ces fichiers poussés sur Git, ArgoCD détectera la nouvelle application et la déploiera automatiquement.

---

## Sujets Avancés

### A. Configurer un Accès Externe (DNS et Certificat TLS)

Pour exposer votre application sur internet avec un nom de domaine et un certificat HTTPS.

1.  **Créez un fichier `ingress.yaml`** dans le dossier de l'overlay (`overlays/dev/`).

**Exemple : `apps/70-tools/filebrowser/overlays/dev/ingress.yaml`**
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: filebrowser-ingress
  annotations:
    # Dit à cert-manager de générer un certificat
    cert-manager.io/cluster-issuer: letsencrypt-staging # ou letsencrypt-prod
spec:
  rules:
    - host: "filebrowser.dev.truxonline.com" # Le nom de domaine pour dev
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: filebrowser-service # Le nom de votre service
                port:
                  number: 80
  tls:
    - hosts:
        - "filebrowser.dev.truxonline.com"
      secretName: filebrowser-tls # Nom du secret qui stockera le certificat
```

2.  **Ajoutez `ingress.yaml`** au `kustomization.yaml` de l'overlay.

### B. Utiliser du Stockage Persistant (Synology CSI)

Pour les applications qui ont besoin de stocker des données (ex: une base de données, un gestionnaire de fichiers).

1.  **Créez un fichier `pvc.yaml`** (PersistentVolumeClaim) dans le dossier `base/`.

**Exemple : `apps/70-tools/filebrowser/base/pvc.yaml`**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: filebrowser-config-pvc
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: synology-iscsi-storage # Le nom de la classe de stockage
  resources:
    requests:
      storage: 1Gi # La taille du volume désiré
```

2.  **Montez ce volume** dans votre `deployment.yaml` (aussi dans `base/`).
    ```yaml
    # ... dans spec.template.spec ...
          volumeMounts:
            - name: config-volume
              mountPath: /config # Le chemin dans le conteneur
      volumes:
        - name: config-volume
          persistentVolumeClaim:
            claimName: filebrowser-config-pvc
    ```

3.  **Ajoutez `pvc.yaml`** au `kustomization.yaml` de `base/`.

### C. Gérer les Secrets avec Infisical

**NE JAMAIS METTRE DE SECRETS EN CLAIR DANS GIT.** Utilisez Infisical.

1.  **Dans l'UI Infisical :** Créez le secret.
    *   Projet: `vixens`
    *   Environnement: `dev`
    *   Path: `/filebrowser`
    *   Secret: `API_KEY` = `valeur-secrete`

2.  **Créez un manifeste `infisical-secret.yaml`** dans le dossier `base/` de votre application.

**Exemple : `apps/70-tools/filebrowser/base/infisical-secret.yaml`**
```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: filebrowser-secrets-sync
spec:
  hostAPI: http://192.168.111.69:8085
  resyncInterval: 60
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens
        # ATTENTION: Le slug de l'environnement sera patché par l'overlay
        envSlug: dev # Valeur par défaut, sera surchargée
        secretsPath: "/filebrowser"
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: infisical-operator-system
  managedSecretReference:
    # Le secret Kubernetes qui sera créé
    secretName: filebrowser-secrets
    creationPolicy: "Owner"
```

3.  **Patch pour chaque environnement :** Créez un patch pour spécifier le `envSlug` correct.

**Exemple : `apps/70-tools/filebrowser/overlays/dev/infisical-patch.yaml`**
```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: filebrowser-secrets-sync
spec:
  authentication:
    universalAuth:
      secretsScope:
        envSlug: dev
```
Ajoutez ce patch au `kustomization.yaml` de l'overlay `dev`.

4.  **Utilisez le secret** dans votre `deployment.yaml`.
    ```yaml
    # ...
          env:
            - name: MY_API_KEY
              valueFrom:
                secretKeyRef:
                  name: filebrowser-secrets # Le secret créé par Infisical
                  key: API_KEY
    ```

### D. Planifier sur les Nœuds Control-Plane (Tolerations)

Par défaut, les applications ne tournent que sur les nœuds `worker`. Si une application doit tourner sur les `control-plane` (ex: agents de monitoring), ajoutez une tolérance.

1.  **Ajoutez la section `tolerations`** à votre `deployment.yaml` (dans `base/`).

    ```yaml
    # ... dans spec.template.spec ...
          tolerations:
            - key: "node-role.kubernetes.io/control-plane"
              operator: "Exists"
              effect: "NoSchedule"
    ```

---

## Résumé du Workflow

1.  ✅ Créer la structure de dossiers dans `apps/`.
2.  ✅ Ajouter les manifestes de base (`deployment.yaml`, `service.yaml`) et un `kustomization.yaml` dans `base/`.
3.  ✅ Créer un `kustomization.yaml` pour chaque `overlay`.
4.  ✅ (Optionnel) Ajouter `ingress.yaml` dans les overlays pour l'accès externe.
5.  ✅ (Optionnel) Ajouter `pvc.yaml` dans la base pour le stockage.
6.  ✅ (Optionnel) Ajouter `infisical-secret.yaml` et les patchs pour les secrets.
7.  ✅ Créer le fichier `app.yaml` dans `argocd/overlays/<env>/apps/`.
8.  ✅ Référencer ce fichier dans `argocd/overlays/<env>/kustomization.yaml`.
9.  ✅ `git add`, `git commit`, `git push`.
10. ✅ Observer le déploiement dans l'UI d'ArgoCD.
