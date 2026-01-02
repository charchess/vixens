# Rapport d'Analyse des Ressources (Prod vs VPA)
**Date :** 2026-01-01
**Source :** Analyse statique des manifestes `overlays/prod` vs Recommandations VPA (données historiques `dev`).

## 1. Synthèse
L'analyse met en évidence des déséquilibres importants dans l'allocation des ressources pour les applications critiques.
- **Risque de Performance :** Lazylibrarian (CPU bridé).
- **Risque de Stabilité (OOM) :** Booklore, Sonarr, Radarr (Marges RAM trop faibles).
- **Gaspillage :** Hydrus Client (RAM sur-allouée).
- **Risque de Sécurité/Noisy Neighbor :** Authentik Worker (Aucune limite définie).

## 2. Détail par Application

| Application | Container | Prod Config (Req / Lim) | VPA Target (Est.) | Status | Recommandation |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Lazylibrarian** | app | `15m` / `259Mi`<br>`15m` / `488Mi` | **CPU: 410m**<br>RAM: 128Mi | 🔴 **CRITIQUE** | Le CPU est bridé à 15m alors que l'app demande 410m. Augmenter Limit CPU à **1000m**. |
| **Booklore** | app | `15m` / `1848Mi`<br>`548m` / `5545Mi` | CPU: 11m<br>**RAM: 3136Mi** | 🟠 **RISQUE** | Request RAM (1.8G) < Target (3.1G). Risque d'éviction si le nœud est plein. Augmenter Request RAM à **3Gi**. |
| **Hydrus** | client | `34m` / `2294Mi`<br>`49m` / `3877Mi` | CPU: 11m<br>**RAM: 587Mi** | 🔵 **GASPILLAGE** | Request RAM (2.3G) >>> Target (0.6G). Réduire Request à **1Gi** pour libérer 1.3Gi au scheduler. |
| **Authentik** | worker | *Non défini* | CPU: 11m<br>RAM: 671Mi | 🔴 **CRITIQUE** | Absence de limites. Risque de consommation illimitée en cas de leak/boucle. Appliquer **Req: 512Mi / Lim: 1Gi**. |
| **Authentik** | server | `200m` / `512Mi`<br>`500m` / `1024Mi` | CPU: 11m<br>RAM: 671Mi | 🟡 **MOYEN** | Request RAM (512M) un peu juste vs Target (671M). Augmenter Request à **768Mi**. |
| **Sonarr** | app | `15m` / `236Mi`<br>`15m` / `236Mi` | CPU: 11m<br>RAM: 203Mi | 🟡 **MOYEN** | Marge RAM très faible (30Mi). Augmenter Limit à **512Mi**. |
| **Radarr** | app | `22m` / `334Mi`<br>`35m` / `362Mi` | CPU: 11m<br>RAM: 272Mi | 🟡 **MOYEN** | Marge RAM faible. Augmenter Limit à **512Mi**. |

## 3. Plan d'Action
1.  **Correctif Prioritaire :** Appliquer les limites sur `authentik-worker` et débrider le CPU de `lazylibrarian`.
2.  **Optimisation :** Réduire la request RAM de `hydrus-client`.
3.  **Fiabilisation :** Ajuster les requests/limits RAM de `booklore`, `sonarr`, `radarr`.

---
*Généré par Coding Agent.*
