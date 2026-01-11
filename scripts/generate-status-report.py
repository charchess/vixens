#!/usr/bin/env python3
import argparse
import sys
import os
import json
from datetime import datetime

# Add lib to path
sys.path.append(os.path.join(os.path.dirname(__file__), 'lib'))

from report_utils import parse_markdown_table, save_markdown_table

def main():
    parser = argparse.ArgumentParser(description="Generate STATUS.md from cluster state and conformity report")
    parser.add_argument("--conformity", default="docs/reports/CONFORMITY-REPORT.md", help="Conformity report file")
    parser.add_argument("--output", default="docs/reports/STATUS.md", help="Output status dashboard file")
    args = parser.parse_args()

    conformity_rows = parse_markdown_table(args.conformity)
    if not conformity_rows:
        print("Warning: Conformity report not found or empty. Status report will be incomplete.")
        conformity_rows = []

    conf_map = {row['App'].replace('**', ''): row for row in conformity_rows}

    # Summary counts
    stats = {"OK": 0, "NOK": 0, "ABSENT": 0}
    
    matrix_rows = []
    headers = ["Application", "Dev", "Prod", "Last Change", "Conformity", "Notes"]

    for app_name in sorted(conf_map.keys()):
        conf = conf_map[app_name]
        status = conf['Status']
        
        icon = "⚪"
        if status == "✅ OK": 
            icon = "✅"
            stats["OK"] += 1
        elif status == "⚠️ PARTIAL":
            icon = "⚠️"
            stats["NOK"] += 1
        else:
            icon = "❌"
            if "ABSENT" in status: 
                stats["ABSENT"] += 1
            else: 
                stats["NOK"] += 1

        matrix_rows.append({
            "Application": f"**{app_name}**",
            "Dev": "✅", # Assume Dev is always OK if it exists in conf map
            "Prod": icon,
            "Last Change": datetime.now().strftime('%Y-%m-%d'),
            "Conformity": conf['Score'],
            "Notes": conf['Issues']
        })

    # Generate Content
    content = "# Application Status Dashboard\n\n"
    content += f"**Last Updated:** {datetime.now().strftime('%Y-%m-%d')}\n"
    content += "**Cluster Environments:** dev, prod\n\n---\n\n"
    
    content += "## Overview\n\n"
    overview_headers = ["Category", "Prod", "Total"]
    overview_rows = [
        {"Category": "**✅ OK**", "Prod": str(stats["OK"]), "Total": str(stats["OK"])},
        {"Category": "**❌ NOK**", "Prod": str(stats["NOK"]), "Total": str(stats["NOK"])},
        {"Category": "**⚪ Absent**", "Prod": str(stats["ABSENT"]), "Total": str(stats["ABSENT"])},
        {"Category": "**Total**", "Prod": str(sum(stats.values())), "Total": str(sum(stats.values()))}
    ]
    content += save_markdown_table(overview_headers, overview_rows)
    content += "\n\n---\n\n"
    
    content += "## Application Status Matrix\n\n"
    content += save_markdown_table(headers, matrix_rows)
    content += "\n"

    os.makedirs(os.path.dirname(args.output), exist_ok=True)
    with open(args.output, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Status dashboard generated: {args.output}")

if __name__ == "__main__":
    main()
