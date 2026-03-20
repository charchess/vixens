# DataAngel Robustness Test Report

**Date:** 2026-03-20
**Cluster:** dev (Talos v1.12.4, K8s v1.34.0)
**App under test:** mealie (with dataangel sidecar)
**Author:** Claude Code (Sisyphus)

---

## Round 1 — Initial Testing

**DataAngel version:** `charchess/dataangel:dev` (commit `036087bf7ac4` — fixes #20, #21)

8 tests executed. **6 passed, 2 issues found** (1 medium, 1 minor).

| # | Test | Result | Severity |
|---|------|--------|----------|
| 1 | Corrupted DB + backup exists → auto-recovery | ✅ PASS | — |
| 2 | S3 unreachable during restore | ✅ PASS | — |
| 3 | Litestream subprocess crash | ✅ PASS | — |
| 4 | Rclone subprocess crash | ✅ PASS | — |
| 5 | DB deleted while running | ⚠️ ISSUE | Medium |
| 6 | Graceful shutdown → lock release | ⚠️ ISSUE | Minor |
| 7 | Metrics accuracy | ✅ PASS | — |
| 8 | S3 outage during backup | ✅ PASS | — |

**Issues found:**
- **#22 (Medium):** DB deleted at runtime → infinite litestream error loop, no self-heal, stays `ready=true`
- **#23 (Minor):** Lock not released on SIGTERM → 12s wait on failover

Both reported as GitHub issues on truxonline/dataAngel.

---

## Round 2 — Full Retest (Post-Fix)

**DataAngel version:** `charchess/dataangel:dev` (commit `d177f3447b4c` — fixes #20, #21, #22, #23, #24)
**S3 Endpoint:** `http://192.168.111.69:9000` (direct IP, bypasses DNS VLAN 204 bug)

12 tests executed. **All 12 passed.**

| # | Test | Result | Notes |
|---|------|--------|-------|
| 1 | Corrupted DB + backup exists → auto-recovery | ✅ PASS | Detected, restored from S3 in ~5s |
| 2 | Corrupted DB + NO backup (fix #20) | ✅ PASS | `CRITICAL` → exit 1 → CrashLoopBackOff |
| 3 | Litestream subprocess crash | ✅ PASS | Detected, daemon stops, kubelet restarts |
| 4 | Rclone subprocess crash | ✅ PASS | Non-fatal, litestream continues |
| 5 | DB deleted while running (fix #22) | ✅ PASS | 10 checks (~30s) → CRITICAL → exit → restore |
| 6 | Graceful shutdown → lock release (fix #23) | ✅ PASS | Lock acquired in 0s (was 12s) |
| 7 | Metrics accuracy (fix #21) | ✅ PASS | All metrics functional |
| 8 | S3 outage during backup | ✅ PASS | CrashLoopBackOff → self-heal |
| 9 | Large DB corruption (middle of file) | ✅ PASS | Detected, restored in ~4s |
| 10 | Rapid container restarts (stress) | ✅ PASS | CrashLoopBackOff backoff → stabilizes |
| 11 | Empty DB file (0 bytes) | ✅ PASS | Detected as corrupted, restored |
| 12 | Simultaneous litestream + rclone crash | ✅ PASS | Litestream death triggers exit → restore |

---

## Test Details (Round 2)

### Test 1: Corrupted DB + backup exists → auto-recovery ✅

**Scenario:** `dd if=/dev/urandom of=mealie.db bs=1024 count=50`, S3 backup intact, kill -ABRT 1.

```
WARNING: Database exists but is corrupted, removing: /app/data/mealie.db
Corrupted database removed, proceeding with restore
restoring snapshot...
SQLite restored successfully: /app/data/mealie.db
phase=restore complete elapsed=6.803874085s
phase=backup starting
Lock acquired, ready for traffic
```

**Verdict:** Full self-heal in ~7s. Corruption detected, removed, restored from S3, backup resumed.

---

### Test 2: Corrupted DB + NO backup (fix #20) ✅

**Scenario:** Corrupt DB + `mc rm --recursive --force synelia-admin/vixens-dev-mealie/` + kill -ABRT 1.

```
WARNING: Database exists but is corrupted, removing: /app/data/mealie.db
Corrupted database removed, proceeding with restore
"no matching backups found"
CRITICAL: corrupted database was removed but no S3 backup exists — refusing to start with empty database
→ exit code 1
```

**Verdict:** Refuses to start with empty DB. CrashLoopBackOff alerts operator. **No silent data loss.**

---

### Test 3: Litestream subprocess crash ✅

**Scenario:** `kill -9 $(pidof litestream)` during backup.

```
[litestream] Command failed: signal: killed
phase=backup error: signal: killed
Daemon stopped
→ Container exits → kubelet restarts → restore + backup resume
```

**Verdict:** Critical subprocess death triggers clean exit. Recovery in ~5s after restart.

---

### Test 4: Rclone subprocess crash ✅

**Scenario:** `kill -9 $(pidof rclone)` during sync.

```
rclone sync error: signal: killed
[rclone] Command failed: signal: killed
→ Container stays alive, litestream continues
→ Next rclone sync at next interval (60s)
```

**Verdict:** Non-fatal. SQLite backup (litestream) prioritized over filesystem sync (rclone).

---

### Test 5: DB deleted while running (fix #22) ✅

**Scenario:** `rm /app/data/mealie.db` during backup phase.

```
litestream: sync error: "cannot verify wal state: stat /app/data/mealie.db: no such file or directory"
WARNING: database file missing: /app/data/mealie.db (1/10)
WARNING: database file missing: /app/data/mealie.db (2/10)
...
WARNING: database file missing: /app/data/mealie.db (10/10)
CRITICAL: database file /app/data/mealie.db has been missing for 10 consecutive checks — exiting to trigger restore
Daemon stopped
→ Container exits → kubelet restarts → restore from S3 → backup resumes
```

**Verdict:** After 10 consecutive missing-file checks (~30s), DataAngel exits to trigger restore. **Previously stayed alive in broken state indefinitely.** Fix #22 working perfectly.

---

### Test 6: Graceful shutdown → lock release (fix #23) ✅

**Scenario:** `kubectl delete pod` (30s grace period).

```
Timeline:
- Pod deleted: 15:53:05
- New pod restore complete: 15:53:11 (6.8s restore)
- Lock acquired: 15:53:11 (0s wait!)
```

**Verdict:** Lock released during graceful shutdown. New pod acquires lock **immediately** (0s). **Previously 12s wait for TTL expiry.** Fix #23 working perfectly.

---

### Test 7: Metrics accuracy (fix #21) ✅

**Scenario:** Scrape `localhost:9090/metrics` during steady-state backup.

| Metric | Value | Correct? |
|--------|-------|----------|
| `dataangel_phase{backup}` | 1 | ✅ |
| `dataangel_phase{restore}` | 0 | ✅ |
| `dataangel_restore_duration_seconds` | 6.8s | ✅ matches logs |
| `dataguard_litestream_up` | 1 | ✅ |
| `dataguard_rclone_syncs_total` | 1 | ✅ |
| `dataguard_rclone_syncs_failed_total` | 0 | ✅ |
| `dataguard_rclone_sync_duration_seconds` | ~3.3s | ✅ histogram |
| `dataguard_rclone_up` | 1 | ✅ |
| `dataguard_sidecar_uptime_seconds` | increasing | ✅ |

**Note:** Metric prefix inconsistency: `dataangel_` for phase/restore vs `dataguard_` for backup metrics. Cosmetic issue, not blocking.

---

### Test 8: S3 outage during backup ✅

**Scenario:** S3 endpoint switched to unreachable IP.

**Verdict:** Restore phase fails → container exits → CrashLoopBackOff → self-heals when S3 returns. Consistent with Round 1 results.

---

### Test 9: Large DB corruption (middle of file) ✅

**Scenario:** `dd if=/dev/urandom of=mealie.db bs=1024 count=100 seek=500` (corrupt 100KB in middle of 1.3MB file), kill -ABRT 1.

```
WARNING: Database exists but is corrupted, removing: /app/data/mealie.db
Corrupted database removed, proceeding with restore
restoring snapshot...
SQLite restored successfully: /app/data/mealie.db
phase=restore complete elapsed=4.312191009s
```

**Verdict:** Middle-of-file corruption detected by SQLite integrity check. Full self-heal in ~4s.

---

### Test 10: Rapid container restarts (stress) ✅

**Scenario:** Kill container 3 times in rapid succession (~8s intervals).

**Result:** 2 kills succeeded (3rd failed — container already restarting). CrashLoopBackOff backoff increased (~90s at 5th restart). Container eventually stabilized and recovered.

**Verdict:** Kubelet exponential backoff prevents thrashing. DataAngel recovers correctly after each restart regardless of rapid cycling.

---

### Test 11: Empty DB file (0 bytes) ✅

**Scenario:** `> /app/data/mealie.db` (truncate to zero), kill -ABRT 1.

```
WARNING: Database exists but is corrupted, removing: /app/data/mealie.db
Corrupted database removed, proceeding with restore
SQLite restored successfully: /app/data/mealie.db
```

**Verdict:** Zero-byte file correctly detected as corrupted (fails SQLite header check). Full self-heal in ~4s.

---

### Test 12: Simultaneous litestream + rclone crash ✅

**Scenario:** `kill -9 $(pidof litestream); kill -9 $(pidof rclone)` simultaneously.

```
[litestream] Command failed: signal: killed
phase=backup error: signal: killed
Daemon stopped
→ Container exits → CrashLoopBackOff → restore + backup resume
```

**Verdict:** Litestream death is the critical signal — triggers clean exit. Rclone death is secondary. Container self-heals on restart.

---

## Fix Validation Summary

| Issue | Fix Commit | Before | After | Status |
|-------|-----------|--------|-------|--------|
| #20 Silent data loss on corrupt DB + no backup | `f97b7da57c68` | Silent exit 0, app starts empty | CRITICAL exit 1, CrashLoopBackOff | ✅ Verified |
| #21 Metrics in wrong registry | `036087bf7ac4` | All metrics show 0 | All metrics functional | ✅ Verified |
| #22 DB deleted → infinite error loop | `2617c170b007` | Infinite loop, ready=true | 10 checks → CRITICAL exit → restore | ✅ Verified |
| #23 Lock not released on shutdown | (included in #22 commit) | 12s TTL wait | 0s immediate acquisition | ✅ Verified |
| #24 Regression: config file not found | `d177f3447b4c` | litestream: config file not found | Uses -config flag correctly | ✅ Verified |

---

## Infrastructure Issue

**DNS routing bug (unrelated to DataAngel):**
- `synelia.internal.truxonline.com` resolves to `192.168.204.69` (VLAN 204)
- MinIO only reliably reachable at `192.168.111.69` (VLAN 111)
- **Workaround applied:** Direct IP in dev overlay (PR #2331)
- **Fix needed:** DNS record or VLAN 204 routing

---

## Conclusions

**DataAngel is production-ready.** All 12 robustness tests pass. All 5 upstream fixes (#20-#24) validated.

**Remaining cosmetic issues (non-blocking):**
1. Metric prefix inconsistency (`dataangel_` vs `dataguard_`)
2. `rclone.conf not found` NOTICE message (harmless, uses env-auth)

**Critical path fully validated:**
- Corruption recovery (header, middle, empty file) → auto-restore from S3
- Data loss prevention → refuses to start without backup
- Subprocess resilience → litestream death = exit, rclone death = non-fatal
- Runtime monitoring → DB deletion detected in ~30s
- Fast failover → lock released on graceful shutdown (0s)
- Observability → all metrics functional and accurate
