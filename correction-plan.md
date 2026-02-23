# Plan de Correction

Voici un plan pour corriger les problèmes identifiés sur le cluster de production.

## 1. Goldilocks: `OutOfSync`/`Progressing`

*   **Problème**: `commonLabels` dans Kustomize tente de modifier le `spec.selector` immuable des déploiements.
*   **Solution**:
    1.  Créer un patch pour ajouter les labels `environment: prod` uniquement aux metadonnées des pods, sans toucher au sélecteur du déploiement.
    2.  Modifier le `kustomization.yaml` de l'overlay `prod` pour utiliser ce patch au lieu de `commonLabels`.

## 2. Velero: `OutOfSync`/`Healthy`

*   **Problème**: ArgoCD détecte les ressources `Backup` créées par Velero comme une dérive par rapport à l'état Git.
*   **Solution**:
    1.  Localiser la définition de l'application ArgoCD pour `velero` dans le dépôt Git.
    2.  Ajouter une règle `ignoreDifferences` pour ignorer les ressources `Backup` (`velero.io/v1/Backup`).

## 3. Birdnet-go: `Synced`/`Progressing`

*   **Problème**: `CrashLoopBackOff` dû à une erreur de `chown` sur un volume qui semble en lecture seule, probablement à cause d'un montage NFS à l'intérieur du volume de données.
*   **Solution**:
    1.  Modifier le `deployment.yaml` de `birdnet-go`.
    2.  Changer le point de montage du volume NFS `internals` pour qu'il ne soit plus un sous-répertoire de `/data`. Par exemple, le monter à la racine du pod, comme `/internals`.
    3.  Mettre à jour le chemin d'accès dans la configuration de l'application si nécessaire.

## 4. Frigate: `Synced`/`Progressing`

*   **Problème**:
    1.  `config.yml` invalide contenant des caractères de contrôle.
    2.  Erreur intermittente `FailedMount` avec le driver CSI Synology.
*   **Solution**:
    1.  **config.yml**:
        *   Identifier la source du fichier `config.yml` dans le dépôt Git (probablement dans `apps/20-media/frigate/`).
        *   Corriger le fichier en supprimant les caractères de contrôle.
    2.  **FailedMount**:
        *   Ce problème est lié à l'infrastructure du cluster. Je recommande une investigation manuelle sur le nœud `powder` pour vérifier l'état des pods du driver CSI `csi.san.synology.com`.

## 5. Robusta: `Synced`/`Progressing`

*   **Problème**: Pods `Pending` en raison d'un manque de mémoire sur le cluster.
*   **Solution**:
    1.  Modifier les `values.yaml` ou les `patches` pour l'application `robusta`.
    2.  Réduire les demandes de mémoire (`requests.memory`) pour les déploiements `robusta-forwarder` et `robusta-runner`.
    3.  Alternativement, si les ressources sont vraiment nécessaires, il faudra envisager d'augmenter la capacité du cluster.

Chacune de ces étapes sera effectuée dans le respect du workflow GitOps, en modifiant les fichiers de configuration dans le dépôt Git et en laissant ArgoCD synchroniser les changements.
