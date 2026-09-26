# Robusta

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|--------|---------|
| Dev         | [x]      | [x]       | [x]    | v0.32.0 |
| Prod        | [x]      | [x]       | [x]    | v0.32.0 |

## Validation
**URL:** N/A (background service)

```bash
kubectl get pods -n robusta
```

For functional validation, trigger a controlled test alert and verify the expected notification path.

## Technical Notes

- **Namespace:** `robusta`
- **Category:** `02-monitoring`
- **Monitoring dependency:** Prometheus-compatible metrics/alerts according to the deployed Robusta configuration.
- **Secrets:** Vixens' canonical secret architecture is OpenBao → External Secrets Operator → Kubernetes Secret. Do not create `InfisicalSecret` resources.
- **Configuration:** use the current manifests/values in `apps/02-monitoring/robusta/` as the authoritative deployment definition; this document must not override Git desired state.

See [`docs/guides/secret-management.md`](../../guides/secret-management.md) for the current secret-management contract.
