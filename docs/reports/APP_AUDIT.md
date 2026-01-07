# Rapport d'Audit Global Exhaustif (Vixens Scoring Model)

*Mis à jour le 07/01/2026 - Rétablissement Complet de la Production (Post-Incident)*
*Référence : [APPLICATION_SCORING_MODEL.md](./APPLICATION_SCORING_MODEL.md)*

## 📊 Tableau des Scores (Base 100)

| Application                   | GitOps (20) | QoS (20) | Sécu (20) | Parité (20) | Data (20) | **Total** | Statut            |
| :---------------------------- | :---------: | :------: | :-------: | :---------: | :-------: | :-------: | :---------------- |
| **adguard-home**              |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **authentik**                 |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **cloudnative-pg**            |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **homeassistant**             |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **lidarr**                    |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **mariadb-shared**            |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **mylar**                     |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **prowlarr**                  |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **radarr**                    |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **sabnzbd**                   |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **sonarr**                    |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **vaultwarden**               |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **whisparr**                  |     20      |    20    |    20     |     20      |    20     |  **100**  | 🏆 Elite          |
| **external-dns**              |     20      |    20    |    20     |     20      |    10     |  **90**   | 🥇 Gold           |
| **alertmanager**              |     20      |    20    |    20     |     20      |    10     |  **90**   | 🥇 Gold           |
| **mealie**                    |     20      |    20    |    20     |     20      |    15     |  **95**   | 🥇 Gold           |
| **argocd**                    |     20      |    10    |    20     |     20      |    20     |  **90**   | 🥇 Gold           |
| **traefik**                   |     20      |    10    |    20     |     20      |    20     |  **90**   | 🥇 Gold           |
| **synology-csi**              |     20      |    10    |    20     |     20      |    20     |  **90**   | 🥇 Gold           |
| **redis-shared**              |     20      |    10    |    20     |     20      |    20     |  **90**   | 🥇 Gold (QoS lost)|
| **postgresql-shared**         |     20      |    10    |    20     |     20      |    20     |  **90**   | 🥇 Gold (QoS lost)|
| **frigate**                   |     20      |    10    |    20     |     20      |    20     |  **90**   | 🥇 Gold (QoS lost)|
| **docspell**                  |     20      |    20    |    20     |     20      |     0     |  **80**   | ✅ Valid          |
| **linkwarden**                |     20      |    10    |    20     |     20      |     0     |  **70**   | ⚠️ To Consolidate |
| **loki**                      |     20      |    10    |    20     |     20      |     0     |  **70**   | ⚠️ To Consolidate |
| **netbox**                    |     20      |    10    |    20     |     20      |     0     |  **70**   | ⚠️ To Consolidate |
| **hydrus-client**             |     20      |    10    |    10     |     20      |    20     |  **80**   | ✅ Valid          |
| **birdnet-go**                |      5      |    10    |    20     |     20      |    10     |  **65**   | ⚠️ To Consolidate |
| **changedetection**           |     20      |    10    |    10     |     20      |    10     |  **70**   | ⚠️ To Consolidate |
| **stirling-pdf**              |     20      |    10    |    10     |     20      |     0     |  **60**   | ⚠️ To Consolidate |
| **it-tools**                  |     20      |    10    |    10     |     20      |     0     |  **60**   | ⚠️ To Consolidate |
| **homepage**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **booklore**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **amule**                     |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **pyload**                    |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **qbittorrent**               |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **lazylibrarian**             |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **music-assistant**           |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **contacts**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **netvisor**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **promtail**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **goldilocks**                |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **grafana**                   |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **prometheus**                |     20      |    10    |    20     |     20      |     0     |  **70**   | ⚠️ To Consolidate |
| **hubble-ui**                 |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **headlamp**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **vpa**                       |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **descheduler**               |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **renovate**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **whoami**                    |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **reloader**                  |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **gitops-revision-controller**|     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **nfs-storage**               |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **cilium-lb**                 |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **metrics-server**            |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **mail-gateway**              |     10      |    10    |    10     |     10      |     0     |  **40**   | ⚠️ To Consolidate |
| **stirling-pdf-ingress**      |      5      |     5    |     5     |      5      |     0     |  **20**   | ❌ Legacy          |
| **it-tools-ingress**          |      5      |     5    |     5     |      5      |     0     |  **20**   | ❌ Legacy          |
| **grafana-ingress**           |      5      |     5    |     5     |      5      |     0     |  **20**   | ❌ Legacy          |
| **prometheus-ingress**        |      5      |     5    |     5     |      5      |     0     |  **20**   | ❌ Legacy          |
| **... (Infrastructure & Sec)**|      -      |     -    |     -     |      -      |     -     |     -     | -                 |

## 🛠️ État de la Production

La production est **stable et synchronisée** (GitOps OK) mais fonctionne en mode dégradé sur l'optimisation des ressources.

### Incidents et Correctifs (07/01/2026) :
*   **GitOps Repair :** Suppression de 58 fichiers `resources-patch.yaml` erronés pour rétablir la synchronisation de 28 applications.
    *   *Impact :* Perte de la QoS (Requests/Limits) et de la configuration VPA pour ces 28 applications (notées "QoS lost" ou score QoS rétrogradé à 10).
*   **Vaultwarden :** Fix du Health Check (passage à `/alive` pour v1.34.3). Service rétabli.
*   **Authentik :** Fix de l'Ingress (middleware Traefik global). Service rétabli.
*   **MariaDB Shared :** Résolution du conflit de duplication ArgoCD. Service rétabli.
*   **Infrastructure :** Rétablissement de VPA, Metrics-Server et Cilium-LB (qui étaient absents du cluster prod).

## 🎯 Prochaines Priorités

1.  **Restauration QoS (Batch Fix) :** Recréer proprement les patchs de ressources pour les 28 applications impactées (VPA, Metrics, Grafana, Loki, etc.) en validant les sélecteurs Kustomize.
2.  **Centralisation Middleware (Batch 4) :** Continuer la migration vers le middleware global pour éliminer les warnings ArgoCD restants.
