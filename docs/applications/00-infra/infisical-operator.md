# Infisical Operator

## Deployment Information
| Environment | Deployed | Configured | Tested | Version   |
|-------------|----------|-----------|-------|-----------|
| Dev         | [x]      | [x]       | [x]   | v0.10.26  |
| Prod        | [x]      | [x]       | [x]   | v0.10.26  |

## Validation
**URL:** N/A (Operator-based service)

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n infisical-operator-system

# Check CRDs
kubectl get crd | grep infisical

# Check InfisicalSecret resources
kubectl get infisicalsecret -A
```

### Manual Validation
1. Verify operator is running: `kubectl get pods -n infisical-operator-system`
2. Check logs: `kubectl logs -n infisical-operator-system -l app.kubernetes.io/name=infisical-operator`
3. Verify secrets are synced: `kubectl get infisicalsecret -A`

## Technical Notes
- **Namespace:** `infisical-operator-system`
- **Category:** `00-infra`
- **Dependencies:**
    - `Infisical` (External secret management service)
- **Specifics:** Kubernetes operator that syncs secrets from Infisical to Kubernetes Secrets. Manages InfisicalSecret custom resources.
