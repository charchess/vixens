# Setup Python Security Tools

Composite GitHub Action to install and cache Python-based security scanning tools.

## Tools Installed

- **checkov** - Infrastructure as Code (IaC) security scanner
  - Scans: Kubernetes, Terraform, CloudFormation, Docker, etc.
  - Checks: 1000+ security best practices
  - Output: SARIF, JSON, JUnit, etc.

## Features

- ✅ Automatic pip caching (faster subsequent runs)
- ✅ Configurable versions
- ✅ Python version selection
- ✅ Verification of installations

## Usage

### Basic Usage

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup Python security tools
    uses: ./.github/actions/setup-python-security
```

### Custom Versions

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup Python security tools
    uses: ./.github/actions/setup-python-security
    with:
      checkov-version: '3.2.0'
      python-version: '3.11'
```

### Selective Installation

```yaml
steps:
  - uses: actions/checkout@v4

  # Only checkov (default)
  - name: Setup security tools
    uses: ./.github/actions/setup-python-security
    with:
      install-checkov: 'true'
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `install-checkov` | Install checkov | No | `true` |
| `checkov-version` | Checkov version (e.g., 3.2.0) | No | `3.2.0` |
| `python-version` | Python version to use | No | `3.11` |

## Outputs

| Output | Description |
|--------|-------------|
| `cache-hit` | Whether pip cache was hit (`true`/`false`) |

## Cache Behavior

- **Cache key**: Based on OS, tool versions, and Python version
- **Cache path**: `~/.cache/pip`
- **Cache duration**: GitHub default (7 days for unused caches)

**First run**: ~60-90s (download and install)  
**Cached run**: ~10-20s (restore from cache)

## Requirements

- Ubuntu runner (`runs-on: ubuntu-latest`)
- Checkout action must run first

## Example Workflows

### Infrastructure Security Scan

```yaml
name: Security Scan

on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup security tools
        uses: ./.github/actions/setup-python-security

      - name: Scan Kubernetes manifests
        run: |
          checkov --directory apps/ \
            --framework kubernetes \
            --output cli \
            --soft-fail
```

### Full Security Pipeline

```yaml
name: Full Security

on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup security tools
        uses: ./.github/actions/setup-python-security

      - name: Scan infrastructure
        run: |
          checkov --directory . \
            --framework kubernetes \
            --compact --quiet \
            --output junitxml \
            --output-file-path checkov-results.xml

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: security-scan-results
          path: checkov-results.xml
```

## Checkov Usage Examples

### Scan Kubernetes manifests

```bash
checkov --directory apps/ --framework kubernetes
```

### Scan with custom policies

```bash
checkov --directory apps/ \
  --framework kubernetes \
  --external-checks-dir ./custom-policies/
```

### Output formats

```bash
# CLI output (default)
checkov --directory apps/ --output cli

# JSON output
checkov --directory apps/ --output json

# SARIF (for GitHub Code Scanning)
checkov --directory apps/ --output sarif

# JUnit XML (for CI/CD)
checkov --directory apps/ --output junitxml
```

### Suppress specific checks

```bash
# Skip specific checks
checkov --directory apps/ --skip-check CKV_K8S_8,CKV_K8S_9

# Use baseline file
checkov --directory apps/ --baseline checkov-baseline.json
```

## Maintenance

### Updating Checkov Version

1. Check latest release: https://github.com/bridgecrewio/checkov/releases
2. Update `checkov-version` default in `action.yaml`
3. Update this README
4. Test in a branch before merging

### Updating Python Version

When updating Python version:
- Ensure checkov supports the new Python version
- Update `python-version` in `action.yaml`
- Test compatibility

## Troubleshooting

### Checkov installation fails

- Check version exists: https://pypi.org/project/checkov/
- Verify Python version compatibility
- Review pip error logs in runner output

### Cache not working

- Check cache key matches (OS + versions)
- Verify pip cache directory exists
- Check cache size < 10GB (GitHub limit)

### Slow installation

- Checkov has many dependencies (~100MB)
- First run is always slower
- Subsequent runs use cache (10-20s)

## Performance Tips

1. **Use caching**: Default behavior, no action needed
2. **Pin versions**: Prevents unexpected updates
3. **Minimize scans**: Only scan changed directories in PRs
4. **Use compact output**: `--compact --quiet` for cleaner logs

## Notes

- **Checkov versions**: Use exact versions for reproducibility
- **Python versions**: Python 3.8+ supported
- **Cache benefits**: ~70% faster on cache hit
- **Default behavior**: Installs checkov only (add more tools as needed)
