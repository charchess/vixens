# Penpot

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [ ]   | 2.12.1  |
| Prod        | [ ]      | [ ]       | [ ]   | 2.12.1  |

## Validation
**URL:** https://design.dev.truxonline.com

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n tools -l app.kubernetes.io/name=penpot

# Check ingress
kubectl get ingress -n tools penpot-ingress
```

### Manual Validation
1. Open URL in browser.
2. Login or create an account.
3. Verify core functionality: Create a new project, add a design element.

## Technical Notes
- **Namespace:** `tools`
- **Category:** `70-tools`
- **Dependencies:**
    - PostgreSQL (managed via Infisical connection string)
    - Redis (managed via Infisical connection string)
    - S3 Bucket (for assets storage)
- **Specifics:**
    - Infisical secrets management for all sensitive configurations.
    - Three-tier deployment: Frontend, Backend, and Exporter.
