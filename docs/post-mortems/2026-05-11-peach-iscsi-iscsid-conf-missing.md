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

Sur Talos Linux, le service iSCSI (`ext-iscsid`) tourne dans son propre namespace de montage via un overlay filesystem. L'overlay se compose de :

- **Lower layer (base image)** : image OCI contenant `open-iscsi`. Contient `/etc/iscsi/nodes/` avec les 255 enregistrements de targets iSCSI (87 targets × 3 portals) créés lors des connexions précédentes. **Ne contient pas `iscsid.conf`.**
- **Upper layer (runtime)** : répertoire inscriptible sur le nœud à `/system/overlays/usr-local-lib-containers-iscsid-diff/`. Vide après chaque reboot.

Le CSI plugin Synology utilise `/csibin/iscsiadm`, un wrapper qui fait `nsenter` dans le namespace de montage d'iscsid pour exécuter les commandes iSCSI. Toute commande `iscsiadm` (discovery, login, etc.) requiert que `/etc/iscsi/iscsid.conf` existe dans ce namespace.

### Pourquoi le fichier manque

`iscsid.conf` n'est pas dans la base image de l'extension Talos `ext-iscsid`. L'upper layer est ephémère (non persisté entre reboots). Résultat : après chaque reboot, `iscsid.conf` est absent, et **toute opération iSCSI échoue silencieusement** du point de vue du kubelet (qui voit juste `NodeStageVolume` qui timeout).

Erreur exacte dans les logs du CSI plugin :
```
[ERROR] [driver/initiator.go:194] Failed in discovery of the target:
  iscsiadm: can't open iscsid.startup configuration file /etc/iscsi/iscsid.conf
```

### Pourquoi l'incident n'a pas été détecté plus tôt

1. **Les autres nœuds (pearl, powder, poison) fonctionnent** car leur overlay upper dir contient déjà `iscsid.conf` — créé lors d'installations/mises à jour antérieures via des chemins non tracés
2. **peach avait un historique plus récent** : son overlay upper dir avait probablement déjà `iscsid.conf` avant le reboot, mais l'upgrade RAM impliquait peut-être un reset de l'overlay (ou peach n'avait jamais eu le fichier depuis sa reconfiguration)
3. **L'erreur iscsiadm n'est pas propagée visiblement** : kubelet affiche `context deadline exceeded` plutôt que l'erreur iscsadm réelle

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

## Fix Permanent (GitOps)

PR #3191 — `apps/01-storage/synology-csi/base/node.yaml`

L'init container `iscsi-lock-cleanup` a été étendu pour créer `iscsid.conf` dans l'overlay upper dir à chaque démarrage du pod CSI node :

```yaml
initContainers:
  - name: iscsi-lock-cleanup
    image: busybox:1.37
    command:
      - sh
      - -c
      - |
        rm -f /host/run/lock/iscsi/lock.write && echo "iSCSI stale lock cleared"
        OVERLAY=/host/system/overlays/usr-local-lib-containers-iscsid-diff
        mkdir -p ${OVERLAY}/etc/iscsi
        cat > ${OVERLAY}/etc/iscsi/iscsid.conf << 'EOF'
        iscsid.startup = /usr/local/sbin/iscsid
        node.startup = manual
        # ... paramètres open-iscsi standards
        EOF
        echo "iSCSI iscsid.conf ensured in overlay"
    securityContext:
      privileged: true
    volumeMounts:
      - name: host-root
        mountPath: /host
```

**Pourquoi ça marche :** YAML strip l'indentation commune (14 espaces), donc le heredoc `EOF` se retrouve en colonne 0 dans le script bash — terminaison correcte. Le fichier est idempotent (écrasé à chaque redémarrage de pod).

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

### 1. L'overlay iscsid est ephémère sur Talos

Tout fichier écrit dans `/system/overlays/usr-local-lib-containers-iscsid-diff/` est perdu au reboot. **La configuration runtime d'iscsid doit être injectée au démarrage du CSI node pod**, pas assumée présente.

**Action :** PR #3191 résout ce point de façon permanente.

### 2. L'erreur iscsiadm réelle est masquée par kubelet

Kubelet logue `context deadline exceeded` ou `failed to stage volume`. L'erreur réelle (iscsadm) est dans les logs du container CSI plugin :

```bash
kubectl logs -n synology-csi <csi-node-pod> -c synology-csi-plugin | grep -i "iscsi\|failed\|error"
```

**Action :** Toujours vérifier les logs CSI plugin en premier lors d'un échec de montage iSCSI, pas seulement `kubectl describe pod`.

### 3. Les 255 records base image ne sont pas dans l'upper dir

Confusion initiale : `iscsadm -m node` affiche 261 entries sur peach, mais elles viennent de la **lower layer** (base image). L'upper dir ne contient que les records créés localement. Supprimer l'upper dir n'aide pas pour les records existants — ils restent disponibles via la lower layer.

### 4. Le reboot post-upgrade matériel nécessite une validation iSCSI

Avant de marquer un nœud comme sain après reboot, vérifier :
```bash
# Depuis un pod debug sur le nœud
iscsiadm -m session   # Sessions actives
iscsiadm -m discoverydb   # DB de discovery accessible (nécessite iscsid.conf)
```

---

## Action Items

- [x] **PR #3191** : iscsid.conf créé dans l'overlay à chaque démarrage CSI node (fix permanent)
- [ ] **VPA memory injection** : désactiver Goldilocks pour namespace `synology-csi` OU augmenter minimum mémoire dans VPA resource policy pour `synology-csi-plugin`
- [ ] **Runbook** : documenter `docs/troubleshooting/iscsi-iscsid-conf-missing.md` avec procédure de diagnostic rapide
- [ ] **Alerte** : pod `ContainerCreating` > 5min avec PVC iSCSI → page oncall (Prometheus alerting sur `kube_pod_status_phase{phase="Pending"}` + PVC storageclass iSCSI)
- [ ] **Validation post-reboot** : ajouter check `iscsid.conf` présent dans la procédure de validation après reboot nœud
