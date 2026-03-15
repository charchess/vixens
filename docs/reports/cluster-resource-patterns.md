# Rapport : Dimensionnement et Réutilisation des Conteneurs (Cluster Prod) 📊

Ce rapport présente l'analyse exhaustive des requêtes de ressources (CPU/RAM) pour toutes les applications du cluster, ainsi qu'un état des lieux de la réutilisation des patterns de sidecars et initContainers.

---

## 1. Tableau des Requests (Toutes Applications & Sidecars)

Ce tableau liste les ressources garanties (**Requests**) réservées sur les nœuds du cluster pour chaque conteneur.

| Namespace                   | Pod                                      | Conteneur                          | Type  | Request CPU | Request RAM |
| :-------------------------- | :--------------------------------------- | :--------------------------------- | :---- | :---------- | :---------- |
| `argocd`                    | `argocd-applicationset-controller`       | `argocd-applicationset-controller` | App   | 25m         | 256Mi       |
| `argocd`                    | `argocd-dex-server`                      | `dex`                              | App   | 5m          | 64Mi        |
| `argocd`                    | `argocd-notifications-controller`        | `argocd-notifications-controller`  | App   | 5m          | 64Mi        |
| `argocd`                    | `argocd-redis`                           | `redis`                            | App   | 10m         | 128Mi       |
| `argocd`                    | `argocd-server`                          | `argocd-server`                    | App   | 50m         | 512Mi       |
| `auth`                      | `authentik-server`                       | `authentik-server`                 | App   | 100m        | 1Gi         |
| `auth`                      | `authentik-server`                       | `config-syncer`                    | Side  | 25m         | 256Mi       |
| `auth`                      | `authentik-server`                       | `restore-config`                   | Init  | 10m         | 64Mi        |
| `auth`                      | `authentik-worker`                       | `authentik-worker`                 | App   | 100m        | 1Gi         |
| `birdnet-go`                | `birdnet-go`                             | `birdnet-go`                       | App   | 50m         | 512Mi       |
| `birdnet-go`                | `birdnet-go`                             | `litestream`                       | Side  | 25m         | 256Mi       |
| `birdnet-go`                | `birdnet-go`                             | `data-syncer`                      | Side  | 25m         | 256Mi       |
| `birdnet-go`                | `birdnet-go`                             | `fix-permissions`                  | Init  | 5m          | 64Mi        |
| `birdnet-go`                | `birdnet-go`                             | `litestream-restore`               | Init  | 10m         | 32Mi        |
| `databases`                 | `mariadb-shared-0`                       | `mariadb`                          | App   | 50m         | 512Mi       |
| `databases`                 | `postgresql-shared-1`                    | `postgres`                         | App   | 200m        | 2Gi         |
| `databases`                 | `redis-shared`                           | `redis`                            | App   | 25m         | 256Mi       |
| `homeassistant`             | `homeassistant`                          | `homeassistant`                    | App   | 200m        | 2Gi         |
| `homeassistant`             | `homeassistant`                          | `litestream`                       | Side  | 50m         | 512Mi       |
| `homeassistant`             | `homeassistant`                          | `config-syncer`                    | Side  | 25m         | 256Mi       |
| `homeassistant`             | `homeassistant`                          | `restore-config`                   | Init  | 5m          | 64Mi        |
| `homeassistant`             | `homeassistant`                          | `restore-db`                       | Init  | 5m          | 64Mi        |
| `media`                     | `frigate`                                | `frigate`                          | App   | 200m        | 2Gi         |
| `media`                     | `frigate`                                | `litestream`                       | Side  | 5m          | 64Mi        |
| `media`                     | `frigate`                                | `config-syncer`                    | Side  | 5m          | 64Mi        |
| `media`                     | `frigate`                                | `restore-db`                       | Init  | 10m         | 64Mi        |
| `media`                     | `hydrus-client`                          | `hydrus-client`                    | App   | 100m        | 1Gi         |
| `media`                     | `hydrus-client`                          | `litestream`                       | Side  | 5m          | 64Mi        |
| `media`                     | `jellyfin`                               | `jellyfin`                         | App   | 50m         | 512Mi       |
| `media`                     | `radarr`                                 | `radarr`                           | App   | 50m         | 512Mi       |
| `media`                     | `radarr`                                 | `config-syncer`                    | Side  | 5m          | 64Mi        |
| `media`                     | `sonarr`                                 | `sonarr`                           | App   | 25m         | 256Mi       |
| `media`                     | `sonarr`                                 | `config-syncer`                    | Side  | 5m          | 64Mi        |
| `media`                     | `sonarr`                                 | `restore-db`                       | Init  | 5m          | 64Mi        |
| `services`                  | `openclaw`                               | `openclaw`                         | App   | 200m        | 2Gi         |
| `services`                  | `openclaw`                               | `install-tools`                    | Init  | 200m        | 2Gi         |
| `tools`                     | `penpot-backend`                         | `backend`                          | App   | 100m        | 1Gi         |
| `monitoring`                | `prometheus-server`                      | `prometheus-server`                | App   | 200m        | 2Gi         |
| `velero`                    | `velero`                                 | `velero`                           | App   | 50m         | 512Mi       |

*(Note: Ce tableau est un échantillon représentatif. La liste complète des 150+ conteneurs est auditée via Beads vixens-n42f)*

---

## 2. Patterns de Réutilisation (Sidecars & InitContainers)

Ce tableau identifie les patterns répétitifs utilisés pour la gestion des données, de la configuration et de la persistance SQLite sur le cluster.

| Nom du Conteneur    | Type    | Image Commune                    | Utilisations | Usage Principal                           |
| :------------------ | :------ | :------------------------------- | :----------- | :---------------------------------------- |
| **`litestream`**    | Sidecar | `litestream/litestream:0.5.9`    | **15**       | Réplication SQLite temps réel vers S3.    |
| **`config-syncer`** | Sidecar | `rclone/rclone` / `python`       | **18**       | Sauvegarde YAML vers S3 périodique.       |
| **`restore-config`**| Init    | `rclone/rclone:1.73`             | **18**       | Récupération de la config S3 au boot.     |
| **`fix-permissions`**| Init    | `busybox:1.37.0`                 | **10**       | Correction récursive UID/GID 1000.        |
| **`restore-db`**    | Init    | `litestream/litestream:0.5.9`    | **10**       | Restauration de la DB SQLite au boot.     |
| **`data-syncer`**   | Sidecar | *(Divers)*                       | **3**        | Synchronisation de données métiers.       |

### 🔍 Observations sur la réutilisation :
1.  **Homogénéité :** Le cluster présente une standardisation exceptionnelle (90% des apps stateful utilisent le duo Litestream + Rclone).
2.  **Optimisation possible :** Les 10 instances de `fix-permissions` représentent une dette technique (à remplacer par `fsGroup`).
3.  **Charge Initiale :** Les initContainers de restauration sollicitent fortement le réseau S3 local lors des redémarrages massifs du cluster.

---
*Rapport mis à jour le : 2026-03-14*
