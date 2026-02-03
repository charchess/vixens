# Firefly III Data Importer

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [ ]       | [ ]   | latest  |
| Prod        | [x]      | [ ]       | [ ]   | latest  |

## Validation
**URL:** [https://importer.truxonline.com](https://importer.truxonline.com)

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n finance -l app.kubernetes.io/name=firefly-iii-importer
```

### Manual Validation
1. Open URL in browser.
2. The UI should display the import options (CSV, Nordigen, etc.).
3. Note: Requires `FIREFLY_III_ACCESS_TOKEN` in Infisical to work.

## Technical Notes
- **Namespace:** `finance`
- **Category:** `60-services`
- **Dependencies:**
    - Firefly III (Internal URL: `http://firefly-iii.finance.svc.cluster.local`)
    - Infisical (Secrets management)
- **Specifics:** 
    - Uses `fireflyiii/data-importer:latest` image.
    - Configuration handled via environment variables and secrets.
    - Automated sync possible via CronJob (planned).
