# Post-Mortem : CrashLoopBackOff en cascade sur prod

**Date :** 2026-03-19 → 2026-03-20
**Durée de l'incident :** ~24h (détection → 84/90 Healthy, 6 Progressing)
**Cluster :** prod (Talos v1.12.4, K8s v1.34.0, ArgoCD v3.3.3)
**Auteur :** Claude Code (Sisyphus)
**PRs mergées :** #2318, #2321 → #2327 (8 PRs)
**Résultat :** 84/90 Healthy (vs ~83/90 au départ), 6 Progressing (gluetun en attente de PR #2327)

---

## 1. Résumé de l'incident

Plusieurs applications prod étaient en état dégradé :

| App | Symptôme | Restarts | Durée |
|-----|----------|----------|-------|
| **13 apps** (authentik, trilium, homepage...) | CrashLoopBackOff (`runAsNonRoot` incompatible) | variable | ~2j |
| **Prometheus** | CrashLoopBackOff (liveness kill pendant WAL replay) | 76 | 10h+ |
| **MariaDB** | CrashLoopBackOff (`mysqladmin: not found` + pas de startup probe) | 819 | 2j+ |
| **Homepage** | Init container `Permission denied` | récurrent | ~2j |
| **Trilium** | `FailedMount` iSCSI I/O error | 0 (bloqué) | 9h+ |
| **Gluetun** | RS crashloop (probes sur serveur de contrôle non activé) | récurrent | ~3j |
| **Promtail** | CrashLoopBackOff (Loki push timeout) | 500+ | 2j+ |

---

## 2. Analyse des causes racines

### 2.1 PR #2318 — `runAsNonRoot: true` sur 13 images qui tournent en root

**Cause :** Un audit Kyverno avait ajouté `runAsNonRoot: true` au `securityContext` de pods dont l'image Docker tourne en root (ex: authentik, netbird-management, homepage).
**Effet :** CrashLoopBackOff immédiat — le kubelet refuse de démarrer un container root avec `runAsNonRoot: true`.
**Fix :** Désactivation de `runAsNonRoot` sur les 13 apps concernées, avec annotation `vixens.io/explicitly-allow-root: "true"` pour documenter le choix.

### 2.2 Homepage — Init container sans droits root

**Cause :** L'init container `copy-initial-config` (busybox) devait nettoyer un PVC contenant des fichiers créés par un pod précédent tournant en root. Le `runAsUser: 1000` au niveau pod empêchait le `rm -rf`.
**Effet :** `rm: can't remove '/config/logs/homepage.log': Permission denied` → init bloqué → pod jamais prêt.
**Fix :** `securityContext: { runAsUser: 0, runAsNonRoot: false }` au niveau du container init uniquement (PR #2321).

### 2.3 Prometheus — Pas de startup probe + OOMKill

**Cause 1 :** Aucune startup probe. Le WAL replay sur 50Gi de données prend 10+ minutes. La liveness probe (`initialDelaySeconds: 30`, 3 failures × 15s) tuait le pod après ~75s.
**Cause 2 :** Le sizing `G-xlarge` (200m/2Gi) était complété par des `resources:` hardcodées (1000m/4Gi limit) — insuffisant pour le WAL replay qui consomme >4Gi en pic.
**Effet :** Boucle de restart infinie — WAL replay jamais terminé.
**Fix :**
1. Ajout startup probe : 90 × 10s = 15 min (PR #2322)
2. Migration vers sizing `SB-2xlarge` (500m/4Gi request, 2000m/16Gi limit) + suppression des `resources:` hardcodées (PR #2325)

### 2.4 MariaDB — `mysqladmin` renommé en `mariadb-admin`

**Cause 1 :** MariaDB 12.2 a renommé `mysqladmin` → `mariadb-admin`. Toutes les probes (startup, liveness, readiness) échouaient avec `mysqladmin: not found`.
**Cause 2 :** Pas de startup probe — l'init InnoDB (buffer pool load) prenait ~80s, dépassant le `initialDelaySeconds: 30` de la liveness.
**Cause 3 :** `resources:` hardcodées au lieu du système de sizing Kyverno (label `SB-micro` déjà présent mais ignoré).
**Effet :** 819 restarts en 2 jours.
**Fix :**
1. Ajout startup probe : 30 × 10s = 5 min (PR #2322)
2. Remplacement `mysqladmin` → `mariadb-admin` dans les 3 probes (PR #2324)
3. Suppression des `resources:` hardcodées — Kyverno `SB-micro` prend le relais (PR #2322)

### 2.5 Trilium — Volume iSCSI I/O error

**Cause :** Erreur I/O sur le volume iSCSI Synology CSI (`open .../mount: input/output error`). La session iSCSI était corrompue.
**Effet :** Pod bloqué en `ContainerCreating` pendant 9h+ (FailedMount).
**Fix :** Suppression du pod → nouveau pod → remontage iSCSI propre. Pas de perte de données.

### 2.6 Gluetun — Probes mal configurées (3 bugs en cascade)

**Bug 1 (PR #2323) :** Les probes ciblaient le port `control` (8000), mais `HTTP_CONTROL_SERVER_ADDRESS` n'était pas défini → serveur de contrôle désactivé → connexion refusée. Le port 8000 n'était pas non plus dans `FIREWALL_INPUT_PORTS`.

**Bug 2 (PR #2326) :** Après activation du contrôle server, celui-ci retourne `401 Unauthorized` par défaut (authentification requise). Migration vers le health server natif sur port 9999.

**Bug 3 (PR #2327, en attente) :** Le health server bind par défaut sur `127.0.0.1:9999` (localhost). Les probes K8s se connectent via l'IP du pod → `connection refused`. Nécessite `HEALTH_SERVER_ADDRESS=:9999` pour binder sur `0.0.0.0`.

**Effet :** Nouvelle RS en CrashLoopBackOff permanent, ancienne RS (sans probes) toujours Running depuis 3 jours.
**Fix :** `HEALTH_SERVER_ADDRESS=:9999` + `FIREWALL_INPUT_PORTS` incluant 9999 + startup probe 30 × 10s = 5 min.

### 2.7 Promtail — Loki push timeout

**Cause :** Loki injoignable par intermittence → `context deadline exceeded` sur le push → promtail crash.
**Effet :** Pods en CrashLoopBackOff sur certains nodes (phoebe, peach).
**Fix :** Suppression des pods (DaemonSet recrée) — problème transitoire résolu au redémarrage.

---

## 3. Chronologie des corrections

| Heure (UTC) | Action | PR | Résultat |
|-------------|--------|-----|---------|
| 2026-03-19 | `runAsNonRoot: false` sur 13 apps | #2318 | 13 apps récupérées |
| 2026-03-20 ~09:30 | Diagnostic trilium → delete pod | — | Trilium Healthy |
| 2026-03-20 ~09:45 | Homepage init container `runAsUser: 0` | #2321 | Homepage Healthy |
| 2026-03-20 ~10:30 | Prometheus startup probe + MariaDB startup probe + sizing cleanup | #2322 | Prometheus: startup probe OK mais OOMKill |
| 2026-03-20 ~10:50 | Gluetun: activation control server + firewall | #2323 | Control server → 401 |
| 2026-03-20 ~11:06 | MariaDB: `mysqladmin` → `mariadb-admin` | #2324 | **MariaDB 1/1 Ready, 0 restarts** |
| 2026-03-20 ~11:11 | Prometheus: `SB-2xlarge` (16Gi limit) | #2325 | **Prometheus 2/2 Ready, 0 restarts** |
| 2026-03-20 ~11:18 | Gluetun: migration control → health server (9999) | #2326 | Health server 127.0.0.1 → échec |
| 2026-03-20 ~11:25 | Gluetun: `HEALTH_SERVER_ADDRESS=:9999` (0.0.0.0) | #2327 | En attente merge |

---

## 4. État final (2026-03-20 11:30 UTC)

| App | Avant | Après | Notes |
|-----|-------|-------|-------|
| 13 apps (runAsNonRoot) | CrashLoop | **Healthy** | PR #2318 |
| Homepage | CrashLoop | **Healthy** | PR #2321 |
| Trilium | FailedMount | **Healthy** | Delete pod |
| Prometheus | CrashLoop (76 restarts) | **2/2 Ready, 0 restarts** | PR #2322 + #2325 |
| MariaDB | CrashLoop (819 restarts) | **1/1 Ready, 0 restarts** | PR #2322 + #2324 |
| Gluetun | Degraded (RS crash) | **Progressing** (PR #2327 pending) | PR #2323 + #2326 + #2327 |
| Promtail | CrashLoop | **Progressing** (transitoire) | Delete pods |
| Goldilocks | Progressing | Auto-heal | Redémarrage |
| Mealie | Pending | **Insufficient memory** | Cluster saturé |

**Score global : 84/90 Healthy** (6 Progressing dont gluetun en attente de fix, mealie bloqué par mémoire cluster)

---

## 5. Leçons apprises

### 5.1 Ne jamais appliquer `runAsNonRoot` sans vérifier l'image

L'ajout bulk de `runAsNonRoot: true` a cassé 13 apps d'un coup. **Action :** Vérifier `USER` dans le Dockerfile ou tester en dev avant de pousser en prod.

### 5.2 Les startup probes sont essentielles pour les apps à démarrage lent

Prometheus (WAL replay 10+ min) et MariaDB (InnoDB buffer pool ~80s) avaient des liveness probes avec `initialDelaySeconds` insuffisants. **Action :** Toujours ajouter une startup probe quand le démarrage dépasse 30s.

### 5.3 Vérifier la compatibilité des commandes après upgrade d'image

MariaDB 12.2 a renommé silencieusement `mysqladmin` → `mariadb-admin`. Les probes exec doivent être testées après chaque upgrade d'image majeure.

### 5.4 Le système de sizing Kyverno doit être la seule source de resources

Plusieurs apps avaient des `resources:` hardcodées ET un label de sizing. Le label était ignoré car le template Helm/Kustomize injectait les resources directement. **Action :** Supprimer tous les `resources:` hardcodés et migrer vers les labels `vixens.io/sizing.*`.

### 5.5 Gluetun : le health server est en localhost par défaut

Le health server de gluetun (port 9999) bind sur `127.0.0.1` par défaut. Les probes K8s ne peuvent pas l'atteindre sans `HEALTH_SERVER_ADDRESS=:9999`. Le control server (port 8000) requiert une auth. **Action :** Toujours documenter les ports de health check des VPN containers.

### 5.6 Cluster saturé en requests mémoire

Avec le sizing Kyverno qui injecte des requests sur TOUS les containers (même ceux qui n'en avaient pas), le cluster frôle les 99% d'allocation mémoire requests sur les 5 nodes. L'usage réel est à ~60%. **Action :** Revoir les tiers de sizing à la baisse ou ajouter des nodes.

---

## 6. Actions de suivi

- [ ] **PR #2327** : Merger et promouvoir pour finaliser gluetun
- [ ] **Cluster capacity** : Revoir l'allocation mémoire — 99% requests vs 60% usage réel
- [ ] **Audit probes** : Vérifier que toutes les apps avec démarrage >30s ont une startup probe
- [ ] **Audit sizing** : Supprimer les `resources:` hardcodées restantes (alertmanager, node-exporter, kube-state-metrics dans prometheus values.yaml)
- [ ] **Test d'upgrade MariaDB** : Ajouter un check de probe post-upgrade dans la CI ou la doc
- [ ] **Promtail/Loki** : Investiguer les timeouts récurrents de push Loki
