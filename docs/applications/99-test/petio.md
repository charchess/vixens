# Petio

## Overview
[Petio](https://petio.tv/) is a third-party companion app for Plex, Emby and Jellyfin. It allows your users to request movies and TV shows.

## Architecture
- **Type:** Candidate deployment for review
- **Environment:** `dev` (deployed in `test` namespace)
- **Database:** MongoDB (dedicated container in the same pod)
- **Deployment:** Kubernetes Deployment with 2 containers

## Configuration
- **Image:** `ghcr.io/petio-team/petio:latest`
- **Port:** 7777 (Internal), 80 (Service)

## Ingress
- **URL:** `https://petio.dev.truxonline.com`
- **Issuer:** `letsencrypt-staging`

## Storage
- **Config/Logs/Images:** `emptyDir` mounted at `/app/api/config`, `/app/logs`, `/app/images`
- **Database Data:** `emptyDir` mounted at `/data/db`
- **Note:** All data is **DISPOSABLE**.

## Validation
### Automatic
- Check pod status: `kubectl -n test get pods -l app=petio`
- Check service: `kubectl -n test get svc petio`

### Manual
- Access `https://petio.dev.truxonline.com`
- Complete the setup wizard (MongoDB connection should be pre-filled or use `mongodb://localhost:27017/petio`).

---
> ⚠️ **APP CANDIDATE**
> Cette application est en phase de revue. Elle est déployée dans le namespace `test` du cluster `dev`.
