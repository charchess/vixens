# Post-Mortem : HA + Frigate iSCSI ext4 emergency_ro → migration XFS (2026-05-01)

**Sévérité :** P1 — Home Assistant non fonctionnel, Frigate en CrashLoop (264 restarts)  
**Durée :** ~4h (détection ~00:00 UTC+2, recovery Home Assistant ~00:51, Frigate ~02:30)  
**Nœuds affectés :** `sakapuss` (prod, HA), `sakapuss` (prod, Frigate)  
**Volumes affectés :** `homeassistant-config` (ext4, `/dev/sd?`), `frigate-config-pvc` + `frigate-cache-pvc` (ext4)

---

## Timeline

| Heure | Événement |
|-------|-----------|
| ~23:00 | HA entre en `emergency_ro` (ext4 I/O error sur iSCSI) — homeassistant pod en CrashLoop |
| 00:00 | Détection — DataAngel init container en cours de WAL restore (litestream replay) |
| 00:25 | WAL replay terminé (25 min) — démarrage de `PRAGMA integrity_check` sur 2.6 GB SQLite |
| 00:51 | integrity_check terminé (26 min) — rclone sync, boot HA → `2/2 Running`, https://homeassistant.truxonline.com 200 OK |
| ~01:00 | Investigation Frigate — 264 restarts, erreur ZMQ `ipc:///tmp/cache/comms: Address already in use` |
| ~01:15 | Root cause ZMQ identifié : socket IPC persistant sur PVC `frigate-cache-pvc` ext4 en `emergency_ro` |
| ~01:30 | Fix ZMQ : init container `cleanup-cache` ajouté (`rm -f /tmp/cache/comms /tmp/cache/.comms`) → PR #3152 |
| ~01:45 | Décision migration XFS pour les PVCs Frigate (ext4 emergency_ro cassé, XFS retourne EIO immédiatement) |
| ~02:00 | Migration PVCs Frigate vers `synelia-iscsi-xfs-retain` → PR #3153 |
| ~02:10 | Deadlock WaitForFirstConsumer détecté : `frigate-config-pvc` (wave 7) attend que `frigate-cache-pvc` (wave 0) soit Bound, mais `frigate-cache-pvc` attend un pod, qui attend `frigate-config-pvc` |
| ~02:15 | Déblocage manuel : `kubectl apply` de `frigate-config-pvc` pour briser le deadlock circulaire |
| ~02:20 | DataAngel restore Frigate depuis S3 MinIO |
| ~02:30 | Frigate `2/2 Running`, 0 restarts — https://frigate.truxonline.com 200 OK |
| ~01:00 | PR #3151 : `formatOptions: "-K"` ajouté aux StorageClasses XFS (skip BLKZEROOUT sur thin LUNs) |

---

## Root Cause

### 1. ext4 emergency_ro
Le filesystem ext4 d'iSCSI bascule en `read-only` silencieux sur erreur I/O, plutôt que de retourner EIO. Cela corrompt l'état des applications sans erreur explicite (ZMQ socket orphelin, base de données inaccessible).

### 2. ZMQ socket orphelin sur PVC
Le socket IPC `/tmp/cache/comms` de Frigate persiste sur le PVC `frigate-cache-pvc` entre les redémarrages. Si le PVC est en `emergency_ro`, le `rm` du socket échoue (EROFS), empêchant le binding ZMQ → crash immédiat au démarrage.

### 3. Deadlock WaitForFirstConsumer
`frigate-config-pvc` avait `argocd.argoproj.io/sync-wave: "7"`, empêchant ArgoCD de le créer avant que `frigate-cache-pvc` (wave 0) soit Bound. Mais `frigate-cache-pvc` utilise `WaitForFirstConsumer` → ne bind que quand un pod est schedulé → le pod ne peut pas scheduler car `frigate-config-pvc` n'existe pas encore → deadlock.

### 4. XFS StorageClass BLKZEROOUT hang
La StorageClass `synelia-iscsi-xfs-retain` ne passait pas `formatOptions: "-K"`, ce qui causait `mkfs.xfs` à appeler `BLKZEROOUT` sur les thin LUNs Synology → hang indéfini (BLKZEROOUT non supporté sur thin LUNs Synology). Corrigé en PR #3151.

---

## Fix

1. **PR #3151** : `formatOptions: "-K"` ajouté à `synelia-iscsi-xfs-retain` et `synelia-iscsi-xfs-delete` dans `apps/01-storage/synology-csi/base/storage-class.yaml`
2. **PR #3152** : Init container `cleanup-cache` ajouté à Frigate (`rm -f /tmp/cache/comms /tmp/cache/.comms`) avant `validate-config`
3. **PR #3153** : PVCs Frigate migrés de `synelia-iscsi-retain` (ext4) vers `synelia-iscsi-xfs-retain` (XFS)
4. **Déblocage manuel** : `kubectl apply` de `frigate-config-pvc` pour briser le deadlock ArgoCD wave/WaitForFirstConsumer

---

## Lessons Learned

### ext4 vs XFS sur iSCSI
- **ext4** : remonte silencieusement en `read-only` (`emergency_ro`) sur erreur I/O — applications continuent sans le savoir
- **XFS** : retourne `EIO` immédiatement sur erreur I/O — application crashe proprement, DataAngel peut restaurer
- **Décision** : migrer tous les PVCs critiques vers XFS progressivement (voir ADR)

### ZMQ socket IPC sur PVC persistant
Un socket IPC persistant sur PVC est une bombe à retardement si le PVC entre en `emergency_ro`. L'init container `cleanup-cache` le supprime avant le démarrage, mais il vaut mieux utiliser `emptyDir` pour les sockets IPC ou le répertoire `/dev/shm` (déjà en Memory). Solution long terme : Frigate devrait utiliser `/dev/shm` pour les sockets ZMQ.

### Deadlock ArgoCD wave + WaitForFirstConsumer
Si un PVC A (wave N) dépend d'un PVC B (wave 0) via `WaitForFirstConsumer`, et que le pod a besoin des deux, ArgoCD créera B mais ne créera pas A avant que B soit Bound — deadlock si B ne bind qu'avec un pod qui a besoin de A.

**Fix structurel** : supprimer `sync-wave` des PVCs ou utiliser `volumeBindingMode: Immediate` pour les PVCs de config statiques.

### Ordre de migration DataAngel
Migrer le PVC AVANT de réparer le ZMQ socket permet à DataAngel de restaurer depuis S3 sur un volume XFS propre.

---

## Action Items

- [ ] Documenter la stratégie de migration ext4 → XFS dans un ADR
- [ ] Supprimer `argocd.argoproj.io/sync-wave` de `frigate-config-pvc` (évite le deadlock)
- [ ] Envisager `emptyDir` ou `/dev/shm` pour les sockets ZMQ Frigate (pas de PVC)
- [ ] Inventorier les PVCs encore sur `synelia-iscsi-retain` (ext4) et planifier la migration XFS
