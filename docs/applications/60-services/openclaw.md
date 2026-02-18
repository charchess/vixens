# OpenClaw

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [x]   | latest  |
| Prod        | [x]      | [x]       | [x]   | latest  |

## Validation
**URL:** https://openclaw.truxonline.com

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl --kubeconfig=.secrets/prod/kubeconfig-prod get pods -n services -l app=openclaw

# Check ingress
kubectl --kubeconfig=.secrets/prod/kubeconfig-prod get ingress -n services openclaw-ingress

# Test connectivity
curl https://openclaw.truxonline.com/
```

### Manual Validation
1. Open URL in browser.
2. Verify the OpenClaw UI loads correctly.

## Technical Notes
- **Namespace:** services
- **Category:** 60-services
- **Dependencies:**
    - MinIO (S3 backup)
    - Infisical (secrets management)
- **Specifics:**
    - Uses gateway mode with lan binding
    - Requires OPENCLAW_GATEWAY_TOKEN for lan binding
    - Data persisted to PVC and synced to MinIO
