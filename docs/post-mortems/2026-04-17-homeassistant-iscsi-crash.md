# Post-Mortem : Home Assistant — Crash iSCSI + récupération LUN

**Date :** 2026-04-17  
**Durée de l'incident :** ~16:49 UTC → ~17:47 UTC (~1h)  
**Cluster :** prod  
**App affectée :** `homeassistant` (namespace `homeassistant`)  
**Node initial :** powder | **Node final :** peach  
**Auteur :** Claude Code + charchess  
**Résultat final :** ✅ `2/2 Running`, `200 OK`, données intactes

---

## 1. CE QUI S'EST PASSÉ

### Déclencheur

À **~16:49 UTC**, le NAS Synology (192.168.111.69) a perdu plusieurs sessions iSCSI simultanément. Les logs kernel des nodes montrent `device offline error` sur plusieurs devices (`/dev/sda`, `/dev/sdf`, etc.). La cause côté NAS est inconnue (storage check automatique ? overload ? redémarrage partiel ?).

### Cascade

1. **Session iSCSI HA coupée** sur powder en plein I/O → le kernel voit le block device disparaître.
2. **NodeStageVolume déclenché** par kubelet pour re-monter le LUN. Le NAS répond, mais **la session iSCSI est à nouveau interrompue pendant l'e2fsck** → timeout gRPC CSI → kubelet marque le montage échoué.
3. **Loop infernale** : kubelet retente avec backoff exponentiel, mais chaque tentative timeout avant que le NAS soit stable → le pod reste bloqué en `Init:0/3`.
4. **DSM port 5000 inaccessible** (`i/o timeout`) pendant la phase de récupération du NAS → le CSI controller ne peut plus provisionner de nouveaux LUNs.

### Tentatives de récupération échouées

- `e2fsck` manuel depuis un pod debug sur powder → échoué (`e2fsck cannot continue`, processus fantôme bloquant `/dev/sda`)
- Cordon de powder → le pod se déplace sur peach, mais le NAS n'est toujours pas stable → mêmes timeouts NodeStageVolume

---

## 2. CE QUI A RÉSOLU L'INCIDENT

### Filesystem réparé automatiquement

À **17:01 UTC**, peach avait réussi un `NodeStageVolume` propre sur ce LUN (confirmé dans dmesg : `EXT4-fs (sda): mounted filesystem` à 16:59 → `unmounting filesystem` à 17:01). **Le filesystem était réparé à ce moment-là** sans intervention manuelle.

### Récupération du PV (Retain policy)

Le StorageClass `synelia-iscsi-retain` utilise `reclaimPolicy: Retain` → le PV et le LUN survivent à la suppression du PVC.

1. Suppression du PVC `homeassistant-config` (géré par ArgoCD)
2. Clear du `claimRef` sur le PV `pvc-39c56cda` (état `Released` → disponible)
3. ArgoCD re-crée le PVC → il se rebind sur le même PV (même LUN `0d605bfe-6643-40cb-b28f-aba0d67a05ff`)

**Aucun nouveau LUN n'a été créé. Les données sont intactes.**

### Déblocage iSCSI manuel

Le `NodeStageVolume` continuait à timeout sur peach malgré le NAS revenu (le port 3260 était ouvert, mais le login iSCSI n'arrivait pas dans le délai CSI gRPC). Solution :

```bash
kubectl -n synology-csi exec synology-csi-node-cl52h -c synology-csi-plugin -- \
  iscsiadm -m node \
  -T iqn.2000-01.com.synology:Synelia.pvc-39c56cda-68a8-4c75-956a-f0fd10be68f0 \
  -p 192.168.111.69:3260 --login
```

Le CSI voit alors "session already exists" → NodeStageVolume passe instantanément → NodePublishVolume → volumes montés.

### Dataangel restore

L'init container `dataangel` s'est exécuté en mode restore :
- DB `home-assistant_v2.db` (2.6 GB) déjà sur le disque, **clean shutdown détecté** → pas de WAL replay nécessaire
- `rclone copy` des fichiers de config non-DB depuis `s3:vixens-prod-homeassistant/config` (MinIO synelia)
- Init containers complétés → pod `2/2 Running`

---

## 3. CAUSES RACINES

| # | Cause | Impact |
|---|-------|--------|
| 1 | **Incident Synology** (cause inconnue) | Sessions iSCSI coupées simultanément sur tous les nodes |
| 2 | **NodeStageVolume timeout trop court** | Le CSI abandonne le login iSCSI avant que le NAS soit stabilisé → loop kubelet infinie |
| 3 | **Pas de mécanisme de re-login iSCSI automatique** | Nécessite une intervention manuelle `iscsiadm --login` pour débloquer |

---

## 4. CE QUI A BIEN FONCTIONNÉ

- **`reclaimPolicy: Retain`** → LUN et données survivent à la suppression du PVC
- **Dataangel backup** → filet de sécurité (non nécessaire ici, DB intacte)
- **Filesystem réparé automatiquement** par le premier NodeStageVolume propre sur peach
- **CNP `host`/`remote-node` fix** (PR #2997, déployé dans cette même session) → readiness probe kubelet désormais correctement autorisée

---

## 5. ACTIONS PRÉVENTIVES

- [ ] **Investiguer les logs DSM Synology** pour identifier la cause de la chute iSCSI à 16:49
- [ ] **Alerte Prometheus** sur la perte de sessions iSCSI (métrique custom via `iscsiadm` ou node-exporter)
- [ ] **Documenter la procédure de déblocage iSCSI** dans `docs/troubleshooting/` (login manuel quand NodeStageVolume timeout)
