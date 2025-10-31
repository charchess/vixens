# Stratégie de Stockage Vixens

## Vue d'Ensemble

Approche **hybride** combinant stockage dynamique iSCSI et volumes NFS statiques.

---

## Architecture Storage

```
┌──────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                         │
│                                                                │
│  ┌─────────────────────┐       ┌────────────────────────┐   │
│  │  Synology CSI       │       │  Static NFS PV/PVC     │   │
│  │  (Dynamic iSCSI)    │       │  (Manual Provisioning) │   │
│  └──────────┬──────────┘       └───────────┬────────────┘   │
│             │                               │                 │
│             │ StorageClass                  │ Direct Mount   │
│             │ synelia-iscsi                 │                 │
│             │                               │                 │
└─────────────┼───────────────────────────────┼─────────────────┘
              │                               │
              │ VLAN 111                      │ VLAN 111
              │ (iSCSI Protocol)              │ (NFS Protocol)
              │                               │
         ┌────▼───────────────────────────────▼─────┐
         │      Synology NAS (Synelia)              │
         │      192.168.111.69                      │
         │                                          │
         │  ┌──────────────┐  ┌─────────────────┐ │
         │  │ iSCSI Targets│  │  NFS Exports    │ │
         │  │ (Dynamic LUN)│  │  /volume1/...   │ │
         │  └──────────────┘  └─────────────────┘ │
         └──────────────────────────────────────────┘
```

---

## Solutions par Use Case

### 1. Stockage Applicatif (Bases de Données, Config)

**Solution** : Synology CSI Driver (iSCSI)

**Pourquoi** :
- ✅ Provisioning dynamique : 1 PVC → 1 LUN automatique
- ✅ Performance block storage pour DB
- ✅ Snapshots supportés (via Synology CSI)
- ✅ Resize dynamique possible

**StorageClass** : `synelia-iscsi`

**Exemple PVC** :
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mariadb-data
  namespace: databases
spec:
  storageClassName: synelia-iscsi
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
```

**Applications cibles** :
- PostgreSQL, MariaDB, MongoDB
- Redis, Valkey (cache persistant)
- Vaultwarden (données sensibles)
- Application state (configs, secrets)

---

### 2. Media Partagés (Priorité Basse)

**Solution** : PersistentVolume Statique NFS

**Pourquoi** :
- ✅ Accès direct aux shares existants (pas de sous-dossiers)
- ✅ ReadWriteMany (multi-pod access)
- ✅ Partage entre apps (Radarr, Sonarr, Plex)
- ✅ Simple (pas de CSI driver pour NFS)

**Note** : **Bas dans la liste de priorité** - Déploiement Phase 3

**PV Statiques** :
- `synelia-content` : `/volume1/content` (media)
- `synelia-downloads` : `/volume1/downloads` (téléchargements)

**Exemple PV + PVC** :
```yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: synelia-content
spec:
  capacity:
    storage: 10Ti
  accessModes:
    - ReadWriteMany
  nfs:
    server: 192.168.111.69
    path: /volume1/content
  mountOptions:
    - nfsvers=4.1
    - hard
    - intr
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: synelia-content
  namespace: media
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Ti
  volumeName: synelia-content
```

**Applications cibles** (Phase 3) :
- Radarr, Sonarr, Lidarr (accès `/volume1/content`)
- SABnzbd (accès `/volume1/downloads`)
- Plex, Jellyfin (lecture media)

---

## Synology CSI Driver

### Version

**Repository** : https://github.com/SynologyOpenSource/synology-csi-talos

**Version cible** : Vérifier dernière release compatible Talos 1.10.7 + K8s 1.30.0

### Configuration

**Fichier Secret (DSM Credentials)** :
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: synology-secret
  namespace: synology-csi
type: Opaque
stringData:
  host: "192.168.111.69"
  port: "5000"  # DSM HTTP port
  username: "kubernetes"  # User dédié avec droits iSCSI
  password: "CHANGE_ME"
  protocol: "http"  # ou https si configuré
```

**StorageClass** :
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: synelia-iscsi
provisioner: csi.san.synology.com
parameters:
  fsType: ext4
  protocol: iscsi
  location: /volume1
reclaimPolicy: Delete
allowVolumeExpansion: true
volumeBindingMode: Immediate
```

### Prérequis Synology

1. **Activer iSCSI Target** :
   - DSM → Storage Manager → iSCSI → Enable iSCSI service
   - Port 3260 ouvert sur VLAN 111

2. **Créer utilisateur Kubernetes** :
   - DSM → Control Panel → User & Group
   - User: `kubernetes`
   - Groupe avec droits : iSCSI Manager, Target LUN management

3. **Configurer Authentication** :
   - CHAP authentication (recommandé) ou None
   - Credentials dans Secret Kubernetes

---

## Tableau Récapitulatif

| Use Case           | Solution         | Protocol | Access Mode | Provisioning | Phase  |
|--------------------|------------------|----------|-------------|--------------|--------|
| Databases          | Synology CSI     | iSCSI    | RWO         | Dynamic      | 2      |
| App State/Config   | Synology CSI     | iSCSI    | RWO         | Dynamic      | 2      |
| Media Partagés     | PV Statique NFS  | NFS v4.1 | RWX         | Manual       | 3 (bas)|
| Downloads          | PV Statique NFS  | NFS v4.1 | RWX         | Manual       | 3 (bas)|

---

## Considérations Opérationnelles

### Backup

**iSCSI Volumes** :
- Snapshots via Synology CSI (VolumeSnapshot CRD)
- Backup Synology Hyper Backup

**NFS Shares** :
- Snapshots Synology (Btrfs)
- Backup Hyper Backup ou rsync externe

### Monitoring

**Métriques à surveiller** :
- PVC usage (kubelet metrics)
- Synology storage capacity (DSM API)
- iSCSI I/O performance (DSM)
- NFS connection count

**Outils** :
- Prometheus + Grafana (Kubernetes metrics)
- Synology DSM (storage backend)

### Performance

**iSCSI** :
- ~10Gbps theoretical (dépend du réseau)
- Latence ~1-2ms (LAN)
- IOPS : dépend des disques Synology

**NFS** :
- ~1Gbps pour large files
- Latence ~2-5ms
- Optimisé pour sequential read (media streaming)

---

## Migration Future

Si besoins évoluent :

**Option 1** : Ajouter Rook-Ceph
- Stockage distribué au lieu de centralisé
- Réplication native Kubernetes
- Plus complexe, nécessite disks locaux

**Option 2** : Longhorn
- Block storage distribué léger
- Interface UI riche
- Moins performant que Ceph

**Décision** : Rester sur Synology pour l'instant (simplicité + fiabilité NAS)

---

## Troubleshooting

### iSCSI Connection Issues

```bash
# Vérifier connexion depuis node
talosctl --nodes 192.168.111.162 shell
iscsiadm -m discovery -t st -p 192.168.111.69:3260

# Vérifier CSI driver logs
kubectl logs -n synology-csi -l app=synology-csi-controller

# Vérifier PVC status
kubectl describe pvc <pvc-name> -n <namespace>
```

### NFS Mount Issues

```bash
# Tester mount depuis node
talosctl --nodes 192.168.111.162 shell
mount -t nfs4 192.168.111.69:/volume1/content /mnt/test

# Vérifier NFS exports Synology
showmount -e 192.168.111.69
```

---

## Changelog

| Date       | Version | Changement                               |
|------------|---------|------------------------------------------|
| 2025-10-30 | 1.0     | Stratégie initiale hybride iSCSI + NFS   |
