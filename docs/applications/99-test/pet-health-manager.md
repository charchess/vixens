# Pet Health Manager

## Overview
[Pet Health Manager](https://github.com/h-koehler/pet-health-manager) is a self-hosted application for tracking pet health information.

## Architecture
- **Type:** Candidate deployment for review
- **Environment:** `dev` (deployed in `test` namespace)
- **Stack:** Python/Django + SQLite (embedded)
- **Deployment:** Kubernetes Deployment (Single container)

## Configuration
- **Image:** `hkoehler/pet-health-manager:latest`
- **Port:** 8000 (Internal), 80 (Service)

## Ingress
- **URL:** `https://pet-health.dev.truxonline.com`
- **Issuer:** `letsencrypt-staging`

## Storage
- **Database:** Embedded SQLite (Ephemeral in this test deployment)
- **Note:** All data is **DISPOSABLE**.

## Validation
### Automatic
- Check pod status: `kubectl -n test get pods -l app=pet-health-manager`
- Check service: `kubectl -n test get svc pet-health-manager`

### Manual
- Access `https://pet-health.dev.truxonline.com`
- Default credentials (if any) are not documented, check logs if setup is required.

---
> ⚠️ **APP CANDIDATE**
> Cette application est en phase de revue. Elle est déployée dans le namespace `test` du cluster `dev`.
