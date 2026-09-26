#!/usr/bin/env python3
"""Generate a lightweight Vixens management report.

Data sources:
- current Kubernetes context/resources via kubectl;
- repository task state via GitHub Issues (`gh issue list`).

Git/ArgoCD remain the desired-state source of truth; this report is observational.
"""

import argparse
import json
import subprocess
from datetime import datetime


def run_command(cmd, description):
    """Run a command and return stdout, or None on failure."""
    print(f"   → {description}...")
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"      ⚠️  Error: {result.stderr.strip()}")
        return None
    return result.stdout


def get_cluster_resources():
    return run_command(
        ["kubectl", "get", "all", "-A", "--no-headers"],
        "Fetching cluster resources",
    )


def get_github_issues():
    output = run_command(
        [
            "gh",
            "issue",
            "list",
            "--state",
            "all",
            "--limit",
            "1000",
            "--json",
            "number,title,state,assignees,labels,url",
        ],
        "Fetching GitHub Issues",
    )
    if output is None:
        return None
    try:
        return json.loads(output)
    except json.JSONDecodeError as exc:
        print(f"      ⚠️  Invalid JSON from GitHub CLI: {exc}")
        return None


def count_resources(resources_output):
    counts = {}
    if not resources_output:
        return counts
    for line in resources_output.splitlines():
        parts = line.split()
        if len(parts) < 2:
            continue
        resource_type = parts[1].split("/")[0]
        counts[resource_type] = counts.get(resource_type, 0) + 1
    return counts


def count_issues_by_state(issues):
    counts = {}
    for issue in issues or []:
        state = issue.get("state", "UNKNOWN").lower()
        counts[state] = counts.get(state, 0) + 1
    return counts


def issue_assignees(issue):
    names = [item.get("login") for item in issue.get("assignees", []) if item.get("login")]
    return ", ".join(names) if names else "unassigned"


def current_context():
    output = run_command(["kubectl", "config", "current-context"], "Reading Kubernetes context")
    return output.strip() if output else "unknown"


def generate_report(output_file):
    print("📊 Génération du rapport chefferie...\n")

    resources = get_cluster_resources()
    issues = get_github_issues()
    resource_counts = count_resources(resources)
    issue_counts = count_issues_by_state(issues)

    content = "# 📊 Rapport Chefferie de Projet\n\n"
    content += f"**Généré:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
    content += f"**Contexte Kubernetes:** {current_context()}\n"
    content += "**Task tracker:** GitHub Issues\n\n"
    content += "> Rapport observationnel : Git et ArgoCD restent les sources de vérité du desired state.\n\n"

    content += "## 🖥️ Vue d'ensemble Cluster\n\n"
    if resource_counts:
        content += "| Type | Count |\n|---|---:|\n"
        for resource_type, count in sorted(resource_counts.items(), key=lambda item: item[1], reverse=True):
            content += f"| {resource_type} | {count} |\n"
        content += "\n"

    content += "## 📋 GitHub Issues\n\n"
    if issue_counts:
        content += "| État | Count |\n|---|---:|\n"
        for state in ("open", "closed"):
            if state in issue_counts:
                content += f"| {state} | {issue_counts[state]} |\n"
        content += "\n"

    open_issues = [issue for issue in (issues or []) if issue.get("state") == "OPEN"]
    if open_issues:
        content += "### Issues ouvertes\n\n"
        content += "| Issue | Titre | Assignee |\n|---|---|---|\n"
        for issue in open_issues[:30]:
            title = str(issue.get("title", "")).replace("|", "\\|")
            content += f"| #{issue.get('number')} | {title} | {issue_assignees(issue)} |\n"
        if len(open_issues) > 30:
            content += f"\n*… et {len(open_issues) - 30} issues ouvertes supplémentaires.*\n"
        content += "\n"

    content += "## 🔍 Ressources Kubernetes (échantillon)\n\n"
    if resources:
        lines = resources.splitlines()
        content += "```text\n"
        content += "\n".join(lines[:20])
        if len(lines) > 20:
            content += f"\n… et {len(lines) - 20} ressources supplémentaires"
        content += "\n```\n\n"

    content += "## 📎 GitHub Issues (JSON)\n\n```json\n"
    content += json.dumps(issues or [], indent=2, ensure_ascii=False)
    content += "\n```\n"

    with open(output_file, "w", encoding="utf-8") as handle:
        handle.write(content)

    print(f"✅ Rapport chefferie généré: {output_file}")


def main():
    parser = argparse.ArgumentParser(description="Generate Vixens management report")
    parser.add_argument("--output", default="docs/reports/MANAGEMENT-REPORT.md", help="Output file")
    args = parser.parse_args()
    generate_report(args.output)


if __name__ == "__main__":
    main()
