# Application Status Dashboard

**Quick reference for application deployment status across environments.**

Last Updated: 2026-03-11

---

## Global Cluster Status

| Cluster | Nodes | Version | Status |
|---------|-------|---------|--------|
| **Prod** | 5 (peach, pearl, phoebe, poison, powder) | Talos v1.12.4 / K8s v1.34.0 | ✅ Active |
| **Dev** | 3 (daphne, diva, dulce) | - | ❌ Cert invalide |

| Component | Status | Description |
|-----------|--------|-------------|
| **ArgoCD Apps** | ✅ 88/89 Healthy | 1 OutOfSync accepted (openclaw PVC volumeName) |
| **Kustomize Build** | ✅ PASSING | |
| **CI/CD Pipelines** | ✅ ACTIVE | |

| **Quality Gates** | ✅ ENFORCED | Pre-commit + CI (secrets, YAML style, K8s validation) |
---

## Système de Maturité (ADR-023)

> **Référence:** [ADR-023: 7-Tier Goldification System v2](adr/023-7-tier-goldification-system-v2.md)

| Niveau | Nom | Description | Count |
|--------|-----|-------------|-------|
| 🥉 1 | **Bronze** | Déployée | 3 |
| 🥈 2 | **Silver** | Production Ready | 17 |
| 🥇 3 | **Gold** | Observable | 48 |
| 💎 4 | **Platinum** | Reliable | 17 |
| 🟢 5 | **Emerald** | Data Durability | 0 |
| 💠 6 | **Diamond** | Secure & Integrated | 0 |
| 🌟 7 | **Orichalcum** | Parfaite | 0 |
| ⚫ | **none** | Non labellisé | 1 |

**Total déploiements labellisés:** 86

---

## Legend

| Symbol | Status | Description |
|--------|--------|-------------|
| ✅ | **Healthy** | Synced, Healthy, pas d'issues |
| ⚠️ | **Degraded** | Fonctionne mais restarts élevés ou policy violations |
| ❌ | **Broken** | Unhealthy ou OutOfSync |
| 🚧 | **Progressing** | Sync ou rollout en cours |
| 💤 | **Paused** | Intentionnellement non déployé |

---

## Infrastructure (00-infra/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| argocd | ✅ | 🥇 Gold | v3.x - Self-Managed |
| kyverno | ✅ | 💎 Platinum | 4 controllers |
| velero | ✅ | 🥇 Gold | v1.17.x + Infisical |
| traefik | ✅ | 🥇 Gold | v3.x Ingress controller |
| cert-manager | ✅ | 🥈 Silver | TLS via Let's Encrypt |
| cert-manager-webhook-gandi | ✅ | 🥇 Gold | DNS-01 challenge |
| cilium-operator | ✅ | 💎 Platinum | CNI |
| synology-csi | ✅ | - | iSCSI storage |
| infisical-operator | ✅ | 🥇 Gold | Secrets management |
| reloader | ✅ | 🥈 Silver | ConfigMap/Secret reload |
| vpa | ✅ | 💎 Platinum | Vertical Pod Autoscaler |
| metrics-server | ✅ | 💎 Platinum | Resource metrics |

---

## Monitoring (02-monitoring/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| prometheus | ✅ | 🥇 Gold | |
| grafana | ✅ | 🥉 Bronze | Needs upgrade |
| loki | ✅ | - | Log aggregation |
| promtail | ✅ | - | Fixed: Probe timeout 1s→5s (PR #1980) |
| goldilocks | ✅ | 🥇 Gold | VPA recommendations |
| descheduler | ✅ | - | Pod rebalancing |
| policy-reporter | ✅ | 🥇 Gold | Kyverno reporting |

---

## Security (03-security/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| authentik | ✅ | 🥈 Silver (server) / 🥇 Gold (worker) | SSO Provider |
| trivy | ✅ | 🥇 Gold | Vulnerability scanning |

---

## Databases (04-databases/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| postgresql-shared | ✅ | - | CloudNativePG |
| redis-shared | ✅ | 🥇 Gold | |
| mariadb-shared | ✅ | - | |
| cloudnative-pg | ✅ | 🥇 Gold | Operator |

---

## Home Automation (10-home/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| homeassistant | ⚠️ | 🥈 Silver | **16 restarts**, PDB manquant |
| mealie | ✅ | 🥈 Silver | |
| mosquitto | ✅ | - | MQTT broker |

---

## Media (20-media/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| jellyfin | ✅ | 🥇 Gold | Media server |
| jellyseerr | ✅ | 🥇 Gold | Request management |
| sabnzbd | ✅ | 🥈 Silver | Usenet downloader |
| radarr | ✅ | 🥈 Silver | Movies |
| sonarr | ✅ | 🥈 Silver | TV Shows |
| prowlarr | ✅ | 🥈 Silver | Indexer manager |
| lidarr | ✅ | 🥈 Silver | Music |
| mylar | ✅ | 🥈 Silver | Comics |
| whisparr | ✅ | 🥈 Silver | Adult content |
| lazylibrarian | ✅ | 🥈 Silver | Books/Audiobooks |
| music-assistant | ✅ | 🥇 Gold | |
| frigate | ✅ | 🥈 Silver | NVR |
| hydrus-client | ✅ | 🥈 Silver | |
| booklore | ✅ | 🥇 Gold | |
| birdnet-go | ✅ | 🥉 Bronze | Bird detection |
| qbittorrent | ✅ | 💎 Platinum | Torrent client |
| pyload | ✅ | 💎 Platinum | Download manager |
| amule | ✅ | ⚫ none | ED2K client |

---

## Network (40-network/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| external-dns-unifi | ✅ | 🥇 Gold | Internal DNS |
| external-dns-gandi | ✅ | 🥇 Gold | Public DNS |
| netbird | ⚠️ | 🥇 Gold | **42 restarts**, SecurityContext manquant |
| netvisor | 🚧 | 🥇 Gold | Progressing |
| adguard-home | ✅ | - | Known: DNS dependency cycle (Litestream→MinIO) |
| contacts | ✅ | - | Redirection |

---

## Services (60-services/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| mail-gateway | ✅ | - | Email gateway |
| vaultwarden | ✅ | 🥈 Silver | Password manager |
| docspell | ✅ | - | Document management |
| gluetun | ✅ | 🥇 Gold | VPN container |
| firefly-iii | ✅ | 🥉 Bronze | Finance - needs upgrade |
| firefly-iii-importer | ✅ | 💎 Platinum | |
| openclaw | ⚠️ | 🥇 Gold | **34 restarts**, probes timeout |

---

## Tools (70-tools/)

| Application | Health | Maturity | Notes |
|-------------|--------|----------|-------|
| whoami | ✅ | 🥇 Gold | Test app |
| homepage | ✅ | 💎 Platinum | Dashboard |
| netbox | ✅ | 🥇 Gold | IPAM |
| changedetection | ✅ | 🥇 Gold | Website monitoring |
| stirling-pdf | ✅ | 🥈 Silver | PDF tools |
| it-tools | ✅ | 🥇 Gold | Dev tools |
| headlamp | ✅ | 🥇 Gold | K8s dashboard |
| linkwarden | ✅ | 🥇 Gold | Bookmark manager |
| vikunja | ⚠️ | 💎 Platinum | **37 restarts** |
| penpot | ✅ | 🥇 Gold | Design platform |
| renovate | ✅ | - | Fixed: OOMKilled - resources 512Mi→2Gi (PR #1980) |
| trilium | ✅ | 💎 Platinum | Notes |
| nocodb | ✅ | 💎 Platinum | Airtable alternative |
| radar | ✅ | 💎 Platinum | |

---

## Applications avec Issues

| Application | Restarts | Issue principale |
|-------------|----------|------------------|
| netbird-management | 42 | SecurityContext non durci |
| vikunja | 37 | À investiguer |
| openclaw | 34 | Probes en timeout |
| homeassistant | 16 | PDB manquant |

---

## Top Policy Failures (Kyverno)

| Policy | Failures | Impact |
|--------|----------|--------|
| check-backup | 317 | Emerald bloqué |
| check-pdb | 237 | Platinum bloqué |
| require-resources | 237 | Silver bloqué |
| require-probes | 198 | Silver bloqué |
| sizing-audit | 126 | Gold bloqué |
| check-security-context | 121 | Diamond bloqué |

---

## Environment Information

### Prod Cluster

- **Nodes:** peach, pearl, phoebe, poison, powder (3 CP + 2 workers)
- **VIP:** 192.168.111.190
- **IP Range:** 192.168.111.191 - 192.168.111.195
- **Status:** ✅ Active

### Dev Cluster

- **Nodes:** daphne, diva, dulce (3 CP HA)
- **VIP:** 192.168.111.160
- **Status:** ❌ Certificat invalide (kubeconfig à regénérer)

---

## Related Documentation

- **[ADR-023: 7-Tier Goldification System v2](adr/023-7-tier-goldification-system-v2.md)** - Source de vérité pour les niveaux de maturité
- **[Application Documentation](applications/)** - Documentation par app
- **[Maturity Standards Matrix](reference/maturity-standards-matrix.md)** - Matrice des exigences par niveau

---

**Last Updated:** 2026-03-11
