# OpenJardin

## Overview
[OpenJardin](https://openjardin.org/) is a desktop software for garden management based on permaculture.

## Architecture
- **Type:** Candidate deployment for review
- **Environment:** `dev` (deployed in `test` namespace)
- **Base Image:** `ghcr.io/linuxserver/webtop:ubuntu-xfce` (Web access via browser)
- **Deployment:** Kubernetes Deployment with custom init script for installation

## Configuration
- **Port:** 3000 (Internal), 80 (Service)
- **Installation:** Automated via `/custom-cont-init.d/install.sh` (ConfigMap)

## Ingress
- **URL:** `https://openjardin.dev.truxonline.com`
- **Issuer:** `letsencrypt-staging`

## Storage
- **Configuration:** `emptyDir` mounted at `/config`
- **Note:** All data is **DISPOSABLE**.

## Validation
### Automatic
- Check pod status: `kubectl -n test get pods -l app=openjardin`
- Check logs: `kubectl -n test logs -l app=openjardin`

### Manual
- Access `https://openjardin.dev.truxonline.com`
- OpenJardin should be installed and available in the application menu or on the desktop.

---
> ⚠️ **APP CANDIDATE**
> Cette application est en phase de revue. Elle est déployée dans le namespace `test` du cluster `dev`.
