# Application Documentation

Documentation for all applications deployed in the Vixens Kubernetes cluster.

---

## Applications by Category

### [00-infra/](00-infra/) - Infrastructure (10 apps)
Core infrastructure components:
- ArgoCD - GitOps continuous delivery
- Cert-Manager - TLS certificate management
- Cilium LB - Load balancer (L2 Announcements + IPAM)
- Metrics Server - Resource metrics
- NFS Storage - NFS provisioner
- Reloader - Auto-reload on ConfigMap/Secret changes
- Synology CSI - Persistent storage (iSCSI)
- Traefik - Ingress controller
- Traefik Dashboard - Ingress monitoring

### [02-monitoring/](02-monitoring/) - Monitoring & Observability (9 apps)
Monitoring stack:
- Alertmanager - Alert management
- Goldilocks - VPA recommendations
- Grafana - Visualization & dashboards
- Headlamp - Kubernetes UI
- Hubble UI - Cilium network visualization
- Loki - Log aggregation
- Promtail - Log shipper
- Prometheus - Metrics & monitoring
- VPA - Vertical Pod Autoscaler

### [10-databases/](10-databases/) - Databases (3 apps)
Shared database services:
- CloudNativePG - PostgreSQL operator
- PostgreSQL Shared - Shared PostgreSQL cluster
- Redis Shared - Shared Redis cluster

### [20-media/](20-media/) - Media Management (16 apps)
Media applications:
- Birdnet-Go - Bird sound identification
- Booklore - Book management
- Frigate - NVR with object detection
- Hydrus Client - Media tagger (client)
- Hydrus Server - Media tagger (server)
- Jellyfin - Media server
- Jellyseerr - Media requests
- Lazylibrarian - Book manager
- Lidarr - Music manager
- Music Assistant - Music server
- Mylar - Comic manager
- Prowlarr - Indexer manager
- Radarr - Movie manager
- Sabnzbd - Usenet downloader
- Sonarr - TV show manager
- Whisparr - Adult content manager

### [40-network/](40-network/) - Network Services (4 apps)
Network infrastructure:
- AdGuard Home - DNS ad-blocker
- Contacts - Contact redirection service
- External-DNS - DNS automation
- Gluetun - VPN client

### [50-services/](50-services/) - General Services (7 apps)
Miscellaneous services:
- Authentik - SSO & authentication
- Home Assistant - Home automation
- Mail Gateway - Email gateway (Roundcube)
- Mosquitto - MQTT broker
- Netbox - IPAM & DCIM
- Netvisor - Network monitoring
- Vaultwarden - Password manager

### [70-tools/](70-tools/) - Tools & Utilities (10 apps)
Utility applications:
- ArgoCD Image Updater - (deprecated)
- Changedetection - Website change monitor
- Docspell - Document management
- GitOps Revision Controller - Git revision tracking
- Homepage - Dashboard
- IT-Tools - Web utilities
- Linkwarden - Bookmark manager
- Renovate - Dependency updater
- Stirling PDF - PDF tools
- Whoami - Test service

---

## Application Structure

Each application documentation includes:
- **Overview** - What the app does
- **Architecture** - How it's deployed
- **Configuration** - Key settings
- **Secrets** - Secret management
- **Ingress** - External access
- **Storage** - Persistent volumes
- **Dependencies** - What it depends on
- **Troubleshooting** - Common issues

---

## Quick Reference

### Adding a New Application
See [guides/adding-new-application.md](../guides/adding-new-application.md)

### Finding Application Code
```
apps/<category>/<app-name>/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml  (optional)
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    └── prod/
```

### Common Namespaces

| Namespace | Apps | Purpose |
|-----------|------|---------|
| `media` | *arr stack, Jellyfin, Jellyseerr | Media management (shared) |
| `tools` | Homepage, Linkwarden, Docspell | Utility apps (shared) |
| `services` | Home Assistant, Vaultwarden | General services (mixed) |
| `monitoring` | Prometheus, Grafana, Loki | Monitoring stack (shared) |
| `homeassistant` | Home Assistant | Dedicated namespace |
| `argocd` | ArgoCD | Dedicated namespace |
| `traefik` | Traefik | Dedicated namespace |

---

## Contributing

When adding new application documentation:
1. Create file in appropriate category directory
2. Use [templates/application-doc-template.md](../templates/application-doc-template.md)
3. Follow naming: `<app-name>.md` (lowercase, kebab-case)
4. Update this README with link
5. Deploy app code to `apps/<category>/<app-name>/`

---

**Last Updated:** 2026-01-20 (After Terraform migration to terravixens)
