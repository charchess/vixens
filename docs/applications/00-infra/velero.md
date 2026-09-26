# Velero

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [ ]      | [ ]       | [ ]   | -       |
| Prod        | [x]      | [x]       | [x]   | v1.17.2 |

## Validation
**URL:** N/A (Backend service)

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n velero

# Check External Secrets projection
kubectl get externalsecret -n velero

# Check backup storage location
kubectl get bsl -n velero

# Check schedules
kubectl get schedules -n velero
```

### Manual Validation
1. Verify backup sync in logs: `kubectl logs -n velero -l app.kubernetes.io/name=velero`
2. Run manual backup: `velero backup create test --from-schedule velero-daily-critical`

## Technical Notes
- **Namespace:** `velero`
- **Category:** `00-infra`
- **Dependencies:**
    - OpenBao + External Secrets Operator for credentials
    - MinIO / S3-compatible external storage
- **Specifics:**
    - **Helm Chart:** v11.3.2
    - **Uploader:** `kopia`
    - **Node Agent:** Enabled (requires `privileged` label on namespace)
    - **Secrets:** projected from OpenBao through External Secrets Operator; production uses `vixens/prod/apps/00-infra/velero`
    - **Kyverno:** Compliant with resource limits and priority classes.

The retired Infisical integration must not be recreated; current GitOps manifests live under `apps/00-infra/velero/external-secrets/`.
