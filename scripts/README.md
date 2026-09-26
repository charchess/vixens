# Scripts Automation

This directory contains optional helper scripts for cluster management, validation, and report generation.

The repository workflow does **not** depend on these helpers. Canonical task tracking and change orchestration live in GitHub Issues, Pull Requests, GitHub Actions, and ArgoCD; see `WORKFLOW.md` and `AGENTS.md`.

## 📂 Directory Structure

| Directory | Purpose |
| :--- | :--- |
| `scripts/analysis/` | Resource analysis and auditing tools |
| `scripts/infra/` | Infrastructure automation and ArgoCD management |
| `scripts/lib/` | Shared libraries and utilities |
| `scripts/reports/` | Report generation for documentation |
| `scripts/testing/` | Test suites and test runners |
| `scripts/utils/` | General purpose CLI utilities |
| `scripts/validation/` | Validation and compliance scripts |

## 📊 Report Generation (`scripts/reports/`)

These scripts automate the generation of infrastructure reports in `docs/reports/`.

- **`generate_actual_state.py`**: Queries the live cluster to generate `STATE-ACTUAL.md`.
- **`conformity_checker.py`**: Compares `STATE-ACTUAL.md` vs `STATE-DESIRED.md`.
- **`generate_status_report.py`**: Generates the `STATUS.md` dashboard.
- **`compare_resources_full.py`**: Detailed resource comparison report.

## 🔍 Analysis & Audit (`scripts/analysis/`)

- **`ultimate_audit.py`**: Comprehensive resource and VPA audit.
- **`audit_resources_and_priorities.py`**: Audit of priorities and resource requests.
- **`analyze_litestream_metrics.py`**: Analysis of backup metrics.
- **`hibernate.py`**: Cluster resource hibernation utility.

## 🛠️ Infrastructure (`scripts/infra/`)

- **`argo-init.sh`**: ArgoCD initialization and bootstrapping.
- **`destroy-namespace.sh`**: Safe namespace deletion.
- **`bootstrap-secrets.sh`**: Secret management bootstrapping.
- **`sync-waves-batch-update.sh`**: Batch update of ArgoCD sync waves.

## ✅ Validation (`scripts/validation/`)

- **`validate.py`**: Core application validation script.
- **`validate-yaml.sh`**: YAML linting and validation.
- **`validate-sync-waves.sh`**: ArgoCD sync waves validation.
- **`apply_resource_compliance.py`**: Automated resource compliance patching.

Important repository checks belong in GitHub Actions. Local scripts are convenience tools and must not become an alternate source of workflow truth.

## 🧰 Utilities (`scripts/utils/`)

- **`check`**: Quick status check utility.
- **`gp`**: Git push helper.
- **`k`**: Kubectl alias with context.
- **`test-deployment-time.sh`**: Performance testing for deployments.

## 🧪 Testing (`scripts/testing/`)

- **`run_tests.py`**: Test runner for the script suite.
- **`requirements-tests.txt`**: Dependencies for testing.

## 🚀 Workflow

Use the repository-native commands or the relevant helper script directly. Do not require `just`, Serena, Archon, or another local orchestration layer for normal contributions.

Before merge, rely on the PR CI checks as the authoritative validation gate. Persistent cluster changes must still go through Git and ArgoCD.
