# Application Status Dashboard

**Quick reference for application deployment status across environments.**

Last Updated: 2026-01-12 (Task vixens-yvs completed - Mail Gateway goldification)

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

## Shared Resources (_shared/)

| Resource | Dev | Prod | Notes |
|----------|-----|------|-------|
| shared-namespaces | ✅ | ✅ | tools, databases, media centralized |
| priority-classes | ✅ | ✅ | Pod priority classes |

## Infrastructure (00-infra/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| argocd | ⚠️ | ✅ | Dev: Recovered from crash (Resource Pressure) |
| traefik | ✅ | ✅ | Ingress controller - v3.x |
| cert-manager | ✅ | ✅ | TLS certificates - Let's Encrypt production |
| cert-manager-webhook-gandi | ✅ | ✅ | Fixed missing secretNamespace |
| cilium | ✅ | ✅ | CNI v1.18.3 - DNS proxy transparent mode disabled |
| cilium-lb | ✅ | ✅ | L2 Announcements + LB IPAM |
| synology-csi | ✅ | ✅ | Persistent storage via iSCSI |
| infisical-operator | ✅ | ✅ | Secrets management operator |
| kubernetes-dashboard | ✅ | 🚧 | Dashboard v7.x (Prod en cours de sync) |

---

## Monitoring (02-monitoring/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| prometheus | ✅ | ✅ | Activé pour monitoring Litestream |
| alertmanager | ✅ | ✅ | Fixed stuck ContainerCreating (secrets) |
| grafana | 💤 | ✅ | Dev: Scaled down to 0 (Resource Pressure) |
| loki | 💤 | ✅ | Dev: Scaled down to 0 (Resource Pressure) |
| promtail | ✅ | ✅ | Fixed missing secretNamespace |
| goldilocks | ✅ | ✅ | Fixed missing secretNamespace |
| hubble-ui | ✅ | ✅ | Fixed secretNamespace error |

---

## Databases (04-databases/)

| Application | Dev | Prod | Notes |
|-------------|-----|------|-------|
| postgresql-shared | ✅ | ✅ | CloudNativePG Shared Cluster |
| redis-shared | ✅ | ✅ | Shared Redis Instance |
| mariadb-shared | ✅ | ✅ | Shared MariaDB Instance |
| cloudnative-pg | ✅ | ✅ | CloudNativePG Operator |

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
| mail-gateway | ✅ | ✅ | Email gateway (External) |
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
- **[reports/validation/RECETTE-FONCTIONNELLE.md](reports/validation/RECETTE-FONCTIONNELLE.md)** - Functional validation checklist
- **[reports/validation/RECETTE-TECHNIQUE.md](reports/validation/RECETTE-TECHNIQUE.md)** - Technical validation checklist
- **[reports/audits/APP_AUDIT.md](reports/audits/APP_AUDIT.md)** - Detailed application audit
- **[reports/audits/ULTIMATE-AUDIT.md](reports/audits/ULTIMATE-AUDIT.md)** - Resource optimization analysis

---

## Notes

- This dashboard is a **quick reference** for deployment status
- For detailed information, see per-application documentation in [docs/applications/](applications/)
- Update this file **immediately** when deploying or discovering issues
- Keep notes column concise (max 80 characters)
- Use emoji symbols consistently

---

**Last Updated:** 2026-01-12
