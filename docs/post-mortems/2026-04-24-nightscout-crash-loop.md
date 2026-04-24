# Post-Mortem : Nightscout — Boucle de redémarrages après récupération MongoDB

**Date :** 2026-04-24  
**Durée de l'incident :** ~06:00 UTC → ~10:12 UTC (~4h12)  
**Cluster :** prod  
**Service impacté :** Nightscout (CGM monitoring)

---

## Résumé

Suite à la récupération de MongoDB (post-mortem séparé), Nightscout s'est retrouvé en boucle de redémarrages. Cinq problèmes distincts se sont enchaînés, nécessitant 7 PRs.

---

## Chronologie

| Heure (UTC) | Événement |
|-------------|-----------|
| ~06:00 | Nightscout découvert en CrashLoop — cause initiale: HTTP→HTTPS redirect (probe → 301 → TLS error) |
| ~06:57 | PR #3054 mergé + promu : `INSECURE_USE_HTTP=true` |
| ~07:05 | Nouveau pod en `Pending` : cluster mémoire à 99-100% sur tous les nœuds |
| ~07:15 | PR #3057 mergé + promu : sizing `V-small→V-micro` (128Mi) — bloqué par Kyverno `sizing-v2-mutate` |
| ~07:26 | Pod avec 128Mi requête toujours Pending (Kyverno override ignoré) |
| ~07:48 | Découverte : `sizing-v2-mutate` override Kyverno force 256Mi via label `V-small` |
| ~08:10 | PR #3058 mergé + promu : `vpa.min-memory: 1Gi→64Mi` pour mylar et pyload — libère ~1.15Gi sur peach |
| ~08:33 | VPA in-place update : mylar 1Gi→128Mi, pyload 512Mi→256Mi |
| ~08:40 | Pod nightscout schedulé sur peach (128Mi) — mais restart boucle (exit 137) |
| ~09:00 | PR #3059 mergé + promu : `initialDelaySeconds` liveness 30→120s |
| ~09:17 | Pod schedulé (grâce à `vixens-medium` priority class PR #3060) |
| ~09:22 | Restart — liveness probe toujours trop courte (startupProbe killed at 210s) |
| ~09:43 | PR #3061 mergé + promu : `startupProbe` (40×15s = 600s) |
| ~09:54 | startupProbe échoue avec **HTTP 401** — `AUTH_DEFAULT_ROLES: denied` bloque `/api/v1/status` |
| ~10:07 | PR #3062 mergé + promu : probe HTTP→TCP socket (port 1337) |
| ~10:12 | Nightscout `1/1 Running` stable, 0 restarts |

---

## Causes racines

### 1. HTTP→HTTPS redirect (probe → TLS error)
`BASE_URL=https://...` active le middleware Express.js de nightscout qui redirige HTTP→HTTPS. Kubelet suit le 301 et envoie TLS sur le port HTTP → `http: server gave HTTP response to HTTPS client`.

**Fix :** `INSECURE_USE_HTTP: "true"` (PR #3054).

### 2. Cluster mémoire à 100% de requêtes
VPA en mode `InPlaceOrRecreate` avait progressivement augmenté les requêtes de tous les pods. Deux pods sur peach avaient des `vpa.min-memory` excessifs :
- mylar: `vpa.min-memory: 1Gi` (usage réel ~121Mi)
- pyload: `vpa.min-memory: 512Mi` (usage réel ~107Mi)

Ces paramètres empêchaient VPA de redescendre les requêtes, occupant ~1.4Gi de plus que nécessaire sur peach.

**Fix :** Baisse des `vpa.min-memory` à `64Mi` (PR #3058) + priority class `vixens-medium` pour nightscout pour activer la préemption (PR #3060).

### 3. Kyverno `sizing-v2-mutate` override les ressources
Le patch kustomize `requests: memory: 128Mi` dans l'overlay prod était ignoré : le webhook Kyverno `sizing-v2-mutate` réécrit les ressources au moment de l'admission selon le label `vixens.io/sizing.nightscout`.

**Fix :** Changer le label de `V-small` (256Mi) à `V-micro` (128Mi) dans le manifest de base (PR #3057).

### 4. Node.js startup >210s (index MongoDB fresh PVC)
Nightscout (Node.js v16) crée tous les index MongoDB au premier démarrage sur un nouveau PVC. Ce processus prend 3-5+ minutes. La liveness probe avec `initialDelaySeconds: 30` (puis 120s) tuait le pod avant que le serveur HTTP démarre.

**Fix :** `startupProbe` avec `failureThreshold: 40, periodSeconds: 15s` = 600s max (PR #3061).

### 5. `/api/v1/status` retourne 401 avec `AUTH_DEFAULT_ROLES: denied`
Quand le serveur HTTP démarre enfin, la probe HTTP GET `/api/v1/status` reçoit **401 Unauthorized** car toutes les routes API sont protégées par authentification.

**Fix :** Remplacement de toutes les probes httpGet par `tcpSocket: port: 1337` (PR #3062). TCP ne vérifie que si le port est ouvert, sans dépendance à l'auth.

---

## Actions préventives

| Action | Priorité | Statut |
|--------|----------|--------|
| Nightscout probes → tcpSocket sur port 1337 | Haute | ✅ PR #3062 |
| Nightscout sizing V-micro (128Mi) | Haute | ✅ PR #3057 |
| Nightscout priority class vixens-medium | Moyenne | ✅ PR #3060 |
| startupProbe 600s pour apps Node.js lentes | Haute | ✅ PR #3061 |
| Baisser vpa.min-memory excessifs (mylar, pyload) | Haute | ✅ PR #3058 |
| Audit cluster : autres pods avec vpa.min-memory trop élevé | Moyenne | Open |
| Audit cluster : apps avec AUTH qui utilisent httpGet probes | Moyenne | Open |

---

## Leçons

1. **`AUTH_DEFAULT_ROLES: denied` casse les probes HTTP** — utiliser `tcpSocket` ou un endpoint sans auth pour les probes dans les apps avec auth forte.
2. **Kyverno `sizing-v2-mutate` surpasse les patches kustomize** — modifier le LABEL de sizing, pas les resources directement.
3. **`vpa.min-memory` mal calibré provoque une inflation de mémoire irréversible** — surveiller les pods dont l'usage réel << requests (goldilocks Off-mode fournit les recommandations non-enforced).
4. **startupProbe >> initialDelaySeconds** pour les apps avec démarrage lent/variable — le failureThreshold donne une fenêtre précise sans bloquer la liveness une fois démarré.
5. **Nightscout sur nouveau PVC prend 3-5min** — index MongoDB créés from scratch.
