#!/usr/bin/env python3
"""
VIXENS ACTUAL STATE GENERATOR (Emerald Shield v5.0)
Réimplémentation Python de vpa.sh avec génération MD + JSON.

Extrait:
- Node summary (capacité, OS, kernel)
- Application details avec VPA recommendations
- Scoring Elite (QoS, limits, priority, backup)
"""

import subprocess
import json
import sys
import os
from datetime import datetime

# Add lib to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'lib'))
from report_utils import save_markdown_table


def get_nodes():
    """Get node information."""
    result = subprocess.run(
        ["kubectl", "get", "nodes", "-o", "json"],
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"❌ Error fetching nodes: {result.stderr}")
        return []

    return json.loads(result.stdout).get('items', [])


def get_vpas():
    """Get all VPA recommendations."""
    result = subprocess.run(
        ["kubectl", "get", "vpa", "-A", "-o", "json"],
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"⚠️  VPA not available or error: {result.stderr}")
        return []

    return json.loads(result.stdout).get('items', [])


def get_pods():
    """Get all pods (excluding kube-system and Succeeded)."""
    result = subprocess.run(
        ["kubectl", "get", "pods", "-A", "-o", "json"],
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"❌ Error fetching pods: {result.stderr}")
        return []

    pods = json.loads(result.stdout).get('items', [])

    # Filter out kube-* namespaces and Succeeded pods
    filtered = [
        p for p in pods
        if not p.get('metadata', {}).get('namespace', '').startswith('kube-')
        and p.get('status', {}).get('phase') != 'Succeeded'
    ]

    return filtered


def extract_app_name(pod):
    """Extract app name from pod labels."""
    labels = pod.get('metadata', {}).get('labels', {})

    # Try various label keys
    for key in ['app.kubernetes.io/name', 'app', 'name']:
        if key in labels:
            return labels[key]

    # Fallback to pod name
    return pod.get('metadata', {}).get('name', 'unknown')


def get_vpa_recommendation(app_name, vpas):
    """Find VPA recommendation for an app."""
    for vpa in vpas:
        vpa_name = vpa.get('metadata', {}).get('name', '')
        target_ref = vpa.get('spec', {}).get('targetRef', {}).get('name', '')

        if vpa_name == app_name or target_ref == app_name:
            return vpa.get('status', {}).get('recommendation', {})

    return None


def format_memory(mem_str):
    """Convert memory string to Mi format."""
    if not mem_str or mem_str == "0":
        return "0Mi"

    # Already in Mi/Gi format
    if mem_str.endswith('Mi') or mem_str.endswith('Gi'):
        return mem_str

    # Ki format
    if mem_str.endswith('Ki'):
        ki = int(mem_str[:-2])
        return f"{ki // 1024}Mi"

    # Bytes (no suffix)
    if mem_str.isdigit():
        bytes_val = int(mem_str)
        return f"{bytes_val // 1048576}Mi"

    return mem_str


def calculate_elite_score(pod_data):
    """Calculate Elite score (0-100) based on best practices."""
    score = 0

    # QoS Class (40 points max)
    qos = pod_data.get('qos', 'BestEffort')
    if qos == 'Guaranteed':
        score += 40
    elif qos == 'Burstable':
        score += 15

    # Resource limits defined (20 points)
    if pod_data.get('cpu_lim') != 'N/A' and pod_data.get('mem_lim') != 'N/A':
        score += 20

    # Priority class (20 points)
    prio = pod_data.get('prio', 'N/A')
    if prio != 'N/A' and ('vixens' in prio or 'homelab' in prio):
        score += 20

    # Backup active (20 points)
    if pod_data.get('backup') == 'Active':
        score += 20

    return score


def process_pods(pods, vpas):
    """Process pods and extract application data."""
    apps_data = {}

    for pod in pods:
        metadata = pod.get('metadata', {})
        spec = pod.get('spec', {})
        status = pod.get('status', {})

        ns = metadata.get('namespace', 'default')
        app_name = extract_app_name(pod)
        key = f"{ns}/{app_name}"

        # Skip if already processed (deduplication)
        if key in apps_data:
            continue

        # Get first container
        containers = spec.get('containers', [])
        if not containers:
            continue

        container = containers[0]
        resources = container.get('resources', {})
        requests = resources.get('requests', {})
        limits = resources.get('limits', {})

        # Extract resource values
        cpu_req = requests.get('cpu', 'N/A')
        cpu_lim = limits.get('cpu', 'N/A')
        mem_req = requests.get('memory', 'N/A')
        mem_lim = limits.get('memory', 'N/A')

        # Priority class
        prio = spec.get('priorityClassName', 'N/A')

        # QoS class
        qos = status.get('qosClass', 'BestEffort')

        # Sync wave (from annotations)
        annotations = metadata.get('annotations', {})
        wave = annotations.get('argocd.argoproj.io/sync-wave', '0')

        # Backup (check for litestream sidecar)
        has_litestream = any('litestream' in c.get('name', '') for c in containers)
        backup = 'Active' if has_litestream else 'None'

        # VPA recommendation
        vpa_rec = get_vpa_recommendation(app_name, vpas)
        vpa_target = "None"
        if vpa_rec:
            container_recs = vpa_rec.get('containerRecommendations', [])
            if container_recs:
                target = container_recs[0].get('target', {})
                cpu_target = target.get('cpu', '0')
                mem_target = format_memory(target.get('memory', '0'))
                vpa_target = f"{cpu_target} / {mem_target}"

        # Build app data
        pod_data = {
            'ns': ns,
            'app': app_name,
            'cpu_req': cpu_req,
            'cpu_lim': cpu_lim,
            'mem_req': mem_req,
            'mem_lim': mem_lim,
            'vpa': vpa_target,
            'prio': prio,
            'wave': wave,
            'backup': backup,
            'qos': qos
        }

        # Calculate Elite score
        pod_data['score'] = calculate_elite_score(pod_data)

        apps_data[key] = pod_data

    # Sort by namespace + app name
    sorted_apps = sorted(apps_data.values(), key=lambda x: (x['ns'], x['app']))

    return sorted_apps


def generate_node_table(nodes):
    """Generate node summary table."""
    rows = []

    for node in nodes:
        metadata = node.get('metadata', {})
        status = node.get('status', {})
        labels = metadata.get('labels', {})

        name = metadata.get('name', 'unknown')

        # Determine role
        if 'node-role.kubernetes.io/control-plane' in labels or 'node-role.kubernetes.io/master' in labels:
            role = 'control-plane'
        else:
            role = 'worker'

        # Capacity
        capacity = status.get('capacity', {})
        cpu_cap = capacity.get('cpu', 'N/A')
        mem_cap = capacity.get('memory', 'N/A')

        # Node info
        node_info = status.get('nodeInfo', {})
        os_image = node_info.get('osImage', 'N/A')
        kernel = node_info.get('kernelVersion', 'N/A')

        rows.append({
            'Node Name': name,
            'Role': role,
            'CPU Cap': cpu_cap,
            'RAM Cap': mem_cap,
            'OS': os_image,
            'Kernel': kernel
        })

    return rows


def generate_app_table(apps):
    """Generate application details table."""
    rows = []

    for app in apps:
        rows.append({
            'App': f"**{app['app']}**",
            'NS': app['ns'],
            'CPU Req': app['cpu_req'],
            'CPU Lim': app['cpu_lim'],
            'Mem Req': app['mem_req'],
            'Mem Lim': app['mem_lim'],
            'VPA Target': app['vpa'],
            'Priority': app['prio'],
            'Wave': app['wave'],
            'Backup': app['backup'],
            'QoS': app['qos'],
            'Score': str(app['score'])
        })

    return rows


def save_markdown_report(nodes, apps, output_file):
    """Save markdown report."""
    content = f"# 📊 État Réel du Cluster - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"

    # Node summary
    content += "## 🖥️ Node Summary\n\n"
    node_rows = generate_node_table(nodes)
    node_headers = ['Node Name', 'Role', 'CPU Cap', 'RAM Cap', 'OS', 'Kernel']
    content += save_markdown_table(node_headers, node_rows)
    content += "\n\n"

    # Application details
    content += "## 📦 Application Details\n\n"
    app_rows = generate_app_table(apps)
    app_headers = ['App', 'NS', 'CPU Req', 'CPU Lim', 'Mem Req', 'Mem Lim',
                   'VPA Target', 'Priority', 'Wave', 'Backup', 'QoS', 'Score']
    content += save_markdown_table(app_headers, app_rows)
    content += "\n"

    # Write file
    os.makedirs(os.path.dirname(output_file), exist_ok=True)
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)


def save_json_report(nodes, apps, output_file):
    """Save JSON report for processing by other tools."""
    # Build JSON structure compatible with generate_status_report.py
    apps_dict = {}

    for app in apps:
        app_name = app['app']
        apps_dict[app_name] = {
            'namespace': app['ns'],
            'sync': 'Synced',  # Placeholder - would need ArgoCD query
            'health': 'Healthy',  # Placeholder - would need ArgoCD query
            'cpu_req': app['cpu_req'],
            'cpu_lim': app['cpu_lim'],
            'mem_req': app['mem_req'],
            'mem_lim': app['mem_lim'],
            'priority': app['prio'],
            'wave': app['wave'],
            'backup': app['backup'],
            'qos': app['qos'],
            'score': app['score'],
            'vpa': app['vpa']
        }

    # Write JSON
    os.makedirs(os.path.dirname(output_file), exist_ok=True)
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(apps_dict, f, indent=2)


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Generate cluster state with VPA (like vpa.sh)")
    parser.add_argument("--env", default="prod", help="Environment (dev/prod)")
    parser.add_argument("--output", default="docs/reports/STATE-ACTUAL.md", help="Output markdown file")
    parser.add_argument("--json-output", help="Output JSON file")
    args = parser.parse_args()

    print("🔍 Selene scanne le chaos avec Python... Et garde tes pattes loin de moi, Charchess !")

    # Fetch data
    nodes = get_nodes()
    vpas = get_vpas()
    pods = get_pods()

    print(f"   ✅ Found {len(nodes)} nodes")
    print(f"   ✅ Found {len(vpas)} VPAs")
    print(f"   ✅ Found {len(pods)} pods (filtered)")

    # Process pods
    apps = process_pods(pods, vpas)
    print(f"   ✅ Processed {len(apps)} unique applications")

    # Generate reports
    save_markdown_report(nodes, apps, args.output)
    print(f"✅ Markdown report: {args.output}")

    if args.json_output:
        save_json_report(nodes, apps, args.json_output)
        print(f"✅ JSON report: {args.json_output}")


if __name__ == "__main__":
    main()
