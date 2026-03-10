# External DNS (Gandi)

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [ ]      | [ ]       | [ ]   | -       |
| Prod        | [x]      | [x]       | [x]   | v0.20.0 |

## Validation
**URL:** N/A (DNS automation service)

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n networking -l app.kubernetes.io/name=external-dns-gandi

# Check logs for DNS updates
kubectl logs -n networking -l app.kubernetes.io/name=external-dns-gandi
```

### Manual Validation
1. Create an Ingress with external-dns annotation.
2. Verify DNS record is created in Gandi.
3. Test DNS resolution: `dig <hostname>.truxonline.com`

## Technical Notes
- **Namespace:** `networking`
- **Category:** `40-network`
- **Dependencies:**
    - `Infisical` (Gandi API token secret)
    - `Traefik` (Ingress resources)
- **Specifics:** Automatically manages DNS records in Gandi based on Kubernetes Ingress resources. Synchronizes DNS entries for exposed services.
