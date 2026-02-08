#!/usr/bin/env python3
"""
Generate comprehensive lint report for Vixens infrastructure.

This script:
1. Runs yamllint on all YAML files
2. Detects DRY violations (duplicated configurations)
3. Checks conformity to ADR standards
4. Generates LINT-REPORT.md with scores and trends
"""

import argparse
import subprocess
import sys
import os
import yaml
import json
from datetime import datetime
from pathlib import Path
from collections import defaultdict
import hashlib

# Add lib to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'lib'))
from report_utils import save_markdown_table


def run_yamllint(paths):
    """Run yamllint on specified paths and return results."""
    print("🔍 Running yamllint...")

    all_files = []
    for path in paths:
        if os.path.isdir(path):
            all_files.extend(Path(path).rglob("*.yaml"))
            all_files.extend(Path(path).rglob("*.yml"))
        else:
            all_files.append(Path(path))

    results = {
        "total_files": len(all_files),
        "errors": [],
        "warnings": [],
        "passed_files": [],
        "failed_files": []
    }

    for file in all_files:
        try:
            result = subprocess.run(
                ["yamllint", "-c", "yamllint-config.yml", "-f", "parsable", str(file)],
                capture_output=True,
                text=True
            )

            if result.returncode == 0:
                results["passed_files"].append(str(file))
            else:
                results["failed_files"].append(str(file))

                # Parse yamllint output
                for line in result.stdout.strip().split("\n"):
                    if not line:
                        continue

                    parts = line.split(":")
                    if len(parts) >= 4:
                        filepath = parts[0]
                        line_num = parts[1]
                        col_num = parts[2]
                        msg = ":".join(parts[3:]).strip()

                        issue = {
                            "file": filepath,
                            "line": line_num,
                            "column": col_num,
                            "message": msg
                        }

                        if "[error]" in msg:
                            results["errors"].append(issue)
                        else:
                            results["warnings"].append(issue)

        except Exception as e:
            print(f"   ⚠️  Error checking {file}: {e}")

    print(f"   ✅ Checked {results['total_files']} files")
    print(f"   ✅ Passed: {len(results['passed_files'])}")
    print(f"   ❌ Failed: {len(results['failed_files'])}")
    print(f"   ⚠️  Warnings: {len(results['warnings'])}")
    print(f"   ❌ Errors: {len(results['errors'])}")

    return results


def detect_dry_violations(paths):
    """Detect duplicated YAML configurations (DRY violations)."""
    print("\n🔍 Detecting DRY violations...")

    configs = defaultdict(list)

    for path in paths:
        yaml_files = []
        if os.path.isdir(path):
            yaml_files.extend(Path(path).rglob("*.yaml"))
            yaml_files.extend(Path(path).rglob("*.yml"))

        for file in yaml_files:
            try:
                with open(file, 'r') as f:
                    content = f.read()

                # Hash the content to detect duplicates
                content_hash = hashlib.md5(content.encode()).hexdigest()
                configs[content_hash].append(str(file))

            except Exception as e:
                print(f"   ⚠️  Error reading {file}: {e}")

    # Find duplicates (same hash, multiple files)
    duplicates = {k: v for k, v in configs.items() if len(v) > 1}

    print(f"   ✅ Scanned {len(configs)} unique configurations")
    print(f"   ❌ Found {len(duplicates)} duplicate groups")

    return duplicates


def check_resource_standards(paths):
    """Check conformity to resource standards (ADR-008)."""
    print("\n🔍 Checking resource standards...")

    violations = []

    for path in paths:
        if not os.path.isdir(path):
            continue

        # Look for Deployment/StatefulSet manifests
        yaml_files = list(Path(path).rglob("*.yaml")) + list(Path(path).rglob("*.yml"))

        for file in yaml_files:
            try:
                with open(file, 'r') as f:
                    docs = list(yaml.safe_load_all(f))

                for doc in docs:
                    if not doc or not isinstance(doc, dict):
                        continue

                    kind = doc.get('kind')
                    if kind not in ['Deployment', 'StatefulSet', 'DaemonSet']:
                        continue

                    name = doc.get('metadata', {}).get('name', 'unknown')

                    # Check containers for resource limits
                    containers = doc.get('spec', {}).get('template', {}).get('spec', {}).get('containers', [])

                    for container in containers:
                        resources = container.get('resources', {})

                        if not resources:
                            violations.append({
                                'file': str(file),
                                'resource': f"{kind}/{name}",
                                'container': container.get('name', 'unknown'),
                                'issue': 'Missing resources section'
                            })
                            continue

                        # Check for requests
                        if not resources.get('requests'):
                            violations.append({
                                'file': str(file),
                                'resource': f"{kind}/{name}",
                                'container': container.get('name', 'unknown'),
                                'issue': 'Missing resource requests'
                            })

                        # Check for limits
                        if not resources.get('limits'):
                            violations.append({
                                'file': str(file),
                                'resource': f"{kind}/{name}",
                                'container': container.get('name', 'unknown'),
                                'issue': 'Missing resource limits'
                            })

            except Exception as e:
                continue

    print(f"   ❌ Found {len(violations)} resource violations")

    return violations


def calculate_quality_score(yamllint_results, dry_violations, resource_violations):
    """Calculate overall quality score (0-100)."""

    # Base score
    score = 100

    # Deduct for yamllint errors (5 points each, max 30)
    error_penalty = min(30, len(yamllint_results['errors']) * 5)
    score -= error_penalty

    # Deduct for yamllint warnings (2 points each, max 20)
    warning_penalty = min(20, len(yamllint_results['warnings']) * 2)
    score -= warning_penalty

    # Deduct for DRY violations (10 points each, max 30)
    dry_penalty = min(30, len(dry_violations) * 10)
    score -= dry_penalty

    # Deduct for resource violations (5 points each, max 20)
    resource_penalty = min(20, len(resource_violations) * 5)
    score -= resource_penalty

    return max(0, score)


def generate_report(yamllint_results, dry_violations, resource_violations, output_file):
    """Generate markdown report."""

    score = calculate_quality_score(yamllint_results, dry_violations, resource_violations)

    content = "# Lint & Quality Report\n\n"
    content += f"**Generated:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
    content += f"**Quality Score:** {score}/100\n\n"

    # Score indicator
    if score >= 90:
        indicator = "🟢 Excellent"
    elif score >= 70:
        indicator = "🟡 Good"
    elif score >= 50:
        indicator = "🟠 Fair"
    else:
        indicator = "🔴 Needs Improvement"

    content += f"**Status:** {indicator}\n\n"
    content += "---\n\n"

    # Summary
    content += "## Summary\n\n"
    summary_rows = [
        {"Category": "Total YAML Files", "Count": str(yamllint_results['total_files']), "Status": "ℹ️"},
        {"Category": "Files Passed", "Count": str(len(yamllint_results['passed_files'])), "Status": "✅"},
        {"Category": "Files Failed", "Count": str(len(yamllint_results['failed_files'])), "Status": "❌" if yamllint_results['failed_files'] else "✅"},
        {"Category": "Yamllint Errors", "Count": str(len(yamllint_results['errors'])), "Status": "❌" if yamllint_results['errors'] else "✅"},
        {"Category": "Yamllint Warnings", "Count": str(len(yamllint_results['warnings'])), "Status": "⚠️" if yamllint_results['warnings'] else "✅"},
        {"Category": "DRY Violations", "Count": str(len(dry_violations)), "Status": "❌" if dry_violations else "✅"},
        {"Category": "Resource Violations", "Count": str(len(resource_violations)), "Status": "❌" if resource_violations else "✅"},
    ]
    content += save_markdown_table(["Category", "Count", "Status"], summary_rows)
    content += "\n\n---\n\n"

    # Yamllint Errors
    if yamllint_results['errors']:
        content += "## Yamllint Errors\n\n"
        error_rows = []
        for error in yamllint_results['errors'][:50]:  # Limit to first 50
            error_rows.append({
                "File": error['file'],
                "Line": error['line'],
                "Message": error['message'].replace('[error] ', '')
            })
        content += save_markdown_table(["File", "Line", "Message"], error_rows)

        if len(yamllint_results['errors']) > 50:
            content += f"\n*... and {len(yamllint_results['errors']) - 50} more errors*\n"
        content += "\n\n---\n\n"

    # Yamllint Warnings
    if yamllint_results['warnings']:
        content += "## Yamllint Warnings\n\n"
        warning_rows = []
        for warning in yamllint_results['warnings'][:50]:  # Limit to first 50
            warning_rows.append({
                "File": warning['file'],
                "Line": warning['line'],
                "Message": warning['message'].replace('[warning] ', '')
            })
        content += save_markdown_table(["File", "Line", "Message"], warning_rows)

        if len(yamllint_results['warnings']) > 50:
            content += f"\n*... and {len(yamllint_results['warnings']) - 50} more warnings*\n"
        content += "\n\n---\n\n"

    # DRY Violations
    if dry_violations:
        content += "## DRY Violations (Duplicated Configs)\n\n"
        content += "*Files with identical content should be consolidated using shared resources.*\n\n"

        for idx, (hash_val, files) in enumerate(list(dry_violations.items())[:20]):
            content += f"### Duplicate Group {idx + 1} ({len(files)} files)\n\n"
            for file in files:
                content += f"- `{file}`\n"
            content += "\n"

        if len(dry_violations) > 20:
            content += f"*... and {len(dry_violations) - 20} more duplicate groups*\n\n"

        content += "---\n\n"

    # Resource Violations
    if resource_violations:
        content += "## Resource Standard Violations (ADR-008)\n\n"
        violation_rows = []
        for violation in resource_violations[:50]:
            violation_rows.append({
                "Resource": violation['resource'],
                "Container": violation['container'],
                "Issue": violation['issue'],
                "File": violation['file']
            })
        content += save_markdown_table(["Resource", "Container", "Issue", "File"], violation_rows)

        if len(resource_violations) > 50:
            content += f"\n*... and {len(resource_violations) - 50} more violations*\n"
        content += "\n\n---\n\n"

    # Recommendations
    content += "## Recommendations\n\n"

    if yamllint_results['errors']:
        content += "### 🔴 Critical: Fix Yamllint Errors\n"
        content += f"- {len(yamllint_results['errors'])} yamllint errors must be fixed\n"
        content += "- Run: `just lint` to see all errors\n\n"

    if dry_violations:
        content += "### 🟡 High Priority: Consolidate Duplicates\n"
        content += f"- {len(dry_violations)} duplicate configuration groups found\n"
        content += "- Move shared configs to `apps/_shared/`\n"
        content += "- Use Kustomize bases/components for reuse\n\n"

    if resource_violations:
        content += "### 🟠 Medium Priority: Add Resource Limits\n"
        content += f"- {len(resource_violations)} containers missing resource specifications\n"
        content += "- Follow ADR-008: All containers must have requests + limits\n"
        content += "- Use VPA recommendations from Goldilocks\n\n"

    if score >= 90:
        content += "### ✅ No Critical Issues\n"
        content += "Infrastructure configuration is in excellent shape!\n\n"

    # Write report
    os.makedirs(os.path.dirname(output_file), exist_ok=True)
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"\n✅ Lint report generated: {output_file}")
    print(f"📊 Quality Score: {score}/100 {indicator}")

    return score


def main():
    parser = argparse.ArgumentParser(description="Generate comprehensive lint report")
    parser.add_argument("--paths", nargs='+', default=["apps", "argocd"], help="Paths to scan")
    parser.add_argument("--output", default="docs/reports/LINT-REPORT.md", help="Output file")
    parser.add_argument("--fail-threshold", type=int, default=50, help="Fail if score below threshold")
    args = parser.parse_args()

    print("🧹 Generating Lint Report...\n")

    # Run checks
    yamllint_results = run_yamllint(args.paths)
    dry_violations = detect_dry_violations(args.paths)
    resource_violations = check_resource_standards(args.paths)

    # Generate report
    score = generate_report(yamllint_results, dry_violations, resource_violations, args.output)

    # Exit with error if below threshold
    if score < args.fail_threshold:
        print(f"\n❌ Quality score {score} is below threshold {args.fail_threshold}")
        sys.exit(1)

    sys.exit(0)


if __name__ == "__main__":
    main()
