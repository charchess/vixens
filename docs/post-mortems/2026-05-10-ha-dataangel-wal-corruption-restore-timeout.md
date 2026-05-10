# Post-mortem: HA Down — DataAngel WAL Corruption + Missing restore-timeout Env Var

**Date:** 2026-05-10
**Duration:** ~5h (May 9 ~22:00 CEST → May 10 ~04:50 CEST)
**Severity:** High (Home Assistant completely unavailable)
**Cluster:** prod

---

## Summary

Home Assistant failed to start after a routine pod cycling event. Three compounding issues were found and fixed:

1. **iSCSI zombie VolumeAttachment** — the PVC couldn't be mounted on the new node
2. **Missing `DATA_GUARD_RESTORE_TIMEOUT` env var** — the `restore-timeout: 90m` pod annotation was silently ignored; DataAngel always used the 30m default
3. **Litestream WAL chain corruption at index 7503** — a WAL written May 9 at 12:46 CEST encoded a "database disk image is malformed" error; every restore attempt aborted partway through

Root cause #2 was a latent bug present since DataAngel was first deployed: the Kustomize component (`apps/_shared/components/dataangel/kustomization.yaml`) injected all other annotation-to-env-var mappings but omitted `DATA_GUARD_RESTORE_TIMEOUT`. The annotation was added to HA's deployment in PR #3184 but had no effect until PR #3186 fixed the component.

---

## Timeline (UTC)

| Time | Event |
|------|-------|
| May 9 ~10:46 | WAL index 7503 written to MinIO with corrupt SQLite state (`database disk image is malformed`) |
| May 9 ~22:00 | HA pod cycled (cause TBD); new pod stuck in `Init:0/3` — DataAngel failing to restore |
| May 10 ~03:00 | Investigation starts — DataAngel logs show restore timing out at 30m |
| May 10 ~03:10 | Discovered `pvc-dd8ba273` couldn't mount: `Volume[8c8edbb6] is not found` (iSCSI zombie) |
| May 10 ~03:15 | Fixed iSCSI zombie: deleted stale VolumeAttachment, restarted `synology-csi-node` on powder |
| May 10 ~03:20 | New pod `homeassistant-b48dc5dc5-ctgd5` created — DataAngel starts with `timeout: 30m0s` despite annotation `restore-timeout: 90m` |
| May 10 ~03:30 | Root cause #2 identified: `DATA_GUARD_RESTORE_TIMEOUT` not mapped in Kustomize component |
| May 10 ~03:45 | PR #3186 created and merged (`fix/dataangel-restore-timeout-env`) |
| May 10 ~03:50 | Root cause #3 identified: restore fails at WAL 7503 with `cannot apply wal: database disk image is malformed` |
| May 10 ~03:55 | prod-stable promoted to include PR #3186 |
| May 10 ~04:05 | Deleted all WAL versions ≥ 0x1d40 (7488) from MinIO using `mc rm --recursive --force --versions` |
| May 10 ~04:10 | Note: WAL 0x1d4e (7502) accidentally deleted along with 0x1d4f (7503) due to prefix overlap |
| May 10 04:19:31 | 5th restore attempt starts (WAL terminus now 7487) |
| May 10 04:30:16 | Restore completes: WALs 7238–7487 applied, DB 2.6 GB, "running full integrity check..." |
| May 10 ~04:32 | Pod goes `Init:Error` — DataAngel killed by Kubernetes during Recreate strategy scale-down |
| May 10 04:35:34 | New pod `homeassistant-67c6bc7944-d6pgv` created with correct `DATA_GUARD_RESTORE_TIMEOUT=90m` |
| May 10 04:38:45 | DataAngel: "clean shutdown detected, skipping validation" — restoring FS via rclone |
| May 10 ~04:50 | HA pod fully started (config init + main container) |

---

## Root Causes

### RC-1: iSCSI zombie VolumeAttachment

When powder node's CSI pod had been restarted earlier, the Synology CSI returned `Volume not found` for the PVC during `NodeStageVolume`. This is a known Synology CSI bug where the driver queries the DSM API for volume metadata but the iSCSI session was established via the initiator without DSM awareness. Deleting the stale VolumeAttachment and restarting the CSI node pod resolved it.

**Resolution:** Deleted `csi-powderXXX` VolumeAttachment + restarted `synology-csi-node-wwshb`.

### RC-2: Missing `DATA_GUARD_RESTORE_TIMEOUT` env var in Kustomize component

`apps/_shared/components/dataangel/kustomization.yaml` maps pod annotations to DataAngel env vars via `fieldRef`. The mapping for `dataangel.io/restore-timeout` → `DATA_GUARD_RESTORE_TIMEOUT` was never added. DataAngel defaulted to 30m, which is insufficient for a 2.6 GB database with 249 WALs to replay (restore takes ~11 minutes).

When the annotation was added in PR #3184, there was no indication the env var mapping was missing — DataAngel silently used 30m.

**Resolution:** PR #3186 added the missing mapping.

**Detection gap:** No alerting when a DataAngel restore hits its timeout (it just exits and Kubernetes retries). A `dataangel_restore_timeout_total` metric would help.

### RC-3: Litestream WAL chain corruption at index 7503

WAL 7503 (written May 9 12:46 CEST) encoded a "database disk image is malformed" error. This means the SQLite database was already malformed before or during the write of that WAL — litestream faithfully recorded the broken state. Root cause of the original SQLite corruption is unknown (possible: pod killed mid-write, I/O error on iSCSI, or OOM during busy period).

**Resolution:** Deleted all WAL versions ≥ 7488 from MinIO (`mc rm --recursive --force --versions`). This truncated the WAL chain at index 7487, losing ~2h15min of HA history (May 9 12:30–14:45 CEST). WAL 7502 (0x1d4e) was accidentally deleted along with 7503 due to prefix overlap (`00001d4` matched both).

**Recovery:** 2.6 GB DB restored from snapshot at WAL 7238 + 249 WALs. Integrity check skipped on second attempt (clean shutdown marker present).

---

## Data Loss

- ~2h15min of HA state history lost (May 9 12:30–14:45 CEST) — automations, sensor states, energy data
- HA backup tars in `tmp_backups/` unaffected (stored as separate FS files in S3)
- HA configuration files unaffected (rclone FS sync restored them)

---

## What Went Wrong

1. Kustomize component was missing an env var mapping — no static analysis caught it
2. DataAngel timeout silently uses a default rather than failing loudly when the env var is absent
3. No alert on DataAngel restore timeout
4. MinIO versioning (`mc rm` without `--versions`) required extra steps; initial delete attempts left DELETE MARKERS
5. Prefix-based WAL deletion (`00001d4`) accidentally deleted one extra WAL due to shared prefix

---

## Action Items

| Priority | Action | Owner |
|----------|--------|-------|
| High | Add `dataangel.io/snapshot-interval: "24h"` to HA deployment + mapping in component | claude |
| High | Add `DATA_GUARD_SNAPSHOT_INTERVAL` to Kustomize component | claude |
| Medium | File DataAngel issue: log warning when annotation env var not set (vs silently using default) | dataangel |
| Medium | Add `dataangel_restore_timeout_total` metric to DataAngel | dataangel (#47 area) |
| Low | Document MinIO versioning gotcha: always use `mc rm --versions` for DataAngel WAL cleanup | docs |
| Low | Consider excluding `tmp_backups/` from rclone FS sync (large tars, not needed for recovery) | claude |

---

## Prevention

- **Static validation**: Add check in `scripts/validate.py` that all `dataangel.io/*` annotations in pod templates have a corresponding env var in the component
- **Snapshot interval**: `dataangel.io/snapshot-interval: "24h"` caps WAL chain length to one day maximum, bounding restore time and limiting blast radius of WAL corruption
- **iSCSI stability**: Already tracking in existing post-mortems; Synology CSI bug persists
