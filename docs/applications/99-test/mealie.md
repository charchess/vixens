# Mealie

## Overview
[Mealie](https://mealie.io/) is a self-hosted recipe manager and meal planner with a RestAPI backend and a reactive frontend application built in Vue for a pleasant user experience.

## Architecture
- **Type:** Candidate deployment for review
- **Environment:** `dev` (deployed in `test` namespace)
- **Database:** SQLite (Default for test)
- **Deployment:** Kubernetes Deployment (Single container)

## Configuration
- **Image:** `ghcr.io/mealie-recipes/mealie:v1.0.0`
- **Port:** 9000 (Internal), 80 (Service)

## Ingress
- **URL:** `https://mealie.dev.truxonline.com`
- **Issuer:** `letsencrypt-staging`

## Storage
- **Data:** `emptyDir` mounted at `/app/data`
- **Note:** All data is **DISPOSABLE**.

## Validation
### Automatic
- Check pod status: `kubectl -n test get pods -l app=mealie`
- Check service: `kubectl -n test get svc mealie`

### Manual
- Access `https://mealie.dev.truxonline.com`
- Default login: `changeme@example.com` / `MyPassword`

---
> ⚠️ **APP CANDIDATE**
> Cette application est en phase de revue. Elle est déployée dans le namespace `test` du cluster `dev`.
