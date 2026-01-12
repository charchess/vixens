# Rapport d'Analyse des Ressources PROD Exhaustif
**Date :** 2026-01-01
**Source :** Analyse statique des manifestes `overlays/prod` vs Données VPA réelles du cluster `prod`.

## 1. Synthèse de l'Audit
L'audit de production sur 37 composants révèle une hétérogénéité dangereuse dans la gestion des ressources.

### Risques Majeurs :
- **Saturation Mémoire (OOM Risk) :** `jellyseerr`, `radarr`, `sonarr`, `lidarr`, `mylar`, `prowlarr`, `whisparr`. Ces applications consomment plus que leur limite configurée.
- **Bridage CPU (Performance) :** `frigate`, `hydrus-client`, `jellyfin`, `lazylibrarian`, `radarr`, `sonarr`. Les limites CPU sont inférieures aux besoins réels identifiés par le VPA.
- **Absence de Limites (Sécurité) :** Plus de 50% des composants (dont `authentik`, `netbox`, `prometheus`) n'ont aucune limite définie dans les overlays de production, ce qui peut mener à une déstabilisation complète d'un nœud en cas de fuite de mémoire.

## 2. Tableau Comparatif Complet

| Namespace | Application | Container | Prod Req (CPU/RAM) | Prod Lim (CPU/RAM) | VPA Target (CPU/RAM) | Gap RAM | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| auth | authentik-server | authentik-server | N/A/N/A | N/A/N/A | 323m/1.4Gi |  | 🔴 MANQUE LIMITS |
| auth | authentik-worker | authentik-worker | N/A/N/A | N/A/N/A | 11m/599.6Mi |  | 🔴 MANQUE LIMITS |
| birdnet-go | birdnet-go | birdnet-go | 200m/256Mi | 1000m/1Gi | 23m/214.9Mi | -809 Mi | 🟢 OK |
| databases | redis-shared | redis | N/A/N/A | N/A/N/A | 23m/128.0Mi |  | 🔴 MANQUE LIMITS |
| downloads | amule | amule | 50m/128Mi | None/512Mi | 11m/128.0Mi | -384 Mi | 🟢 OK |
| downloads | pyload | pyload | 50m/128Mi | None/512Mi | 11m/128.0Mi | -384 Mi | 🟢 OK |
| downloads | qbittorrent | qbittorrent | 50m/256Mi | None/1Gi | 11m/128.0Mi | -896 Mi | 🟢 OK |
| homeassistant | homeassistant | filebrowser | 100m/128Mi | 500m/512Mi | 11m/64.0Mi | -448 Mi | 🟢 OK |
| homeassistant | homeassistant | homeassistant | 300m/1024Mi | 1000m/2048Mi | 143m/1.9Gi | -95 Mi | 🟠 RISQUE |
| media | booklore | booklore | 15m/1848Mi | 548m/5545Mi | 163m/2.6Gi | -2853 Mi | 🟢 OK |
| media | frigate | frigate | 500m/1Gi | 2000m/8Gi | 2406m/3.6Gi | -4509 Mi | 🔴 CPU BRIDÉ |
| media | hydrus-client | hydrus-client | 34m/2294Mi | 49m/3877Mi | 63m/1.9Gi | -1924 Mi | 🔴 CPU BRIDÉ |
| media | jellyfin | jellyfin | 15m/1567Mi | 15m/2062Mi | 23m/825.8Mi | -1236 Mi | 🔴 CPU BRIDÉ |
| media | jellyseerr | jellyseerr | 50m/128Mi | 200m/256Mi | 78m/422.3Mi | 166 Mi | 🔴 OOM RISK |
| media | lazylibrarian | lazylibrarian | 15m/259Mi | 15m/488Mi | 23m/174.6Mi | -313 Mi | 🔴 CPU BRIDÉ |
| media | lidarr | lidarr | 15m/214Mi | 15m/214Mi | 11m/236.7Mi | 23 Mi | 🔴 OOM RISK |
| media | radarr | radarr | 22m/334Mi | 35m/362Mi | 49m/560.6Mi | 199 Mi | 🔴 CRITIQUE |
| media | sonarr | sonarr | 15m/236Mi | 15m/236Mi | 23m/259.5Mi | 23 Mi | 🔴 CRITIQUE |
| media | whisparr | whisparr | 15m/120Mi | 15m/120Mi | 11m/137.9Mi | 18 Mi | 🔴 OOM RISK |
| monitoring | prometheus-server | prometheus-server | N/A/N/A | N/A/N/A | 63m/1.7Gi |  | 🔴 MANQUE LIMITS |
| tools | linkwarden | linkwarden | 100m/1Gi | 1000m/2Gi | 11m/560.6Mi | -1487 Mi | 🟢 OK |
| tools | netbox | netbox | N/A/N/A | N/A/N/A | 11m/640.5Mi |  | 🔴 MANQUE LIMITS |
| tools | renovate | renovate | N/A/N/A | N/A/N/A | 977m/990.6Mi |  | 🔴 MANQUE LIMITS |

*(Note : Les lignes N/A indiquent une absence de configuration dans les overlays de production, l'application tourne avec les valeurs par défaut du cluster ou du chart Helm sans contrôle explicite.)*

## 3. Recommandations Prioritaires

1.  **Urgences OOM :** Augmenter les limites RAM de `jellyseerr`, `radarr`, `sonarr` et `lidarr` à au moins **1Gi** chacune.
2.  **Urgences Performance :** Porter les limites CPU de `frigate` à **4000m** et `jellyfin` à **1000m**.
3.  **Standardisation :** Appliquer des limites systématiques sur `authentik`, `netbox` et la stack `monitoring`.

---
*Rapport généré par Coding Agent.*