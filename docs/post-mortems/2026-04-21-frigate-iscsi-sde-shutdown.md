# Post-Mortem: Frigate iSCSI Mount Failure (sde shutdown loop)

**Date:** 2026-04-21  
**Duration:** ~14 hours (00:00 → 12:35 UTC)  
**Severity:** High — Frigate NVR down in prod  
**Node:** poison (192.168.111.194)  
**Volume:** frigate-cache-pvc (`pvc-9d89a9b3-8cf9-4f6f-8d09-6938f471a918`, 10Gi iSCSI, `/dev/sde`)

---

## Summary

Frigate failed to start in prod due to a cascading iSCSI mount failure on the `frigate-cache-pvc` volume. The ext4 filesystem on `/dev/sde` repeatedly entered `shutdown` mode due to iSCSI session drops, causing I/O errors. The root cause was a stale iSCSI session state combined with the Synology refusing `SendTargets` discovery requests after repeated reconnection attempts.

---

## Timeline

| Time | Event |
|------|-------|
| 2026-04-17 21:44 | First `device offline error` on `/dev/sde` (correlated with `synology-csi-node` pod restart) |
| 2026-04-21 00:00 | Frigate deployment updated (sizing fix) — pod rescheduled |
| 00:11 | Second `device offline error` on sde → ext4 shutdown |
| 00:45 | Third `device offline error` → ext4 shutdown again |
| ~01:00 | iSCSI session manually logged out, fsck ran, filesystem repaired |
| 01:00–12:30 | Repeated NodeStageVolume failures: Synology refusing discovery (`Connection to Discovery Address closed`), kubelet 2-min timeout exceeded |
| 12:21 | Session confirmed clean (`Session doesn't exist` after logout) |
| 12:31 | NodeStageVolume succeeds via pre-created node DB entry + direct login |
| 12:35 | Frigate `2/2 Running` |

---

## Root Cause

**Two compounding issues:**

1. **Synology refusing iSCSI SendTargets discovery** after too many reconnection attempts from the same initiator. The CSI driver's NodeStageVolume flow does `iscsiadm -m discovery -t sendtargets` first; when the Synology closes this connection, the driver fails immediately instead of falling back to direct login.

2. **Stale globalmount with `shutdown` flag** persisting across pod restarts. Once the ext4 filesystem enters shutdown mode (triggered by an iSCSI session drop), the mount path retains the `shutdown` flag even after the session reconnects. A `fsck` is required to clear the `orphan_present` flag.

**Why sde specifically:** The `synology-csi-node` pod restarted on 2026-04-17, disrupting the iSCSI session for `sde` mid-write. This corrupted the journal and caused the initial `device offline` error chain. Other volumes (sdb, sdf, sdh) were not affected because their sessions survived the restart cleanly.

---

## Resolution

1. `iscsiadm --logout` on the stale session
2. `fsck.ext4 -y /dev/sde` via the `synology-csi-plugin` container (fixed `orphan_present` + journal)
3. Manually created the iSCSI node DB entry (`--op new`) to bypass discovery
4. Pre-logged in the session so NodeStageVolume found it ready
5. Force-deleted stuck pod — CSI mounted cleanly on fresh attempt

---

## What Did NOT Work

- Restarting `ext-iscsid` service on poison (session issue was not daemon-side)
- Waiting for auto-retry (kubelet 2-min timeout consistently exceeded due to slow discovery)
- `kubectl debug node/` (blocked by PodSecurity baseline policy)

---

## Action Items

- [ ] Investigate why Synology refuses `SendTargets` discovery after repeated reconnects — consider increasing rate limits in DSM iSCSI settings
- [ ] Add monitoring/alert on ext4 `shutdown` flag on iSCSI volumes (check `/proc/mounts` for `,shutdown` option)
- [ ] Consider pre-seeding iSCSI node DB entries on `synology-csi-node` pod startup to avoid discovery dependency
- [ ] Investigate the 2026-04-17 `synology-csi-node` pod restart root cause (see post-mortem `2026-04-17-homeassistant-iscsi-crash.md`)
- [ ] Consider moving frigate-cache-pvc to NFS instead of iSCSI (cache data is reconstructible, NFS is more resilient)
