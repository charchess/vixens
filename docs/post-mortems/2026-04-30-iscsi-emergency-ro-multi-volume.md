# Post-Mortem : iSCSI emergency_ro multi-volume (2026-04-30)

**Sévérité :** P1 — Home Assistant non fonctionnel, recorder KO, cascade DiskPressure cluster  
**Durée :** ~26h (détection initiale tardive + recovery prolongée)  
**Nœuds affectés :** `poison`, `pearl`, `peach`  
**Volumes affectés :** Home Assistant (`/dev/sde` sur poison), Jellyfin (`/dev/sdh` sur poison), Hydrus Client (`/dev/sdg` sur poison), Loki (`/dev/sdb` sur pearl), Netbird Management (`/dev/sdc` sur peach)

---

## Timeline

| Heure | Événement |
|-------|-----------|
| ~19:00 | Événement réseau/NAS probable (non confirmé) — sessions iSCSI interrompues au-delà du replacement_timeout (120s) |
| ~19:00 | ext4 détecte des erreurs I/O → remonte 3 volumes en `emergency_ro` |
| ~20:45 | HA recorder commence à logguer `OSError: [Errno 30] Read-only file system` |
| ~20:50 | Détection via inspection manuelle des logs HA |
| ~21:00 | Début de la recovery |
| ~21:10 | Jellyfin et Hydrus : e2fsck via pod privilégié → OK |
| ~21:20 | HA : zombie iSCSI session → logout/login via nsenter → e2fsck → OK |
| ~23:05 | Home Assistant 2/2 Running (DataAngel restore inclus) |

---

## Root Cause

Un événement **NAS-side** (redémarrage Synology ou maintenace?) a interrompu les sessions iSCSI au-delà du `replacement_timeout` par défaut (120s). L'initiateur iscsid a déclaré les sessions mortes, les I/Os en attente ont échoué, et ext4 a basculé 5 filesystems en `emergency_ro` / `shutdown`.

**Confirmation NAS-side** : 5 volumes sur 3 nœuds différents (`poison`, `pearl`, `peach`) ont été impactés simultanément, ce qui exclut une cause purement réseau locale.

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

1. **Détection tardive** : aucune alerte sur `node_filesystem_readonly`. → Issue #3135
2. **replacement_timeout trop court** : 120s insuffisant pour un reboot NAS (~2-3 min). → Issue #3136
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
