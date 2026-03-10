# External DNS (UniFi)

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
kubectl get pods -n networking -l app.kubernetes.io/name=external-dns-unifi

# Check logs for DNS updates
kubectl logs -n networking -l app.kubernetes.io/name=external-dns-unifi
```

### Manual Validation
1. Create an Ingress with external-dns annotation for UniFi zone.
2. Verify DNS record is created in UniFi Controller.
3. Test DNS resolution from internal network.

## Technical Notes
- **Namespace:** `networking`
- **Category:** `40-network`
- **Dependencies:**
    - `Infisical` (UniFi API credentials secret)
    - `Traefik` (Ingress resources)
- **Specifics:** Automatically manages internal DNS records in UniFi Controller based on Kubernetes Ingress resources. Used for internal zone management.
