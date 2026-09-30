# TrueNAS CSI

This app group manages `democratic-csi` releases for TrueNAS-backed storage.

- `truenas-csi-iscsi` provisions RWO block volumes through TrueNAS iSCSI/zvols.
- `truenas-csi-nfs` provisions RWX/RWO filesystem volumes through TrueNAS NFS datasets.

The Helm values are safe to commit. Driver connection configs are secrets and must never be committed.

## Environments

### Production

Production uses dedicated values files and OpenBao-backed `ExternalSecret` resources:

- `values/iscsi-prod.yaml`
- `values/nfs-prod.yaml`
- `vixens/prod/apps/01-storage/truenas-csi/iscsi`
- `vixens/prod/apps/01-storage/truenas-csi/nfs`

The resulting Kubernetes Secrets are:

- `truenas-csi-iscsi-driver-config`
- `truenas-csi-nfs-driver-config`

Both expect the key `driver-config-file.yaml`.

The production ArgoCD overlay must use the production applications (`truenas-csi-iscsi.yaml` and `truenas-csi-nfs.yaml`). UMI applications are development-only and must not be enabled in the production overlay.

### UMI development

UMI keeps its own development values and OpenBao paths. The `truenas-csi-secrets-umi` Application points at the UMI OpenBao store and rewrites the secret paths to `vixens/dev/...`.

Read-only scan on 2026-09-01 showed:

- `nfs-common` is installed.
- `open-iscsi` is not installed and `iscsid` is inactive.
- snapshot CRDs are not present except k3s etcd snapshots.
- current cluster storage class is only `local-path`.

Before testing iSCSI PVCs on UMI, install/enable the host iSCSI initiator and confirm the TrueNAS target portal/initiator group IDs.

## StorageClasses

Names are aligned with the existing retention split:

- `truenas-iscsi-retain`
- `truenas-iscsi-delete`
- `truenas-iscsi-xfs-retain`
- `truenas-iscsi-xfs-delete`
- `truenas-nfs-retain`
- `truenas-nfs-delete`

None is default initially; `local-path` stays default until an explicit migration decision.

TXO Fabric shared-workspace profiles currently use `truenas-nfs-retain`, so that StorageClass must exist in every environment where those profiles are enabled.

## Secret hygiene

Use the templates in `secret-templates/` only as shape references. Do not commit filled copies. Production and UMI credentials/configuration are sourced from their respective OpenBao paths through External Secrets.
