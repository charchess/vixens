# Setup Kubernetes Tools

Composite GitHub Action to install and cache Kubernetes validation tools.

## Tools Installed

- **Kustomize** - Kubernetes configuration management
- **Helm** - Kubernetes package manager
- **Kubeconform** - Kubernetes manifest validation (replaces deprecated kubeval)

## Features

- ✅ Automatic caching (30-50% faster on subsequent runs)
- ✅ Configurable versions
- ✅ Verification of installations

## Usage

### Basic Usage

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup K8s tools
    uses: ./.github/actions/setup-k8s-tools
```

### Custom Versions

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup K8s tools
    uses: ./.github/actions/setup-k8s-tools
    with:
      kustomize-version: '5.5.0'
      helm-version: '3.16.0'
      kubeconform-version: '0.6.7'
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `kustomize-version` | Kustomize version to install | No | `5.5.0` |
| `helm-version` | Helm version to install | No | `3.16.0` |
| `kubeconform-version` | Kubeconform version to install | No | `0.6.7` |

## Outputs

| Output | Description |
|--------|-------------|
| `cache-hit` | Whether the cache was hit (`true`/`false`) |

## Cache Behavior

- **Cache key**: Based on OS and tool versions
- **Cache paths**: `/usr/local/bin/{kustomize,helm,kubeconform}`
- **Cache duration**: GitHub default (7 days for unused caches)

**First run**: ~30-60s (download and install)  
**Cached run**: ~5-10s (restore from cache)

## Requirements

- Ubuntu runner (`runs-on: ubuntu-latest`)
- Checkout action must run first

## Example Workflow

```yaml
name: Validate Kubernetes Manifests

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup K8s tools
        uses: ./.github/actions/setup-k8s-tools

      - name: Build with Kustomize
        run: kustomize build overlays/dev

      - name: Validate with Kubeconform
        run: |
          kustomize build overlays/dev | \
          kubeconform -strict -kubernetes-version 1.30.0
```

## Maintenance

To update tool versions:

1. Update default versions in `action.yaml`
2. Update this README with new versions
3. Test in a branch before merging to main

## Troubleshooting

### Cache not working

If tools are being downloaded every run:
- Check cache key matches (OS + versions)
- Verify cache size < 10GB (GitHub limit)
- Check cache hasn't expired (7 days for unused)

### Installation failures

- Check network connectivity to GitHub
- Verify tool versions exist (check releases)
- Review runner logs for specific errors
