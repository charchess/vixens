# Post-Mortem : mongodb-shared-0 — Récupération PVC iSCSI + probe kill loop

**Date :** 2026-04-24  
**Durée de l'incident :** ~00:00 UTC → ~04:30 UTC (~4h30)  
**Cluster :** prod  
**Service impacté :** mongodb-shared (Nightscout)

---

## Résumé

Après suppression et recréation du PVC `pvc-24f68afa` (suite à un blocage du pod précédent), `mongodb-shared-0` est resté bloqué dans une boucle de redémarrages pendant ~4h30. Trois problèmes distincts se sont enchaînés.

---

## Chronologie

| Heure (UTC) | Événement |
|-------------|-----------|
| ~00:00 | PVC `pvc-b2d49d48` supprimé, nouveau PVC `pvc-24f68afa` créé |
| ~00:30 | NodeStageVolume bloqué (iscsiadm thundering herd sur lock.write) |
| ~00:34 | Sessions iSCSI nettoyées manuellement, NodeStageVolume réussi |
| ~00:41 | MongoDB démarre mais startup probe timeout en boucle (exit 100 puis mongosh 55s) |
| ~02:57 | PR #3051 mergé : `DO_NOT_TRACK=1` + egress CNP |
| ~03:52 | PR #3052 mergé : probe mongosh → bash `/dev/tcp` |
| ~04:05 | Filesystem `emergency_ro` puis `shutdown` sur `/dev/sdb` |
| ~04:15 | e2fsck + remontage sur `/dev/sdh` (session125) |
| ~04:26 | `mongodb-shared-0` atteint **3/3 Running** stable |

---

## Causes racines

### 1. Thundering herd iscsiadm (lock.write)
Plusieurs tentatives kubelet concurrentes de NodeStageVolume ont créé des files d'attente sur le fichier `lock.write` iSCSI, dépassant systématiquement le timeout gRPC de 2 minutes.

**Fix :** Tuer les processus iscsiadm zombies, effacer le lock.write manuellement depuis le namespace mount d'iscsid (`/proc/30294/root/run/lock/iscsi/lock.write`).

### 2. Mongosh startup probe timeout (Cilium default-deny + Talos seccomp)
La probe `mongosh --quiet --eval "db.adminCommand('ping')"` prenait **30–55 secondes** à s'exécuter :
- Cilium default-deny bloque les connexions egress de mongosh vers telemetry.mongodb.com (~25s timeout TCP silencieux)
- Node.js (mongosh) prend **34s** à démarrer sur `peach` (control-plane) même avec `--nodb`, probablement dû au profil seccomp strict de Talos

**Fix court terme :** `DO_NOT_TRACK=1` (PR #3051) réduit le délai à ~30s mais insuffisant.  
**Fix définitif :** Remplacement de la probe mongosh par un check TCP bash `exec 3<>/dev/tcp/127.0.0.1/27017` (PR #3052) — ~1ms, aucun appel réseau.

### 3. Filesystem emergency_ro / I/O errors sur /dev/sdb
Après plusieurs kills forcés du pod MongoDB (pendant journal actif), ext4 a détecté des erreurs et remonté le volume en lecture seule (`emergency_ro`), puis le device `/dev/sdb` a complètement disparu (SCSI layer drop après trop d'erreurs I/O).

La session iSCSI s'est automatiquement reconnectée sur un nouveau device (`session125 → /dev/sdh`).

**Fix :** `e2fsck -y /dev/sdh` pour récupérer le journal, remontage manuel sur le staging path, puis suppression du pod pour déclencher un redémarrage propre.

---

## Actions préventives

| Action | Priorité | Statut |
|--------|----------|--------|
| Remplacer toutes les probes mongosh par des checks TCP bash | Haute | ✅ Fait (PR #3052) |
| Ajouter egress DNS + S3 au CNP mongodb-shared | Haute | ✅ Fait (PR #3051) |
| Investiguer pourquoi Node.js prend 34s sur peach (seccomp Talos) | Moyenne | Open |
| Configurer NAS pour n'exposer que les portails IPv4 iSCSI | Moyenne | Open |
| Supprimer le target orphelin `pvc-b2d49d48` sur le NAS | Basse | Manuel (user) |

---

## Leçons

1. **Mongosh est inadapté comme probe en environnement Cilium default-deny** — toujours utiliser des checks TCP ou des outils sans dépendances réseau pour les probes dans ce cluster.
2. **Tuer un pod MongoDB en force corrompt le journal** — préférer un arrêt gracieux (`mongod --shutdown`) ou accepter que fsck sera nécessaire.
3. **Le SCSI layer peut droper un device après trop d'erreurs I/O** — la reconnexion iSCSI assigne un nouveau device name ; le staging path reste valide mais vide.
4. **Thundering herd iSCSI** → voir post-mortem 2026-04-17 pour la procédure de récupération.
