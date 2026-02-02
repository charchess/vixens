# Robusta

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [ ]   | v0.15.0 |
| Prod        | [x]      | [x]       | [ ]   | v0.15.0 |

## Validation
**URL:** N/A (Background Service)

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n robusta
```

### Manual Validation
1. Trigger a test alert (e.g., create a crashing pod).
2. Verify notification in Discord channel.

## Technical Notes
- **Namespace:** `robusta`
- **Category:** `02-monitoring`
- **Dependencies:**
    - Prometheus (Robusta connects to it)
    - Infisical (for Secrets)
- **Specifics:** 
    - Configuration handled via `values.yaml` in overlays.
    - Secrets (`DISCORD_WEBHOOK_URL`, `ROBUSTA_SIGNING_KEY`) injected via `InfisicalSecret`.
