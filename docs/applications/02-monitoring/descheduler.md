# Descheduler (Compliant Version)

## Status
- **Dev**: Week 1 - DryRun Mode
- **Prod**: Pending Week 4

## Policy
- **Schedule**: 0 */4 * * *
- **Mode**: DryRun Enabled (--dry-run)
- **Strategy**: LowNodeUtilization (Thresholds: 30% / Targets: 60%)
- **Constraints**: numberOfNodes: 3, priorityThreshold: 10000
- **Exclusions**: kube-system, homeassistant namespaces

## Protections
- Home Assistant: homelab-critical (100000) - OK
- Frigate: homelab-important (50000) - OK
- Jellyfin: homelab-important (50000) - OK
- Traefik/ArgoCD: homelab-critical - PENDING (Requires Terraform update)
