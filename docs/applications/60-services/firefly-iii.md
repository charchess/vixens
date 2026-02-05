# Firefly III

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [ ]   | v6.4.16 |
| Prod        | [ ]      | [ ]       | [ ]   | -       |

## Validation
**URL:** https://firefly.dev.truxonline.com

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n finance

# Check ingress
kubectl get ingress -n finance
```

### Manual Validation
1. Open URL in browser.
2. Login with standard credentials.
3. Verify core functionality: Create a transaction, View dashboard.

## Technical Notes
- **Namespace:** `finance`
- **Category:** `60-services`
- **Dependencies:**
    - `postgresql-shared` (Database)
    - `infisical` (Secrets)
    - `synology-csi` (Backup storage/Litestream)
- **Specifics:**
    - Uses `rclone` sidecar for Litestream S3 sync.
    - Uses `firefly-iii/core` image.
    - **Elite Status:** VPA enabled, Security Context hardened, Resources defined.
