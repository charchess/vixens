# Velero - Kubernetes Backup Solution

Velero provides backup and restore capabilities for Kubernetes cluster resources and persistent volumes.

## Configuration

### Credentials

Velero uses the **shared litestream credentials** from `/shared/litestream` in Infisical.

No additional secret creation needed - the existing litestream S3 credentials are reused.

The Helm values map `LITESTREAM_ACCESS_KEY_ID` → `AWS_ACCESS_KEY_ID` automatically.

### Backup Schedules (Production)

| Schedule | Frequency | Retention | Namespaces |
|----------|-----------|-----------|------------|
| daily-critical | 2 AM daily | 30 days | databases, tools, security |
| weekly-full | 3 AM Sunday | 90 days | All (except kube-*) |
| daily-home | 4 AM daily | 7 days | home |

### Storage

Backups are stored in MinIO on the Synology NAS:
- **Endpoint:** `http://192.168.111.69:9000`
- **Bucket:** `backups-vixens-prod`
- **Prefix:** `velero/`

## Manual Operations

### Create a manual backup
```bash
velero backup create my-backup --include-namespaces=my-namespace
```

### Restore from backup
```bash
velero restore create --from-backup my-backup
```

### List backups
```bash
velero backup get
```

### Check backup logs
```bash
velero backup logs my-backup
```

## References

- [Velero Documentation](https://velero.io/docs/)
- [Velero Helm Chart](https://github.com/vmware-tanzu/helm-charts/tree/main/charts/velero)
