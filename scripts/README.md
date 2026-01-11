# Scripts Automation

This directory contains various scripts for cluster management, validation, and report generation.

## Report Generation

These scripts automate the generation of infrastructure reports in `docs/reports/`.

### 1. Actual State Generation
Queries the live cluster (Pods, VPAs, ArgoCD) to generate the production reality report.

```bash
python3 scripts/generate-actual-state.py --env dev
```

### 2. Conformity Checking
Compares the `STATE-ACTUAL.md` against `STATE-DESIRED.md` and generates a conformity score per application.

```bash
python3 scripts/conformity-checker.py
```

### 3. Status Dashboard Generation
Generates the high-level `STATUS.md` dashboard based on conformity results.

```bash
python3 scripts/generate-status-report.py
```

## Shared Utilities

`scripts/lib/report_utils.py` provides shared functions for:
- CPU/Memory parsing and formatting
- Kubernetes API interaction (via kubectl JSON)
- Markdown table parsing and generation

## Workflow

To update all reports at once, use:

```bash
just reports
```
