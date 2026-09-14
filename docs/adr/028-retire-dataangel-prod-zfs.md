# ADR-028: Retire DataAngel Restore Init from Production Workloads

**Date:** 2026-09-14
**Status:** Accepted
**Deciders:** Vixens maintainers
**Tags:** backup, disaster-recovery, truenas, zfs, gitops

---

## Context

DataAngel injects a filesystem backup/restore init container into application
Pods. Its automatic restore happens before the application starts. This makes
boot dependent on object-store availability and on an independently managed
readiness endpoint. Recent recovery incidents showed that an interrupted or
orphaned DataAngel process can retain its listener and block unrelated
workload startup; automatic restore can also overwrite an intentional local
recovery change.

Production persistent volumes are provisioned on TrueNAS. The durable storage
strategy is now ZFS snapshots with GFS retention and replication through
`zfs send` to an independent target. This protects the volume layer without
an application init container altering its contents during every startup.

## Decision

- Remove the DataAngel Kustomize component and its paired `dataangel.yaml`
  patch from every production overlay that currently enables it.
- Keep the shared DataAngel component and non-production overlays intact so a
  controlled test/recovery path remains available.
- Preserve PVCs, retained PVs, historical S3 objects, and existing
  application-native backups. This decision does not delete any data or S3
  credentials.
- Use application-native logical backups where supported (for example Home
  Assistant backups and database-native backups) in addition to ZFS volume
  snapshots. ZFS replication must target a failure domain independent of the
  source pool/NAS.
- Restore data by selecting and cloning/rolling back a ZFS snapshot through a
  deliberate recovery runbook; workloads must not automatically restore a
  filesystem at ordinary Pod startup.

## Affected production overlays

- `apps/03-security/authentik/overlays/prod`
- `apps/10-home/mealie/overlays/prod`
- `apps/10-home/mosquitto/overlays/prod`
- `apps/20-media/amule/overlays/prod`
- `apps/20-media/birdnet-go/overlays/prod`
- `apps/20-media/booklore/overlays/prod`
- `apps/20-media/bookshelf/overlays/prod`
- `apps/20-media/frigate/overlays/prod`
- `apps/20-media/hydrus-client/overlays/prod`
- `apps/20-media/jellyseerr/overlays/prod`
- `apps/20-media/lidarr/overlays/prod`
- `apps/20-media/music-assistant/overlays/prod`
- `apps/20-media/mylar/overlays/prod`
- `apps/20-media/prowlarr/overlays/prod`
- `apps/20-media/pyload/overlays/prod`
- `apps/20-media/qbittorrent/overlays/prod`
- `apps/20-media/radarr/overlays/prod`
- `apps/20-media/sabnzbd/overlays/prod`
- `apps/20-media/sonarr/overlays/prod`
- `apps/20-media/whisparr/overlays/prod`
- `apps/40-network/adguard-home/overlays/prod`
- `apps/40-network/netbird/overlays/prod`
- `apps/60-services/firefly-iii/overlays/prod`
- `apps/60-services/n8n/overlays/prod`
- `apps/60-services/vaultwarden/overlays/prod`
- `apps/70-tools/changedetection/overlays/prod`
- `apps/70-tools/linkwarden/overlays/prod`
- `apps/70-tools/netbox/overlays/prod`
- `apps/70-tools/nexterm/overlays/prod`
- `apps/70-tools/nocodb/overlays/prod`
- `apps/70-tools/trilium/overlays/prod`
- `apps/70-tools/vikunja/overlays/prod`

## Consequences

### Positives

- Workload startup is no longer gated on DataAngel/S3 restore availability.
- A failed/orphaned DataAngel init container cannot block these Pods.
- Operator-driven ZFS recovery preserves a clearly selected recovery point
  instead of applying an implicit object-store state at every restart.

### Constraints and migration gates

- A same-NAS snapshot alone is not offsite backup. GFS retention and successful
  `zfs send` replication must be monitored and periodically restore-tested.
- Stateful databases still require their native consistency/recovery policy;
  this change removes DataAngel injection, not database backup obligations.
- Production rollout recreates affected workloads. Merge and promotion require
  explicit owner approval, followed by application-by-application health
  verification; no imperative Pod deletion is used to enforce the change.

## Validation

Before merge, build every affected production overlay, lint only changed YAML,
and assert no affected rendered Deployment has `dataangel` in
`spec.template.spec.initContainers`. After an approved promotion, require Argo
CD to resolve the promoted revision and inspect each affected Application/Pod;
report workloads that remain unhealthy separately from DataAngel removal.

## References

- [ADR-013: Layered Configuration and Disaster Recovery](013-layered-configuration-disaster-recovery.md)
- [Application status dashboard](../STATUS.md)
- [ADR-017: Pure Trunk-Based Development](017-pure-trunk-based-single-branch.md)
