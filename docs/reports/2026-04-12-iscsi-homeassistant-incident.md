# Rapport d'Incident : Perte de connectivité iSCSI — Home Assistant (Production)

**Date :** 12 Avril 2026  
**Durée :** ~7h (00:07 → 07:30 environ)  
**Statut :** Résolu  
**Sévérité :** P2 — Service indisponible (Home Assistant down)

---

## Résumé

À 00:07, le LUN iSCSI du PVC `homeassistant-config` (250 Gi, `pvc-39c56cda`) a perdu sa connexion brutalement sur le nœud `powder`. Le kernel ext4 a détecté des erreurs d'écriture critiques et a fait un shutdown d'urgence du filesystem. Malgré la reconnexion iSCSI automatique à 00:11, le filesystem était corrompu (journal dirty) et refusait de se monter (`exit status 32`). Home Assistant est resté down pendant ~7h jusqu'à résolution manuelle.

---

## Timeline

| Heure | Événement |
|-------|-----------|
| 00:07 | `device offline error, dev sdd` — session iSCSI coupée brutalement |
| 00:07 | `EXT4-fs (sdd): failed to convert unwritten extents — potential data loss!` |
| 00:07 | `EXT4-fs (sdd): shut down requested (2)` — kernel shutdown du filesystem |
| 00:07 | `JBD2: I/O error when updating journal superblock for sdd-8` — journal corrompu |
| 00:11 | Nouvelle session iSCSI établie — `sdc` reconnecté |
| 00:11 | Kubelet tente de remonter → `mount failed: exit status 32` (fs dirty, refuse) |
| 00:11+ | DataAngel startup probe en boucle (503) — 8721+ tentatives sur 5h |
| ~07:00 | Détection de l'incident |
| ~07:10 | Diagnostic kernel dmesg — identification cause racine |
| ~07:15 | Intervention DSM Synology — LUN remis en ligne |
| 07:14 | Nouvelle session iSCSI : `sd 5:0:0:1: [sdd] Attached SCSI disk` |
| 07:14 | `mount failed: exit status 32` — journal toujours dirty après reconnect |
| ~07:20 | Tentative fsck via pod privilégié — bloqué par stale mounts CSI |
| ~07:23 | `talosctl service kubelet restart` sur `powder` |
| 07:23 | `EXT4-fs (sdd): mounted filesystem r/w` — journal replay automatique réussi |
| 07:27 | DataAngel restore depuis MinIO — OK, pas d'I/O errors |
| 07:30 | Home Assistant `2/2 Running` |

---

## Cause Racine

**Déconnexion iSCSI brutale** du LUN `homeassistant-config` sur le nœud `powder`. La cause exacte côté Synology DSM n'est pas identifiée (problème réseau, overload du NAS, bug firmware ?).

**Note :** C'est la **deuxième déconnexion iSCSI en 2 jours** sur le même nœud :
- 10 Avril 16:53 : `sdc` (autre LUN) — `device offline error` → remontage automatique OK
- 12 Avril 00:07 : `sdd` (homeassistant-config) — remontage **impossible** (journal corrompu)

La différence entre les deux incidents : l'arrêt brutal lors de writes actifs sur le journal ext4 → inode 44, JBD2 superblock corrompu → ext4 refuse de monter sans fsck ou journal replay.

---

## Procédure de Résolution

1. **Intervention DSM** (utilisateur) — LUN remis en ligne dans iSCSI Manager
2. **Scale down HA** : `kubectl scale deployment -n homeassistant homeassistant --replicas=0`
3. **Suppression VolumeAttachment** : `kubectl delete volumeattachment csi-...`
4. **Suppression pod fsck** stale (mounts bloquants dans kubelet staging path)
5. **Restart kubelet** : `talosctl -n 192.168.111.193 service kubelet restart`
   - Déclenche le journal replay ext4 automatique au remontage
   - `EXT4-fs (sdd): mounted filesystem r/w` confirmé
6. **Scale up HA** : `kubectl scale deployment -n homeassistant homeassistant --replicas=1`
7. DataAngel restore depuis MinIO → HA boot normal

---

## Ce Qui a Bien Fonctionné

- **DataAngel** : backup complet disponible dans MinIO, restore propre en ~4 min
- **Diagnostic rapide** via `talosctl dmesg` — cause identifiée en <5 min
- **Pas de perte de données** : filesystem récupéré intégralement via journal replay

---

## Ce Qui a Posé Problème

- **Stale mounts CSI** : après suppression du VolumeAttachment et force-delete du pod, les staging mounts kubelet restaient (`/var/lib/kubelet/plugins/.../globalmount`). e2fsck bloqué.
- **e2fsck impossible sur Talos** : pas de `umount` userspace sur le host, `nsenter` ne peut pas exécuter les binaires du container dans le mount namespace hôte.
- **Solution finale contre-intuitive** : restart kubelet (pas fsck manuel) → journal replay automatique ext4 au montage.

---

## Actions de Suivi

- [ ] **Vérifier la santé des disques Synology** — Storage Manager → HDD/SSD → Health Info (SMART)
- [ ] **Investiguer les logs Synology** autour de 00:07 et 16:53 le 10 avril — pourquoi les LUNs tombent ?
- [ ] **Configurer des alertes iSCSI** sur le NAS (DSM notification si LUN offline)
- [ ] **Envisager `iSCSI multipath`** pour les LUNs critiques (HA) si le NAS le supporte
- [ ] **Documenter la procédure de recovery** dans `docs/troubleshooting/iscsi-recovery.md`

---

## Leçons Apprises

1. **Restart kubelet = journal replay** — sur Talos, si ext4 refuse de monter (`exit status 32`), restart kubelet force le journal recovery automatique. Plus rapide que fsck manuel.
2. **Stale mounts CSI** — force-delete pod + delete VolumeAttachment ne suffit pas à nettoyer les staging mounts kubelet. Nécessite un restart kubelet.
3. **Deux déconnexions iSCSI en 2 jours** = signal fort d'un problème sous-jacent à investiguer (NAS, réseau, ou firmware).
4. **DataAngel a prouvé sa valeur** — restore complet disponible, aucune perte de données même avec journal corrompu.
