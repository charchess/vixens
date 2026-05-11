# Post-Mortem : iSCSI total failure on peach — iscsid.conf missing from overlay (2026-05-11)

**Sévérité :** P1 — 4 pods inaccessibles (netbird-router, netbird-management, g4f, hydrus-client)  
**Durée :** ~12h (reboot peach 2026-05-10 ~20:00 UTC → fix 2026-05-11 ~08:00 UTC)  
**Nœud affecté :** `peach` (192.168.111.191) — rebooté pour upgrade RAM  
**Volumes affectés :** 4 PVCs iSCSI (pvc-3dfcab97, pvc-8fdd58fa, pvc-be76d48b, pvc-40661ded)

---

## Timeline

| Heure (UTC) | Événement |
|-------------|-----------|
| 2026-05-10 ~20:00 | Reboot de peach pour upgrade RAM DDR5 |
| 20:05 | peach rejoint le cluster — ext-iscsid redémarre, overlay upper dir vide |
| 20:05+ | Pods reschedule sur peach (netbird-router, netbird-management, g4f, hydrus-client) |
| 20:05+ | Tous les pods en `ContainerCreating` — kubelet bloqué sur NodeStageVolume |
| 20:05+ | CSI plugin logs : `iscsiadm: can't open iscsid.startup configuration file /etc/iscsi/iscsid.conf` |
| 2026-05-11 ~02:00 | Investigation démarrée — logs CSI plugin analysés |
| 02:15 | Root cause identifié : `/etc/iscsi/iscsid.conf` absent de l'overlay iscsid sur peach |
| 02:30 | iscsid.conf créé manuellement dans l'overlay upper dir via pod debug |
| 02:45 | 4e node record iSCSI (pvc-40661ded) ajouté dans l'overlay (manquant) |
| 03:00 | 4 sessions iSCSI loggées manuellement via `iscsiadm --login` |
| 03:05 | netbird-router : `Running` |
| 03:07 | netbird-management : `2/2 Running` |
| 03:10 | g4f : CSI volumes montés, startup probe en cours |
| 03:15 | hydrus-client : Init:0/1 (dataangel S3 restore en cours, iSCSI OK) |
| ~04:30 | hydrus-client : `2/2 Running` (restore terminé) |
| 08:00 | PR #3191 créé avec fix permanent (`node.yaml`) |

---

## Root Cause

### Architecture overlay ext-iscsid sur Talos Linux

Sur Talos Linux, le service iSCSI (`ext-iscsid`) tourne dans son propre namespace de montage via un overlay filesystem :

- **Mount point** : `/usr/local/lib/containers/iscsid/` (vue merged)
- **Lower layer** : `/usr/local/lib/containers/iscsid/` (image OCI ext-iscsid) — contient `etc/iscsi/iscsid.conf` + `etc/iscsi/nodes/` vide
- **Upper layer** : `/system/overlays/usr-local-lib-containers-iscsid-diff/` (partition state, persistante entre reboots) — accumule les node records créés par le CSI driver et les fichiers écrits par iscsid lui-même (hosts, resolv.conf)

**`iscsid.conf` EST dans la lower layer** — les nœuds pearl et powder ont un upper dir vide et lisent `iscsid.conf` depuis la lower layer sans problème. L'upper dir persiste entre les reboots normaux (partition state).

### Vraie root cause : répertoire opaque dans l'upper dir

Le répertoire `/system/overlays/usr-local-lib-containers-iscsid-diff/etc/iscsi/` sur peach avait probablement l'attribut **opaque** (`trusted.overlay.opaque = "y"`), ce qui bloque l'accès à tous les fichiers de la lower layer correspondante — y compris `iscsid.conf`.

Un répertoire devient opaque dans un overlay quand il est **supprimé puis recréé** à l'intérieur du namespace. Cela survient lors d'opérations de recovery agressives : si une procédure précédente a fait `rm -rf /etc/iscsi` puis `mkdir -p /etc/iscsi/nodes` à l'intérieur du namespace iscsid (via nsenter ou iscsadm), le kernel overlay a marqué le nouveau répertoire comme opaque, cachant definitivement la lower layer.

L'attribut opaque survit aux reboots (l'upper dir est persistant). Après le reboot RAM upgrade, peach redémarre avec l'upper dir intact, l'opaque est toujours là, et `iscsid.conf` reste invisible dans la vue merged.

Erreur dans les logs du CSI plugin :
```
[ERROR] [driver/initiator.go:194] Failed in discovery of the target:
  iscsiadm: can't open iscsid.startup configuration file /etc/iscsi/iscsid.conf
```

### Pourquoi l'incident n'a pas été détecté avant le reboot

L'attribut opaque existait peut-être déjà avant le reboot, mais iscsid avait ses sessions iSCSI maintenues en mémoire (le daemon n'a pas besoin de relire `iscsid.conf` pour les sessions existantes). Au reboot, iscsid repart de zéro et essaie d'ouvrir le fichier — qui n'est plus visible.

**L'erreur iscsiadm n'est pas propagée visiblement** : kubelet affiche `context deadline exceeded` plutôt que l'erreur iscsadm réelle — ce qui a retardé le diagnostic.

---

## Fix Immédiat (manuel)

```bash
# 1. Créer pod debug avec accès host-root
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: iscsi-db-init
  namespace: kube-system
spec:
  nodeName: peach
  hostPID: true
  tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
  volumes:
  - name: overlay
    hostPath:
      path: /system/overlays/usr-local-lib-containers-iscsid-diff
  containers:
  - name: init
    image: busybox:1.37
    securityContext:
      privileged: true
    volumeMounts:
    - name: overlay
      mountPath: /overlay
    command: ["sleep", "3600"]
EOF

# 2. Créer iscsid.conf dans l'overlay
kubectl exec -n kube-system iscsi-db-init -- sh -c '
mkdir -p /overlay/etc/iscsi
cat > /overlay/etc/iscsi/iscsid.conf << "CONF"
iscsid.startup = /usr/local/sbin/iscsid
node.startup = manual
node.session.timeo.replacement_timeout = 120
node.conn[0].timeo.login_timeout = 30
node.conn[0].timeo.logout_timeout = 15
node.conn[0].timeo.noop_out_interval = 5
node.conn[0].timeo.noop_out_timeout = 5
node.conn[0].iscsi.MaxRecvDataSegmentLength = 262144
node.session.iscsi.InitialR2T = No
node.session.iscsi.ImmediateData = Yes
node.session.iscsi.FirstBurstLength = 262144
node.session.iscsi.MaxBurstLength = 16776192
node.session.cmds_max = 128
node.session.queue_depth = 32
node.session.err_timeo.abort_timeout = 15
node.session.err_timeo.lu_reset_timeout = 30
node.session.err_timeo.tgt_reset_timeout = 30
discovery.sendtargets.auth.authmethod = None
discovery.sendtargets.timeo.login_timeout = 30
discovery.sendtargets.timeo.auth_timeout = 45
discovery.sendtargets.timeo.active_timeout = 30
CONF
echo "iscsid.conf created"
'

# 3. Ajouter les node records manquants dans l'overlay
# (chaque PVC schedulé sur peach pour la première fois après reboot)
# Format : /overlay/etc/iscsi/nodes/<iqn>/192.168.111.69,3260,1/default

# 4. Login iSCSI via le CSI plugin (nsenter dans le namespace iscsid)
ISCSID_PID=$(kubectl exec -n synology-csi <csi-node-pod> -c synology-csi-plugin -- pgrep iscsid | head -1)
kubectl exec -n synology-csi <csi-node-pod> -c synology-csi-plugin -- \
  /csibin/iscsiadm -m node \
  -T iqn.2000-01.com.synology:Synelia.pvc-<UUID> \
  -p 192.168.111.69:3260 --login

# 5. Nettoyage
kubectl delete pod -n kube-system iscsi-db-init
```

---

## Fix Permanent

**Aucun changement de code dans vixens.** L'incident est spécifique à peach (répertoire opaque dans l'upper dir persistant).

**Fix one-shot sur peach** : créer `iscsid.conf` dans l'upper dir (fait manuellement lors de la recovery) suffit à restaurer la visibilité — le fichier dans l'upper dir prend la précédence sur l'opaque et permet à iscsadm de fonctionner.

**Fix structurel** : supprimer l'attribut opaque du répertoire `etc/iscsi/` dans l'upper dir, ce qui permettrait de voir à nouveau `iscsid.conf` depuis la lower layer sans avoir le fichier en double. Mais c'est cosmétique — le fichier en double ne cause pas de problème fonctionnel.

```bash
# Diagnostic : vérifier si le répertoire est opaque
kubectl debug node/<node> -it --image=busybox -- \
  sh -c 'cat /proc/filesystems'  # vérifier support overlay
# Via pod debug avec accès state partition :
# getfattr -n trusted.overlay.opaque /host/system/overlays/usr-local-lib-containers-iscsid-diff/etc/iscsi/
# Si retourne "y" → répertoire opaque → bloque lower layer
```

**À ne pas faire** : créer `iscsid.conf` depuis un init container Kubernetes à chaque démarrage. `iscsid.conf` est fourni par la lower layer de l'extension Talos ext-iscsid — l'écrire depuis K8s bypass le modèle de configuration Talos et écrase silencieusement la version bundlée à chaque mise à jour du CSI node pod.

---

## Problème Secondaire : VPA injecte 64Mi sur le CSI plugin

Lors de l'investigation, un second problème a été identifié mais non résolu dans ce fix :

**Symptôme** : Le pod `synology-csi-plugin` a `memory limit: 64Mi` en live malgré `256Mi` dans le manifest.

**Cause** : Le VPA admission controller (`verticalPodAutoscaler`) injecte des limites basées sur l'historique des métriques. Le namespace `synology-csi` a le label `goldilocks.fairwinds.com/enabled: "true"`, et un VPA Kyverno-généré (`sizing-vpa-generate`) cible le DaemonSet `synology-csi-node`.

**Impact** : Lors d'un reboot de nœud, 4 appels `NodeStageVolume` concurrents → 4 processus `iscsadm` simultanés (8-11 MB chacun) + overhead plugin → OOM kill (exit 137). La montée en charge provoquait des redémarrages en boucle jusqu'à ce que les sessions iSCSI soient pré-établies.

**Workaround actuel** : Les sessions iSCSI pré-existantes (de la session manuelle) survivent aux redémarrages du CSI plugin. Le plugin n'a besoin que de les *vérifier* (faible mémoire) plutôt que de les *créer* (mémoire élevée).

**Fix long terme** : À définir — soit désactiver Goldilocks pour `synology-csi`, soit augmenter le minimum dans la resource policy VPA.

---

## Lessons Learned

### 1. L'overlay upper dir iscsid est persistant entre reboots

`/system/overlays/usr-local-lib-containers-iscsid-diff/` est sur la **partition state** de Talos — elle survit aux reboots. Les nœuds sains ont un upper dir vide et lisent `iscsid.conf` depuis la lower layer (bundlée avec ext-iscsid). Les attributs xattr (comme opaque) survivent aussi — c'est ce qui a rendu l'incident invisible jusqu'au reboot.

### 2. Un répertoire opaque dans l'upper dir bloque silencieusement la lower layer

Overlay filesystem : si un répertoire dans l'upper dir a l'attribut `trusted.overlay.opaque = "y"`, **tous** les fichiers de la lower layer correspondante deviennent invisibles. Il n'y a aucun log, aucune erreur — ils n'existent simplement plus dans la vue merged. Cela arrive quand un répertoire est supprimé puis recréé à l'intérieur du namespace overlay.

**Diagnostic rapide :** si `iscsid.conf` manque mais le lower layer `/usr/local/lib/containers/iscsid/etc/iscsi/iscsid.conf` existe → chercher un opaque sur le upper dir.

### 3. L'erreur iscsiadm réelle est masquée par kubelet

Kubelet logue `context deadline exceeded` ou `failed to stage volume`. L'erreur réelle est dans les logs du container CSI plugin :

```bash
kubectl logs -n synology-csi <csi-node-pod> -c synology-csi-plugin | grep -i "iscsi\|failed\|error"
```

Toujours commencer par les logs CSI plugin, pas `kubectl describe pod`.

### 4. Ne pas écrire la config OS depuis Kubernetes

`iscsid.conf` est fourni par Talos ext-iscsid. L'écrire depuis un init container K8s n'est pas la bonne approche — ça bypass le modèle de configuration Talos et écrase silencieusement la version bundlée. Si un fichier de la lower layer est absent ou masqué, le bon endroit pour corriger est le nœud Talos (machine.files ou suppression du whiteout), pas le CSI driver.

### 5. Les opérations de recovery inside le namespace iscsid créent des whiteouts

Toute opération `rm` ou `rmdir` exécutée **à l'intérieur** du namespace iscsid (via nsenter ou dans le container iscsid) peut créer des whiteouts dans l'upper dir. À éviter lors des recoveries iSCSI — préférer les opérations via le chemin hostPath de l'upper dir (`/host/system/overlays/...`).

---

## Action Items

- [x] **Recovery peach** : iscsid.conf créé manuellement dans l'upper dir — peach opérationnel
- [ ] **Supprimer l'opaque sur peach** : `setfattr -x trusted.overlay.opaque /system/overlays/usr-local-lib-containers-iscsid-diff/etc/iscsi/` (cosmétique — fonctionnel déjà OK)
- [ ] **VPA memory injection** : désactiver Goldilocks pour namespace `synology-csi` OU augmenter minimum mémoire dans VPA resource policy pour `synology-csi-plugin`
- [ ] **Runbook** : documenter procédure de diagnostic opaque overlay dans `docs/troubleshooting/iscsi-opaque-overlay.md`
- [ ] **Alerte** : pod `ContainerCreating` > 5min avec PVC iSCSI → page oncall
- [ ] **Procédure recovery** : explicitement interdire `rm` inside namespace iscsid — utiliser uniquement le chemin hostPath de l'upper dir
