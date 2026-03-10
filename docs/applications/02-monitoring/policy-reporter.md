# Policy Reporter

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [x]   | 2.20.2  |
| Prod        | [x]      | [x]       | [x]   | 2.20.2  |

## Validation
**URL:** https://policy-reporter.[env].truxonline.com

### Automatic Validation (CLI)
```bash
# 1. Verify HTTP -> HTTPS redirect
curl -I http://policy-reporter.dev.truxonline.com
# Expected: HTTP 301/302/307/308 (Location: https://...)

# 2. Verify HTTPS access and content
curl -L -k https://policy-reporter.dev.truxonline.com | grep "Policy Reporter"
# Expected: "Policy Reporter" in the body or title
```

### Manual Validation
1. Open URL in browser.
2. Verify Policy Reporter dashboard displays correctly.
3. Check policy violations are visible.

## Technical Notes
- **Namespace:** `policy-reporter`
- **Category:** `02-monitoring`
- **Dependencies:**
    - `Kyverno` (Policy engine)
    - `Traefik` (Ingress)
- **Specifics:** Web UI for visualizing Kyverno policy reports. Provides dashboard for policy violations and compliance status.
