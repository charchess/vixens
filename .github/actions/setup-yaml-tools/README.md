# Setup YAML Tools

Composite GitHub Action to install YAML validation and processing tools.

## Tools Installed

- **yamllint** - YAML syntax and style validator
- **yq** - YAML processor (jq for YAML)

## Features

- ✅ Selective installation (install only what you need)
- ✅ Configurable yq version
- ✅ Fast installation (yamllint via apt, yq via dedicated action)

## Usage

### Install Both Tools

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup YAML tools
    uses: ./.github/actions/setup-yaml-tools
```

### Install Only yamllint

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup YAML tools
    uses: ./.github/actions/setup-yaml-tools
    with:
      install-yq: 'false'
```

### Install Only yq

```yaml
steps:
  - uses: actions/checkout@v4

  - name: Setup YAML tools
    uses: ./.github/actions/setup-yaml-tools
    with:
      install-yamllint: 'false'
```


## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `install-yamllint` | Install yamllint | No | `true` |
| `install-yq` | Install yq | No | `true` |

## Outputs

None.

## Installation Time

- **yamllint**: ~10-20s (via apt-get)
- **yq**: ~5-10s (via mikefarah/yq action)
- **Total**: ~15-30s

## Requirements

- Ubuntu runner (`runs-on: ubuntu-latest`)
- Checkout action must run first

## Example Workflows

### YAML Linting

```yaml
name: Lint YAML Files

on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup YAML tools
        uses: ./.github/actions/setup-yaml-tools
        with:
          install-yq: 'false'  # Don't need yq for linting

      - name: Run yamllint
        run: yamllint -c yamllint-config.yml .
```

### YAML Processing

```yaml
name: Process YAML

on: [push]

jobs:
  process:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup YAML tools
        uses: ./.github/actions/setup-yaml-tools
        with:
          install-yamllint: 'false'  # Don't need linting

      - name: Extract value from YAML
        run: |
          VERSION=$(yq eval '.version' config.yaml)
          echo "Version: $VERSION"
```

### Combined Validation and Processing

```yaml
name: Validate and Process YAML

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup YAML tools
        uses: ./.github/actions/setup-yaml-tools

      - name: Lint YAML
        run: yamllint -c yamllint-config.yml apps/

      - name: Validate structure
        run: |
          yq eval '.kind' apps/*/base/*.yaml | \
          grep -qE '^(Deployment|StatefulSet|DaemonSet)$'
```

## Maintenance

### Updating yq Version

1. Check latest release: https://github.com/mikefarah/yq/releases
2. Update `yq-version` default in `action.yaml`
3. Update this README

### Updating yamllint

yamllint is installed via `apt-get` and uses the version available in Ubuntu's repository. To use a specific version, consider installing via pip instead:

```yaml
- name: Install yamllint
  run: pip3 install yamllint==1.35.1
```

## Troubleshooting

### yamllint not found

- Verify Ubuntu runner (`runs-on: ubuntu-latest`)
- Check apt-get update succeeded
- Review runner logs for installation errors

### yq version mismatch

- Ensure `yq-version` matches format `vX.Y.Z` (e.g., `v4.44.1`)
- Check version exists: https://github.com/mikefarah/yq/releases
- Verify mikefarah/yq action is accessible

## Notes

- **yamllint**: System package, no caching needed
- **yq**: Installed via dedicated action, handles caching internally
- Both tools are small and install quickly
