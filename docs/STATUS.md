# Application Status Dashboard

**Quick reference for application deployment status across environments.**

Last Updated: 2026-01-10 (Task vixens-04mo completed)

---

## Legend

| Symbol | Status | Description |
|--------|--------|-------------|
| ✅ | **Working** | Deployed, configured, tested, no known issues |
| ⚠️ | **Degraded** | Working but needs attention (resources, config, minor issues) |
| ❌ | **Broken** | Not working, needs immediate fix |
| 🚧 | **WIP** | Work in progress, deployment ongoing |
| 💤 | **Paused** | Intentionally not deployed (planned for future) |
| ⏳ | **Planned** | Not yet deployed, planned for future sprint |

---

## Infrastructure (00-infra/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| argocd | ✅ | ✅ | GitOps controller - v7.7.7 |
| traefik | ✅ | ✅ | Ingress controller - v3.x |
| cert-manager | ✅ | ✅ | TLS certificates - Let's Encrypt production |
| cert-manager-webhook-gandi | ✅ | ✅ | Fixed missing secretNamespace |
| cilium-lb | ✅ | ✅ | L2 Announcements + LB IPAM |
| synology-csi | ✅ | ✅ | Persistent storage via iSCSI |
| infisical-operator | ✅ | ✅ | Secrets management operator |

---

## Monitoring (02-monitoring/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| prometheus | ✅ | ✅ | Fixed missing InfisicalSecrets |
| alertmanager | ✅ | ✅ | Fixed stuck ContainerCreating (secrets) |
| grafana | ✅ | ✅ | Fixed missing secretNamespace |
| loki | ✅ | ✅ | Fixed missing secretNamespace |
| promtail | ✅ | ✅ | Fixed missing secretNamespace |
| goldilocks | ✅ | ✅ | Fixed missing secretNamespace |
| hubble-ui | ✅ | ✅ | Fixed secretNamespace error |

---

## Databases (10-databases/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| postgresql | ⏳ | 💤 | CloudNativePG (planned) |
| redis | ⏳ | 💤 | In-memory cache (planned) |

---

## Home Automation (10-home/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| homeassistant | ✅ | ✅ | Fixed Kustomization syntax error |
| mealie | ✅ | ✅ | Fixed DNS resolution (removed target annotation) |
| mosquitto | ✅ | ✅ | MQTT broker |

---

## Media (20-media/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| jellyfin | ⏳ | 💤 | Media server (planned) |
| sabnzbd | ⏳ | ✅ | Prod fixed and synced |
| radarr | ⏳ | ✅ | Prod fixed 
| sonarr | ⏳ | ✅ | Prod fixed 
| prowlarr | ⏳ | ✅ | Prod fixed 
| jellyseerr | ⏳ | 💤 | Media request management (planned) |
| hydrus-client | ✅ | ✅ | Metrics Prometheus activées (v0.5.5) |

---

## Network (40-network/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| external-dns-unifi | ✅ | ✅ | Internal DNS management |
| external-dns-gandi | ✅ | ✅ | Public DNS management |
| contacts | ✅ | 💤 | Contacts redirection service |
| netvisor | ✅ | ✅ | Network monitoring (fixed syntax error) |
| adguard | ⏳ | ✅ | DNS-based ad blocking (planned) |
| gluetun | ✅ | ✅ | Fixed missing secretNamespace |

---

## Services (50-services/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| mail-gateway | ✅ | 💤 | Email gateway (Roundcube) |
| vaultwarden | ✅ | ✅ | Migrated to standardized middleware |
| authentik | 🚧 | ✅ | Prod fixed (Redis auth solved) |
| docspell-native | ✅ | ✅ | Fixed missing secretNamespace |

---

## Tools (70-tools/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| whoami | ✅ | ✅ | Migrated to centralized middleware |
| homepage | ✅ | 💤 | Prod fixed and synced |
| netbox | ✅ | ✅ | Migrated to centralized middleware |
| changedetection | ✅ | ✅ | Migrated to centralized middleware |
| stirling-pdf | ✅ | ✅ | Migrated to centralized middleware |
| it-tools | ✅ | ✅ | Migrated to centralized middleware |
| headlamp | ✅ | ✅ | Migrated to centralized middleware |
| linkwarden | ✅ | ✅ | Migrated to centralized middleware |
| renovate | ✅ | ✅ | Auto-dependency updates (ADR-017) |

---

## Update Protocol

**MANDATORY:** When deploying or discovering issues, update this dashboard.

### When to Update

| Event | Action |
|-------|--------|
| Deploy to dev | Update dev column |
| Deploy to prod | Update prod column |
| Discover issue | Change status to ⚠️ or ❌ |
| Fix issue | Change status to ✅ |
| Remove service | Mark as 💤 (if temporary) or delete row |

### How to Update

```bash
# Edit this file
vim docs/STATUS.md

# Update status symbols and notes
# Example: | jellyfin | ✅ | ⚠️ | Dev OK, Prod needs resource tuning |
| sabnzbd | ⏳ | ✅ | Prod fixed and synced |

# Commit changes
git add docs/STATUS.md
git commit -m "docs: update STATUS.md - <application> <status>"
git push origin main
```

---

## Quick Stats

**Dev Environment:**
- ✅ Working: 11 applications
- ⚠️ Degraded: 0 applications
- ❌ Broken: 0 applications
- 🚧 WIP: 1 application (authentik)
- ⏳ Planned: 12 applications
- 💤 Paused: 0 applications

**Prod Environment:**
- ✅ Working: many applications (Phase 3 active)

---

## Environment Information

### Dev Cluster

- **Nodes:** daphne, diva, dulce (3 CP HA)
- **VIP:** 192.168.111.160
- **VLAN Internal:** 111
- **VLAN Services:** 208
- **Status:** ✅ Active

### Prod Cluster

- **Nodes:** pearl, phoebe, poison, powder
- **VIP:** 192.168.111.200
- **VLAN Internal:** 111
- **VLAN Services:** 201
- **Status:** ✅ Active

---

## Related Documentation

- **[Application Documentation](applications/)** - Detailed per-app documentation
- **[RECETTE-FONCTIONNELLE.md](RECETTE-FONCTIONNELLE.md)** - Functional validation checklist
- **[RECETTE-TECHNIQUE.md](RECETTE-TECHNIQUE.md)** - Technical validation checklist
- **[reports/APP_AUDIT.md](reports/APP_AUDIT.md)** - Detailed application audit
- **[reports/ULTIMATE-AUDIT.md](reports/ULTIMATE-AUDIT.md)** - Resource optimization analysis

---

## Notes

- This dashboard is a **quick reference** for deployment status
- For detailed information, see per-application documentation in [docs/applications/](applications/)
- Update this file **immediately** when deploying or discovering issues
- Keep notes column concise (max 80 characters)
- Use emoji symbols consistently

---

**Last Updated:** 2026-01-08
