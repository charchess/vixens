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
    - `infisical-operator` (Secrets management)
    - `minio` (External S3 storage)
- **Specifics:** 
    - **Helm Chart:** v11.3.2
    - **Uploader:** `kopia`
    - **Node Agent:** Enabled (requires `privileged` label on namespace)
    - **Secrets:** Integrated with Infisical (path: `/00-infra/velero`)
    - **Kyverno:** Compliant with resource limits and priority classes.
