# {Application Name}

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [ ]      | [ ]       | [ ]   | -       |
| Prod        | [ ]      | [ ]       | [ ]   | -       |

## Validation
**URL:** {https://app.dev.truxonline.com}

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl get pods -n {namespace}

# Check ingress
kubectl get ingress -n {namespace}
```

### Manual Validation
1. Open URL in browser.
2. Login with standard credentials.
3. Verify core functionality: {Action A}, {Action B}.

## Technical Notes
- **Namespace:** `{namespace}`
- **Category:** `{category}` (e.g., 20-media)
- **Dependencies:**
    - {Dependency A}
    - {Dependency B}
- **Specifics:** {Any special configuration, e.g., iSCSI, custom middlewares}
