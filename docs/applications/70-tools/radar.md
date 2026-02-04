# Radar

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [x]   | v0.4.2  |
| Prod        | [x]      | [x]       | [ ]   | v0.4.2  |

## Validation
**URL:** [https://radar.truxonline.com](https://radar.truxonline.com)

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n tools -l app.kubernetes.io/name=radar

# Check ingress
kubectl get ingress -n tools radar
```

### Manual Validation
1. Open URL in browser.
2. Login via Authentik.
3. Verify cluster topology and Hubble traffic are visible.

## Technical Notes
- **Namespace:** `tools`
- **Category:** `70-tools`
- **Purpose:** Real-time cluster visibility and network traffic visualization.
- **Security:** Protected by `forward-auth` middleware.
- **Specifics:** 
    - Uses Hubble Relay to visualize service-to-service traffic.
    - Zero cluster-side CRDs or heavy agents.
