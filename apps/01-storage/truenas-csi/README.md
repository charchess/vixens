# TrueNAS CSI for UMI/k3s

This app group prepares two `democratic-csi` Helm releases for the UMI laboratory cluster using the current TrueNAS test appliance.

- `truenas-csi-iscsi` provisions RWO block volumes through TrueNAS iSCSI/zvols.
- `truenas-csi-nfs` provisions RWX/RWO filesystem volumes through TrueNAS NFS datasets.

The Helm values are safe to commit. The driver connection config is **not** committed and must be created as Kubernetes Secrets before syncing the Applications:

```bash
kubectl -n truenas-csi create secret generic truenas-csi-iscsi-driver-config \
  --from-file=driver-config-file.yaml=/secure/path/truenas-iscsi-driver-config.yaml

kubectl -n truenas-csi create secret generic truenas-csi-nfs-driver-config \
  --from-file=driver-config-file.yaml=/secure/path/truenas-nfs-driver-config.yaml
```

Use the templates in `secret-templates/` as shape references only. Do not commit filled copies.

## UMI prerequisites

Read-only scan on 2026-09-01 showed:

- `nfs-common` is installed.
- `open-iscsi` is not installed and `iscsid` is inactive.
- snapshot CRDs are not present except k3s etcd snapshots.
- current cluster storage class is only `local-path`.

Before testing iSCSI PVCs, install/enable the host iSCSI initiator on UMI and confirm the TrueNAS target portal/initiator group IDs. The bootstrap Application `truenas-csi-secrets-umi` creates the privileged namespace first; create the two driver config Secrets after that and before syncing the CSI Applications.

## StorageClasses

Proposed names, aligned with the existing Synology `synelia-*` retention split:

- `truenas-iscsi-retain`
- `truenas-iscsi-delete`
- `truenas-iscsi-xfs-retain`
- `truenas-iscsi-xfs-delete`
- `truenas-nfs-retain`
- `truenas-nfs-delete`

None is default initially; `local-path` stays default until an explicit migration decision.


## Secret hygiene

The `secret-templates/.gitignore` file ignores every non-example file in that directory so filled driver configs are not accidentally committed. Keep real rendered configs under a secure local state directory and feed them to `kubectl create secret --from-file`.
