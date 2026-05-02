# Post-Mortem : Hydrus iSCSI Multi-Attach zombie session (2026-05-02)

**Sévérité :** P2 — Hydrus Client inaccessible depuis 18h+  
**Durée :** 18h+ (dernier restart 2026-05-01 19:49 UTC+2 → fix 2026-05-02 ~13:10 UTC)  
**Nœud affecté :** `poison` (session zombie), `powder` (pod schedulé mais bloqué)  
**Volume affecté :** `hydrus-client-config-pvc` (ext4, PV `pvc-40661ded-1800-48ad-b3fe-ddcf1ec302c5`)

---

## Timeline

| Heure | Événement |
|-------|-----------|
| 2026-05-01 19:49 | Pod hydrus-client recréé sur `powder` après force-delete |
| 19:49+ | Pod bloqué en `Init:0/1` — kubelet powder : `unmounted volumes=[config], unattached volumes=[config], context deadline exceeded` |
| 18h+ | kube-controller-manager (sur `poison`) : `VerifyVolumesAreAttached: nil spec for volume 87b05d98...` toutes les 60s |
| 18h+ | Aucun VolumeAttachment créé pour `pvc-40661ded` |
| 2026-05-02 12:48 | kube-controller-manager : `Multi-Attach error: volume already exclusively attached to "poison"` |
| 12:55 | Diagnostic confirmé : `87b05d98` dans `poison.status.volumesInUse` ET `poison.status.volumesAttached`, mais aucun pod sur poison ne l'utilise |
| 12:55 | CSI node pod (poison) : `NodeUnstageVolume` pour `87b05d98` toutes les ~2min, session iSCSI qui connect/disconnect |
| 12:58 | Pod debug privilégié créé sur poison → répertoire stale trouvé : `/var/lib/kubelet/plugins/kubernetes.io/csi/csi.san.synology.com/1d56e109.../globalmount/` |
| 12:58 | Répertoire stale supprimé (pas de mount actif) |
| 13:00 | `poison.status.volumesInUse` se vide automatiquement (kubelet détecte la suppression) |
| 13:01 | VolumeAttachment créé sur `powder` → `attached: true` en 39s |
| 13:01 | iSCSI login sur powder (90s de délai — session reconstituée) |
| 13:02 | DataAngel démarre — validation 4 bases SQLite (~35min total) |
| 13:37 | DataAngel terminé — pull image ghcr.io/hydrusnetwork/hydrus:v670 (15min, 893 MB) |
| 13:52 | Pod `2/2 Running`, 0 restarts |

---

## Root Cause

### Session iSCSI zombie sur `poison`

Le pod hydrus-client a précédemment tourné sur `poison`. Lors d'un redémarrage, le kubelet sur `poison` a tenté de démonter (`NodeUnstageVolume`) le volume CSI, mais le driver Synology a retourné `OK` sans supprimer le répertoire de staging kubelet :

```
/var/lib/kubelet/plugins/kubernetes.io/csi/csi.san.synology.com/1d56e109.../globalmount/
```

**Conséquence en cascade :**

1. Le répertoire existe → kubelet `poison` pense le volume toujours stagé → le signale dans `volumesInUse` → node status mis à jour → `poison.status.volumesAttached` conserve `87b05d98`

2. kube-controller-manager (attach-detach controller) voit le volume comme attaché à `poison` → refuse de créer un VolumeAttachment pour `powder` (violation RWO)

3. L'erreur `nil spec` se produit car le controller essaie de vérifier le volume dans son ASW (actual state of world) mais ne parvient pas à reconstruire la VolumeSpec depuis l'entrée stale du node status

4. Cycle infini : `NodeUnstageVolume` → session manquante (`doesn't exist`) → driver retourne OK → répertoire non supprimé → kubelet retry dans 2min → loop

### Pourquoi la session n'a pas été nettoyée

Le driver CSI Synology retourne `OK` sur `NodeUnstageVolume` quand la session iSCSI n'existe pas, sans supprimer le répertoire de staging. C'est un bug du driver qui devrait supprimer le répertoire même en l'absence de session active.

---

## Fix

```bash
# 1. Créer un pod debug privilégié sur poison
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: iscsi-cleanup-poison
  namespace: kube-system
spec:
  nodeName: poison
  tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
  volumes:
  - name: kubelet
    hostPath:
      path: /var/lib/kubelet
  containers:
  - name: cleanup
    image: busybox:1.37.0
    securityContext:
      privileged: true
    volumeMounts:
    - name: kubelet
      mountPath: /var/lib/kubelet
    command: ["rm", "-rf", "/var/lib/kubelet/plugins/kubernetes.io/csi/csi.san.synology.com/1d56e109.../"]
EOF

# 2. Supprimer le pod debug
kubectl delete pod -n kube-system iscsi-cleanup-poison

# 3. Attendre que kubelet vide volumesInUse (~60s)
# 4. VolumeAttachment créé automatiquement sur le bon nœud
```

---

## Diagnostic

Commandes clés pour identifier ce type de zombie :

```bash
# Vérifier si un PVC est signalé attaché à un nœud sans VolumeAttachment correspondant
kubectl get volumeattachment | grep <pvc-name>  # doit retourner une ligne
kubectl get node <node> -o jsonpath='{.status.volumesAttached}'  # vs VolumeAttachments

# Trouver l'erreur dans kube-controller-manager
kubectl logs -n kube-system kube-controller-manager-<leader> | grep "nil spec for volume"
kubectl logs -n kube-system kube-controller-manager-<leader> | grep "Multi-Attach error"

# Vérifier le répertoire stale sur le nœud (via pod debug privilégié)
ls /var/lib/kubelet/plugins/kubernetes.io/csi/csi.san.synology.com/
```

---

## Lessons Learned

### Bug driver CSI Synology : NodeUnstageVolume incomplet

Le driver ne supprime pas le répertoire de staging quand la session iSCSI n'existe plus. Cela laisse un répertoire orphelin qui fait croire au kubelet que le volume est toujours stagé.

**Workaround** : suppression manuelle via pod debug privilégié.

**Fix structurel** : contribuer un fix upstream au driver CSI Synology, ou créer un job de cleanup périodique qui vérifie les répertoires CSI staging sans session iSCSI active.

### Diagnostic de Multi-Attach zombie

L'erreur `nil spec for volume` dans le kube-controller-manager est le premier signal. Elle indique que le controller essaie de vérifier un volume dans son ASW mais ne peut pas reconstruire sa spec — souvent causé par un nœud ayant le volume dans `volumesAttached` sans VolumeAttachment correspondant.

Vérifier immédiatement :
1. `kubectl get volumeattachment` — le VolumeAttachment existe-t-il ?
2. `kubectl get node <node> -o jsonpath='{.status.volumesAttached}'` — sur quel nœud le volume est-il listé ?
3. Y a-t-il un pod actif sur ce nœud utilisant ce PVC ?

### Impact de la durée

18h de downtime pour Hydrus Client, une application non-critique. L'absence d'alerte proactive sur `Init:0/1` prolongé a retardé le diagnostic.

**Action** : ajouter une alerte Prometheus sur `kube_pod_init_container_status_running == 0` pendant > 30 minutes.

---

## Action Items

- [ ] Bug report upstream driver CSI Synology : `NodeUnstageVolume` doit supprimer le répertoire de staging même si la session iSCSI est absente
- [ ] Alerte : `kube_pod_init_container_status_running == 0` pendant > 30min (exclude dataangel restore)
- [ ] Script de diagnostic : détecter les volumes dans `volumesAttached` sans VolumeAttachment correspondant
- [ ] Procédure documentée dans `docs/troubleshooting/iscsi-zombie-session.md`
