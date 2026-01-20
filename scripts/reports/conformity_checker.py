#!/usr/bin/env python3
import argparse
import sys
import os
import json

# Add lib to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'lib'))

from report_utils import parse_markdown_table, save_markdown_table

def compare_values(actual, desired, name, app):
    if actual == desired:
        return True, ""
    return False, f"{name} mismatch: {actual} vs {desired}"

def main():
    parser = argparse.ArgumentParser(description="Check conformity between ACTUAL and DESIRED state")
    parser.add_argument("--actual", default="docs/reports/STATE-ACTUAL.md", help="Actual state file")
    parser.add_argument("--desired", default="docs/reports/STATE-DESIRED.md", help="Desired state file")
    parser.add_argument("--output", default="docs/reports/CONFORMITY-REPORT.md", help="Output report file")
    args = parser.parse_args()

    actual_rows = parse_markdown_table(args.actual)
    desired_rows = parse_markdown_table(args.desired)

    if not desired_rows:
        print("Error: Could not parse desired state file")
        sys.exit(1)
    
    if not actual_rows:
        print("Warning: Actual state file is empty or could not be parsed")
        actual_rows = []

    # Index by App name (strip markdown bold)
    actual_map = {row['App'].replace('**', ''): row for row in actual_rows if 'App' in row}
    desired_map = {row['App'].replace('**', ''): row for row in desired_rows if 'App' in row}

    results = []
    summary = {"total": 0, "compliant": 0, "partial": 0, "non_compliant": 0}

    # Sort apps by name for deterministic report
    for app_name in sorted(desired_map.keys()):
        summary["total"] += 1
        desired = desired_map[app_name]
        actual = actual_map.get(app_name)
        
        if not actual:
            results.append({
                "App": f"**{app_name}**",
                "Status": "🔴 ABSENT",
                "Score": "0/100",
                "Issues": "Application not found in actual state"
            })
            summary["non_compliant"] += 1
            continue

        issues = []
        # Compare key fields
        checks = [
            ("CPU Req", "CPU Req"),
            ("CPU Lim", "CPU Lim"),
            ("Mem Req", "Mem Req"),
            ("Mem Lim", "Mem Lim"),
            ("Priority", "Priority"),
            ("Sync Wave", "Sync Wave"),
            ("Backup Profile", "Backup Profile")
        ]

        score = 100
        for actual_field, desired_field in checks:
            act_val = actual.get(actual_field, "N/A")
            des_val = desired.get(desired_field, "N/A")
            ok, msg = compare_values(act_val, des_val, desired_field, app_name)
            if not ok:
                issues.append(msg)
                score -= 10 # Rough scoring logic

        if score == 100:
            status = "✅ OK"
            summary["compliant"] += 1
        elif score > 70:
            status = "⚠️ PARTIAL"
            summary["partial"] += 1
        else:
            status = "❌ NOK"
            summary["non_compliant"] += 1

        results.append({
            "App": f"**{app_name}**",
            "Status": status,
            "Score": f"{max(0, score)}/100",
            "Issues": "; ".join(issues) if issues else "Full compliance"
        })

    # Generate Report
    content = "# Conformity Report\n\n"
    content += f"**Total Apps:** {summary['total']}\n"
    content += f"- ✅ Compliant: {summary['compliant']}\n"
    content += f"- ⚠️ Partial: {summary['partial']}\n"
    content += f"- ❌ Non-compliant: {summary['non_compliant']}\n\n"
    
    headers = ["App", "Status", "Score", "Issues"]
    content += "## Conformity Details\n\n"
    content += save_markdown_table(headers, results)
    content += "\n"

    os.makedirs(os.path.dirname(args.output), exist_ok=True)
    with open(args.output, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Conformity report generated: {args.output}")

if __name__ == "__main__":
    main()
