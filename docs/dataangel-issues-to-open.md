# Issues à ouvrir sur truxonline/dataangel

Ces issues n'ont pas pu être créées automatiquement (token sans scope `issues:write`).
À créer manuellement sur https://github.com/truxonline/dataangel/issues

---

## Issue 1 — Restore rclone killed par timeout hardcodé 2 min (SIGKILL)

**Titre** : `fix: restore phase rclone killed by hardcoded 2-minute context timeout (SIGKILL)`

**Body** :

### Bug Description

`restoreFilesystem()` in `cmd/dataangel/restore.go` wraps the rclone subprocess in `context.WithTimeout(ctx, 2*time.Minute)`. When the deadline fires, Go's default `exec.CommandContext` sends **SIGKILL** (not SIGTERM), instantly killing rclone.

### Root Cause

```go
// cmd/dataangel/restore.go:~100
restoreCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)  // hardcoded
defer cancel()
cmd := exec.CommandContext(restoreCtx, "rclone", args...)
// NO cmd.Cancel → Go default = SIGKILL
// NO cmd.WaitDelay → immediate kill
```

The backup phase (`internal/sidecar/litestream.go`) correctly uses graceful shutdown:
```go
cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
cmd.WaitDelay = 15 * time.Second
```

The same issue affects `restoreSQLite()`.

### Observed Behavior

```
06:49:10 Running: rclone copy :s3:vixens-dev-mealie/data /app/data --s3-env-auth --exclude *.db* --timeout 60s
06:51:10 phase=restore failed: rclone copy failed: signal: killed
```

Exactly 120 seconds. Bucket has ~26KB under `data/` prefix (5 objects), but rclone listing is slow on MinIO.

### Impact

- CrashLoopBackOff on first boot with FS paths configured
- Pod stabilizes after 2-3 restarts (partial restores accumulate in emptyDir)
- Not idempotent: each restart re-downloads already-restored files

### Suggested Fix

1. **Make timeout configurable**: `DATA_GUARD_RESTORE_TIMEOUT` (default 10m)
2. **Graceful shutdown**: set `cmd.Cancel = SIGTERM` + `cmd.WaitDelay = 15s`
3. **Increase default**: 2 min too aggressive for FS restores

### Comparison

| Property | Restore (broken) | Backup (correct) |
|---|---|---|
| Timeout | 2 min hardcoded | 3 min hardcoded |
| Kill signal | SIGKILL | SIGTERM |
| Grace period | None | 15s |
| Configurable | No | No |

**Version**: `charchess/dataangel:dev` (post-commit 97cd875c)

---

## Issue 2 — Lock renewal fails with context deadline exceeded at startup

**Titre** : `fix: lock renewal context deadline exceeded during initial backup startup`

**Body** :

### Bug Description

After the backup phase starts, S3 lock renewals fail consistently for ~6 minutes with `context deadline exceeded`, then stabilize. This creates a window where the lock may expire (TTL 60s) while renewals fail.

### Observed Behavior

```
06:54:01 Lock acquired, ready for traffic
06:54:01 Starting litestream replicator + rclone sync loop
06:54:41 Failed to renew lock: operation error S3: PutObject, StatusCode: 0, canceled, context deadline exceeded
06:55:11 Failed to renew lock: [same]
... every 30s for ~6 minutes ...
07:00:05 wal segment written (elapsed=5m1.49s)  ← litestream finally writes
```

### Root Cause Hypothesis

At startup, litestream (snapshot upload), rclone (full sync), and lock renewal all compete for S3/MinIO connections simultaneously. The lock renewal has a 10s per-renewal context timeout which is insufficient when MinIO is saturated by the initial litestream snapshot + rclone sync.

### Impact

- **Split-brain risk**: Lock TTL is 60s, but renewals fail for 6+ minutes → lock expires → another pod could acquire it
- Pod remains Ready despite expired lock (readiness probe doesn't check lock validity)
- Eventual consistency: system recovers after ~6 min when S3 load decreases

### Suggested Fix

1. **Sequence startup**: Acquire lock → start litestream → wait for first snapshot → start rclone. Don't start everything simultaneously.
2. **Increase renewal timeout**: 10s → 30s (or configurable)
3. **Readiness probe should check lock**: If lock is lost, mark pod as not-ready

**Version**: `charchess/dataangel:dev` (post-commit 97cd875c)
**Environment**: MinIO on Synology NAS, mealie app, Kubernetes 1.34.0
