# Secrets Management

⚠️ **TEMPORARY SOLUTION** - Secrets are currently committed in Git for simplicity.

This will be revisited in a future sprint with proper encryption (Minio + age, Sealed Secrets, or SOPS).

## Structure

```
.secrets/
├── dev/
│   └── gandi-credentials.yaml
├── test/
│   └── gandi-credentials.yaml
├── staging/
└── prod/
```

## Usage

After cluster deployment:

```bash
./scripts/bootstrap-secrets.sh dev
# or
kubectl apply -f .secrets/dev/
```

## Future Improvements

- [ ] Encrypt secrets (Minio + age)
- [ ] Or use Sealed Secrets
- [ ] Or use SOPS
- [ ] Remove from Git
