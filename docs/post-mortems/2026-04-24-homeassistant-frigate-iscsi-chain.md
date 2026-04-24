# Post-Mortem : Home Assistant + Frigate — Cascade iSCSI + migration local-path

**Date :** 2026-04-24  
**Durée de l'incident :** ~18:00 UTC → ~21:30 UTC (~3h30)  
**Cluster :** prod  
**Apps affectées :** `homeassistant`, `frigate`, `jellyfin`, `renovate`  
**Auteur :** Claude Code + charchess  
**Résultat final :** ✅ HA 2/2, Frigate 2/2, Jellyfin 1/1 — toutes Running

---

## 1. CE QUI S'EST PASSÉ

### Déclencheur initial — HA bloqué en Init:0/3

Home Assistant (prod) est passé en `Init:0/3` après un redémarrage du pod. La cause profonde : le LUN iSCSI `pvc-39c56cda` (HA config) avait une **session fantôme sur pearl** (nœud d'une ancienne exécution), bloquant le nouveau login sur peach (`max_sessions=1`).

### Cascade de problèmes iSCSI sur plusieurs nodes

En tentant de débloquer HA, plusieurs problèmes iSCSI additionnels ont été découverts :

1. **Pearl** : session fantôme sur session112 pour pvc-39c56cda → logout forcé via CSI pod
2. **Peach** : fichier de configuration iSCSI corrompu (`/var/lib/iscsi/nodes/pvc-9d89a9b3/.../default`, 36 lignes, tronqué milieu d'écriture) → bloquait TOUS les logins iscsiadm sur peach
3. **Peach** : fsck requis sur /dev/sdh (pvc-39c56cda, 250Gi) après déconnexion non-propre → ~40 minutes de fsck automatique par le CSI driver
4. **Poison** : VolumeAttachment stale après migration → supprimé manuellement
5. **Poison** : cascade de processus iscsiadm zombies (~20 processus D-state) + fichiers lock → iscsid tué + fichiers nettoyés
6. **Poison** : fichier corrompu pour pvc-2495ad1a (36 lignes) + répertoires IPv6 vides pour pvc-2d8775d6 → supprimés
7. **Poison (fin de session)** : fichiers pvc-b245bb50 (36 lignes) + 4 fichiers 0-octet dans répertoires fe80 → supprimés

### HA récupéré — puis Frigate bloqué

Après récupération d'HA sur poison, Frigate est passé en `Pending: 0/5 nodes available: 5 Insufficient memory` à cause de deux problèmes combinés :

**Problème 1 — PVC local-path ancrait Frigate sur poison**  
Le PVC `frigate-config-pvc` utilisait `local-path-delete` avec un PV ayant une `nodeAffinity` dure sur poison. Frigate ne pouvait donc se reschedule que sur poison — et pendant la migration, ce verrou a été levé mais sans résoudre le problème de mémoire.

**Problème 2 — Saturation mémoire du cluster**  
Après récupération d'HA (qui s'était déplacé de pearl vers poison), le cluster n'avait plus de nœud avec 6Gi de requests libres. La principale cause cachée : `openclaw` sur powder consomme 7.3 GiB réels mais ne déclare que 2Gi en requests (cap VPA à 2Gi), rendant powder "secrètement plein".

| Node | Requests | Réel |
|------|----------|------|
| poison | 89% | 88% |
| powder | 91% | 87% |
| phoebe | 87% | 63% |
| peach | 89% | 71% |
| pearl | 106% | 70% |

**Problème 3 — Deadlock ArgoCD PVC wave 0 vs Deployment wave 7**  
Après migration de `frigate-config-pvc` vers iSCSI (`synelia-iscsi-retain`, WaitForFirstConsumer), ArgoCD est entré en deadlock :
- PVC sans annotation = wave 0 → ArgoCD attend qu'il soit Bound avant d'appliquer wave 7
- Deployment en wave 7 → créerait le pod qui déclencherait le binding du PVC
- Le PVC ne peut jamais binder sans le pod → deadlock complet

---

## 2. CE QUI A RÉSOLU L'INCIDENT

### Récupération HA

1. Logout session fantôme pearl (pvc-39c56cda)
2. Suppression fichier corrompu pvc-9d89a9b3 sur peach via pod debug
3. Attente fsck automatique (~40 min) sur /dev/sdh sur peach
4. Suppression VolumeAttachment stale + nettoyage iscsiadm zombies sur poison
5. Suppression fichiers iSCSI corrompus sur poison

### Récupération Frigate

1. **Migration PVCs local-path → iSCSI** (PRs #3065 + #3066) : frigate-config-pvc, jellyfin-config-pvc, renovate-cache, bookshelf PVCs
2. **Réduction request mémoire Frigate** (PR #3067) : 6Gi → 2Gi (limit toujours 10Gi). Poison avait 2.5Gi libre après suppression du pod renovate échoué
3. **Patch direct du Deployment** pour briser le deadlock ArgoCD (wave 0 PVC bloquant wave 7 Deployment)
4. **Nettoyage fichiers iSCSI corrompus sur poison** (5 fichiers : 1×36 lignes + 4×0 octets dans répertoires fe80)
5. **Fix permanente deadlock** (PR #3068) : annotation `argocd.argoproj.io/sync-wave: "7"` sur frigate-config-pvc

---

## 3. CHRONOLOGIE

| Heure UTC | Événement |
|-----------|-----------|
| ~18:00 | HA détecté Init:0/3 prod |
| ~18:15 | Session fantôme pearl identifiée et déconnectée |
| ~18:20 | Fichier corrompu peach identifié, supprimé |
| ~18:25 | fsck démarré automatiquement sur /dev/sdh (peach) |
| ~19:05 | fsck terminé, HA monte sur poison → 2/2 Running |
| ~19:10 | Frigate détecté Pending (0/5 Insufficient memory) |
| ~19:30 | Root cause: local-path PVCs ancrent les pods → PRs #3065/#3066 |
| ~20:20 | PRs #3065/#3066 mergés, PVCs migrés, prod-stable mis à jour |
| ~20:50 | PR #3067 mergé (requests 6Gi→2Gi), prod-stable mis à jour |
| ~21:00 | ArgoCD deadlock identifié (wave 0 PVC vs wave 7 Deployment) |
| ~21:12 | Deployment patché directement → nouveau pod créé → PVC bind |
| ~21:15 | Fichiers iSCSI corrompus sur poison nettoyés (5 fichiers) |
| ~21:28 | Frigate 2/2 Running sur poison |
| ~21:30 | PR #3068 mergé (sync-wave fix), prod-stable final mis à jour |

---

## 4. ROOT CAUSES

### RC1 — Fichiers de configuration iSCSI corrompus persistants

Lors de crashs précédents, l'écriture du fichier `default` dans `/var/lib/iscsi/nodes/` est interrompue en plein milieu. Le fichier tronqué empêche iscsiadm de fonctionner pour TOUTES les cibles iSCSI sur ce node.

**Impact :** Bloque tous les montages CSI sur le node concerné.

**Mitigation déployée :** Le pod `iscsi-lock-cleanup` (DaemonSet) nettoie le fichier lock au démarrage, mais ne détecte pas les fichiers `default` corrompus.

**Action requise :** Améliorer le DaemonSet `iscsi-lock-cleanup` pour scanner et supprimer aussi les fichiers `default` tronqués/vides.

### RC2 — PVCs local-path ancrent les pods sur un nœud unique

`local-path` provisioner crée des PVs avec `nodeAffinity` permanente sur le nœud de provisionnement. Un pod ne peut donc se reschedule QUE sur ce nœud, même si celui-ci est défaillant.

**Impact :** Frigate, Jellyfin, Renovate étaient bloqués sur leurs nœuds respectifs.

**Correction déployée :** Migration vers `synelia-iscsi-retain` pour tous les PVCs concernés (PRs #3065/#3066/#3066).

**Règle :** Ne plus utiliser `local-path` pour des apps mobiles. Seul l'éphémère (cache pur, données recréables) peut utiliser local-path.

### RC3 — openclaw sous-déclare sa consommation mémoire (cap VPA 2Gi, usage réel 7-9Gi)

La cap VPA `maxAllowed.memory: 2Gi` pour openclaw force une request à 2Gi alors que l'usage réel est de 7-9 GiB. Le scheduler pense powder est moins plein qu'il ne l'est, ce qui a permis l'accumulation de workloads supplémentaires.

**Impact :** powder était "secrètement" saturé en mémoire réelle, faussant les décisions de scheduling pour tout le cluster.

**Action requise :** Revoir la cap VPA openclaw. Envisager d'augmenter la cap à 10Gi pour que le scheduler ait une vue honnête de la consommation.

### RC4 — Deadlock ArgoCD : PVC WaitForFirstConsumer en wave 0 bloque Deployment en wave 7

Un PVC `storageClassName: synelia-iscsi-retain` (WaitForFirstConsumer) sans annotation de sync-wave se retrouve en wave 0. ArgoCD attend qu'il soit Bound avant d'appliquer les resources de wave supérieure (Deployment). Mais le PVC ne peut binder que quand un pod (créé par le Deployment) est schedulé.

**Impact :** Frigate ne pouvait pas démarrer via ArgoCD après recréation du PVC.

**Correction déployée :** Annotation `argocd.argoproj.io/sync-wave: "7"` sur frigate-config-pvc (PR #3068).

**Règle :** Tout PVC `WaitForFirstConsumer` doit avoir le même sync-wave que son Deployment.

---

## 5. ACTIONS DE SUIVI

| Action | Priorité | Owner |
|--------|----------|-------|
| Améliorer iscsi-lock-cleanup : détecter et supprimer les fichiers `default` corrompus (≤100 octets) | P1 | |
| Audit de tous les PVCs local-path restants sur tous les clusters | P1 | |
| Revoir cap VPA openclaw (2Gi → 10Gi) pour honnêteté scheduler | P2 | |
| Vérifier sync-wave sur tous les PVCs WaitForFirstConsumer dans les overlays prod | P2 | |
| Documenter règle : WaitForFirstConsumer PVC = même sync-wave que son Deployment | P3 | |
