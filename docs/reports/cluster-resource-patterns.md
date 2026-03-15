# Rapport : Dimensionnement et Réutilisation des Conteneurs (Cluster Prod) 📊

Ce rapport présente l'analyse des requêtes de ressources (CPU/RAM) pour toutes les applications du cluster, ainsi qu'un état des lieux de la réutilisation des patterns de sidecars et initContainers.

---

## 1. Tableau des Requests (Toutes Applications & Sidecars)

Ce tableau liste les ressources garanties (**Requests**) réservées sur les nœuds du cluster pour chaque conteneur.

| Namespace | Pod | Conteneur | Type | Request CPU | Request RAM |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `auth` | `authentik-server` | `authentik-server` | App | 100m | 1Gi |
| `auth` | `authentik-server` | `config-syncer` | Sidecar | 25m | 256Mi |
| `birdnet-go` | `birdnet-go` | `birdnet-go` | App | 50m | 512Mi |
| `birdnet-go` | `birdnet-go` | `litestream` | Sidecar | 25m | 256Mi |
| `birdnet-go` | `birdnet-go` | `data-syncer` | Sidecar | 25m | 256Mi |
| `databases` | `mariadb-shared` | `mariadb` | App | 50m | 512Mi |
| `databases` | `postgresql-shared` | `postgres` | App | 200m | 2Gi |
| `homeassistant` | `homeassistant` | `homeassistant` | App | 200m | 2Gi |
| `homeassistant` | `homeassistant` | `litestream` | Sidecar | 50m | 512Mi |
| `homeassistant` | `homeassistant` | `config-syncer` | Sidecar | 25m | 256Mi |
| `media` | `frigate` | `frigate` | App | 200m | 2Gi |
| `media` | `frigate` | `litestream` | Sidecar | 5m | 64Mi |
| `media` | `frigate` | `config-syncer` | Sidecar | 5m | 64Mi |
| `media` | `radarr` | `radarr` | App | 50m | 512Mi |
| `media` | `radarr` | `litestream` | Sidecar | 5m | 64Mi |
| `media` | `radarr` | `config-syncer` | Sidecar | 5m | 64Mi |
| `media` | `sonarr` | `sonarr` | App | 25m | 256Mi |
| `media` | `sonarr` | `litestream` | Sidecar | 5m | 64Mi |
| `media` | `sonarr` | `config-syncer` | Sidecar | 5m | 64Mi |
| `services` | `openclaw` | `openclaw` | App | 200m | 2Gi |
| `services` | `openclaw` | `install-tools` | Init | 200m | 2Gi |
| `tools` | `penpot-backend` | `backend` | App | 100m | 1Gi |
| `monitoring` | `prometheus-server` | `prometheus-server` | App | 200m | 2Gi |
| *(...)`* | *Liste complète* | *disponible via* | *kubectl* | *...* | *...* |

---

## 2. Patterns de Réutilisation (Sidecars & InitContainers)

Ce tableau identifie les patterns répétitifs utilisés pour la gestion des données, de la configuration et de la persistance SQLite sur le cluster.

| Nom du Conteneur | Type | Image Commune | Nombre d'utilisations | Usage |
| :--- | :--- | :--- | :--- | :--- |
| **`litestream`** | Sidecar | `litestream/litestream` | **14** | Réplication SQLite temps réel vers S3. |
| **`config-syncer`** | Sidecar | `python:3.14-alpine` / `rclone` | **12** | Sauvegarde périodique de la config YAML vers S3. |
| **`restore-config`** | Init | `rclone/rclone` | **11** | Restauration de la configuration au démarrage du pod. |
| **`fix-permissions`** | Init | `busybox` | **9** | Ajustement récursif des droits (UID/GID 1000) sur les PVC. |
| **`restore-db`** | Init | `litestream/litestream` | **8** | Restauration de la base SQLite depuis le dernier replica S3. |
| **`data-syncer`** | Sidecar | *(Divers)* | **3** | Synchronisation de données applicatives spécifiques. |

### 🔍 Observations sur la réutilisation :
1.  **Standardisation forte :** Le cluster utilise des patterns très cohérents pour la résilience des données (Litestream + Rclone).
2.  **Dette Technique identifiée :** Les conteneurs `fix-permissions` sont encore très présents (9 pods), alors que le standard 2026 vers lequel nous tendons (`fsGroup`) devrait permettre de les supprimer pour réduire le temps de démarrage.
3.  **Complexité opérationnelle :** Un pod "Standard Vixens" comporte désormais en moyenne **3 à 4 conteneurs** (1 App + 2 Sidecars + 1 Init), augmentant la pression sur le scheduler.

---
*Rapport généré le : 2026-03-14*
