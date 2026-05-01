# Post-Mortem : iSCSI emergency_ro multi-volume (2026-04-30)

**Sévérité :** P1 — Home Assistant non fonctionnel, recorder KO, cascade DiskPressure cluster  
**Durée :** ~26h (événement réel 03:34, détection 20:45 soit +17h, recovery prolongée)  
**Nœuds affectés :** `poison`, `pearl`, `peach`  
**Volumes affectés :** Home Assistant (`/dev/sde` sur poison), Jellyfin (`/dev/sdh` sur poison), Hydrus Client (`/dev/sdg` sur poison), Loki (`/dev/sdb` sur pearl), Netbird Management (`/dev/sdc` sur peach)

---

## Timeline

| Heure | Événement |
|-------|-----------|
| **03:34:09** | **Tous les NICs du NAS tombent simultanément** (`eth3`, `eth1`, `eth0`, `eth0.204`) — événement réseau physique (switch?) |
| 03:34:41–03:35:09 | Avalanche de `iscsit_handle_nopin_response_timeout` côté NAS sur **toutes** les sessions iSCSI (tous PVCs, tous nœuds) |
| **03:36:03–03:36:13** | **NICs NAS remontent** — durée d'interruption : ~114 secondes |
| ~03:35–03:36 | ext4 détecte des erreurs I/O sur les 5 PVCs dont le `replacement_timeout` (120s) avait expiré → `emergency_ro` / `shutdown` |
| 06:01:49 | `iscsi_post_login_handler` — bulk reconnect des sessions récupérées |
| 19:00:12 | `synowin net ads test join fail` (non lié à l'incident iSCSI) |
| 19:04:56 | `syslog-ng reload` × 2 (maintenance DSM planifiée, non liée) |
| 19:44:07 | `talos-csi has session timeout` — conséquence : CSI tente d'accéder aux volumes déjà emergency_ro |
| **20:45** | **Première détection** : HA recorder logue `OSError: [Errno 30] Read-only file system` → gap de ~17h |
| 20:50 | Détection via inspection manuelle des logs HA |
| 21:00 | Début de la recovery |
| 21:10 | Jellyfin et Hydrus : e2fsck via pod privilégié → OK |
| 21:20 | HA : zombie iSCSI session → logout/login via nsenter → e2fsck → OK |
| 23:05 | Home Assistant 2/2 Running (DataAngel restore inclus) |

---

## Root Cause

**Confirmé par les logs NAS** (`/var/log/kern.log`, `/var/log/messages` sur Synelia) :

À **03:34:09**, tous les NICs du NAS (`eth3`, `eth1`, `eth0`, `eth0.204`) sont tombés simultanément pendant **~114 secondes** (réseau revenu à 03:36:03–13). Cet événement réseau physique — probablement un redémarrage de switch ou une micro-coupure électrique — a déclenché une avalanche de `iscsit_handle_nopin_response_timeout` côté target sur la totalité des sessions iSCSI actives.

Le `replacement_timeout` initiateur est de 120s. La panne réseau ayant duré ~114s, la marge était infime. Les 5 PVCs affectés sont ceux dont la session n'a pas réussi à être rétablie dans le délai (légère variance de timing, ordre des reconnexions, charge nœud) → `replacement_timeout` expiré → iscsid déclare les sessions mortes → ext4 `errors=remount-ro` → `emergency_ro` / `shutdown`.

**La majorité des PVCs a survécu** car leurs sessions se sont reconnectées juste à temps (confirmation : `iscsi_post_login_handler` en masse à 06:01:49).

**Les événements ~19:00 dans syslog sont non liés** : `synowin` domain-join check (AD), reload syslog-ng (maintenance DSM planifiée). Le `talos-csi has session timeout` à 19:44 est une conséquence — le CSI tentait d'accéder à des volumes déjà morts depuis 03:34.

**Gap de détection : ~17 heures** (03:34 → 20:45) — les pods restaient en état "Running" avec filesystems silencieusement en read-only.

**Cascade secondaire (2026-05-01)** : La recovery prolongée + les evictions mass sur `powder` (DiskPressure sur NVMe /var) ont provoqué :
- Réallocation de tous les pods powder sur les autres nœuds → surcharge mémoire cluster
- DiskPressure temporaire sur `peach` (ephemeral storage des pods evictés)
- CSI mount timeouts sur `peach` et `powder` (system I/O stress)
- OOM kills sur `homepage`, pods Pending (`nocodb`)

---

## Impact

- **Home Assistant** : recorder SQLite inaccessible → perte d'historique pendant la durée de l'incident. Interface web accessible mais toutes les automations écrivant en base échouaient silencieusement.
- **Jellyfin** : pod Running mais config inaccessible en écriture. Lecture des médias NFS non affectée.
- **Hydrus Client** : pod Running mais données non accessibles en écriture.
- **Loki** : pod en emergency_ro sur `pearl` — ingestion logs interrompue ~19h.
- **Netbird Management** : pod bloqué en Init (I/O error PVC) sur `peach` — management VPN inaccessible.
- **Cascade cluster** : DiskPressure `powder` + `peach` → OOM kills, pods Pending, CSI mount timeouts sur l'ensemble du cluster pendant ~2h.

---

## Recovery effectuée

### Pré-requis : pod privilégié

Le namespace `homeassistant` autorise les pods privilégiés. Pour contourner la mutation Kyverno qui injecte `allowPrivilegeEscalation: false`, utiliser l'annotation :

```yaml
annotations:
  vixens.io/explicitly-allow-root: "true"
```

### 1. Scale down + suspension ArgoCD

```bash
kubectl -n argocd patch app homeassistant --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}'
kubectl -n argocd patch app jellyfin --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}'
kubectl -n argocd patch app hydrus-client --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}'
kubectl -n homeassistant scale deployment homeassistant --replicas=0
kubectl -n media scale deployment jellyfin hydrus-client --replicas=0
```

### 2. Identifier les devices

```bash
talosctl -n poison read /proc/mounts | grep emergency_ro
# → /dev/sdg globalmount (Hydrus), /dev/sdh (Jellyfin), /dev/sde (HA)
```

### 3. e2fsck Jellyfin + Hydrus (sessions encore actives)

Pod privilégié avec `mountPropagation: Bidirectional` sur `/var/lib/kubelet` :

```bash
umount -l <podmount> <globalmount>
e2fsck -y /dev/sdg   # Hydrus — journal recovered, corrections faites (exit 1)
e2fsck -y /dev/sdh   # Jellyfin — session disparue, traité au scale up
```

### 4. HA : session iSCSI zombie

Le device `/dev/sde` était visible mais le mount échouait avec `Can't open blockdev`. La session iSCSI était dans un état zombie (kernel marquait le device "in use" sans qu'il soit monté).

Fix via nsenter dans le namespace du process iscsid Talos :

```bash
ISCSID_PID=$(talosctl -n poison processes | grep "/usr/local/sbin/iscsid" | awk '{print $2}')
IQN="iqn.2000-01.com.synology:Synelia.pvc-39c56cda-68a8-4c75-956a-f0fd10be68f0"
PORTAL="192.168.111.69:3260"

# Depuis un pod privilégié avec hostPID: true
nsenter -t ${ISCSID_PID} -m -n -- iscsiadm -m node -T "${IQN}" -p "${PORTAL}" --logout
nsenter -t ${ISCSID_PID} -m -n -- iscsiadm -m node -T "${IQN}" -p "${PORTAL}" -o delete
nsenter -t ${ISCSID_PID} -m -n -- iscsiadm -m discovery -t sendtargets -p "${PORTAL}"
nsenter -t ${ISCSID_PID} -m -n -- iscsiadm -m node -T "${IQN}" -p "${PORTAL}" --login
```

Suivi du restart ext-iscsid et CSI node pod :

```bash
talosctl -n poison service ext-iscsid restart
kubectl -n synology-csi delete pod synology-csi-node-<hash>
```

Puis e2fsck :

```bash
e2fsck -y /dev/sde
# → journal recovered, Free blocks count wrong (corrigé), exit 1 (corrections faites)
```

### 5. Scale up + re-enable ArgoCD

```bash
kubectl -n argocd patch app homeassistant --type merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":true,"prune":true}}}}'
kubectl -n argocd patch app jellyfin --type merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":true,"prune":true}}}}'
kubectl -n argocd patch app hydrus-client --type merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":true,"prune":true}}}}'
kubectl -n homeassistant scale deployment homeassistant --replicas=1
kubectl -n media scale deployment jellyfin hydrus-client --replicas=1
```

---

## Lessons Learned

1. **Détection tardive critique** : gap de ~17h entre l'incident (03:34) et la détection (20:45). Aucune alerte sur `node_filesystem_readonly`. Les pods en `emergency_ro` restent `Running` → Kubernetes ne voit rien. → Issue #3135
2. **replacement_timeout trop court** : 120s insuffisant — la panne a duré ~114s, laissant une marge de ~6s. Avec un switch qui redémarre ou une micro-coupure, c'est systématiquement fatal. → Issue #3136
3. **Procédure non documentée** : chaque recovery se fait à tâtons. → Issue #3137
4. **Un seul chemin réseau** : un seul chemin VLAN 111 → un point de défaillance unique. → Issue #3138 (multipath)
5. **La Kyverno annotation `vixens.io/explicitly-allow-root`** est nécessaire pour tout pod debug privilégié.
6. **DiskPressure powder pré-existant** : NVMe /var presque plein → moindre perturbation = cascade d'evictions cluster-wide. Corriger la taille /var (Terraform) est critique.
7. **VolumeAttachment stale** : après scale-down/up d'un pod avec PVC RWO, vérifier que l'ancien VA (sur l'ancien nœud) a bien été supprimé avant de scale up sur un nouveau nœud.
8. **CSI node plugin** : redémarrer le pod CSI sur le nœud impacté si les mount timeouts persistent après recovery iSCSI.
9. **WaitForFirstConsumer deadlock ArgoCD** : avec `storageClassName: synelia-iscsi-retain` (WaitForFirstConsumer), ArgoCD attend que le PVC soit Healthy avant de syncer le Deployment → deadlock. Fix : `kubectl apply -n <ns> -f <deployment.yaml>` pour forcer l'état git.

---

## Actions correctives

| # | Action | Issue | Priorité |
|---|--------|-------|----------|
| 1 | Alerte Prometheus `node_filesystem_readonly` | #3135 | P1 |
| 2 | `replacement_timeout = 300s` via MachineConfig | #3136 | P1 |
| 3 | Runbook iSCSI recovery + post-mortem | #3137 | P1 |
| 4 | ADR multipath iSCSI (investigation) | #3138 | P2 |
| 5 | StorageClass XFS pour nouvelles PVCs | #3139 | P3 |
| 6 | Extend powder NVMe /var (Terraform) | — | P1 (user scope) |
| 7 | Alerte Prometheus `node_disk_pressure` (kubelet) | — | P1 |
