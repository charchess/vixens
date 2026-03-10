---
name: minio-admin
description: "MinIO administration on the homelab. Manage buckets, users, policies, service accounts, lifecycle, and object operations via mc CLI. Use when: any MinIO/S3 storage task on the homelab cluster."
---

# MinIO Admin — Homelab Skill

> Full MinIO administration via `mc` CLI for the Vixens homelab cluster.

## Connection

```bash
# Alias already configured
mc alias set minio http://192.168.111.69:9000 admin 'S0d0m!e69'
```

- **API endpoint**: `http://192.168.111.69:9000` (S3 API)
- **Console**: NOT exposed (port 9090 unreachable)
- **Version**: `2022-10-24T18-35-07Z` (single-node, single-drive)
- **Alias**: `minio`
- **mc version**: `RELEASE.2025-08-13`

## Server Info

```bash
mc admin info minio
# 1.1 TiB Used, 32 Buckets, 470K+ Objects, 586K+ Versions
# 1 drive online, EC:0 (no erasure coding — single drive)
```

---

## ⚠️ Version Compatibility Notes

This MinIO instance (2022-10-24) has API limitations with modern `mc` (2025-08-13).
**Root cause**: Server is too old for the new `idp/builtin/policy/attach` API, but new `mc` rejects the legacy `policy set`.

| Feature | Status | Workaround |
|---------|---------| -----------|
| `mc admin policy attach/detach` | ❌ Server too old (needs ≥2023-03) | Use `mc-compat` (see below) |
| `mc admin policy set/unset` | ❌ Rejected by new mc client | Use `mc-compat` (see below) |
| `mc admin user svcacct list` | ❌ JSON parse error (version mismatch) | Use `mc-compat` or `svcacct info` per key |
| `mc admin heal` | ❌ Single-drive mode | Not applicable |
| SSE encryption | ❌ KMS not configured | Cannot enable server-side encryption |
| Object retention | ⚠️ Requires `--with-lock` at bucket creation | Cannot enable on existing buckets |
| Replication | ❌ Not configured | Single-node, no replication target |
| Event notifications | ❌ No webhook configured | Must configure `notify_webhook` first |

### mc-compat: Legacy Client Workaround

An old `mc` binary (RELEASE.2022-10-29) is installed at `~/bin/mc-compat` for commands
broken by the version mismatch. It has its own alias config:

```bash
# Setup (already done)
mc-compat alias set minio http://192.168.111.69:9000 admin 'S0d0m!e69'

# Attach/detach policies (WORKING with mc-compat)
mc-compat admin policy set minio <policy-name> user=<username>
mc-compat admin policy set minio <policy-name> group=<groupname>
mc-compat admin policy unset minio <policy-name> user=<username>
mc-compat admin policy unset minio <policy-name> group=<groupname>

# List service accounts (WORKING with mc-compat)
mc-compat admin user svcacct list minio <username>
```

**Long-term fix**: Upgrade MinIO server to ≥ RELEASE.2023-03-xx, then use modern `mc`.

**Use modern `mc` for everything else** (buckets, objects, users, groups, policies CRUD, etc.).
Only use `mc-compat` for: policy attach/detach and svcacct list.

---

## Buckets

### List

```bash
mc ls minio/                              # List all buckets
mc ls minio/bucket-name/                  # List objects in bucket
mc ls minio/bucket-name/ --recursive      # All objects recursively
mc ls minio/bucket-name/ --versions       # Include version IDs
```

### Create / Delete

```bash
mc mb minio/my-new-bucket                 # Create bucket
mc mb minio/my-locked-bucket --with-lock  # Create with Object Lock enabled
mc rb minio/my-bucket                     # Delete empty bucket
mc rb --force minio/my-bucket             # Delete bucket + all contents
```

### Versioning

```bash
mc version info minio/bucket-name         # Check versioning status
mc version enable minio/bucket-name       # Enable versioning
mc version suspend minio/bucket-name      # Suspend versioning (keeps existing versions)
```

**Current state**: backups-vixens-{dev,prod}, terraform-state-* have versioning enabled. Others are un-versioned.

### Lifecycle (ILM)

```bash
# Add lifecycle rule — expire noncurrent versions after N days
mc ilm add minio/bucket-name --noncurrent-expire-days 30

# Add expiration for current versions
mc ilm add minio/bucket-name --expire-days 90

# Add with prefix filter
mc ilm add minio/bucket-name --prefix "logs/" --expire-days 7

# List lifecycle rules
mc ilm ls minio/bucket-name

# Remove all lifecycle rules
mc ilm rm --all --force minio/bucket-name
```

**Current state**: backups-vixens-prod has `PurgeOldVersions` (noncurrent expire 7d, delete markers purged).

### Quota

```bash
mc quota set minio/bucket-name --size 10GB   # Set hard quota
mc quota info minio/bucket-name               # Check quota
mc quota clear minio/bucket-name              # Remove quota
```

### Tags

```bash
mc tag set minio/bucket-name "env=prod&app=frigate"   # Set tags
mc tag list minio/bucket-name                          # List tags
mc tag remove minio/bucket-name                        # Remove tags
```

### Object Lock / Retention (requires --with-lock at creation)

```bash
mc mb minio/immutable-bucket --with-lock              # Must create with lock
mc retention set --default compliance 30d minio/immutable-bucket  # Set default retention
mc retention info minio/immutable-bucket               # Check retention policy
```

### Anonymous / Public Access

```bash
mc anonymous set download minio/bucket-name        # Allow public downloads
mc anonymous set upload minio/bucket-name          # Allow public uploads
mc anonymous set public minio/bucket-name          # Full public access
mc anonymous set none minio/bucket-name            # Remove all public access (private)
mc anonymous get minio/bucket-name                 # Check current policy

# Custom bucket policy (JSON)
mc anonymous set-json /path/to/policy.json minio/bucket-name
mc anonymous get-json minio/bucket-name
```

---

## Objects

### Upload / Download

```bash
mc cp local-file.txt minio/bucket/path/file.txt       # Upload single file
mc cp minio/bucket/path/file.txt ./local-file.txt     # Download
mc cp --recursive ./local-dir/ minio/bucket/prefix/   # Upload directory
mc cp minio/bucket/file.txt minio/other-bucket/file.txt  # Copy between buckets
mc mv minio/bucket/old.txt minio/bucket/new.txt       # Move/rename
```

### Delete

```bash
mc rm minio/bucket/file.txt                           # Delete object
mc rm --recursive --force minio/bucket/prefix/        # Delete all under prefix
mc rm --older-than 30d minio/bucket/logs/             # Delete older than 30 days
mc rm --versions minio/bucket/file.txt                # Delete all versions
```

### Read / Inspect

```bash
mc cat minio/bucket/file.txt                          # Print file contents
mc head minio/bucket/file.txt                         # First 10 lines
mc stat minio/bucket/file.txt                         # Metadata (size, ETag, version, content-type)
mc du minio/bucket-name                               # Disk usage
mc find minio/bucket-name --name "*.log"              # Find files by pattern
mc find minio/bucket-name --larger 100MB              # Find by size
mc find minio/bucket-name --older 30d                 # Find by age
```

### Presigned URLs

```bash
mc share download minio/bucket/file.txt               # 7-day download link (default)
mc share download --expire 24h minio/bucket/file.txt  # Custom expiry
mc share upload minio/bucket/prefix/                  # Upload presigned URL
```

### Mirror (sync)

```bash
mc mirror ./local-dir/ minio/bucket/prefix/           # Sync local → MinIO
mc mirror minio/bucket/ ./local-backup/               # Sync MinIO → local
mc mirror --overwrite minio/src/ minio/dst/           # Sync between buckets
mc mirror --dry-run ./local/ minio/bucket/            # Preview only
```

### Tags (object-level)

```bash
mc tag set minio/bucket/file.txt "type=backup&retention=30d"
mc tag list minio/bucket/file.txt
mc tag remove minio/bucket/file.txt
```

### SQL Select (CSV/JSON/Parquet only)

```bash
mc sql --query "SELECT * FROM s3object WHERE status='error'" minio/bucket/data.csv
```

---

## Users

### CRUD

```bash
mc admin user add minio username 'SecurePassword123!'  # Create user
mc admin user list minio                                # List all users
mc admin user info minio username                       # User details + policies
mc admin user disable minio username                    # Disable user
mc admin user enable minio username                     # Enable user
mc admin user remove minio username                     # Delete user
```

### Current Users

| User | Policy | Groups | Purpose |
|------|--------|--------|---------|
| terraform | terraform-state | — | Terraform state storage |
| velero-prod | — | — | K8s backup (Velero) |
| litestream | backups-vixens | — | SQLite streaming backup |
| birdnet-go-access | — | — | BirdNET-Go audio storage |
| openclaw-user | — | openclaw-access-group | OpenClaw app |
| vixens-openclaw-prod | — | — | OpenClaw prod |

---

## Groups

```bash
mc admin group add minio group-name user1 user2        # Create group + add members
mc admin group remove minio group-name user1            # Remove member from group
mc admin group list minio                               # List all groups
mc admin group info minio group-name                    # Group details
mc admin group disable minio group-name                 # Disable group
mc admin group enable minio group-name                  # Enable group
```

---

## Policies

### Built-in Policies

| Policy | Description |
|--------|-------------|
| `readonly` | Read-only access to all buckets |
| `readwrite` | Full read/write to all buckets |
| `writeonly` | Write-only access to all buckets |
| `diagnostics` | Server diagnostics access |
| `consoleAdmin` | Full admin console access |

### Custom Policy CRUD

```bash
# Create custom policy from JSON
mc admin policy create minio policy-name /path/to/policy.json

# List all policies
mc admin policy list minio

# View policy details
mc admin policy info minio policy-name

# Update policy (recreate)
mc admin policy create minio policy-name /path/to/updated-policy.json

# Delete policy
mc admin policy remove minio policy-name
```

### Policy Template (per-app pattern)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListBucket"
      ],
      "Resource": ["arn:aws:s3:::BUCKET-NAME"]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:DeleteObject",
        "s3:GetObject",
        "s3:ListMultipartUploadParts",
        "s3:PutObject"
      ],
      "Resource": ["arn:aws:s3:::BUCKET-NAME/*"]
    }
  ]
}
```

### Attaching Policies to Users/Groups

Use `mc-compat` (legacy mc client) for policy attach/detach:

```bash
# Attach policy to user
mc-compat admin policy set minio <policy-name> user=<username>

# Attach policy to group
mc-compat admin policy set minio <policy-name> group=<groupname>

# Detach policy from user
mc-compat admin policy unset minio <policy-name> user=<username>

# Detach policy from group
mc-compat admin policy unset minio <policy-name> group=<groupname>

# Verify
mc admin user info minio <username>
```

**Alternative 1**: Create service account with scoped policy (see below).
**Alternative 2**: Bucket-level policies via `mc anonymous set-json` for public access.

### Existing Custom Policies

| Policy | Scope | Used by |
|--------|-------|---------|
| terraform-state | `terraform-state-*` buckets (CRUD) | terraform |
| backups-vixens | `backups-vixens*` buckets (CRUD) | litestream |
| velero-access | `vixens-prod-velero` bucket (multipart + CRUD) | velero-prod |
| birdnet-go-policy | `vixens-prod-birdnet-go` bucket (full) | birdnet-go-access |
| birdnet-go-litestream | — | — |
| firefly-iii-policy | — | — |
| vixens-prod-openclaw | `vixens-prod-openclaw` bucket (CRUD) | — |
| vixens-openclaw-access | — | openclaw-user (via group) |

---

## Service Accounts

```bash
# Create service account for a user (returns access key + secret key)
mc admin user svcacct add minio parent-username
# Output: Access Key: XXXX, Secret Key: YYYY

# Create with custom policy
mc admin user svcacct add minio parent-username --policy /path/to/policy.json

# Create with expiry
mc admin user svcacct add minio parent-username --expiry 2026-12-31T00:00:00Z

# Info
mc admin user svcacct info minio ACCESS_KEY

# Disable / Enable
mc admin user svcacct disable minio ACCESS_KEY
mc admin user svcacct enable minio ACCESS_KEY

# Delete
mc admin user svcacct rm minio ACCESS_KEY
```

⚠️ `mc admin user svcacct list` is broken with modern mc (JSON parse error). Use `mc-compat admin user svcacct list minio username` or `mc admin user svcacct info minio ACCESS_KEY` per key.

---

## Server Administration

### Configuration

```bash
mc admin config get minio                              # All config
mc admin config get minio region                       # Specific subsystem
mc admin config set minio region name=us-east-1        # Set config
mc admin config reset minio region                     # Reset to default
```

Available subsystems: `api`, `cache`, `compression`, `etcd`, `identity_ldap`, `identity_openid`, `notify_webhook`, `notify_redis`, `notify_kafka`, `notify_mqtt`, `notify_postgres`, `scanner`, `site`, `subnet`

### Monitoring

```bash
mc admin info minio                                    # Server info + capacity
mc admin trace minio                                   # Live API trace (Ctrl+C to stop)
mc admin trace minio --call s3.PutObject               # Filter by API call
mc admin prometheus generate minio                     # Prometheus scrape config
```

### Service

```bash
mc admin service restart minio                         # Restart server
mc admin service stop minio                            # Stop server
```

---

## Naming Conventions (Homelab)

Existing bucket naming patterns:

| Pattern | Example | Purpose |
|---------|---------|---------|
| `vixens-prod-<app>` | `vixens-prod-frigate` | K8s app persistent storage |
| `backups-vixens-<env>` | `backups-vixens-prod` | Backup storage (versioned) |
| `terraform-state-<env>` | `terraform-state-prod` | Terraform state (versioned) |

User naming: `<app-name>` or `<app-name>-access` (e.g., `velero-prod`, `birdnet-go-access`)
Policy naming: `<app-name>-policy` or descriptive (e.g., `velero-access`, `terraform-state`)

## Standard Onboarding Pattern (New App)

When adding a new app to the cluster that needs MinIO storage:

```bash
# 1. Create bucket
mc mb minio/vixens-prod-<app>
mc version enable minio/vixens-prod-<app>

# 2. Create policy JSON
cat > /tmp/<app>-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {"Effect": "Allow", "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
     "Resource": ["arn:aws:s3:::vixens-prod-<app>"]},
    {"Effect": "Allow", "Action": ["s3:*Object", "s3:ListMultipartUploadParts", "s3:AbortMultipartUpload"],
     "Resource": ["arn:aws:s3:::vixens-prod-<app>/*"]}
  ]
}
EOF
mc admin policy create minio <app>-policy /tmp/<app>-policy.json

# 3b. Attach policy to user (requires mc-compat)
mc-compat admin policy set minio <app>-policy user=<app>-access

# 3. Create user
mc admin user add minio <app>-access 'GENERATED_PASSWORD'

# 4. Create service account with scoped policy
mc admin user svcacct add minio <app>-access --policy /tmp/<app>-policy.json

# 5. Set lifecycle (optional)
mc ilm add minio/vixens-prod-<app> --noncurrent-expire-days 30
```
