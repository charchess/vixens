# TrueNAS static NFS shared storage

This app declares static Kubernetes PVs for existing TrueNAS trees. It intentionally does **not** use the democratic-csi NFS dynamic provisioner, because these paths already exist and must not be created/deleted/mutated by CSI.

TrueNAS exports are expected on `192.168.200.244`, restricted to UMI's observed NFS source `192.168.199.119/32`.

## Static PVs

- `truenas-scratch-downloads` -> `/mnt/tank/scratch/downloads`
- `truenas-scratch-import-staging` -> `/mnt/tank/scratch/import-staging`
- `truenas-scratch-temp` -> `/mnt/tank/scratch/temp`
- `truenas-scratch-transcode` -> `/mnt/tank/scratch/transcode`
- `truenas-data-media` -> `/mnt/tank/data/media`
- `truenas-data-documents` -> `/mnt/tank/data/documents`
- `truenas-data-photos` -> `/mnt/tank/data/photos`

## Consumer PVC pattern

PVCs are namespace-scoped, so each app namespace that needs one of these paths should define its own PVC bound by `volumeName` and empty `storageClassName`.

Example:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: truenas-media
  namespace: media
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: ""
  volumeName: truenas-data-media
  resources:
    requests:
      storage: 8Ti
```

For multiple namespaces needing the same NFS path, create additional PV objects with distinct names pointing at the same `nfs.path`, then bind each namespace's PVC to its matching PV. Kubernetes PV/PVC binding is one-to-one even for NFS.
