# farmOS

## Overview
[farmOS](https://farmos.org) is a web-based application for farm management, planning, and record keeping.

## Architecture
- **Type:** Candidate deployment for review
- **Environment:** `dev` (deployed in `test` namespace)
- **Database:** PostgreSQL (dedicated container in the same pod)
- **Deployment:** Kubernetes Deployment with 2 containers

## Configuration
- **Image:** `farmos/farmos:2.x`
- **Port:** 80

## Secrets
None for this test deployment (credentials passed via env vars in plaintext for simplicity).

## Ingress
- **URL:** `https://farmos.dev.truxonline.com`
- **Issuer:** `letsencrypt-staging`

## Storage
- **Application Data:** `emptyDir` mounted at `/opt/drupal/web/sites`
- **Database Data:** `emptyDir` mounted at `/var/lib/postgresql/data`
- **Note:** All data is **DISPOSABLE** and will be lost if the pod is deleted.

## Validation
### Automatic
- Check pod status: `kubectl -n test get pods -l app=farmos`
- Check service: `kubectl -n test get svc farmos`

### Manual
- Access `https://farmos.dev.truxonline.com`
- Proceed with Drupal/farmOS installation if required.
- Use DB credentials: Host `localhost`, User `farm`, Pass `farm`, DB `farm`.

---
> ⚠️ **APP CANDIDATE**
> Cette application est en phase de revue. Elle est déployée dans le namespace `test` du cluster `dev`.
