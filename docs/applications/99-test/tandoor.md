# Tandoor Recipes

## Overview
[Tandoor Recipes](https://tandoor.dev/) is a powerful recipe manager that allows you to manage your growing collection of recipes as a digital cookbook.

## Architecture
- **Type:** Candidate deployment for review
- **Environment:** `dev` (deployed in `test` namespace)
- **Database:** PostgreSQL 16 (dedicated container in the same pod)
- **Deployment:** Kubernetes Deployment with 2 containers

## Configuration
- **Image:** `vabene1111/recipes:latest`
- **Port:** 8080 (Internal), 80 (Service)

## Secrets
None for this test deployment (credentials passed via env vars in plaintext).

## Ingress
- **URL:** `https://tandoor.dev.truxonline.com`
- **Issuer:** `letsencrypt-staging`

## Storage
- **Static/Media Files:** `emptyDir` mounted at `/opt/recipes/staticfiles` and `/opt/recipes/mediafiles`
- **Database Data:** `emptyDir` mounted at `/var/lib/postgresql/data`
- **Note:** All data is **DISPOSABLE**.

## Validation
### Automatic
- Check pod status: `kubectl -n test get pods -l app=tandoor`
- Check service: `kubectl -n test get svc tandoor`

### Manual
- Access `https://tandoor.dev.truxonline.com`
- Default login: Usually requires creating an admin account on first run.

---
> ⚠️ **APP CANDIDATE**
> Cette application est en phase de revue. Elle est déployée dans le namespace `test` du cluster `dev`.
