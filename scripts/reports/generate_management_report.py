#!/usr/bin/env python3
"""
MANAGEMENT REPORT GENERATOR
Rapport consolidé pour la chefferie de projet.

Contient:
- Vue d'ensemble cluster (kubectl get all -A)
- État des tâches Beads (tous statuts)
- Métriques clés (nodes, pods, services, etc.)
"""

import subprocess
import json
import sys
from datetime import datetime


def run_command(cmd, description):
    """Run a command and return output."""
    print(f"   → {description}...")
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        shell=isinstance(cmd, str)
    )

    if result.returncode != 0:
        print(f"      ⚠️  Error: {result.stderr}")
        return None

    return result.stdout


def get_cluster_resources():
    """Get all cluster resources."""
    return run_command(
        "kubectl get all -A --no-headers",
        "Fetching cluster resources"
    )


def get_beads_tasks():
    """Get all Beads tasks."""
    result = subprocess.run(
        ["bd", "list", "--status", "all", "--limit", "0", "--json"],
        capture_output=True,
        text=True,
        cwd="/root/vixens"
    )

    if result.returncode != 0:
        print(f"      ⚠️  Error fetching Beads: {result.stderr}")
        return None

    try:
        tasks = json.loads(result.stdout)
        return tasks
    except json.JSONDecodeError:
        print(f"      ⚠️  Invalid JSON from Beads")
        return None


def count_resources(resources_output):
    """Count resources by type."""
    if not resources_output:
        return {}

    counts = {}
    for line in resources_output.strip().split('\n'):
        if not line:
            continue

        parts = line.split()
        if len(parts) < 2:
            continue

        # Format: NAMESPACE TYPE/NAME ...
        resource_type = parts[1].split('/')[0]

        if resource_type not in counts:
            counts[resource_type] = 0
        counts[resource_type] += 1

    return counts


def count_beads_by_status(tasks):
    """Count Beads tasks by status."""
    if not tasks:
        return {}

    counts = {}
    for task in tasks:
        status = task.get('status', 'unknown')
        if status not in counts:
            counts[status] = 0
        counts[status] += 1

    return counts


def count_beads_by_assignee(tasks):
    """Count Beads tasks by assignee."""
    if not tasks:
        return {}

    counts = {}
    for task in tasks:
        assignee = task.get('assignee') or 'unassigned'
        if assignee not in counts:
            counts[assignee] = 0
        counts[assignee] += 1

    return counts


def generate_report(output_file):
    """Generate management report."""
    print("📊 Génération du rapport chefferie...")
    print("")

    # Fetch data
    resources = get_cluster_resources()
    tasks = get_beads_tasks()

    # Count metrics
    resource_counts = count_resources(resources)
    task_status_counts = count_beads_by_status(tasks)
    task_assignee_counts = count_beads_by_assignee(tasks)

    # Build report
    content = "# 📊 Rapport Chefferie de Projet\n\n"
    content += f"**Généré:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
    content += f"**Environnement:** {subprocess.run(['kubectl', 'config', 'current-context'], capture_output=True, text=True).stdout.strip()}\n\n"

    content += "---\n\n"

    # === CLUSTER OVERVIEW ===
    content += "## 🖥️ Vue d'ensemble Cluster\n\n"

    if resource_counts:
        content += "### Ressources Kubernetes\n\n"
        content += "| Type | Count |\n"
        content += "|------|-------|\n"

        # Sort by count (descending)
        sorted_resources = sorted(resource_counts.items(), key=lambda x: x[1], reverse=True)
        for resource_type, count in sorted_resources:
            content += f"| {resource_type} | {count} |\n"

        content += "\n"

    # === BEADS TASKS ===
    content += "## 📋 État des Tâches (Beads)\n\n"

    if task_status_counts:
        content += "### Par Statut\n\n"
        content += "| Statut | Count |\n"
        content += "|--------|-------|\n"

        status_order = ['open', 'in_progress', 'blocked', 'closed']
        for status in status_order:
            if status in task_status_counts:
                count = task_status_counts[status]
                emoji = {
                    'open': '⚪',
                    'in_progress': '🟡',
                    'blocked': '🔴',
                    'closed': '✅'
                }.get(status, '❓')
                content += f"| {emoji} {status} | {count} |\n"

        content += "\n"

    if task_assignee_counts:
        content += "### Par Assignee\n\n"
        content += "| Assignee | Count |\n"
        content += "|----------|-------|\n"

        sorted_assignees = sorted(task_assignee_counts.items(), key=lambda x: x[1], reverse=True)
        for assignee, count in sorted_assignees:
            content += f"| {assignee} | {count} |\n"

        content += "\n"

    # === DETAILED RESOURCES (Top 20) ===
    content += "## 🔍 Détails Ressources (Top 20)\n\n"

    if resources:
        lines = resources.strip().split('\n')[:20]
        content += "```\n"
        content += "NAMESPACE          TYPE/NAME                                    STATUS\n"
        content += "─" * 80 + "\n"

        for line in lines:
            parts = line.split()
            if len(parts) >= 3:
                ns = parts[0]
                resource = parts[1]
                status = ' '.join(parts[2:])
                content += f"{ns[:18]:<18} {resource[:44]:<44} {status[:12]:<12}\n"

        if len(resources.strip().split('\n')) > 20:
            remaining = len(resources.strip().split('\n')) - 20
            content += f"\n... et {remaining} ressources supplémentaires\n"

        content += "```\n\n"

    # === DETAILED TASKS (Open + In Progress) ===
    if tasks:
        active_tasks = [t for t in tasks if t.get('status') in ['open', 'in_progress']]

        if active_tasks:
            content += "## 📝 Tâches Actives\n\n"
            content += "| ID | Titre | Statut | Assignee | Priorité |\n"
            content += "|----|-------|--------|----------|----------|\n"

            for task in active_tasks[:20]:
                task_id = task.get('id', 'N/A')
                title = task.get('title', 'N/A')[:50]
                status = task.get('status', 'N/A')
                assignee = task.get('assignee') or 'unassigned'
                priority = task.get('priority', 'N/A')

                status_emoji = {
                    'open': '⚪',
                    'in_progress': '🟡',
                    'blocked': '🔴'
                }.get(status, '❓')

                content += f"| {task_id} | {title} | {status_emoji} {status} | {assignee} | P{priority} |\n"

            if len(active_tasks) > 20:
                content += f"\n*... et {len(active_tasks) - 20} tâches actives supplémentaires*\n"

            content += "\n"

    # === RAW DATA (APPENDIX) ===
    content += "---\n\n"
    content += "## 📎 Annexes\n\n"

    content += "### Beads Tasks (JSON)\n\n"
    content += "```json\n"
    content += json.dumps(tasks if tasks else [], indent=2)
    content += "\n```\n\n"

    # Write report
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"✅ Rapport chefferie généré: {output_file}")


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Generate management report for project leadership")
    parser.add_argument("--output", default="docs/reports/MANAGEMENT-REPORT.md", help="Output file")
    args = parser.parse_args()

    generate_report(args.output)


if __name__ == "__main__":
    main()
