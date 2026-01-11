#!/usr/bin/env python3
import argparse
import sys
import os
import json
from datetime import datetime

# Add lib to path
sys.path.append(os.path.join(os.path.dirname(__file__), 'lib'))

from report_utils import (
    get_kubectl_json,
    format_cpu,
    format_memory,
    parse_cpu,
    parse_memory,
    save_markdown_table
)

def main():
    parser = argparse.ArgumentParser(description="Generate STATE-ACTUAL.md from cluster state")
    parser.add_argument("--env", default="dev", choices=["dev", "prod"], help="Environment to query")
    parser.add_argument("--dry-run", action="store_true", help="Preview without writing files")
    parser.add_argument("--output", default="docs/reports/STATE-ACTUAL.md", help="Output file path")
    args = parser.parse_args()

    # Set KUBECONFIG if not set or according to env
    if args.env == "dev":
        os.environ['KUBECONFIG'] = "terraform/environments/dev/kubeconfig-dev"
        # We also need the extra opts for the TLS handshake issue
        if 'KUBECONFIG_EXTRA_OPTS' not in os.environ:
            os.environ['KUBECONFIG_EXTRA_OPTS'] = "--server=https://192.168.111.160:6443 --insecure-skip-tls-verify"
    else:
        os.environ['KUBECONFIG'] = "terraform/environments/prod/kubeconfig-prod"

    print(f"Collecting data for {args.env} environment...")

    # 1. Get all applications from ArgoCD
    apps_data = get_kubectl_json(["get", "applications", "-n", "argocd"])
    if not apps_data:
        print("Error: Could not fetch ArgoCD applications")
        # Fallback to pods if ArgoCD not available? No, apps are central.
        sys.exit(1)

    # 2. Get all VPAs
    vpa_data = get_kubectl_json(["get", "vpa", "-A"]) or {"items": []}
    vpa_map = {}
    for item in vpa_data.get('items', []):
        ns = item['metadata']['namespace']
        target_name = item['spec']['targetRef']['name']
        recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
        for r in recs:
            vpa_map[f"{ns}/{target_name}/{r['containerName']}"] = {
                'cpu': r['target'].get('cpu', '0'),
                'mem': r['target'].get('memory', '0')
            }

    # 3. Get all pods to extract priority and current resources
    pods_data = get_kubectl_json(["get", "pods", "-A"]) or {"items": []}
    pod_info = {} # (ns, app_name) -> info
    for pod in pods_data.get('items', []):
        ns = pod['metadata']['namespace']
        # Try to find app name from labels
        labels = pod['metadata'].get('labels', {})
        # Use common app labels
        app_name = labels.get('app.kubernetes.io/instance') or labels.get('app.kubernetes.io/name') or labels.get('app') or pod['metadata'].get('generateName', '').split('-')[0]
        if not app_name: continue
        
        containers = pod['spec'].get('containers', [])
        # We take the first container for now or combine them
        main_container = containers[0] if containers else {}
        resources = main_container.get('resources', {})
        
        # If we have multiple pods, we might want to aggregate or just take one
        if f"{ns}/{app_name}" not in pod_info:
            pod_info[f"{ns}/{app_name}"] = {
                'priority': pod['spec'].get('priorityClassName', 'N/A'),
                'req_cpu': resources.get('requests', {}).get('cpu', 'N/A'),
                'req_mem': resources.get('requests', {}).get('memory', 'N/A'),
                'lim_cpu': resources.get('limits', {}).get('cpu', 'N/A'),
                'lim_mem': resources.get('limits', {}).get('memory', 'N/A'),
                'backup': 'None'
            }
        
        # Detect Litestream sidecar
        for c in containers:
            if 'litestream' in c.get('image', '').lower() or c.get('name') == 'litestream':
                pod_info[f"{ns}/{app_name}"]['backup'] = 'Standard'

    # 4. Process Applications
    rows = []
    headers = ["App", "NS", "CPU Req", "CPU Lim", "Mem Req", "Mem Lim", "VPA Target", "Profile", "Priority", "Sync Wave", "Backup Profile", "Score"]
    
    # Sort applications by name
    sorted_apps = sorted(apps_data.get('items', []), key=lambda x: x['metadata']['name'])

    for app in sorted_apps:
        name = app['metadata']['name']
        ns = app['spec']['destination']['namespace']
        
        info = pod_info.get(f"{ns}/{name}", {
            'priority': 'N/A',
            'req_cpu': 'N/A', 'req_mem': 'N/A',
            'lim_cpu': 'N/A', 'lim_mem': 'N/A',
            'backup': 'None'
        })
        
        # VPA Target
        vpa_target = "No VPA"
        vpa_key_prefix = f"{ns}/{name}"
        for k, v in vpa_map.items():
            if k.startswith(vpa_key_prefix):
                vpa_target = f"{v['cpu']} / {v['mem']}"
                break
        
        # Sync Wave from app annotations
        annotations = app['metadata'].get('annotations', {})
        sync_wave = annotations.get('argocd.argoproj.io/sync-wave', '0')
        
        # Profile detection
        mem_lim_bytes = parse_memory(info['lim_mem'])
        profile = "Unknown"
        if mem_lim_bytes > 0:
            if mem_lim_bytes <= 128 * 1024 * 1024:
                profile = "Micro"
            elif mem_lim_bytes <= 512 * 1024 * 1024:
                profile = "Small"
            elif mem_lim_bytes <= 2 * 1024 * 1024 * 1024:
                profile = "Medium"
            else:
                profile = "Large"

        row = {
            "App": f"**{name}**",
            "NS": ns,
            "CPU Req": info['req_cpu'],
            "CPU Lim": info['lim_cpu'],
            "Mem Req": info['req_mem'],
            "Mem Lim": info['lim_mem'],
            "VPA Target": vpa_target,
            "Profile": profile,
            "Priority": info['priority'],
            "Sync Wave": sync_wave,
            "Backup Profile": info['backup'],
            "Score": "0" 
        }
        rows.append(row)

    # Generate Markdown
    content = f"# Application State - Actual ({args.env.capitalize()} Reality)\n\n"
    content += f"**Last Updated:** {datetime.now().strftime('%Y-%m-%d')}\n"
    content += f"**Environment:** {args.env.capitalize()} Cluster\n"
    content += f"**Data Source:** Live cluster state + GitOps manifests\n\n---\n\n"
    content += "## Production Application Inventory\n\n"
    content += save_markdown_table(headers, rows)
    content += "\n"

    if args.dry_run:
        print("DRY RUN: Content generated:")
        print(content[:500] + "...")
    else:
        os.makedirs(os.path.dirname(args.output), exist_ok=True)
        with open(args.output, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"Report generated: {args.output}")

if __name__ == "__main__":
    main()
