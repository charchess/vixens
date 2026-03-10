# NocoDB

## Deployment Information
| Environment | Deployed | Configured | Tested | Version  |
|-------------|----------|-----------|-------|----------|
| Dev         | [ ]      | [ ]       | [ ]   | -        |
| Prod        | [x]      | [x]       | [x]   | 0.301.3  |

## Validation
**URL:** https://nocodb.[env].truxonline.com

### Automatic Validation (CLI)
```bash
# 1. Verify HTTP -> HTTPS redirect
curl -I http://nocodb.truxonline.com
# Expected: HTTP 301/302/307/308 (Location: https://...)

# 2. Verify HTTPS access and content
curl -L -k https://nocodb.truxonline.com | grep "NocoDB"
# Expected: "NocoDB" in the body or title
```

### Manual Validation
1. Open URL in browser.
2. Login with standard credentials.
3. Verify core functionality: database access, table creation.

## Technical Notes
- **Namespace:** `tools`
- **Category:** `70-tools`
- **Dependencies:**
    - `PostgreSQL Shared` (Database backend)
    - `Traefik` (Ingress)
    - `Infisical` (Secrets)
- **Specifics:** Open-source Airtable alternative. Provides spreadsheet interface for databases. Used for lightweight database management and API generation.
