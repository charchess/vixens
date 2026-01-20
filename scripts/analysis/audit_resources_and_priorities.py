import json
import subprocess
import os

def run_kubectl(cmd):
    try:
        result = subprocess.run(cmd, shell=True, check=True, stdout=subprocess.PIPE, text=True)
        return json.loads(result.stdout)
    except subprocess.CalledProcessError as e:
        print(f"Error running cmd: {cmd}")
        return {}

def parse_cpu(cpu_str):
    if not cpu_str or cpu_str == '0': return 0
    if str(cpu_str).endswith('m'):
        return int(cpu_str[:-1])
    try:
        return int(float(cpu_str) * 1000)
    except:
        return 0

def parse_mem(mem_str):
    if not mem_str or mem_str == '0': return 0
    units = {'Ki': 1024, 'Mi': 1024**2, 'Gi': 1024**3, 'Ti': 1024**4}
    for unit, mult in units.items():
        if str(mem_str).endswith(unit):
            return int(mem_str[:-len(unit)]) * mult
    try:
        return int(mem_str)
    except:
        return 0

def format_mem(bytes_val):
    if not isinstance(bytes_val, int): 
        try: bytes_val = int(bytes_val)
        except: return str(bytes_val)
    if bytes_val > 1024**3: return f"{bytes_val/1024**3:.1f}Gi"
    if bytes_val > 1024**2: return f"{bytes_val/1024**2:.0f}Mi"
    if bytes_val > 1024: return f"{bytes_val/1024:.0f}Ki"
    return f"{bytes_val}B"

def get_recommended_priority(ns, name):
    infra_ns = ['kube-system', 'argocd', 'cert-manager', 'monitoring', 'networking', 'synology-csi']
    critical_apps = ['authentik', 'postgresql-shared', 'redis-shared', 'traefik', 'cilium', 'vpa', 'goldilocks']
    low_apps = ['hydrus', 'jellyfin', 'plex', 'radarr', 'sonarr', 'lidarr', 'downloads', 'media', 'amule', 'pyload', 'qbittorrent']
    
    if ns in infra_ns or any(c in name for c in critical_apps):
        return "vixens-critical"
    if any(l in name or l in ns for l in low_apps):
        return "vixens-low"
    return "vixens-medium"

print("Collecting Pods...")
pods = run_kubectl("kubectl get pods -A -o json --kubeconfig terraform/environments/prod/kubeconfig-prod")

print("Collecting ArgoCD Apps...")
apps = run_kubectl("kubectl get applications -n argocd -o json --kubeconfig terraform/environments/prod/kubeconfig-prod")
app_health = {a['metadata']['name']: a.get('status', {}).get('health', {}).get('status', 'Unknown') for a in apps.get('items', [])}

print("Collecting VPAs...")
vpas = run_kubectl("kubectl get vpa -A -o json --kubeconfig terraform/environments/prod/kubeconfig-prod")

vpa_map = {}
if vpas:
    for vpa in vpas.get('items', []):
        ns = vpa['metadata']['namespace']
        target_name = vpa['spec']['targetRef']['name']
        recs = vpa.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
        vpa_map[f"{ns}/{target_name}"] = recs

data = []
processed_controllers = set()

for pod in pods.get('items', []):
    ns = pod['metadata']['namespace']
    pod_name = pod['metadata']['name']
    phase = pod['status']['phase']
    
    # Get restarts
    restarts = 0
    if 'containerStatuses' in pod['status']:
        restarts = sum(c['restartCount'] for c in pod['status']['containerStatuses'])

    owner = pod['metadata'].get('ownerReferences', [{}])[0]
    controller_name = owner.get('name', pod_name)
    controller_kind = owner.get('kind', 'Pod')
    
    if controller_kind == 'ReplicaSet':
        controller_name = '-'.join(controller_name.split('-')[:-1])
        controller_kind = 'Deployment'
    
    key = f"{ns}/{controller_kind}/{controller_name}"
    if key in processed_controllers: continue
    processed_controllers.add(key)

    priority = pod['spec'].get('priorityClassName', 'default (0)')
    
    # Logic for "App Status"
    argo_status = app_health.get(controller_name, app_health.get(controller_name.replace('-prod', ''), 'N/A'))
    if phase != 'Running' or argo_status in ['Degraded', 'Missing']:
        functional_status = "❌ Broken"
    elif restarts > 50:
        functional_status = f"⚠️ Unstable ({restarts} restarts)"
    else:
        functional_status = "✅ OK"

    for container in pod['spec']['containers']:
        c_name = container['name']
        req = container.get('resources', {}).get('requests', {})
        
        req_cpu = req.get('cpu', '0')
        req_mem = req.get('memory', '0')
        
        vpa_target_cpu = "N/A"
        vpa_target_mem = "N/A"
        
        vpa_recs = vpa_map.get(f"{ns}/{controller_name}", [])
        for r in vpa_recs:
            if r['containerName'] == c_name:
                vpa_target_cpu = r['target']['cpu']
                vpa_target_mem = format_mem(parse_mem(r['target']['memory']))
                break
        
        rec_prio = get_recommended_priority(ns, controller_name)

        data.append({
            "Namespace": ns,
            "App": controller_name,
            "Container": c_name,
            "Req CPU": req_cpu,
            "Req RAM": req_mem,
            "VPA CPU": vpa_target_cpu,
            "VPA RAM": vpa_target_mem,
            "Pod Phase": phase,
            "App Status": functional_status,
            "Current Prio": priority,
            "Rec. Prio": rec_prio
        })

data.sort(key=lambda x: (x['Namespace'], x['App']))

# Alignment logic
headers = ["Namespace", "App", "Container", "Req CPU", "Req RAM", "VPA CPU", "VPA RAM", "Pod Phase", "App Status", "Current Prio", "Rec. Prio"]
widths = {h: len(h) for h in headers}
for row in data:
    for h in headers:
        widths[h] = max(widths[h], len(str(row[h])))

def pad(text, width):
    return str(text).ljust(width)

# Generate Markdown
md = "# Resource & Priority Reference Map (Prod)\n\n"
md += "**Last Updated:** 2026-01-03\n\n"

header_row = "| " + " | ".join(pad(h, widths[h]) for h in headers) + " |\n"
separator_row = "|- " + "-|- ".join("-" * widths[h] for h in headers) + " -|\n"

md += header_row
md += separator_row

for row in data:
    md += "| " + " | ".join(pad(row[h], widths[h]) for h in headers) + " |\n"

with open("docs/reference/resources-and-priorities.md", "w") as f:
    f.write(md)

print("Pretty report generated at docs/reference/resources-and-priorities.md")